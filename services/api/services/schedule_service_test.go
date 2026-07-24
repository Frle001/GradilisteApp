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

// ── Test infrastructure ───────────────────────────────────────────────────────

type schedMockTx struct {
	pgx.Tx    // nil embed; only Commit/Rollback are called
	committed bool
	commitErr error
}

func (m *schedMockTx) Commit(_ context.Context) error   { m.committed = true; return m.commitErr }
func (m *schedMockTx) Rollback(_ context.Context) error { return nil }

type schedTxBeginner struct {
	tx  pgx.Tx
	err error
}

func (b *schedTxBeginner) Begin(_ context.Context) (pgx.Tx, error) { return b.tx, b.err }

type mockSchedRepo struct {
	projectBelongsFn    func(context.Context, string, string) (bool, error)
	createShiftFn       func(context.Context, pgx.Tx, string, string, string, *string, *string, *string, string) (string, error)
	getShiftFn          func(context.Context, string, string) (*repositories.ShiftRow, error)
	getShiftForUpdateFn func(context.Context, pgx.Tx, string, string) (*repositories.ShiftRow, error)
	listShiftsFn        func(context.Context, string, string, string, *string) ([]repositories.ShiftRow, error)
	updateShiftFn       func(context.Context, pgx.Tx, string, string, *string, *string, *string) error
	cancelShiftFn       func(context.Context, pgx.Tx, string, string, string) error
	listAssignmentsFn   func(context.Context, string, string) ([]repositories.ShiftAssignmentRow, error)
	deleteAssignmentsFn func(context.Context, pgx.Tx, string) error
	createAssignmentFn  func(context.Context, pgx.Tx, string, string, string, string, bool, *string) error
	employeesForDateFn  func(context.Context, string, string, string) ([]repositories.EmployeeForDateRow, error)
}

func (m *mockSchedRepo) ProjectBelongsToCompany(ctx context.Context, companyID, projectID string) (bool, error) {
	if m.projectBelongsFn != nil {
		return m.projectBelongsFn(ctx, companyID, projectID)
	}
	return true, nil
}

func (m *mockSchedRepo) CreateShift(ctx context.Context, tx pgx.Tx, companyID, projectID, shiftDate string, startTime, endTime *string, notes *string, createdBy string) (string, error) {
	if m.createShiftFn != nil {
		return m.createShiftFn(ctx, tx, companyID, projectID, shiftDate, startTime, endTime, notes, createdBy)
	}
	return "shift-id", nil
}

func (m *mockSchedRepo) GetShift(ctx context.Context, companyID, shiftID string) (*repositories.ShiftRow, error) {
	if m.getShiftFn != nil {
		return m.getShiftFn(ctx, companyID, shiftID)
	}
	return testSchedShiftRow(shiftID), nil
}

func (m *mockSchedRepo) GetShiftForUpdate(ctx context.Context, tx pgx.Tx, companyID, shiftID string) (*repositories.ShiftRow, error) {
	if m.getShiftForUpdateFn != nil {
		return m.getShiftForUpdateFn(ctx, tx, companyID, shiftID)
	}
	return testSchedShiftRow(shiftID), nil
}

func (m *mockSchedRepo) ListShifts(ctx context.Context, companyID, dateFrom, dateTo string, projectID *string) ([]repositories.ShiftRow, error) {
	if m.listShiftsFn != nil {
		return m.listShiftsFn(ctx, companyID, dateFrom, dateTo, projectID)
	}
	return nil, nil
}

func (m *mockSchedRepo) UpdateShift(ctx context.Context, tx pgx.Tx, shiftID, shiftDate string, startTime, endTime *string, notes *string) error {
	if m.updateShiftFn != nil {
		return m.updateShiftFn(ctx, tx, shiftID, shiftDate, startTime, endTime, notes)
	}
	return nil
}

func (m *mockSchedRepo) CancelShift(ctx context.Context, tx pgx.Tx, companyID, shiftID, cancelledBy string) error {
	if m.cancelShiftFn != nil {
		return m.cancelShiftFn(ctx, tx, companyID, shiftID, cancelledBy)
	}
	return nil
}

func (m *mockSchedRepo) ListAssignments(ctx context.Context, companyID, shiftID string) ([]repositories.ShiftAssignmentRow, error) {
	if m.listAssignmentsFn != nil {
		return m.listAssignmentsFn(ctx, companyID, shiftID)
	}
	return nil, nil
}

func (m *mockSchedRepo) DeleteAssignments(ctx context.Context, tx pgx.Tx, shiftID string) error {
	if m.deleteAssignmentsFn != nil {
		return m.deleteAssignmentsFn(ctx, tx, shiftID)
	}
	return nil
}

func (m *mockSchedRepo) CreateAssignment(ctx context.Context, tx pgx.Tx, companyID, shiftID, employeeID, assignedBy string, overlapOverridden bool, overriddenBy *string) error {
	if m.createAssignmentFn != nil {
		return m.createAssignmentFn(ctx, tx, companyID, shiftID, employeeID, assignedBy, overlapOverridden, overriddenBy)
	}
	return nil
}

func (m *mockSchedRepo) EmployeesForDate(ctx context.Context, companyID, shiftDate, excludeShiftID string) ([]repositories.EmployeeForDateRow, error) {
	if m.employeesForDateFn != nil {
		return m.employeesForDateFn(ctx, companyID, shiftDate, excludeShiftID)
	}
	return nil, nil
}

func testSchedShiftRow(id string) *repositories.ShiftRow {
	st := "08:00"
	et := "16:00"
	return &repositories.ShiftRow{
		ID:        id,
		CompanyID: "company-1",
		ProjectID: "project-1",
		ShiftDate: "2025-01-15",
		StartTime: &st,
		EndTime:   &et,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newSchedSvc(repo *mockSchedRepo) *ScheduleService {
	return &ScheduleService{
		db:   &schedTxBeginner{tx: &schedMockTx{}},
		repo: repo,
	}
}

// ── CreateShift ───────────────────────────────────────────────────────────────

func TestSchedule_CreateShift_Success(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	shift, err := svc.CreateShift(context.Background(), "company-1", "project-1", "user-1",
		&dto.CreateShiftRequest{ProjectID: "project-1", ShiftDate: "2025-01-15"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shift == nil || shift.Status != "active" {
		t.Errorf("expected active shift, got %+v", shift)
	}
}

func TestSchedule_CreateShift_InvalidDate(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	_, err := svc.CreateShift(context.Background(), "company-1", "project-1", "user-1",
		&dto.CreateShiftRequest{ProjectID: "project-1", ShiftDate: "not-a-date"})
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestSchedule_CreateShift_ProjectNotFound(t *testing.T) {
	repo := &mockSchedRepo{
		projectBelongsFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := newSchedSvc(repo)
	_, err := svc.CreateShift(context.Background(), "company-1", "project-1", "user-1",
		&dto.CreateShiftRequest{ProjectID: "project-1", ShiftDate: "2025-01-15"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSchedule_CreateShift_SameDayAssignmentAllowed(t *testing.T) {
	var created []string
	repo := &mockSchedRepo{
		createAssignmentFn: func(_ context.Context, _ pgx.Tx, _, _, employeeID, _ string, _ bool, _ *string) error {
			created = append(created, employeeID)
			return nil
		},
	}
	svc := newSchedSvc(repo)
	shift, err := svc.CreateShift(context.Background(), "company-1", "project-1", "user-1",
		&dto.CreateShiftRequest{ProjectID: "project-1", ShiftDate: "2025-01-15", EmployeeIDs: []string{"emp-1"}})
	if err != nil {
		t.Fatalf("expected no error for same-day assignment, got %v", err)
	}
	if shift == nil {
		t.Fatal("expected non-nil shift")
	}
	if len(created) != 1 || created[0] != "emp-1" {
		t.Errorf("expected emp-1 to be assigned, got %v", created)
	}
}

// ── GetShift ──────────────────────────────────────────────────────────────────

func TestSchedule_GetShift_Success(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	shift, err := svc.GetShift(context.Background(), "company-1", "shift-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shift.ID != "shift-abc" {
		t.Errorf("expected shift-abc, got %s", shift.ID)
	}
}

func TestSchedule_GetShift_NotFound(t *testing.T) {
	repo := &mockSchedRepo{
		getShiftFn: func(_ context.Context, _, _ string) (*repositories.ShiftRow, error) {
			return nil, repositories.ErrNotFound
		},
	}
	svc := newSchedSvc(repo)
	_, err := svc.GetShift(context.Background(), "company-1", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── ListShifts ────────────────────────────────────────────────────────────────

func TestSchedule_ListShifts_Success(t *testing.T) {
	repo := &mockSchedRepo{
		listShiftsFn: func(_ context.Context, _, _, _ string, _ *string) ([]repositories.ShiftRow, error) {
			return []repositories.ShiftRow{*testSchedShiftRow("s1"), *testSchedShiftRow("s2")}, nil
		},
	}
	svc := newSchedSvc(repo)
	shifts, err := svc.ListShifts(context.Background(), "company-1", "2025-01-13", "2025-01-19", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shifts) != 2 {
		t.Errorf("expected 2 shifts, got %d", len(shifts))
	}
}

func TestSchedule_ListShifts_InvalidDateFrom(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	_, err := svc.ListShifts(context.Background(), "company-1", "bad", "2025-01-19", nil)
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

// ── UpdateShift ───────────────────────────────────────────────────────────────

func TestSchedule_UpdateShift_Success(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	shift, err := svc.UpdateShift(context.Background(), "company-1", "shift-1",
		&dto.UpdateShiftRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shift == nil {
		t.Fatal("expected non-nil shift")
	}
}

func TestSchedule_UpdateShift_CancelledShift(t *testing.T) {
	repo := &mockSchedRepo{
		getShiftFn: func(_ context.Context, _, id string) (*repositories.ShiftRow, error) {
			r := testSchedShiftRow(id)
			r.Status = "cancelled"
			return r, nil
		},
	}
	svc := newSchedSvc(repo)
	_, err := svc.UpdateShift(context.Background(), "company-1", "shift-1", &dto.UpdateShiftRequest{})
	if !errors.Is(err, ErrShiftCancelled) {
		t.Errorf("expected ErrShiftCancelled, got %v", err)
	}
}

func TestSchedule_UpdateShift_NotFound(t *testing.T) {
	repo := &mockSchedRepo{
		getShiftFn: func(_ context.Context, _, _ string) (*repositories.ShiftRow, error) {
			return nil, repositories.ErrNotFound
		},
	}
	svc := newSchedSvc(repo)
	_, err := svc.UpdateShift(context.Background(), "company-1", "missing", &dto.UpdateShiftRequest{})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── CancelShift ───────────────────────────────────────────────────────────────

func TestSchedule_CancelShift_Success(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	if err := svc.CancelShift(context.Background(), "company-1", "shift-1", "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchedule_CancelShift_AlreadyCancelled(t *testing.T) {
	repo := &mockSchedRepo{
		getShiftFn: func(_ context.Context, _, id string) (*repositories.ShiftRow, error) {
			r := testSchedShiftRow(id)
			r.Status = "cancelled"
			return r, nil
		},
	}
	svc := newSchedSvc(repo)
	if err := svc.CancelShift(context.Background(), "company-1", "shift-1", "user-1"); !errors.Is(err, ErrShiftCancelled) {
		t.Errorf("expected ErrShiftCancelled, got %v", err)
	}
}

func TestSchedule_CancelShift_NotFound(t *testing.T) {
	repo := &mockSchedRepo{
		getShiftFn: func(_ context.Context, _, _ string) (*repositories.ShiftRow, error) {
			return nil, repositories.ErrNotFound
		},
	}
	svc := newSchedSvc(repo)
	if err := svc.CancelShift(context.Background(), "company-1", "missing", "user-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── SyncAssignments ───────────────────────────────────────────────────────────

func TestSchedule_SyncAssignments_NoConflicts_Success(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	result, err := svc.SyncAssignments(context.Background(), "company-1", "shift-1", "user-1",
		&dto.AssignEmployeesRequest{EmployeeIDs: []string{"emp-1", "emp-2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequiresOverride {
		t.Error("RequiresOverride should always be false")
	}
}

func TestSchedule_SyncAssignments_SameDayAssignmentAllowed(t *testing.T) {
	var created []string
	repo := &mockSchedRepo{
		createAssignmentFn: func(_ context.Context, _ pgx.Tx, _, _, employeeID, _ string, _ bool, _ *string) error {
			created = append(created, employeeID)
			return nil
		},
	}
	svc := newSchedSvc(repo)
	result, err := svc.SyncAssignments(context.Background(), "company-1", "shift-1", "user-1",
		&dto.AssignEmployeesRequest{EmployeeIDs: []string{"emp-1"}})
	if err != nil {
		t.Fatalf("expected no error for same-day assignment, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(created) != 1 || created[0] != "emp-1" {
		t.Errorf("expected emp-1 to be assigned, got %v", created)
	}
}

func TestSchedule_SyncAssignments_CancelledShift(t *testing.T) {
	repo := &mockSchedRepo{
		getShiftFn: func(_ context.Context, _, id string) (*repositories.ShiftRow, error) {
			r := testSchedShiftRow(id)
			r.Status = "cancelled"
			return r, nil
		},
	}
	svc := newSchedSvc(repo)
	_, err := svc.SyncAssignments(context.Background(), "company-1", "shift-1", "user-1",
		&dto.AssignEmployeesRequest{EmployeeIDs: []string{"emp-1"}})
	if !errors.Is(err, ErrShiftCancelled) {
		t.Errorf("expected ErrShiftCancelled, got %v", err)
	}
}

func TestSchedule_SyncAssignments_DeduplicatesEmployeeIDs(t *testing.T) {
	var created []string
	repo := &mockSchedRepo{
		createAssignmentFn: func(_ context.Context, _ pgx.Tx, _, _, employeeID, _ string, _ bool, _ *string) error {
			created = append(created, employeeID)
			return nil
		},
	}
	svc := newSchedSvc(repo)
	_, err := svc.SyncAssignments(context.Background(), "company-1", "shift-1", "user-1",
		&dto.AssignEmployeesRequest{EmployeeIDs: []string{"emp-1", "emp-1", "emp-2"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(created) != 2 {
		t.Errorf("expected 2 unique assignments, got %d: %v", len(created), created)
	}
}

// ── DuplicateShift ────────────────────────────────────────────────────────────

func TestSchedule_DuplicateShift_Success(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	shift, err := svc.DuplicateShift(context.Background(), "company-1", "shift-1", "user-1",
		&dto.DuplicateShiftRequest{TargetDate: "2025-01-20"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shift == nil {
		t.Fatal("expected non-nil shift")
	}
}

func TestSchedule_DuplicateShift_NotFound(t *testing.T) {
	repo := &mockSchedRepo{
		getShiftFn: func(_ context.Context, _, _ string) (*repositories.ShiftRow, error) {
			return nil, repositories.ErrNotFound
		},
	}
	svc := newSchedSvc(repo)
	_, err := svc.DuplicateShift(context.Background(), "company-1", "missing", "user-1",
		&dto.DuplicateShiftRequest{TargetDate: "2025-01-20"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSchedule_DuplicateShift_InvalidDate(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	_, err := svc.DuplicateShift(context.Background(), "company-1", "shift-1", "user-1",
		&dto.DuplicateShiftRequest{TargetDate: "not-a-date"})
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

// ── CopyDay ───────────────────────────────────────────────────────────────────

func TestSchedule_CopyDay_Success(t *testing.T) {
	repo := &mockSchedRepo{
		listShiftsFn: func(_ context.Context, _, _, _ string, _ *string) ([]repositories.ShiftRow, error) {
			return []repositories.ShiftRow{*testSchedShiftRow("s1")}, nil
		},
	}
	svc := newSchedSvc(repo)
	result, err := svc.CopyDay(context.Background(), "company-1", "user-1",
		&dto.CopyDayRequest{SourceDate: "2025-01-15", TargetDate: "2025-01-16"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created shift, got %d", result.Created)
	}
}

func TestSchedule_CopyDay_SameDateError(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	_, err := svc.CopyDay(context.Background(), "company-1", "user-1",
		&dto.CopyDayRequest{SourceDate: "2025-01-15", TargetDate: "2025-01-15"})
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError for same-date copy, got %v", err)
	}
}

// ── CopyWeek ──────────────────────────────────────────────────────────────────

func TestSchedule_CopyWeek_Success(t *testing.T) {
	repo := &mockSchedRepo{
		listShiftsFn: func(_ context.Context, _, _, _ string, _ *string) ([]repositories.ShiftRow, error) {
			r := testSchedShiftRow("s1")
			r.ShiftDate = "2025-01-13"
			return []repositories.ShiftRow{*r}, nil
		},
	}
	svc := newSchedSvc(repo)
	result, err := svc.CopyWeek(context.Background(), "company-1", "user-1",
		&dto.CopyWeekRequest{SourceWeekStart: "2025-01-13", TargetWeekStart: "2025-01-20"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created shift, got %d", result.Created)
	}
}

// ── EmployeesForDate ──────────────────────────────────────────────────────────

func TestSchedule_EmployeesForDate_Success(t *testing.T) {
	proj := "Projekt A"
	sid := "shift-x"
	repo := &mockSchedRepo{
		employeesForDateFn: func(_ context.Context, _, _, _ string) ([]repositories.EmployeeForDateRow, error) {
			return []repositories.EmployeeForDateRow{
				{ID: "emp-1", Name: "Ana Anić", Role: "radnik", Assigned: false},
				{ID: "emp-2", Name: "Ivo Ivić", Role: "poslovoda", Assigned: true, ShiftID: &sid, ProjectName: &proj},
			}, nil
		},
	}
	svc := newSchedSvc(repo)
	items, err := svc.EmployeesForDate(context.Background(), "company-1", "2025-01-15",
		"00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Assigned {
		t.Error("emp-1 should not be assigned")
	}
	if !items[1].Assigned {
		t.Error("emp-2 should be assigned")
	}
	if items[1].ProjectName == nil || *items[1].ProjectName != "Projekt A" {
		t.Errorf("expected ProjectName='Projekt A', got %v", items[1].ProjectName)
	}
}

func TestSchedule_EmployeesForDate_InvalidDate(t *testing.T) {
	svc := newSchedSvc(&mockSchedRepo{})
	_, err := svc.EmployeesForDate(context.Background(), "company-1", "bad-date",
		"00000000-0000-0000-0000-000000000000")
	if AsValidationError(err) == nil {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestSchedule_EmployeesForDate_AssignedShiftIDAndProjectForwarded(t *testing.T) {
	proj := "Projekt B"
	sid := "shift-y"
	repo := &mockSchedRepo{
		employeesForDateFn: func(_ context.Context, _, _, _ string) ([]repositories.EmployeeForDateRow, error) {
			return []repositories.EmployeeForDateRow{
				{ID: "emp-3", Name: "Marko Marić", Role: "radnik", Assigned: true, ShiftID: &sid, ProjectName: &proj},
			}, nil
		},
	}
	svc := newSchedSvc(repo)
	items, err := svc.EmployeesForDate(context.Background(), "company-1", "2025-01-15",
		"00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	emp := items[0]
	if !emp.Assigned {
		t.Error("employee should be assigned=true")
	}
	if emp.ShiftID == nil || *emp.ShiftID != "shift-y" {
		t.Errorf("expected ShiftID=shift-y, got %v", emp.ShiftID)
	}
	if emp.ProjectName == nil || *emp.ProjectName != "Projekt B" {
		t.Errorf("expected ProjectName=Projekt B, got %v", emp.ProjectName)
	}
}

func TestSchedule_EmployeesForDate_ExcludeShiftIDForwarded(t *testing.T) {
	var capturedExclude string
	repo := &mockSchedRepo{
		employeesForDateFn: func(_ context.Context, _, _, excludeID string) ([]repositories.EmployeeForDateRow, error) {
			capturedExclude = excludeID
			return nil, nil
		},
	}
	svc := newSchedSvc(repo)
	_, err := svc.EmployeesForDate(context.Background(), "company-1", "2025-01-15", "shift-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedExclude != "shift-abc" {
		t.Errorf("expected excludeShiftID=shift-abc forwarded to repo, got %q", capturedExclude)
	}
}

// ── Schedule visibility: no role-based filtering ──────────────────────────────

// TestSchedule_ListShifts_CompanyIDForwardedToRepo verifies that ListShifts
// forwards the caller's companyID to the repository unchanged, providing
// company isolation without any additional role-based scoping.
func TestSchedule_ListShifts_CompanyIDForwardedToRepo(t *testing.T) {
	var capturedCompanyID string
	repo := &mockSchedRepo{
		listShiftsFn: func(_ context.Context, companyID, _, _ string, _ *string) ([]repositories.ShiftRow, error) {
			capturedCompanyID = companyID
			return nil, nil
		},
	}
	if _, err := newSchedSvc(repo).ListShifts(context.Background(), "co-42", "2025-01-13", "2025-01-19", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedCompanyID != "co-42" {
		t.Errorf("expected repo to receive companyID='co-42', got %q", capturedCompanyID)
	}
}

// TestSchedule_ListShifts_AllProjectShiftsReturned verifies that ListShifts
// returns shifts from every project in the company, not only projects managed
// by the requesting user. The service performs no project-ownership filtering.
func TestSchedule_ListShifts_AllProjectShiftsReturned(t *testing.T) {
	row1 := testSchedShiftRow("s-proj-A")
	row1.ProjectID = "project-A"
	row2 := testSchedShiftRow("s-proj-B")
	row2.ProjectID = "project-B"

	repo := &mockSchedRepo{
		listShiftsFn: func(_ context.Context, _, _, _ string, _ *string) ([]repositories.ShiftRow, error) {
			return []repositories.ShiftRow{*row1, *row2}, nil
		},
	}
	shifts, err := newSchedSvc(repo).ListShifts(context.Background(), "company-1", "2025-01-13", "2025-01-19", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shifts) != 2 {
		t.Errorf("expected 2 shifts from different projects, got %d", len(shifts))
	}
	projects := map[string]bool{}
	for _, s := range shifts {
		projects[s.ProjectID] = true
	}
	if !projects["project-A"] || !projects["project-B"] {
		t.Errorf("expected shifts from both projects, got IDs: %v", projects)
	}
}
