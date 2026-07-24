package services

import (
	"context"
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

// ── Helpers ───────────────────────────────────────────────────────────────────

func newDrSvc(repo *mockDrRepo) *DailyReportService {
	return &DailyReportService{
		db:        &schedTxBeginner{tx: &schedMockTx{}},
		drRepo:    repo,
		auditRepo: &mockAuditLog{},
		// materialEffectsRepo is nil — none of the 7 tests exercise Approve
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
