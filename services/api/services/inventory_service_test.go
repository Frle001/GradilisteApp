package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Mock inventory repository ─────────────────────────────────────────────────

type mockInvRepo struct {
	getEmployeeFn                func(context.Context, string, string) (*dto.InventoryEmployeeSummary, error)
	getProjectMaterialInfoFn     func(context.Context, pgx.Tx, string, string) (*repositories.ProjectMaterialInfo, error)
	findOrCreateDestProjMatFn    func(context.Context, pgx.Tx, string, string, string, *string, string) (string, error)
	deductProjectMaterialAvailFn func(context.Context, pgx.Tx, string, string, float64) error
	addProjectMaterialAvailFn    func(context.Context, pgx.Tx, string, string, float64) error
	lockMaterialRowsFn           func(context.Context, pgx.Tx, string, string, string) ([]repositories.MaterialResponsibilityRow, float64, error)
	deductMaterialRowFn          func(context.Context, pgx.Tx, string, float64) error
	upsertDestMaterialRowFn      func(context.Context, pgx.Tx, string, string, string, string, string, float64) (string, error)
	insertTransferRecordFn       func(context.Context, pgx.Tx, string, string, string, string, *string, *string, float64, *string, string, string) (string, error)
	getTransferTargetsFn         func(context.Context, string, string, string, string) ([]dto.TransferTarget, error)
	lockAssetFn                  func(context.Context, pgx.Tx, string, string, string) (string, error)
	assignAssetFn                func(context.Context, pgx.Tx, string, string) error
}

func (m *mockInvRepo) GetEmployee(ctx context.Context, companyID, empID string) (*dto.InventoryEmployeeSummary, error) {
	if m.getEmployeeFn != nil {
		return m.getEmployeeFn(ctx, companyID, empID)
	}
	return &dto.InventoryEmployeeSummary{ID: empID, Role: "poslovoda"}, nil
}
func (m *mockInvRepo) GetAssets(_ context.Context, _, _ string) ([]dto.InventoryAssetItem, error) {
	return nil, nil
}
func (m *mockInvRepo) GetMaterials(_ context.Context, _, _ string) ([]dto.InventoryMaterialItem, error) {
	return nil, nil
}
func (m *mockInvRepo) GetTransferTargets(ctx context.Context, companyID, callerEmpID, callerRole, transferType string) ([]dto.TransferTarget, error) {
	if m.getTransferTargetsFn != nil {
		return m.getTransferTargetsFn(ctx, companyID, callerEmpID, callerRole, transferType)
	}
	return []dto.TransferTarget{{ID: "to-emp-1", Role: "poslovoda"}}, nil
}
func (m *mockInvRepo) GetEmployeeActiveProjects(_ context.Context, _, _ string) ([]dto.InventoryProject, error) {
	return nil, nil
}
func (m *mockInvRepo) GetProjectMaterialInfo(ctx context.Context, tx pgx.Tx, companyID, pmID string) (*repositories.ProjectMaterialInfo, error) {
	if m.getProjectMaterialInfoFn != nil {
		return m.getProjectMaterialInfoFn(ctx, tx, companyID, pmID)
	}
	return &repositories.ProjectMaterialInfo{ID: pmID, ProjectID: "project-src", MaterialName: "Kabel NYY", Unit: "m"}, nil
}
func (m *mockInvRepo) FindOrCreateDestProjectMaterial(ctx context.Context, tx pgx.Tx, companyID, destProjID, name string, code *string, unit string) (string, error) {
	if m.findOrCreateDestProjMatFn != nil {
		return m.findOrCreateDestProjMatFn(ctx, tx, companyID, destProjID, name, code, unit)
	}
	return "dest-pm-1", nil
}
func (m *mockInvRepo) DeductProjectMaterialAvailable(ctx context.Context, tx pgx.Tx, companyID, pmID string, qty float64) error {
	if m.deductProjectMaterialAvailFn != nil {
		return m.deductProjectMaterialAvailFn(ctx, tx, companyID, pmID, qty)
	}
	return nil
}
func (m *mockInvRepo) AddProjectMaterialAvailable(ctx context.Context, tx pgx.Tx, companyID, pmID string, qty float64) error {
	if m.addProjectMaterialAvailFn != nil {
		return m.addProjectMaterialAvailFn(ctx, tx, companyID, pmID, qty)
	}
	return nil
}
func (m *mockInvRepo) LockAsset(ctx context.Context, tx pgx.Tx, companyID, assetID, fromEmpID string) (string, error) {
	if m.lockAssetFn != nil {
		return m.lockAssetFn(ctx, tx, companyID, assetID, fromEmpID)
	}
	return "tool", nil
}
func (m *mockInvRepo) AssignAsset(ctx context.Context, tx pgx.Tx, assetID, toEmpID string) error {
	if m.assignAssetFn != nil {
		return m.assignAssetFn(ctx, tx, assetID, toEmpID)
	}
	return nil
}
func (m *mockInvRepo) LockMaterialRows(ctx context.Context, tx pgx.Tx, companyID, empID, pmID string) ([]repositories.MaterialResponsibilityRow, float64, error) {
	if m.lockMaterialRowsFn != nil {
		return m.lockMaterialRowsFn(ctx, tx, companyID, empID, pmID)
	}
	row := repositories.MaterialResponsibilityRow{ID: "emr-1", ProjectID: "project-src", Unit: "m", Quantity: 100}
	return []repositories.MaterialResponsibilityRow{row}, 100, nil
}
func (m *mockInvRepo) DeductMaterialRow(ctx context.Context, tx pgx.Tx, rowID string, remaining float64) error {
	if m.deductMaterialRowFn != nil {
		return m.deductMaterialRowFn(ctx, tx, rowID, remaining)
	}
	return nil
}
func (m *mockInvRepo) UpsertDestMaterialRow(ctx context.Context, tx pgx.Tx, companyID, toEmpID, projID, pmID, unit string, addQty float64) (string, error) {
	if m.upsertDestMaterialRowFn != nil {
		return m.upsertDestMaterialRowFn(ctx, tx, companyID, toEmpID, projID, pmID, unit, addQty)
	}
	return "emr-dest-1", nil
}
func (m *mockInvRepo) InsertTransferRecord(ctx context.Context, tx pgx.Tx, companyID, fromEmpID, toEmpID, assetType string, assetID *string, responsibilityID *string, qty float64, projID *string, by, notes string) (string, error) {
	if m.insertTransferRecordFn != nil {
		return m.insertTransferRecordFn(ctx, tx, companyID, fromEmpID, toEmpID, assetType, assetID, responsibilityID, qty, projID, by, notes)
	}
	return "transfer-1", nil
}
func (m *mockInvRepo) CountTransfers(_ context.Context, _ string, _ *string) (int, error) {
	return 0, nil
}
func (m *mockInvRepo) ListTransfers(_ context.Context, _ string, _ *string, _, _ int) ([]dto.TransferListItem, error) {
	return nil, nil
}

// newInvSvc constructs an InventoryService with a mock repo and mock tx pool.
func newInvSvc(repo *mockInvRepo) *InventoryService {
	return &InventoryService{db: &schedTxBeginner{tx: &schedMockTx{}}, repo: repo}
}

// materialTransferBody builds the JSON request bytes used for material transfer tests.
func materialTransferBody(toEmpID, projectMaterialID, destProjectID string, qty float64) []byte {
	return []byte(`{
		"to_employee_id": "` + toEmpID + `",
		"type": "material",
		"project_material_id": "` + projectMaterialID + `",
		"destination_project_id": "` + destProjectID + `",
		"quantity": ` + ftoa(qty) + `,
		"notes": ""
	}`)
}

func ftoa(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestTransfer_InsufficientResponsibility verifies that requesting more than
// the caller's total responsibility returns ErrInsufficientMaterial (HTTP 422).
func TestTransfer_InsufficientResponsibility(t *testing.T) {
	repo := &mockInvRepo{
		lockMaterialRowsFn: func(_ context.Context, _ pgx.Tx, _, _, _ string) ([]repositories.MaterialResponsibilityRow, float64, error) {
			row := repositories.MaterialResponsibilityRow{ID: "emr-1", Quantity: 5}
			return []repositories.MaterialResponsibilityRow{row}, 5, nil
		},
	}
	svc := newInvSvc(repo)
	body := materialTransferBody("to-emp-1", "pm-1", "proj-dest", 10)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrInsufficientMaterial) {
		t.Errorf("expected ErrInsufficientMaterial when responsibility total < requested, got: %v", err)
	}
}

// TestTransfer_StaleQuantity_Returns409Sentinel verifies that when responsibility
// check passes but the project-level DeductProjectMaterialAvailable guard fails
// (project quantity was reduced by a concurrent approval), ErrStaleTransferQuantity
// is returned — which the handler maps to HTTP 409.
func TestTransfer_StaleQuantity_Returns409Sentinel(t *testing.T) {
	repo := &mockInvRepo{
		// Responsibility shows 100 (stale — not yet decremented by report approval).
		lockMaterialRowsFn: func(_ context.Context, _ pgx.Tx, _, _, _ string) ([]repositories.MaterialResponsibilityRow, float64, error) {
			row := repositories.MaterialResponsibilityRow{ID: "emr-1", Quantity: 100}
			return []repositories.MaterialResponsibilityRow{row}, 100, nil
		},
		// Project material only has 20 (already decremented by the approval).
		deductProjectMaterialAvailFn: func(_ context.Context, _ pgx.Tx, _, _ string, _ float64) error {
			return repositories.ErrInsufficientQuantity
		},
	}
	svc := newInvSvc(repo)
	body := materialTransferBody("to-emp-1", "pm-1", "proj-dest", 50)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrStaleTransferQuantity) {
		t.Errorf("expected ErrStaleTransferQuantity when project quantity is stale, got: %v", err)
	}
	if errors.Is(err, ErrInsufficientMaterial) {
		t.Error("stale quantity must NOT map to ErrInsufficientMaterial (wrong HTTP status)")
	}
}

// TestTransfer_ToRadnik_Rejected verifies that material cannot be transferred to
// an employee with role 'radnik'. The target must pass the allowed-list check
// (so we include them in GetTransferTargets), but the role validation inside
// createMaterialTransfer must still reject the transfer.
func TestTransfer_ToRadnik_Rejected(t *testing.T) {
	repo := &mockInvRepo{
		// Include "radnik-emp" in the transfer-target list so verifyTransferTarget passes,
		// letting createMaterialTransfer's explicit role check run.
		getTransferTargetsFn: func(_ context.Context, _, _, _, _ string) ([]dto.TransferTarget, error) {
			return []dto.TransferTarget{{ID: "radnik-emp", Role: "radnik"}}, nil
		},
		getEmployeeFn: func(_ context.Context, _, empID string) (*dto.InventoryEmployeeSummary, error) {
			return &dto.InventoryEmployeeSummary{ID: empID, Role: "radnik"}, nil
		},
	}
	svc := newInvSvc(repo)
	body := materialTransferBody("radnik-emp", "pm-1", "proj-dest", 10)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrMaterialRecipientNotPoslovoda) {
		t.Errorf("expected ErrMaterialRecipientNotPoslovoda for radnik recipient, got: %v", err)
	}
}

// TestTransfer_CannotTransferToSelf verifies self-transfer is rejected before any
// DB call is made.
func TestTransfer_CannotTransferToSelf(t *testing.T) {
	svc := newInvSvc(&mockInvRepo{})
	body := materialTransferBody("same-emp", "pm-1", "proj-dest", 10)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "same-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrCannotTransferToSelf) {
		t.Errorf("expected ErrCannotTransferToSelf, got: %v", err)
	}
}

// TestTransfer_Success_RecordsTransferAndReturnsID verifies the happy path:
// both responsibility and project quantities are updated and a transfer ID is returned.
func TestTransfer_Success_RecordsTransferAndReturnsID(t *testing.T) {
	var deductedQty, addedQty float64
	var insertCalled bool
	repo := &mockInvRepo{
		deductProjectMaterialAvailFn: func(_ context.Context, _ pgx.Tx, _, _ string, qty float64) error {
			deductedQty = qty
			return nil
		},
		addProjectMaterialAvailFn: func(_ context.Context, _ pgx.Tx, _, _ string, qty float64) error {
			addedQty = qty
			return nil
		},
		insertTransferRecordFn: func(_ context.Context, _ pgx.Tx, _, _, _, _ string, _ *string, _ *string, _ float64, _ *string, _, _ string) (string, error) {
			insertCalled = true
			return "transfer-abc", nil
		},
	}
	svc := newInvSvc(repo)
	body := materialTransferBody("to-emp-1", "pm-1", "proj-dest", 30)
	id, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "transfer-abc" {
		t.Errorf("expected transfer-abc, got %q", id)
	}
	if deductedQty != 30 {
		t.Errorf("expected deducted qty=30, got %.2f", deductedQty)
	}
	if addedQty != 30 {
		t.Errorf("expected added qty=30, got %.2f", addedQty)
	}
	if !insertCalled {
		t.Error("expected InsertTransferRecord to be called")
	}
}

// TestTransfer_NoResponsibilityRows_ReturnsNoResponsibilityError verifies that
// when the caller has no active responsibility rows for the material,
// ErrNoMaterialResponsibility is returned.
func TestTransfer_NoResponsibilityRows_ReturnsNoResponsibilityError(t *testing.T) {
	repo := &mockInvRepo{
		lockMaterialRowsFn: func(_ context.Context, _ pgx.Tx, _, _, _ string) ([]repositories.MaterialResponsibilityRow, float64, error) {
			return []repositories.MaterialResponsibilityRow{}, 0, nil
		},
	}
	svc := newInvSvc(repo)
	body := materialTransferBody("to-emp-1", "pm-1", "proj-dest", 10)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrNoMaterialResponsibility) {
		t.Errorf("expected ErrNoMaterialResponsibility, got: %v", err)
	}
}

// TestTransfer_CrossCompany_TargetNotAllowed verifies that the transfer target
// verification prevents cross-company transfers (target not in company's employee list).
func TestTransfer_CrossCompany_TargetNotAllowed(t *testing.T) {
	repo := &mockInvRepo{
		getTransferTargetsFn: func(_ context.Context, _, _, _, _ string) ([]dto.TransferTarget, error) {
			// Returns no targets → recipient is not in the allowed set.
			return []dto.TransferTarget{}, nil
		},
	}
	svc := newInvSvc(repo)
	body := materialTransferBody("foreign-emp", "pm-1", "proj-dest", 10)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrTransferTargetNotAllowed) {
		t.Errorf("expected ErrTransferTargetNotAllowed for cross-company target, got: %v", err)
	}
}

// TestTransfer_ZeroQuantity_Rejected verifies that requesting quantity ≤ 0 is rejected
// before any DB call.
func TestTransfer_ZeroQuantity_Rejected(t *testing.T) {
	svc := newInvSvc(&mockInvRepo{})
	body := materialTransferBody("to-emp-1", "pm-1", "proj-dest", 0)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrInvalidTransferQuantity) {
		t.Errorf("expected ErrInvalidTransferQuantity, got: %v", err)
	}
}

// TestTransfer_MissingDestinationProject_Rejected verifies that omitting the
// destination project is rejected with ErrInvalidDestinationProject.
func TestTransfer_MissingDestinationProject_Rejected(t *testing.T) {
	svc := newInvSvc(&mockInvRepo{})
	body := []byte(`{
		"to_employee_id": "to-emp-1",
		"type": "material",
		"project_material_id": "pm-1",
		"quantity": 10,
		"notes": ""
	}`)
	_, err := svc.CreateTransfer(context.Background(), "co-1", "from-emp", "poslovoda", "user-1", body)
	if !errors.Is(err, ErrInvalidDestinationProject) {
		t.Errorf("expected ErrInvalidDestinationProject, got: %v", err)
	}
}
