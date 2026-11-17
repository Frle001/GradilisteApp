package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

var (
	ErrWarehouseNotFound        = errors.New("warehouse item not found")
	ErrWarehouseInsufficientQty = errors.New("Nedovoljna količina u skladištu.")
	ErrWarehouseAccessDenied    = errors.New("nemate pristup ovoj funkciji")
	ErrIdempotencyConflict      = errors.New("isti ključ je već korišten s različitim podacima")
)

// payloadHash returns a hex SHA-256 of the JSON-serialised request struct.
func payloadHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// ── Interfaces ────────────────────────────────────────────────────────────────

type warehouseRepoIface interface {
	GetOverview(ctx context.Context, companyID string) (*dto.WarehouseOverview, error)
	ListStock(ctx context.Context, companyID, search string, activeOnly bool) ([]dto.WarehouseStockItem, error)
	GetStockByID(ctx context.Context, companyID, balanceID string) (*dto.WarehouseStockDetail, error)
	FindOrCreateBalance(ctx context.Context, tx pgx.Tx, companyID, materialName, unit string, materialCode *string) (*repositories.BalanceSnapshot, error)
	LockBalance(ctx context.Context, tx pgx.Tx, companyID, balanceID string) (*repositories.BalanceSnapshot, error)
	UpdateBalance(ctx context.Context, tx pgx.Tx, balanceID string, newQty, newAvgCost float64) error
	InsertReceipt(ctx context.Context, tx pgx.Tx, companyID, createdBy string, receiptNumber, supplier, notes *string, receiptDate time.Time) (string, error)
	InsertReceiptLine(ctx context.Context, tx pgx.Tx, receiptID, balanceID string, qty, unitPrice float64) (string, error)
	InsertMovement(ctx context.Context, tx pgx.Tx, p repositories.InsertMovementParams) (string, error)
	CountMovements(ctx context.Context, companyID string, movementType, balanceID *string) (int, error)
	ListMovements(ctx context.Context, companyID string, movementType, balanceID *string, limit, offset int) ([]dto.WarehouseMovementItem, error)
	IsProjectAssigned(ctx context.Context, companyID, projectID, employeeID string) (bool, error)
	TryInsertIdempotencyKey(ctx context.Context, tx pgx.Tx, companyID, idempotencyKey, opType, payloadHash string) (*repositories.IdempotencyRecord, bool, error)
	UpdateIdempotencyResult(ctx context.Context, tx pgx.Tx, companyID, idempotencyKey, resultID string) error
}

// projectStockIface covers the project_materials operations needed for
// atomic warehouse↔project transfers. Satisfied by *repositories.InventoryRepository.
type projectStockIface interface {
	FindOrCreateDestProjectMaterial(ctx context.Context, tx pgx.Tx, companyID, destProjectID, materialName string, materialCode *string, unit string) (string, error)
	DeductProjectMaterialAvailable(ctx context.Context, tx pgx.Tx, companyID, projectMaterialID string, qty float64) error
	AddProjectMaterialAvailable(ctx context.Context, tx pgx.Tx, companyID, projectMaterialID string, qty float64) error
}

// ── Service ───────────────────────────────────────────────────────────────────

type WarehouseService struct {
	db      txBeginner
	repo    warehouseRepoIface
	matRepo projectStockIface
}

func NewWarehouseService(db *pgxpool.Pool, repo *repositories.WarehouseRepository, matRepo *repositories.InventoryRepository) *WarehouseService {
	return &WarehouseService{db: db, repo: repo, matRepo: matRepo}
}

// ── Role checks ───────────────────────────────────────────────────────────────

// canRead: all roles can view warehouse stock
func canReadWarehouse(role string) bool {
	switch role {
	case "direktor", "inzenjer", "administracija", "poslovoda", "radnik":
		return true
	}
	return false
}

// canWrite: only management roles can create receipts, issues, returns, adjustments
func canWriteWarehouse(role string) bool {
	switch role {
	case "direktor", "inzenjer", "administracija":
		return true
	}
	return false
}

// canIssue: direktor, inzenjer, administracija, poslovoda may issue/return
func canIssueWarehouse(role string) bool {
	switch role {
	case "direktor", "inzenjer", "administracija", "poslovoda":
		return true
	}
	return false
}

// ── Overview ──────────────────────────────────────────────────────────────────

func (s *WarehouseService) GetOverview(ctx context.Context, companyID, role string) (*dto.WarehouseOverview, error) {
	if !canReadWarehouse(role) {
		return nil, ErrWarehouseAccessDenied
	}
	return s.repo.GetOverview(ctx, companyID)
}

// ── Stock ─────────────────────────────────────────────────────────────────────

func (s *WarehouseService) ListStock(ctx context.Context, companyID, role, search string, activeOnly bool) ([]dto.WarehouseStockItem, error) {
	if !canReadWarehouse(role) {
		return nil, ErrWarehouseAccessDenied
	}
	items, err := s.repo.ListStock(ctx, companyID, search, activeOnly)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []dto.WarehouseStockItem{}
	}
	return items, nil
}

func (s *WarehouseService) GetStockByID(ctx context.Context, companyID, role, balanceID string) (*dto.WarehouseStockDetail, error) {
	if !canReadWarehouse(role) {
		return nil, ErrWarehouseAccessDenied
	}
	detail, err := s.repo.GetStockByID(ctx, companyID, balanceID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrWarehouseNotFound
	}
	return detail, err
}

// ── Receipt (incoming goods) ──────────────────────────────────────────────────

func (s *WarehouseService) CreateReceipt(ctx context.Context, companyID, role, userID, idempotencyKey string, req dto.CreateReceiptRequest) (*dto.ReceiptResponse, bool, error) {
	if !canWriteWarehouse(role) {
		return nil, false, ErrWarehouseAccessDenied
	}

	receiptDate := time.Now().UTC()
	if req.ReceiptDate != nil && *req.ReceiptDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.ReceiptDate)
		if err != nil {
			return nil, false, fmt.Errorf("nevaljan format datuma (koristite YYYY-MM-DD): %w", err)
		}
		receiptDate = parsed
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if idempotencyKey != "" {
		hash := payloadHash(req)
		rec, isNew, err := s.repo.TryInsertIdempotencyKey(ctx, tx, companyID, idempotencyKey, "receipt", hash)
		if err != nil {
			return nil, false, fmt.Errorf("idempotency: %w", err)
		}
		if !isNew {
			if rec.PayloadHash != hash {
				return nil, false, ErrIdempotencyConflict
			}
			if rec.ResultID != nil {
				return &dto.ReceiptResponse{ID: *rec.ResultID}, true, nil
			}
			return nil, false, fmt.Errorf("idempotency: original result unavailable")
		}
	}

	receiptID, err := s.repo.InsertReceipt(ctx, tx, companyID, userID, req.ReceiptNumber, req.Supplier, req.Notes, receiptDate)
	if err != nil {
		return nil, false, fmt.Errorf("insert receipt: %w", err)
	}

	var totalValue float64
	for _, line := range req.Lines {
		snap, err := s.repo.FindOrCreateBalance(ctx, tx, companyID, line.MaterialName, line.Unit, line.MaterialCode)
		if err != nil {
			return nil, false, fmt.Errorf("find/create balance for %q: %w", line.MaterialName, err)
		}

		// Moving weighted-average cost: new_avg = (old_qty × old_avg + in_qty × in_price) / (old_qty + in_qty)
		newQty := snap.Quantity + line.Quantity
		newAvgCost := snap.AvgCost
		if newQty > 0 {
			newAvgCost = (snap.Quantity*snap.AvgCost + line.Quantity*line.UnitPrice) / newQty
		}

		if err := s.repo.UpdateBalance(ctx, tx, snap.ID, newQty, newAvgCost); err != nil {
			return nil, false, fmt.Errorf("update balance: %w", err)
		}

		lineID, err := s.repo.InsertReceiptLine(ctx, tx, receiptID, snap.ID, line.Quantity, line.UnitPrice)
		if err != nil {
			return nil, false, fmt.Errorf("insert receipt line: %w", err)
		}

		if _, err := s.repo.InsertMovement(ctx, tx, repositories.InsertMovementParams{
			CompanyID:      companyID,
			BalanceID:      snap.ID,
			MovementType:   "receipt",
			QuantityDelta:  line.Quantity,
			QuantityBefore: snap.Quantity,
			QuantityAfter:  newQty,
			UnitPrice:      line.UnitPrice,
			AvgCostBefore:  snap.AvgCost,
			AvgCostAfter:   newAvgCost,
			ReceiptLineID:  &lineID,
			Notes:          req.Notes,
			CreatedBy:      userID,
		}); err != nil {
			return nil, false, fmt.Errorf("insert movement: %w", err)
		}

		totalValue += line.Quantity * line.UnitPrice
	}

	if idempotencyKey != "" {
		if err := s.repo.UpdateIdempotencyResult(ctx, tx, companyID, idempotencyKey, receiptID); err != nil {
			return nil, false, fmt.Errorf("update idempotency: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit tx: %w", err)
	}

	return &dto.ReceiptResponse{
		ID:            receiptID,
		ReceiptNumber: req.ReceiptNumber,
		Supplier:      req.Supplier,
		ReceiptDate:   receiptDate.Format("2006-01-02"),
		Notes:         req.Notes,
		LineCount:     len(req.Lines),
		TotalValue:    totalValue,
		CreatedAt:     time.Now().UTC(),
	}, false, nil
}

// ── Issue (warehouse → project) ────────────────────────────────────────────────

func (s *WarehouseService) IssueToProject(ctx context.Context, companyID, role, userID, employeeID, idempotencyKey string, req dto.CreateIssueRequest) (string, bool, error) {
	if !canIssueWarehouse(role) {
		return "", false, ErrWarehouseAccessDenied
	}

	if role == "poslovoda" {
		if employeeID == "" {
			return "", false, ErrWarehouseAccessDenied
		}
		assigned, err := s.repo.IsProjectAssigned(ctx, companyID, req.ProjectID, employeeID)
		if err != nil {
			return "", false, fmt.Errorf("check project assignment: %w", err)
		}
		if !assigned {
			return "", false, ErrWarehouseAccessDenied
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if idempotencyKey != "" {
		hash := payloadHash(req)
		rec, isNew, err := s.repo.TryInsertIdempotencyKey(ctx, tx, companyID, idempotencyKey, "issue", hash)
		if err != nil {
			return "", false, fmt.Errorf("idempotency: %w", err)
		}
		if !isNew {
			if rec.PayloadHash != hash {
				return "", false, ErrIdempotencyConflict
			}
			if rec.ResultID != nil {
				return *rec.ResultID, true, nil
			}
			return "", false, fmt.Errorf("idempotency: original result unavailable")
		}
	}

	snap, err := s.repo.LockBalance(ctx, tx, companyID, req.BalanceID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", false, ErrWarehouseNotFound
		}
		return "", false, fmt.Errorf("lock balance: %w", err)
	}

	if snap.Quantity < req.Quantity {
		return "", false, ErrWarehouseInsufficientQty
	}

	newQty := snap.Quantity - req.Quantity

	if err := s.repo.UpdateBalance(ctx, tx, snap.ID, newQty, snap.AvgCost); err != nil {
		return "", false, fmt.Errorf("update balance: %w", err)
	}

	projMatID, err := s.matRepo.FindOrCreateDestProjectMaterial(ctx, tx, companyID, req.ProjectID, snap.MaterialName, snap.MaterialCode, snap.Unit)
	if err != nil {
		return "", false, fmt.Errorf("find/create project material: %w", err)
	}

	if err := s.matRepo.AddProjectMaterialAvailable(ctx, tx, companyID, projMatID, req.Quantity); err != nil {
		return "", false, fmt.Errorf("add project material available: %w", err)
	}

	projIDPtr := &req.ProjectID
	movID, err := s.repo.InsertMovement(ctx, tx, repositories.InsertMovementParams{
		CompanyID:      companyID,
		BalanceID:      snap.ID,
		MovementType:   "issue",
		QuantityDelta:  -req.Quantity,
		QuantityBefore: snap.Quantity,
		QuantityAfter:  newQty,
		UnitPrice:      snap.AvgCost,
		AvgCostBefore:  snap.AvgCost,
		AvgCostAfter:   snap.AvgCost,
		ProjectID:      projIDPtr,
		ProjectMatID:   &projMatID,
		Notes:          req.Notes,
		CreatedBy:      userID,
	})
	if err != nil {
		return "", false, fmt.Errorf("insert movement: %w", err)
	}

	if idempotencyKey != "" {
		if err := s.repo.UpdateIdempotencyResult(ctx, tx, companyID, idempotencyKey, movID); err != nil {
			return "", false, fmt.Errorf("update idempotency: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("commit tx: %w", err)
	}
	return movID, false, nil
}

// ── Return (project → warehouse) ──────────────────────────────────────────────

func (s *WarehouseService) ReturnFromProject(ctx context.Context, companyID, role, userID, employeeID, idempotencyKey string, req dto.CreateReturnRequest) (string, bool, error) {
	if !canIssueWarehouse(role) {
		return "", false, ErrWarehouseAccessDenied
	}

	if role == "poslovoda" {
		if employeeID == "" {
			return "", false, ErrWarehouseAccessDenied
		}
		assigned, err := s.repo.IsProjectAssigned(ctx, companyID, req.ProjectID, employeeID)
		if err != nil {
			return "", false, fmt.Errorf("check project assignment: %w", err)
		}
		if !assigned {
			return "", false, ErrWarehouseAccessDenied
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if idempotencyKey != "" {
		hash := payloadHash(req)
		rec, isNew, err := s.repo.TryInsertIdempotencyKey(ctx, tx, companyID, idempotencyKey, "return", hash)
		if err != nil {
			return "", false, fmt.Errorf("idempotency: %w", err)
		}
		if !isNew {
			if rec.PayloadHash != hash {
				return "", false, ErrIdempotencyConflict
			}
			if rec.ResultID != nil {
				return *rec.ResultID, true, nil
			}
			return "", false, fmt.Errorf("idempotency: original result unavailable")
		}
	}

	// Deduct from project material first (guarded against going negative).
	if err := s.matRepo.DeductProjectMaterialAvailable(ctx, tx, companyID, req.ProjectMaterialID, req.Quantity); err != nil {
		if errors.Is(err, repositories.ErrInsufficientQuantity) {
			return "", false, ErrWarehouseInsufficientQty
		}
		return "", false, fmt.Errorf("deduct project material: %w", err)
	}

	snap, err := s.repo.LockBalance(ctx, tx, companyID, req.BalanceID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", false, ErrWarehouseNotFound
		}
		return "", false, fmt.Errorf("lock balance: %w", err)
	}

	// Returned goods are valued at the current avg_cost (no revaluation on return).
	newQty := snap.Quantity + req.Quantity
	newAvgCost := snap.AvgCost

	if err := s.repo.UpdateBalance(ctx, tx, snap.ID, newQty, newAvgCost); err != nil {
		return "", false, fmt.Errorf("update balance: %w", err)
	}

	projIDPtr := &req.ProjectID
	projMatPtr := &req.ProjectMaterialID
	movID, err := s.repo.InsertMovement(ctx, tx, repositories.InsertMovementParams{
		CompanyID:      companyID,
		BalanceID:      snap.ID,
		MovementType:   "return",
		QuantityDelta:  req.Quantity,
		QuantityBefore: snap.Quantity,
		QuantityAfter:  newQty,
		UnitPrice:      newAvgCost,
		AvgCostBefore:  snap.AvgCost,
		AvgCostAfter:   newAvgCost,
		ProjectID:      projIDPtr,
		ProjectMatID:   projMatPtr,
		Notes:          req.Notes,
		CreatedBy:      userID,
	})
	if err != nil {
		return "", false, fmt.Errorf("insert movement: %w", err)
	}

	if idempotencyKey != "" {
		if err := s.repo.UpdateIdempotencyResult(ctx, tx, companyID, idempotencyKey, movID); err != nil {
			return "", false, fmt.Errorf("update idempotency: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("commit tx: %w", err)
	}
	return movID, false, nil
}

// ── Adjustment (inventory count) ──────────────────────────────────────────────

func (s *WarehouseService) AdjustInventory(ctx context.Context, companyID, role, userID, idempotencyKey string, req dto.CreateAdjustmentRequest) (string, bool, error) {
	if !canWriteWarehouse(role) {
		return "", false, ErrWarehouseAccessDenied
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if idempotencyKey != "" {
		hash := payloadHash(req)
		rec, isNew, err := s.repo.TryInsertIdempotencyKey(ctx, tx, companyID, idempotencyKey, "adjustment", hash)
		if err != nil {
			return "", false, fmt.Errorf("idempotency: %w", err)
		}
		if !isNew {
			if rec.PayloadHash != hash {
				return "", false, ErrIdempotencyConflict
			}
			if rec.ResultID != nil {
				return *rec.ResultID, true, nil
			}
			return "", false, fmt.Errorf("idempotency: original result unavailable")
		}
	}

	snap, err := s.repo.LockBalance(ctx, tx, companyID, req.BalanceID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", false, ErrWarehouseNotFound
		}
		return "", false, fmt.Errorf("lock balance: %w", err)
	}

	delta := req.NewQuantity - snap.Quantity
	if err := s.repo.UpdateBalance(ctx, tx, snap.ID, req.NewQuantity, snap.AvgCost); err != nil {
		return "", false, fmt.Errorf("update balance: %w", err)
	}

	movID, err := s.repo.InsertMovement(ctx, tx, repositories.InsertMovementParams{
		CompanyID:      companyID,
		BalanceID:      snap.ID,
		MovementType:   "adjustment",
		QuantityDelta:  delta,
		QuantityBefore: snap.Quantity,
		QuantityAfter:  req.NewQuantity,
		UnitPrice:      snap.AvgCost,
		AvgCostBefore:  snap.AvgCost,
		AvgCostAfter:   snap.AvgCost,
		Notes:          req.Notes,
		CreatedBy:      userID,
	})
	if err != nil {
		return "", false, fmt.Errorf("insert movement: %w", err)
	}

	if idempotencyKey != "" {
		if err := s.repo.UpdateIdempotencyResult(ctx, tx, companyID, idempotencyKey, movID); err != nil {
			return "", false, fmt.Errorf("update idempotency: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("commit tx: %w", err)
	}
	return movID, false, nil
}

// ── Movement list ─────────────────────────────────────────────────────────────

func (s *WarehouseService) ListMovements(ctx context.Context, companyID, role string, movementType, balanceID *string, page, perPage int) (*dto.WarehouseMovementListResponse, error) {
	if !canReadWarehouse(role) {
		return nil, ErrWarehouseAccessDenied
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	if movementType != nil {
		mt := strings.TrimSpace(*movementType)
		movementType = &mt
	}

	total, err := s.repo.CountMovements(ctx, companyID, movementType, balanceID)
	if err != nil {
		return nil, fmt.Errorf("count movements: %w", err)
	}

	totalPages := (total + perPage - 1) / perPage
	offset := (page - 1) * perPage

	items, err := s.repo.ListMovements(ctx, companyID, movementType, balanceID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	if items == nil {
		items = []dto.WarehouseMovementItem{}
	}

	return &dto.WarehouseMovementListResponse{
		Data: items,
		Pagination: dto.WarehousePagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
