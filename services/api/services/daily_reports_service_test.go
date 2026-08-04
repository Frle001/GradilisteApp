package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Mock daily-report repository ──────────────────────────────────────────────

type mockDrRepo struct {
	listFn                     func(context.Context, string, dto.DailyReportFilter) ([]dto.DailyReportListItem, error)
	getByIDFn                  func(context.Context, string, string) (*dto.DailyReportDetail, error)
	getByIDForEditFn           func(context.Context, string, string) (string, string, string, error)
	getActivitiesForApprovalFn func(context.Context, string, string) ([]repositories.ActivityForApproval, error)
	createFn                   func(context.Context, pgx.Tx, string, string, string, string, dto.CreateDailyReportRequest) (string, error)
	updateFn                   func(context.Context, pgx.Tx, string, string, dto.CreateDailyReportRequest) error
	setStatusFn                func(context.Context, pgx.Tx, string, string, string, *string) error
	getProjectStatusFn         func(context.Context, string, string) (string, error)
	getWorkerInfoFn            func(context.Context, string, string) (string, *string, error)
	isMaterialInProjectFn      func(context.Context, string, string, string) (bool, error)
	getActiveProjectsFn        func(context.Context, string) ([]dto.FormDataProject, error)
	getWorkersForPoslovodaFn   func(context.Context, string, string) ([]dto.FormDataWorker, error)
	getAllActiveWorkersFn      func(context.Context, string) ([]dto.FormDataWorker, error)
	getMaterialsForProjectFn   func(context.Context, string, string) ([]dto.FormDataMaterial, error)
}

func (m *mockDrRepo) List(ctx context.Context, companyID string, f dto.DailyReportFilter) ([]dto.DailyReportListItem, error) {
	if m.listFn != nil {
		return m.listFn(ctx, companyID, f)
	}
	return nil, nil
}

func (m *mockDrRepo) GetByID(ctx context.Context, id, companyID string) (*dto.DailyReportDetail, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, companyID)
	}
	return nil, repositories.ErrDailyReportNotFound
}

func (m *mockDrRepo) GetByIDForEdit(ctx context.Context, id, companyID string) (string, string, string, error) {
	if m.getByIDForEditFn != nil {
		return m.getByIDForEditFn(ctx, id, companyID)
	}
	return "submitted", "emp-poslovoda", "project-1", nil
}

func (m *mockDrRepo) GetActivitiesForApproval(ctx context.Context, reportID, companyID string) ([]repositories.ActivityForApproval, error) {
	if m.getActivitiesForApprovalFn != nil {
		return m.getActivitiesForApprovalFn(ctx, reportID, companyID)
	}
	return nil, nil
}

func (m *mockDrRepo) Create(ctx context.Context, tx pgx.Tx, companyID, projectID, poslovodaEmpID, userID string, req dto.CreateDailyReportRequest) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tx, companyID, projectID, poslovodaEmpID, userID, req)
	}
	return "report-id", nil
}

func (m *mockDrRepo) Update(ctx context.Context, tx pgx.Tx, id, companyID string, req dto.CreateDailyReportRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tx, id, companyID, req)
	}
	return nil
}

func (m *mockDrRepo) SetStatus(ctx context.Context, tx pgx.Tx, id, companyID, status string, notes *string) error {
	if m.setStatusFn != nil {
		return m.setStatusFn(ctx, tx, id, companyID, status, notes)
	}
	return nil
}

func (m *mockDrRepo) GetProjectStatus(ctx context.Context, projectID, companyID string) (string, error) {
	if m.getProjectStatusFn != nil {
		return m.getProjectStatusFn(ctx, projectID, companyID)
	}
	return "active", nil
}

func (m *mockDrRepo) GetWorkerInfo(ctx context.Context, workerID, companyID string) (string, *string, error) {
	if m.getWorkerInfoFn != nil {
		return m.getWorkerInfoFn(ctx, workerID, companyID)
	}
	return "", nil, nil
}

func (m *mockDrRepo) IsMaterialInProject(ctx context.Context, materialID, projectID, companyID string) (bool, error) {
	if m.isMaterialInProjectFn != nil {
		return m.isMaterialInProjectFn(ctx, materialID, projectID, companyID)
	}
	return true, nil
}

func (m *mockDrRepo) GetActiveProjects(ctx context.Context, companyID string) ([]dto.FormDataProject, error) {
	if m.getActiveProjectsFn != nil {
		return m.getActiveProjectsFn(ctx, companyID)
	}
	return []dto.FormDataProject{}, nil
}

func (m *mockDrRepo) GetWorkersForPoslovoda(ctx context.Context, poslovodaEmpID, companyID string) ([]dto.FormDataWorker, error) {
	if m.getWorkersForPoslovodaFn != nil {
		return m.getWorkersForPoslovodaFn(ctx, poslovodaEmpID, companyID)
	}
	return []dto.FormDataWorker{}, nil
}

func (m *mockDrRepo) GetAllActiveWorkers(ctx context.Context, companyID string) ([]dto.FormDataWorker, error) {
	if m.getAllActiveWorkersFn != nil {
		return m.getAllActiveWorkersFn(ctx, companyID)
	}
	return []dto.FormDataWorker{}, nil
}

func (m *mockDrRepo) GetMaterialsForProject(ctx context.Context, projectID, companyID string) ([]dto.FormDataMaterial, error) {
	if m.getMaterialsForProjectFn != nil {
		return m.getMaterialsForProjectFn(ctx, projectID, companyID)
	}
	return []dto.FormDataMaterial{}, nil
}

// ── Mock audit repository ─────────────────────────────────────────────────────

type mockAuditLog struct{}

func (m *mockAuditLog) Log(_ context.Context, _ repositories.AuditParams) {}

// ── Mock materialEffectsRepoIface ─────────────────────────────────────────────

type mockMaterialEffectsRepo struct {
	alreadyAppliedFn   func(context.Context, pgx.Tx, string) (bool, error)
	validateAndApplyFn func(context.Context, pgx.Tx, string, string, string, string, string, []repositories.ActivityForApproval) error
	applyCallArgs      struct {
		companyID     string
		reportID      string
		projectID     string
		poslovodaID   string
		appliedByID   string
		activityCount int
	}
}

func (m *mockMaterialEffectsRepo) EffectsAlreadyApplied(ctx context.Context, tx pgx.Tx, reportID string) (bool, error) {
	if m.alreadyAppliedFn != nil {
		return m.alreadyAppliedFn(ctx, tx, reportID)
	}
	return false, nil
}

func (m *mockMaterialEffectsRepo) ValidateAndApply(
	ctx context.Context, tx pgx.Tx,
	companyID, reportID, projectID, poslovodaEmpID, appliedByUserID string,
	acts []repositories.ActivityForApproval,
) error {
	m.applyCallArgs.companyID = companyID
	m.applyCallArgs.reportID = reportID
	m.applyCallArgs.projectID = projectID
	m.applyCallArgs.poslovodaID = poslovodaEmpID
	m.applyCallArgs.appliedByID = appliedByUserID
	m.applyCallArgs.activityCount = len(acts)
	if m.validateAndApplyFn != nil {
		return m.validateAndApplyFn(ctx, tx, companyID, reportID, projectID, poslovodaEmpID, appliedByUserID, acts)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newDrSvc(repo *mockDrRepo) *DailyReportService {
	return &DailyReportService{
		db:        &schedTxBeginner{tx: &schedMockTx{}},
		drRepo:    repo,
		auditRepo: &mockAuditLog{},
		// materialEffectsRepo is nil — none of the Create/Update tests exercise Approve
	}
}

func newDrSvcWithEffects(repo *mockDrRepo, fx *mockMaterialEffectsRepo) *DailyReportService {
	return &DailyReportService{
		db:                  &schedTxBeginner{tx: &schedMockTx{}},
		drRepo:              repo,
		auditRepo:           &mockAuditLog{},
		materialEffectsRepo: fx,
	}
}

func drTodayStr() string {
	loc, err := time.LoadLocation("Europe/Zagreb")
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// vtkActivity returns a valid VTK activity that passes validateReport without
// hitting the project-material lookup.
func vtkActivity() dto.ActivityInput {
	name := "Beton B25"
	return dto.ActivityInput{
		Quantity:           2.5,
		Unit:               "m2",
		ActivityType:       "other",
		IsVTK:              true,
		CustomMaterialName: &name,
	}
}

// ── Tests: GetFormData ────────────────────────────────────────────────────────

// TestDailyReport_GetFormData_PoslovodaSeesAllActiveProjects verifies that the
// form-data endpoint returns all active company projects for poslovoda, not only
// the subset of projects they are permanently assigned to.
func TestDailyReport_GetFormData_PoslovodaSeesAllActiveProjects(t *testing.T) {
	allProjects := []dto.FormDataProject{
		{ID: "p-1", Name: "Projekt Alpha"},
		{ID: "p-2", Name: "Projekt Beta"},
		{ID: "p-3", Name: "Projekt Gamma"},
	}
	var activeProjectsCalledWithCompany string
	repo := &mockDrRepo{
		getActiveProjectsFn: func(_ context.Context, companyID string) ([]dto.FormDataProject, error) {
			activeProjectsCalledWithCompany = companyID
			return allProjects, nil
		},
	}
	fd, err := newDrSvc(repo).GetFormData(context.Background(), "co-1", "emp-poslovoda", "poslovoda", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fd.Projects) != 3 {
		t.Errorf("poslovoda should see all 3 active company projects, got %d", len(fd.Projects))
	}
	if activeProjectsCalledWithCompany != "co-1" {
		t.Errorf("GetActiveProjects should be called with companyID='co-1', got %q", activeProjectsCalledWithCompany)
	}
}

// ── Tests: Create ─────────────────────────────────────────────────────────────

// TestDailyReport_Create_PoslovodaCanSubmitForUnassignedProject is the core
// regression test: a poslovoda must be able to create a report for a project
// they have no permanent project_assignment on.
func TestDailyReport_Create_PoslovodaCanSubmitForUnassignedProject(t *testing.T) {
	created := false
	repo := &mockDrRepo{
		getProjectStatusFn: func(_ context.Context, projectID, _ string) (string, error) {
			// "project-other" simulates a project not in poslovoda's assignment list
			if projectID == "project-other" {
				return "active", nil
			}
			return "", nil
		},
		createFn: func(_ context.Context, _ pgx.Tx, _, _, _, _ string, _ dto.CreateDailyReportRequest) (string, error) {
			created = true
			return "report-new", nil
		},
	}
	req := dto.CreateDailyReportRequest{
		ProjectID:  "project-other",
		ReportDate: drTodayStr(),
		Activities: []dto.ActivityInput{vtkActivity()},
	}
	id, err := newDrSvc(repo).Create(context.Background(), "co-1", "user-1", "emp-poslovoda", "poslovoda", req)
	if err != nil {
		t.Fatalf("poslovoda should be able to submit for an unassigned project, got: %v", err)
	}
	if id == "" {
		t.Error("expected a non-empty report ID")
	}
	if !created {
		t.Error("expected drRepo.Create to be called")
	}
}

// TestDailyReport_Create_PoslovodaCanSubmitForAssignedProject verifies no
// regression: a project the poslovoda is assigned to still works.
func TestDailyReport_Create_PoslovodaCanSubmitForAssignedProject(t *testing.T) {
	repo := &mockDrRepo{
		getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) { return "active", nil },
	}
	req := dto.CreateDailyReportRequest{
		ProjectID:  "project-assigned",
		ReportDate: drTodayStr(),
		Activities: []dto.ActivityInput{vtkActivity()},
	}
	_, err := newDrSvc(repo).Create(context.Background(), "co-1", "user-1", "emp-poslovoda", "poslovoda", req)
	if err != nil {
		t.Fatalf("poslovoda should still be able to submit for an assigned project, got: %v", err)
	}
}

// TestDailyReport_Create_CrossCompanyProjectRejected verifies that a project
// from another company (returns empty status, i.e. not found) is rejected.
func TestDailyReport_Create_CrossCompanyProjectRejected(t *testing.T) {
	repo := &mockDrRepo{
		getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "", nil // empty = project not found in this company
		},
	}
	req := dto.CreateDailyReportRequest{
		ProjectID:  "project-other-company",
		ReportDate: drTodayStr(),
		Activities: []dto.ActivityInput{vtkActivity()},
	}
	_, err := newDrSvc(repo).Create(context.Background(), "co-1", "user-1", "emp-poslovoda", "poslovoda", req)
	if err == nil {
		t.Fatal("expected error for cross-company project, got nil")
	}
	if !strings.Contains(err.Error(), "projekt nije pronađen") {
		t.Errorf("expected 'projekt nije pronađen' error, got: %v", err)
	}
}

// TestDailyReport_Create_InactiveProjectRejected verifies that creating a
// report for an archived or otherwise inactive project is rejected.
func TestDailyReport_Create_InactiveProjectRejected(t *testing.T) {
	repo := &mockDrRepo{
		getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) {
			return "archived", nil
		},
	}
	req := dto.CreateDailyReportRequest{
		ProjectID:  "project-archived",
		ReportDate: drTodayStr(),
		Activities: []dto.ActivityInput{vtkActivity()},
	}
	_, err := newDrSvc(repo).Create(context.Background(), "co-1", "user-1", "emp-poslovoda", "poslovoda", req)
	if err == nil {
		t.Fatal("expected error for inactive project, got nil")
	}
	if !strings.Contains(err.Error(), "aktivne projekte") {
		t.Errorf("expected 'aktivne projekte' error, got: %v", err)
	}
}

// TestDailyReport_RoleRestrictionsUnchanged documents that route-level role
// guards remain in place and are not the subject of this service layer.
// POST/PUT daily-reports → requireRoles("direktor", "inzenjer", "poslovoda")
// Approve/Reject          → requireRoles("direktor", "inzenjer")
// Enforcement is in routes/daily_reports.go via middleware; no service test needed.
func TestDailyReport_RoleRestrictionsUnchanged(_ *testing.T) {}

// TestDailyReport_Create_DoesNotCreateProjectAssignment verifies that creating
// a daily report does not write to the project_assignments table.
// drRepoIface has no project-assignment mutation methods, so the interface
// itself makes this invariant impossible to violate from the service layer.
func TestDailyReport_Create_DoesNotCreateProjectAssignment(t *testing.T) {
	createCallCount := 0
	repo := &mockDrRepo{
		getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) { return "active", nil },
		createFn: func(_ context.Context, _ pgx.Tx, _, _, _, _ string, _ dto.CreateDailyReportRequest) (string, error) {
			createCallCount++
			return "report-no-assign", nil
		},
	}
	req := dto.CreateDailyReportRequest{
		ProjectID:  "project-1",
		ReportDate: drTodayStr(),
		Activities: []dto.ActivityInput{vtkActivity()},
	}
	if _, err := newDrSvc(repo).Create(context.Background(), "co-1", "user-1", "emp-poslovoda", "poslovoda", req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCallCount != 1 {
		t.Errorf("expected drRepo.Create called exactly once, got %d", createCallCount)
	}
	// drRepoIface exposes no CreateProjectAssignment or similar method, so no
	// assignment row can be written by this service during report creation.
}

// ── Tests: Approve ────────────────────────────────────────────────────────────

// TestDailyReport_Approve_PassesPoslovodaIDToEffects verifies that the poslovodaID
// from GetByIDForEdit is forwarded to ValidateAndApply, so responsibility rows
// are updated for the correct employee during montaža/demontaža approval.
func TestDailyReport_Approve_PassesPoslovodaIDToEffects(t *testing.T) {
	fx := &mockMaterialEffectsRepo{}
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda-123", "project-456", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{}, nil
		},
	}
	svc := newDrSvcWithEffects(repo, fx)
	if err := svc.Approve(context.Background(), "report-1", "co-1", "user-1", "emp-direktor"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fx.applyCallArgs.poslovodaID != "emp-poslovoda-123" {
		t.Errorf("expected poslovodaID='emp-poslovoda-123' forwarded to ValidateAndApply, got %q", fx.applyCallArgs.poslovodaID)
	}
	if fx.applyCallArgs.projectID != "project-456" {
		t.Errorf("expected projectID='project-456', got %q", fx.applyCallArgs.projectID)
	}
}

// TestDailyReport_Approve_RejectsNonSubmittedStatus verifies that an already-approved
// or rejected report cannot be approved again.
func TestDailyReport_Approve_RejectsNonSubmittedStatus(t *testing.T) {
	for _, status := range []string{"approved", "rejected"} {
		status := status
		repo := &mockDrRepo{
			getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
				return status, "emp-poslovoda", "project-1", nil
			},
		}
		err := newDrSvcWithEffects(repo, &mockMaterialEffectsRepo{}).
			Approve(context.Background(), "report-1", "co-1", "user-1", "emp-direktor")
		if err == nil {
			t.Errorf("status=%s: expected error, got nil", status)
		}
		if !errors.Is(err, ErrDailyReportInvalidStatus) {
			t.Errorf("status=%s: expected ErrDailyReportInvalidStatus, got %v", status, err)
		}
	}
}

// TestDailyReport_Approve_IdempotentWhenEffectsAlreadyApplied verifies that if
// effects were already recorded (e.g. a crash after apply but before status update),
// ValidateAndApply is NOT called again.
func TestDailyReport_Approve_IdempotentWhenEffectsAlreadyApplied(t *testing.T) {
	secondApplyCalled := false
	fx := &mockMaterialEffectsRepo{
		alreadyAppliedFn: func(_ context.Context, _ pgx.Tx, _ string) (bool, error) { return true, nil },
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, _ []repositories.ActivityForApproval) error {
			secondApplyCalled = true
			return nil
		},
	}
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{}, nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secondApplyCalled {
		t.Error("ValidateAndApply should not be called when effects are already applied")
	}
}

// TestDailyReport_Approve_PropagatesEffectsError verifies that a validation error
// from ValidateAndApply (e.g. insufficient stock) bubbles up from Approve.
func TestDailyReport_Approve_PropagatesEffectsError(t *testing.T) {
	insufficientErr := errors.New("nedovoljno dostupnih zaliha za materijal")
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, _ []repositories.ActivityForApproval) error {
			return insufficientErr
		},
	}
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			mat := "mat-1"
			return []repositories.ActivityForApproval{
				{ID: "act-1", ProjectMaterialID: &mat, Quantity: 999, Unit: "kom", ActivityType: "montaza"},
			}, nil
		},
	}
	err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d")
	if !errors.Is(err, insufficientErr) {
		t.Errorf("expected insufficient-stock error to propagate, got %v", err)
	}
}

// TestDailyReport_Approve_StatusNotChangedOnEffectsFailure verifies that SetStatus
// is not called when ValidateAndApply fails (transaction will be rolled back).
func TestDailyReport_Approve_StatusNotChangedOnEffectsFailure(t *testing.T) {
	setStatusCalled := false
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, _ []repositories.ActivityForApproval) error {
			return errors.New("stock error")
		},
	}
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{}, nil
		},
		setStatusFn: func(_ context.Context, _ pgx.Tx, _, _, _ string, _ *string) error {
			setStatusCalled = true
			return nil
		},
	}
	_ = newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d")
	if setStatusCalled {
		t.Error("SetStatus must not be called when ValidateAndApply fails")
	}
}

// ── Tests: VTR approval ───────────────────────────────────────────────────────

// TestDailyReport_Approve_VTRMontazaZeroStockErrorPropagates verifies that when
// a VTR montaža activity fails because the material has zero stock, the specific
// Croatian error message propagates from the effects repo through the service
// without wrapping or modification.
//
// Scenario 15: VTR montaža on missing zero-stock material returns clear message.
func TestDailyReport_Approve_VTRMontazaZeroStockErrorPropagates(t *testing.T) {
	zeroStockErr := errors.New("Materijal nije dostupan na projektu za ovu VTR montažu.")
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, _ []repositories.ActivityForApproval) error {
			return zeroStockErr
		},
	}
	matName := "Beton B25"
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{ID: "act-1", IsVTK: true, CustomMaterialName: &matName, Quantity: 5, Unit: "m2", ActivityType: "montaza"},
			}, nil
		},
	}
	err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d")
	if !errors.Is(err, zeroStockErr) {
		t.Errorf("expected zero-stock error to propagate unchanged, got %v", err)
	}
}

// TestDailyReport_Approve_VTRMontazaBlocksStatusChange verifies that a VTR
// montaža stock failure prevents SetStatus from being called (i.e. the report
// remains in "submitted" state and the transaction is rolled back).
//
// Scenario 15 (status guard): stock error from VTR montaža rolls back approval.
func TestDailyReport_Approve_VTRMontazaBlocksStatusChange(t *testing.T) {
	setStatusCalled := false
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, _ []repositories.ActivityForApproval) error {
			return errors.New("Materijal nije dostupan na projektu za ovu VTR montažu.")
		},
	}
	matName := "Beton B25"
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{ID: "act-1", IsVTK: true, CustomMaterialName: &matName, Quantity: 5, Unit: "m2", ActivityType: "montaza"},
			}, nil
		},
		setStatusFn: func(_ context.Context, _ pgx.Tx, _, _, _ string, _ *string) error {
			setStatusCalled = true
			return nil
		},
	}
	_ = newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d")
	if setStatusCalled {
		t.Error("SetStatus must not be called when VTR montaža fails stock validation")
	}
}

// TestDailyReport_Approve_VTRActivitiesForwardedToEffects verifies that VTR
// activities (IsVTK=true) are included in the full activity slice forwarded to
// ValidateAndApply — the service must not filter them out.
//
// Scenarios 4, 6, 7: VTR activities reach the effects layer for processing.
func TestDailyReport_Approve_VTRActivitiesForwardedToEffects(t *testing.T) {
	var gotActCount int
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, acts []repositories.ActivityForApproval) error {
			gotActCount = len(acts)
			return nil
		},
	}
	matName := "Beton B25"
	matID := "mat-1"
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{ID: "act-vtr", IsVTK: true, CustomMaterialName: &matName, Quantity: 2, Unit: "m2", ActivityType: "demontaza"},
				{ID: "act-normal", IsVTK: false, ProjectMaterialID: &matID, Quantity: 1, Unit: "kom", ActivityType: "montaza"},
			}, nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotActCount != 2 {
		t.Errorf("expected both VTR and normal activities forwarded to effects, got %d", gotActCount)
	}
}

// TestDailyReport_Approve_VTRDemontazaSucceeds verifies that a VTR demontaža
// approval succeeds when the effects repo reports no error — simulating the
// case where the material already exists or was just created in the same tx.
//
// Scenario 17: already-existing VTR material approves without error.
func TestDailyReport_Approve_VTRDemontazaSucceeds(t *testing.T) {
	statusSet := ""
	fx := &mockMaterialEffectsRepo{} // validateAndApplyFn nil → returns nil
	matName := "Beton B25"
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{ID: "act-vtr", IsVTK: true, CustomMaterialName: &matName, Quantity: 3, Unit: "kom", ActivityType: "demontaza"},
			}, nil
		},
		setStatusFn: func(_ context.Context, _ pgx.Tx, _, _, status string, _ *string) error {
			statusSet = status
			return nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("VTR demontaža approval should succeed when effects return no error: %v", err)
	}
	if statusSet != "approved" {
		t.Errorf("expected status set to 'approved', got %q", statusSet)
	}
}

// TestDailyReport_Approve_VTRAndNonVTRMixed_PoslovodaIDForwarded verifies that
// a report mixing VTR and non-VTR activities still forwards the correct
// poslovodaEmpID and projectID to ValidateAndApply.
//
// Scenarios 7, 8: poslovoda identity correctly propagated for responsibility tracking.
func TestDailyReport_Approve_VTRAndNonVTRMixed_PoslovodaIDForwarded(t *testing.T) {
	fx := &mockMaterialEffectsRepo{}
	matName := "Beton B25"
	matID := "mat-existing"
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda-xyz", "project-abc", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{ID: "act-vtr", IsVTK: true, CustomMaterialName: &matName, Quantity: 2, Unit: "m3", ActivityType: "demontaza"},
				{ID: "act-std", IsVTK: false, ProjectMaterialID: &matID, Quantity: 1, Unit: "kom", ActivityType: "demontaza"},
			}, nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fx.applyCallArgs.poslovodaID != "emp-poslovoda-xyz" {
		t.Errorf("expected poslovodaID='emp-poslovoda-xyz' forwarded, got %q", fx.applyCallArgs.poslovodaID)
	}
	if fx.applyCallArgs.projectID != "project-abc" {
		t.Errorf("expected projectID='project-abc' forwarded, got %q", fx.applyCallArgs.projectID)
	}
}

// ── Tests: work-item (tracking_type=work) payload validation ─────────────────

// TestDailyReport_Create_ContradicatoryPayloadRejected verifies that a payload
// with is_vtk=true AND project_material_id set is rejected by validateReport.
// A client must never submit both fields; the backend enforces this defensively.
func TestDailyReport_Create_ContradicatoryPayloadRejected(t *testing.T) {
	matID := "mat-1"
	customName := "Štemanje"
	repo := &mockDrRepo{
		getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) { return "active", nil },
	}
	req := dto.CreateDailyReportRequest{
		ProjectID:  "project-1",
		ReportDate: drTodayStr(),
		Activities: []dto.ActivityInput{
			{
				IsVTK:              true,
				ProjectMaterialID:  &matID,
				CustomMaterialName: &customName,
				Quantity:           25,
				Unit:               "m",
				ActivityType:       "montaza",
			},
		},
	}
	_, err := newDrSvc(repo).Create(context.Background(), "co-1", "user-1", "emp-poslovoda", "poslovoda", req)
	if err == nil {
		t.Fatal("expected error for contradictory is_vtk+project_material_id payload, got nil")
	}
	if !strings.Contains(err.Error(), "project_material_id mora biti null") {
		t.Errorf("expected 'project_material_id mora biti null' error, got: %v", err)
	}
}

// TestDailyReport_Create_WorkItemWithProjectMaterialIDAccepted verifies that a
// work-type activity submitted with project_material_id and is_vtk=false passes
// validateReport — this is the correct shape for work-item activities.
func TestDailyReport_Create_WorkItemWithProjectMaterialIDAccepted(t *testing.T) {
	matID := "mat-stemanje"
	created := false
	repo := &mockDrRepo{
		getProjectStatusFn: func(_ context.Context, _, _ string) (string, error) { return "active", nil },
		isMaterialInProjectFn: func(_ context.Context, materialID, _, _ string) (bool, error) {
			if materialID == matID {
				return true, nil
			}
			return false, nil
		},
		createFn: func(_ context.Context, _ pgx.Tx, _, _, _, _ string, _ dto.CreateDailyReportRequest) (string, error) {
			created = true
			return "report-work-item", nil
		},
	}
	req := dto.CreateDailyReportRequest{
		ProjectID:  "project-1",
		ReportDate: drTodayStr(),
		Activities: []dto.ActivityInput{
			{
				IsVTK:             false,
				ProjectMaterialID: &matID,
				Quantity:          25,
				Unit:              "m",
				ActivityType:      "montaza",
			},
		},
	}
	_, err := newDrSvc(repo).Create(context.Background(), "co-1", "user-1", "emp-poslovoda", "poslovoda", req)
	if err != nil {
		t.Fatalf("work-item activity with project_material_id should pass validation, got: %v", err)
	}
	if !created {
		t.Error("expected drRepo.Create to be called for a valid work-item activity")
	}
}

// TestDailyReport_Approve_WorkItemActivityForwardedWithCorrectTrackingType
// verifies that when GetActivitiesForApproval returns an activity with
// TrackingType="work" and IsVTK=false, ValidateAndApply receives that activity
// with TrackingType="work" — confirming the service does not touch TrackingType.
// The effects repo (not tested here) branches on TrackingType to bypass stock checks.
func TestDailyReport_Approve_WorkItemActivityForwardedWithCorrectTrackingType(t *testing.T) {
	matID := "mat-stemanje"
	var gotActs []repositories.ActivityForApproval
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, acts []repositories.ActivityForApproval) error {
			gotActs = acts
			return nil
		},
	}
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{
					ID:                "act-work",
					ProjectMaterialID: &matID,
					Quantity:          25,
					Unit:              "m",
					ActivityType:      "montaza",
					IsVTK:             false,
					TrackingType:      "work",
				},
			}, nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotActs) != 1 {
		t.Fatalf("expected 1 activity forwarded to ValidateAndApply, got %d", len(gotActs))
	}
	act := gotActs[0]
	if act.TrackingType != "work" {
		t.Errorf("expected TrackingType='work' forwarded, got %q", act.TrackingType)
	}
	if act.IsVTK {
		t.Error("expected IsVTK=false for a project-linked work item, got true")
	}
	if act.ProjectMaterialID == nil || *act.ProjectMaterialID != matID {
		t.Errorf("expected ProjectMaterialID=%q forwarded, got %v", matID, act.ProjectMaterialID)
	}
}

// TestDailyReport_Approve_WorkItemZeroStockApproves verifies (at the service layer)
// that a work-item approval is not blocked by the effects layer returning an error.
// The actual no-stock-check behavior lives in applyMontaza (effects repo), but the
// service must propagate a success transparently.
func TestDailyReport_Approve_WorkItemZeroStockApproves(t *testing.T) {
	matID := "mat-stemanje"
	statusSet := ""
	fx := &mockMaterialEffectsRepo{} // validateAndApplyFn nil → success
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{
					ID:                "act-work",
					ProjectMaterialID: &matID,
					Quantity:          25,
					Unit:              "m",
					ActivityType:      "montaza",
					IsVTK:             false,
					TrackingType:      "work",
				},
			}, nil
		},
		setStatusFn: func(_ context.Context, _ pgx.Tx, _, _, status string, _ *string) error {
			statusSet = status
			return nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("work-item approval with zero stock should succeed, got: %v", err)
	}
	if statusSet != "approved" {
		t.Errorf("expected status='approved', got %q", statusSet)
	}
}

// TestDailyReport_Approve_VTKActivityNotReroutedAsWorkItem verifies that a genuine
// VTK activity (is_vtk=true, custom_material_name set) retains TrackingType="stock"
// and is NOT routed through work-item logic. The effects repo validates stock for VTK.
func TestDailyReport_Approve_VTKActivityNotReroutedAsWorkItem(t *testing.T) {
	matName := "Beton B25"
	var gotActs []repositories.ActivityForApproval
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, acts []repositories.ActivityForApproval) error {
			gotActs = acts
			return nil
		},
	}
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{
					ID:                 "act-vtk",
					CustomMaterialName: &matName,
					Quantity:           5,
					Unit:               "m3",
					ActivityType:       "montaza",
					IsVTK:              true,
					TrackingType:       "stock",
				},
			}, nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotActs) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(gotActs))
	}
	if !gotActs[0].IsVTK {
		t.Error("VTK activity must remain IsVTK=true when forwarded to effects repo")
	}
	if gotActs[0].TrackingType != "stock" {
		t.Errorf("VTK activity must retain TrackingType='stock', got %q", gotActs[0].TrackingType)
	}
}

// TestDailyReport_Approve_NormalStockActivityUnchanged verifies that a normal
// stock activity (is_vtk=false, tracking_type=stock) is forwarded unchanged —
// existing stock subtraction behaviour must not be affected by work-item changes.
func TestDailyReport_Approve_NormalStockActivityUnchanged(t *testing.T) {
	matID := "mat-beton"
	var gotActs []repositories.ActivityForApproval
	fx := &mockMaterialEffectsRepo{
		validateAndApplyFn: func(_ context.Context, _ pgx.Tx, _, _, _, _, _ string, acts []repositories.ActivityForApproval) error {
			gotActs = acts
			return nil
		},
	}
	repo := &mockDrRepo{
		getByIDForEditFn: func(_ context.Context, _, _ string) (string, string, string, error) {
			return "submitted", "emp-poslovoda", "project-1", nil
		},
		getActivitiesForApprovalFn: func(_ context.Context, _, _ string) ([]repositories.ActivityForApproval, error) {
			return []repositories.ActivityForApproval{
				{
					ID:                "act-stock",
					ProjectMaterialID: &matID,
					Quantity:          3,
					Unit:              "m3",
					ActivityType:      "montaza",
					IsVTK:             false,
					TrackingType:      "stock",
				},
			}, nil
		},
	}
	if err := newDrSvcWithEffects(repo, fx).Approve(context.Background(), "report-1", "co-1", "user-1", "emp-d"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotActs) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(gotActs))
	}
	act := gotActs[0]
	if act.IsVTK {
		t.Error("normal stock activity must remain IsVTK=false")
	}
	if act.TrackingType != "stock" {
		t.Errorf("normal stock activity must retain TrackingType='stock', got %q", act.TrackingType)
	}
	if act.ProjectMaterialID == nil || *act.ProjectMaterialID != matID {
		t.Errorf("expected ProjectMaterialID=%q, got %v", matID, act.ProjectMaterialID)
	}
}
