package services

import (
	"context"
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
)

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

// canIssue: direktor, inzenjer, poslovoda may issue from warehouse to a project
func canIssueWarehouse(role string) bool {
	switch role {
	case "direktor", "inzenjer", "poslovoda":
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

func (s *WarehouseService) CreateReceipt(ctx context.Context, companyID, role, userID string, req dto.CreateReceiptRequest) (*dto.ReceiptResponse, error) {
	if !canWriteWarehouse(role) {
		return nil, ErrWarehouseAccessDenied
	}

	receiptDate := time.Now().UTC()
	if req.ReceiptDate != nil && *req.ReceiptDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.ReceiptDate)
		if err != nil {
			return nil, fmt.Errorf("nevaljan format datuma (koristite YYYY-MM-DD): %w", err)
		}
		receiptDate = parsed
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	receiptID, err := s.repo.InsertReceipt(ctx, tx, companyID, userID, req.ReceiptNumber, req.Supplier, req.Notes, receiptDate)
	if err != nil {
		return nil, fmt.Errorf("insert receipt: %w", err)
	}

	var totalValue float64
	for _, line := range req.Lines {
		snap, err := s.repo.FindOrCreateBalance(ctx, tx, companyID, line.MaterialName, line.Unit, line.MaterialCode)
		if err != nil {
			return nil, fmt.Errorf("find/create balance for %q: %w", line.MaterialName, err)
		}

		// Moving weighted-average cost: new_avg = (old_qty × old_avg + in_qty × in_price) / (old_qty + in_qty)
		newQty := snap.Quantity + line.Quantity
		newAvgCost := snap.AvgCost
		if newQty > 0 {
			newAvgCost = (snap.Quantity*snap.AvgCost + line.Quantity*line.UnitPrice) / newQty
		}

		if err := s.repo.UpdateBalance(ctx, tx, snap.ID, newQty, newAvgCost); err != nil {
			return nil, fmt.Errorf("update balance: %w", err)
		}

		lineID, err := s.repo.InsertReceiptLine(ctx, tx, receiptID, snap.ID, line.Quantity, line.UnitPrice)
		if err != nil {
			return nil, fmt.Errorf("insert receipt line: %w", err)
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
			return nil, fmt.Errorf("insert movement: %w", err)
		}

		totalValue += line.Quantity * line.UnitPrice
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
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
	}, nil
}

// ── Issue (warehouse → project) ────────────────────────────────────────────────

func (s *WarehouseService) IssueToProject(ctx context.Context, companyID, role, userID string, req dto.CreateIssueRequest) (string, error) {
	if !canIssueWarehouse(role) {
		return "", ErrWarehouseAccessDenied
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	snap, err := s.repo.LockBalance(ctx, tx, companyID, req.BalanceID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", ErrWarehouseNotFound
		}
		return "", fmt.Errorf("lock balance: %w", err)
	}

	if snap.Quantity < req.Quantity {
		return "", ErrWarehouseInsufficientQty
	}

	newQty := snap.Quantity - req.Quantity

	if err := s.repo.UpdateBalance(ctx, tx, snap.ID, newQty, snap.AvgCost); err != nil {
		return "", fmt.Errorf("update balance: %w", err)
	}

	// Ensure the destination project has a matching project_materials row.
	// We need the material name and unit from the warehouse balance — read them.
	var matName, unit string
	var matCode *string
	err = tx.QueryRow(ctx,
		`SELECT material_name, material_code, unit FROM warehouse_balances WHERE id = $1::uuid`,
		snap.ID,
	).Scan(&matName, &matCode, &unit)
	if err != nil {
		return "", fmt.Errorf("read balance metadata: %w", err)
	}

	projMatID, err := s.matRepo.FindOrCreateDestProjectMaterial(ctx, tx, companyID, req.ProjectID, matName, matCode, unit)
	if err != nil {
		return "", fmt.Errorf("find/create project material: %w", err)
	}

	if err := s.matRepo.AddProjectMaterialAvailable(ctx, tx, companyID, projMatID, req.Quantity); err != nil {
		return "", fmt.Errorf("add project material available: %w", err)
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
		return "", fmt.Errorf("insert movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return movID, nil
}

// ── Return (project → warehouse) ──────────────────────────────────────────────

func (s *WarehouseService) ReturnFromProject(ctx context.Context, companyID, role, userID string, req dto.CreateReturnRequest) (string, error) {
	if !canIssueWarehouse(role) {
		return "", ErrWarehouseAccessDenied
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Deduct from project material first (guarded against going negative).
	if err := s.matRepo.DeductProjectMaterialAvailable(ctx, tx, companyID, req.ProjectMaterialID, req.Quantity); err != nil {
		if errors.Is(err, repositories.ErrInsufficientQuantity) {
			return "", ErrWarehouseInsufficientQty
		}
		return "", fmt.Errorf("deduct project material: %w", err)
	}

	snap, err := s.repo.LockBalance(ctx, tx, companyID, req.BalanceID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", ErrWarehouseNotFound
		}
		return "", fmt.Errorf("lock balance: %w", err)
	}

	// Returned goods are valued at current avg_cost (no price change on return).
	newQty := snap.Quantity + req.Quantity
	newAvgCost := snap.AvgCost
	if newQty > 0 && snap.Quantity == 0 {
		// Balance was empty; keep existing avg_cost as the return valuation.
		newAvgCost = snap.AvgCost
	}

	if err := s.repo.UpdateBalance(ctx, tx, snap.ID, newQty, newAvgCost); err != nil {
		return "", fmt.Errorf("update balance: %w", err)
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
		return "", fmt.Errorf("insert movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return movID, nil
}

// ── Adjustment (inventory count) ──────────────────────────────────────────────

func (s *WarehouseService) AdjustInventory(ctx context.Context, companyID, role, userID string, req dto.CreateAdjustmentRequest) (string, error) {
	if !canWriteWarehouse(role) {
		return "", ErrWarehouseAccessDenied
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	snap, err := s.repo.LockBalance(ctx, tx, companyID, req.BalanceID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", ErrWarehouseNotFound
		}
		return "", fmt.Errorf("lock balance: %w", err)
	}

	delta := req.NewQuantity - snap.Quantity
	if err := s.repo.UpdateBalance(ctx, tx, snap.ID, req.NewQuantity, snap.AvgCost); err != nil {
		return "", fmt.Errorf("update balance: %w", err)
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
		return "", fmt.Errorf("insert movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return movID, nil
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
