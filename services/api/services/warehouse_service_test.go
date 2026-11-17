package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

type mockWarehouseRepo struct {
	overview        *dto.WarehouseOverview
	overviewErr     error
	stock           []dto.WarehouseStockItem
	stockErr        error
	stockDetail     *dto.WarehouseStockDetail
	stockDetailErr  error
	findOrCreateSnap *repositories.BalanceSnapshot
	findOrCreateErr  error
	lockSnap        *repositories.BalanceSnapshot
	lockErr         error
	updateBalErr    error
	insertReceiptID string
	insertReceiptErr error
	insertLineID    string
	insertLineErr   error
	insertMovID     string
	insertMovErr    error
	countMovements  int
	countMovErr     error
	movements       []dto.WarehouseMovementItem
	movementsErr    error
}

func (m *mockWarehouseRepo) GetOverview(_ context.Context, _ string) (*dto.WarehouseOverview, error) {
	return m.overview, m.overviewErr
}
func (m *mockWarehouseRepo) ListStock(_ context.Context, _, _ string, _ bool) ([]dto.WarehouseStockItem, error) {
	return m.stock, m.stockErr
}
func (m *mockWarehouseRepo) GetStockByID(_ context.Context, _, _ string) (*dto.WarehouseStockDetail, error) {
	return m.stockDetail, m.stockDetailErr
}
func (m *mockWarehouseRepo) FindOrCreateBalance(_ context.Context, _ pgx.Tx, _, _, _ string, _ *string) (*repositories.BalanceSnapshot, error) {
	return m.findOrCreateSnap, m.findOrCreateErr
}
func (m *mockWarehouseRepo) LockBalance(_ context.Context, _ pgx.Tx, _, _ string) (*repositories.BalanceSnapshot, error) {
	return m.lockSnap, m.lockErr
}
func (m *mockWarehouseRepo) UpdateBalance(_ context.Context, _ pgx.Tx, _ string, _, _ float64) error {
	return m.updateBalErr
}
func (m *mockWarehouseRepo) InsertReceipt(_ context.Context, _ pgx.Tx, _, _ string, _, _, _ *string, _ time.Time) (string, error) {
	return m.insertReceiptID, m.insertReceiptErr
}
func (m *mockWarehouseRepo) InsertReceiptLine(_ context.Context, _ pgx.Tx, _, _ string, _, _ float64) (string, error) {
	return m.insertLineID, m.insertLineErr
}
func (m *mockWarehouseRepo) InsertMovement(_ context.Context, _ pgx.Tx, _ repositories.InsertMovementParams) (string, error) {
	return m.insertMovID, m.insertMovErr
}
func (m *mockWarehouseRepo) CountMovements(_ context.Context, _ string, _, _ *string) (int, error) {
	return m.countMovements, m.countMovErr
}
func (m *mockWarehouseRepo) ListMovements(_ context.Context, _ string, _, _ *string, _, _ int) ([]dto.WarehouseMovementItem, error) {
	return m.movements, m.movementsErr
}

type mockProjectStock struct {
	findOrCreateID  string
	findOrCreateErr error
	deductErr       error
	addErr          error
}

func (m *mockProjectStock) FindOrCreateDestProjectMaterial(_ context.Context, _ pgx.Tx, _, _, _ string, _ *string, _ string) (string, error) {
	return m.findOrCreateID, m.findOrCreateErr
}
func (m *mockProjectStock) DeductProjectMaterialAvailable(_ context.Context, _ pgx.Tx, _, _ string, _ float64) error {
	return m.deductErr
}
func (m *mockProjectStock) AddProjectMaterialAvailable(_ context.Context, _ pgx.Tx, _, _ string, _ float64) error {
	return m.addErr
}

// newWarehouseSvc builds a testable WarehouseService with mocked dependencies.
func newWarehouseSvc(repo warehouseRepoIface, mat projectStockIface) *WarehouseService {
	return &WarehouseService{
		db:      &mockTxBeginner{tx: &mockTx{}},
		repo:    repo,
		matRepo: mat,
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestWarehouse_GetOverview_AllRolesCanRead(t *testing.T) {
	repo := &mockWarehouseRepo{overview: &dto.WarehouseOverview{TotalItems: 5}}
	svc := newWarehouseSvc(repo, &mockProjectStock{})

	for _, role := range []string{"direktor", "inzenjer", "administracija", "poslovoda", "radnik"} {
		overview, err := svc.GetOverview(context.Background(), "company-1", role)
		if err != nil {
			t.Errorf("role %q should be able to read overview, got: %v", role, err)
		}
		if overview == nil || overview.TotalItems != 5 {
			t.Errorf("role %q: unexpected overview", role)
		}
	}
}

func TestWarehouse_GetOverview_UnknownRoleDenied(t *testing.T) {
	svc := newWarehouseSvc(&mockWarehouseRepo{}, &mockProjectStock{})
	_, err := svc.GetOverview(context.Background(), "company-1", "nepostojeci")
	if !errors.Is(err, ErrWarehouseAccessDenied) {
		t.Errorf("unknown role should be denied, got: %v", err)
	}
}

func TestWarehouse_CreateReceipt_UpdatesAvgCost(t *testing.T) {
	// Initial: 10 units @ avg 5 €. Receipt: +5 units @ 8 €.
	// New avg = (10×5 + 5×8) / 15 = 90/15 = 6.
	snap := &repositories.BalanceSnapshot{ID: "bal-1", Quantity: 10, AvgCost: 5}
	repo := &mockWarehouseRepo{
		findOrCreateSnap: snap,
		insertReceiptID:  "receipt-1",
		insertLineID:     "line-1",
		insertMovID:      "mov-1",
	}
	var capturedQty, capturedAvg float64
	repo.updateBalErr = nil

	// Intercept UpdateBalance via a custom mock
	type captureRepo struct{ *mockWarehouseRepo; capturedQty, capturedAvg float64 }
	cr := &captureRepo{mockWarehouseRepo: repo}
	captureSvc := &WarehouseService{
		db: &mockTxBeginner{tx: &mockTx{}},
		repo: &captureUpdateRepo{mockWarehouseRepo: repo, onUpdate: func(qty, avg float64) {
			capturedQty = qty
			capturedAvg = avg
		}},
		matRepo: &mockProjectStock{},
	}

	req := dto.CreateReceiptRequest{
		Lines: []dto.CreateReceiptLineRequest{
			{MaterialName: "Cement", Unit: "kg", Quantity: 5, UnitPrice: 8},
		},
	}
	_, err := captureSvc.CreateReceipt(context.Background(), "company-1", "direktor", "user-1", req)
	if err != nil {
		t.Fatalf("CreateReceipt failed: %v", err)
	}

	_ = cr
	const wantQty = 15.0
	const wantAvg = 6.0
	if capturedQty != wantQty {
		t.Errorf("expected qty=%v, got %v", wantQty, capturedQty)
	}
	if capturedAvg != wantAvg {
		t.Errorf("expected avg=%v, got %v", wantAvg, capturedAvg)
	}
}

// captureUpdateRepo wraps mockWarehouseRepo to intercept UpdateBalance.
type captureUpdateRepo struct {
	*mockWarehouseRepo
	onUpdate func(qty, avg float64)
}

func (r *captureUpdateRepo) UpdateBalance(_ context.Context, _ pgx.Tx, _ string, qty, avg float64) error {
	r.onUpdate(qty, avg)
	return nil
}

func TestWarehouse_CreateReceipt_RadnikDenied(t *testing.T) {
	svc := newWarehouseSvc(&mockWarehouseRepo{}, &mockProjectStock{})
	req := dto.CreateReceiptRequest{Lines: []dto.CreateReceiptLineRequest{{MaterialName: "X", Unit: "kom", Quantity: 1}}}
	_, err := svc.CreateReceipt(context.Background(), "company-1", "radnik", "user-1", req)
	if !errors.Is(err, ErrWarehouseAccessDenied) {
		t.Errorf("radnik should not be able to create receipt, got: %v", err)
	}
}

func TestWarehouse_CreateReceipt_PoslovodaDenied(t *testing.T) {
	svc := newWarehouseSvc(&mockWarehouseRepo{}, &mockProjectStock{})
	req := dto.CreateReceiptRequest{Lines: []dto.CreateReceiptLineRequest{{MaterialName: "X", Unit: "kom", Quantity: 1}}}
	_, err := svc.CreateReceipt(context.Background(), "company-1", "poslovoda", "user-1", req)
	if !errors.Is(err, ErrWarehouseAccessDenied) {
		t.Errorf("poslovoda should not be able to create receipt, got: %v", err)
	}
}

func TestWarehouse_IssueToProject_InsufficientQty(t *testing.T) {
	repo := &mockWarehouseRepo{
		lockSnap: &repositories.BalanceSnapshot{ID: "bal-1", Quantity: 2, AvgCost: 5},
	}
	svc := newWarehouseSvc(repo, &mockProjectStock{})

	req := dto.CreateIssueRequest{BalanceID: "bal-1", ProjectID: "proj-1", Quantity: 10}
	_, err := svc.IssueToProject(context.Background(), "company-1", "direktor", "user-1", req)
	if !errors.Is(err, ErrWarehouseInsufficientQty) {
		t.Errorf("expected ErrWarehouseInsufficientQty, got: %v", err)
	}
}

func TestWarehouse_IssueToProject_RadnikDenied(t *testing.T) {
	svc := newWarehouseSvc(&mockWarehouseRepo{}, &mockProjectStock{})
	req := dto.CreateIssueRequest{BalanceID: "bal-1", ProjectID: "proj-1", Quantity: 1}
	_, err := svc.IssueToProject(context.Background(), "company-1", "radnik", "user-1", req)
	if !errors.Is(err, ErrWarehouseAccessDenied) {
		t.Errorf("radnik should not be able to issue, got: %v", err)
	}
}

func TestWarehouse_AdjustInventory_AdminOnlyDeniedForPoslovoda(t *testing.T) {
	svc := newWarehouseSvc(&mockWarehouseRepo{}, &mockProjectStock{})
	req := dto.CreateAdjustmentRequest{BalanceID: "bal-1", NewQuantity: 5}
	_, err := svc.AdjustInventory(context.Background(), "company-1", "poslovoda", "user-1", req)
	if !errors.Is(err, ErrWarehouseAccessDenied) {
		t.Errorf("poslovoda should not adjust inventory, got: %v", err)
	}
}

func TestWarehouse_ReturnFromProject_InsufficientProjectQty(t *testing.T) {
	mat := &mockProjectStock{deductErr: repositories.ErrInsufficientQuantity}
	svc := newWarehouseSvc(&mockWarehouseRepo{}, mat)

	req := dto.CreateReturnRequest{
		BalanceID: "bal-1", ProjectID: "proj-1",
		ProjectMaterialID: "pm-1", Quantity: 99,
	}
	_, err := svc.ReturnFromProject(context.Background(), "company-1", "direktor", "user-1", req)
	if !errors.Is(err, ErrWarehouseInsufficientQty) {
		t.Errorf("expected ErrWarehouseInsufficientQty, got: %v", err)
	}
}
