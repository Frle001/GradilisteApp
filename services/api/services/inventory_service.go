package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

var (
	ErrInventoryEmployeeNotFound     = errors.New("employee not found")
	ErrAssetNotFound                 = errors.New("asset not found or not assigned to you")
	ErrInsufficientMaterial          = errors.New("Nedovoljna dostupna količina za prijenos.")
	ErrNoMaterialResponsibility      = errors.New("no active material responsibility found")
	ErrCannotTransferToSelf          = errors.New("cannot transfer to yourself")
	ErrTransferTargetNotAllowed      = errors.New("you are not allowed to transfer to this employee")
	ErrInvalidTransferType           = errors.New("transfer type must be 'asset' or 'material'")
	ErrMissingTransferFields         = errors.New("missing required fields for transfer type")
	ErrInvalidTransferQuantity       = errors.New("transfer quantity must be greater than zero")
	ErrMaterialRecipientNotPoslovoda = errors.New("materijal se može prenijeti samo drugom poslovođi")
	ErrInvalidDestinationProject     = errors.New("odaberite odredišni projekt za prijenos materijala")
)

type InventoryService struct {
	db   *pgxpool.Pool
	repo *repositories.InventoryRepository
}

func NewInventoryService(db *pgxpool.Pool, repo *repositories.InventoryRepository) *InventoryService {
	return &InventoryService{db: db, repo: repo}
}

// ── Get inventory ─────────────────────────────────────────────────────────────

func (s *InventoryService) GetMyInventory(ctx context.Context, companyID, employeeID string) (*dto.InventoryResponse, error) {
	if employeeID == "" {
		return nil, ErrForbidden
	}
	return s.getInventory(ctx, companyID, employeeID)
}

func (s *InventoryService) GetEmployeeInventory(ctx context.Context, companyID, callerRole, targetEmployeeID string) (*dto.InventoryResponse, error) {
	// Only management roles may view others' inventories
	switch callerRole {
	case "direktor", "inzenjer", "administracija":
	default:
		return nil, ErrForbidden
	}
	return s.getInventory(ctx, companyID, targetEmployeeID)
}

func (s *InventoryService) getInventory(ctx context.Context, companyID, employeeID string) (*dto.InventoryResponse, error) {
	emp, err := s.repo.GetEmployee(ctx, companyID, employeeID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrInventoryEmployeeNotFound
		}
		return nil, fmt.Errorf("get employee: %w", err)
	}

	assets, err := s.repo.GetAssets(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get assets: %w", err)
	}

	materials, err := s.repo.GetMaterials(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get materials: %w", err)
	}

	if assets == nil {
		assets = []dto.InventoryAssetItem{}
	}
	if materials == nil {
		materials = []dto.InventoryMaterialItem{}
	}

	return &dto.InventoryResponse{
		Employee:  *emp,
		Assets:    assets,
		Materials: materials,
	}, nil
}

// ── Transfer targets ───────────────────────────────────────────────────────────

func (s *InventoryService) GetTransferTargets(ctx context.Context, companyID, callerEmployeeID, callerRole, transferType string) ([]dto.TransferTarget, error) {
	if callerEmployeeID == "" {
		return nil, ErrForbidden
	}
	targets, err := s.repo.GetTransferTargets(ctx, companyID, callerEmployeeID, callerRole, transferType)
	if err != nil {
		return nil, fmt.Errorf("get transfer targets: %w", err)
	}
	if targets == nil {
		targets = []dto.TransferTarget{}
	}
	return targets, nil
}

func (s *InventoryService) GetEmployeeProjects(ctx context.Context, companyID, callerEmployeeID, targetEmployeeID string) ([]dto.InventoryProject, error) {
	if callerEmployeeID == "" {
		return nil, ErrForbidden
	}
	projects, err := s.repo.GetEmployeeActiveProjects(ctx, companyID, targetEmployeeID)
	if err != nil {
		return nil, fmt.Errorf("get employee projects: %w", err)
	}
	if projects == nil {
		projects = []dto.InventoryProject{}
	}
	return projects, nil
}

// ── Create transfer ────────────────────────────────────────────────────────────

func (s *InventoryService) CreateTransfer(
	ctx context.Context,
	companyID, callerEmployeeID, callerRole, callerUserID string,
	reqJSON []byte,
) (string, error) {
	if callerEmployeeID == "" {
		return "", ErrForbidden
	}

	var req dto.CreateTransferRequest
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		return "", fmt.Errorf("invalid request body: %w", err)
	}

	if req.ToEmployeeID == "" {
		return "", ErrMissingTransferFields
	}
	if req.ToEmployeeID == callerEmployeeID {
		return "", ErrCannotTransferToSelf
	}

	// Permission: verify caller can transfer to this target
	if err := s.verifyTransferTarget(ctx, companyID, callerEmployeeID, callerRole, req.ToEmployeeID, req.Type); err != nil {
		return "", err
	}

	switch req.Type {
	case "asset":
		return s.createAssetTransfer(ctx, companyID, callerEmployeeID, req.ToEmployeeID, callerUserID, req)
	case "material":
		return s.createMaterialTransfer(ctx, companyID, callerEmployeeID, req.ToEmployeeID, callerUserID, req)
	default:
		return "", ErrInvalidTransferType
	}
}

func (s *InventoryService) verifyTransferTarget(ctx context.Context, companyID, callerEmployeeID, callerRole, toEmployeeID, transferType string) error {
	targets, err := s.repo.GetTransferTargets(ctx, companyID, callerEmployeeID, callerRole, transferType)
	if err != nil {
		return fmt.Errorf("get transfer targets: %w", err)
	}
	for _, t := range targets {
		if t.ID == toEmployeeID {
			return nil
		}
	}
	return ErrTransferTargetNotAllowed
}

func (s *InventoryService) createAssetTransfer(
	ctx context.Context,
	companyID, fromEmployeeID, toEmployeeID, userID string,
	req dto.CreateTransferRequest,
) (string, error) {
	if req.EmployeeAssetID == nil || *req.EmployeeAssetID == "" {
		return "", ErrMissingTransferFields
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	assetType, err := s.repo.LockAsset(ctx, tx, companyID, *req.EmployeeAssetID, fromEmployeeID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", ErrAssetNotFound
		}
		return "", fmt.Errorf("lock asset: %w", err)
	}

	if err := s.repo.AssignAsset(ctx, tx, *req.EmployeeAssetID, toEmployeeID); err != nil {
		return "", fmt.Errorf("assign asset: %w", err)
	}

	id, err := s.repo.InsertTransferRecord(
		ctx, tx,
		companyID, fromEmployeeID, toEmployeeID, assetType,
		req.EmployeeAssetID, nil,
		1, nil,
		userID, req.Notes,
	)
	if err != nil {
		return "", fmt.Errorf("insert transfer record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return id, nil
}

func (s *InventoryService) createMaterialTransfer(
	ctx context.Context,
	companyID, fromEmployeeID, toEmployeeID, userID string,
	req dto.CreateTransferRequest,
) (string, error) {
	if req.ProjectMaterialID == nil || *req.ProjectMaterialID == "" {
		return "", ErrMissingTransferFields
	}
	if req.Quantity == nil || *req.Quantity <= 0 {
		return "", ErrInvalidTransferQuantity
	}
	if req.DestinationProjectID == nil || *req.DestinationProjectID == "" {
		return "", ErrInvalidDestinationProject
	}
	transferQty := *req.Quantity
	destProjectID := *req.DestinationProjectID

	// Validate recipient role before opening a transaction.
	toEmp, err := s.repo.GetEmployee(ctx, companyID, toEmployeeID)
	if err != nil || toEmp.Role != "poslovoda" {
		return "", ErrMaterialRecipientNotPoslovoda
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Fetch source material metadata needed for dest project material lookup/creation.
	srcMat, err := s.repo.GetProjectMaterialInfo(ctx, tx, companyID, *req.ProjectMaterialID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return "", ErrNoMaterialResponsibility
		}
		return "", fmt.Errorf("get source material info: %w", err)
	}

	sourceRows, totalAvailable, err := s.repo.LockMaterialRows(ctx, tx, companyID, fromEmployeeID, *req.ProjectMaterialID)
	if err != nil {
		return "", fmt.Errorf("lock material rows: %w", err)
	}
	if len(sourceRows) == 0 {
		return "", ErrNoMaterialResponsibility
	}
	if totalAvailable < transferQty {
		return "", ErrInsufficientMaterial
	}

	// Ensure the destination project has a project_materials row for this material.
	// If the dest project is different from the source, this may create a new row
	// with planned/used/available = 0 (material arrives via transfer, not purchase).
	destProjectMaterialID, err := s.repo.FindOrCreateDestProjectMaterial(
		ctx, tx, companyID, destProjectID,
		srcMat.MaterialName, srcMat.MaterialCode, srcMat.Unit,
	)
	if err != nil {
		return "", fmt.Errorf("find/create dest project material: %w", err)
	}

	// Deduct transferred quantity from source project_materials.available_quantity.
	// The guarded UPDATE prevents negative dostupno on the source project.
	if err := s.repo.DeductProjectMaterialAvailable(ctx, tx, companyID, *req.ProjectMaterialID, transferQty); err != nil {
		if errors.Is(err, repositories.ErrInsufficientQuantity) {
			return "", ErrInsufficientMaterial
		}
		return "", fmt.Errorf("deduct source available quantity: %w", err)
	}

	// Add transferred quantity to destination project_materials.available_quantity.
	if err := s.repo.AddProjectMaterialAvailable(ctx, tx, companyID, destProjectMaterialID, transferQty); err != nil {
		return "", fmt.Errorf("add dest available quantity: %w", err)
	}

	// Greedy deduction from source responsibility rows (oldest first)
	remaining := transferQty
	for _, row := range sourceRows {
		if remaining <= 0 {
			break
		}
		deduct := remaining
		if deduct > row.Quantity {
			deduct = row.Quantity
		}
		newQty := row.Quantity - deduct
		if err := s.repo.DeductMaterialRow(ctx, tx, row.ID, newQty); err != nil {
			if errors.Is(err, repositories.ErrInsufficientQuantity) {
				return "", ErrInsufficientMaterial
			}
			return "", fmt.Errorf("deduct material row: %w", err)
		}
		remaining -= deduct
	}

	destRowID, err := s.repo.UpsertDestMaterialRow(ctx, tx, companyID, toEmployeeID, destProjectID, destProjectMaterialID, srcMat.Unit, transferQty)
	if err != nil {
		return "", fmt.Errorf("upsert dest material row: %w", err)
	}

	id, err := s.repo.InsertTransferRecord(
		ctx, tx,
		companyID, fromEmployeeID, toEmployeeID, "material",
		nil, &destRowID,
		transferQty, &destProjectID,
		userID, req.Notes,
	)
	if err != nil {
		return "", fmt.Errorf("insert transfer record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return id, nil
}

// ── List transfers ─────────────────────────────────────────────────────────────

func (s *InventoryService) ListTransfers(
	ctx context.Context,
	companyID, callerEmployeeID, callerRole string,
	page, perPage int,
) (*dto.TransferListResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}

	// Management sees all; poslovoda/radnik see only their own
	var scopeEmpID *string
	switch callerRole {
	case "direktor", "inzenjer", "administracija":
		// no scope
	default:
		if callerEmployeeID == "" {
			return nil, ErrForbidden
		}
		scopeEmpID = &callerEmployeeID
	}

	total, err := s.repo.CountTransfers(ctx, companyID, scopeEmpID)
	if err != nil {
		return nil, fmt.Errorf("count transfers: %w", err)
	}

	totalPages := (total + perPage - 1) / perPage
	offset := (page - 1) * perPage

	items, err := s.repo.ListTransfers(ctx, companyID, scopeEmpID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	if items == nil {
		items = []dto.TransferListItem{}
	}

	return &dto.TransferListResponse{
		Data: items,
		Pagination: dto.TransferPagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
