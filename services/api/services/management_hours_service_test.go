package services

import (
	"context"
	"errors"
	"testing"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Mock repository ───────────────────────────────────────────────────────────

type mockMgmtRepo struct {
	upsertFn          func(context.Context, string, string, string, string, float64, *string, *string) (*dto.ManagementHoursEntry, error)
	getByIDFn         func(context.Context, string, string) (*dto.ManagementHoursEntry, error)
	listForEmployeeFn func(context.Context, string, string) ([]dto.ManagementHoursEntry, error)
	listForDirectorFn func(context.Context, string, string) ([]dto.ManagementHoursEntry, error)
	updateFn          func(context.Context, string, string, string, float64, *string) error
	deleteFn          func(context.Context, string, string, string) error
}

func (m *mockMgmtRepo) Upsert(ctx context.Context, companyID, employeeID, userID, workDate string, h float64, notes, projectID *string) (*dto.ManagementHoursEntry, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, companyID, employeeID, userID, workDate, h, notes, projectID)
	}
	return &dto.ManagementHoursEntry{ID: "entry-1", EmployeeID: employeeID}, nil
}
func (m *mockMgmtRepo) GetByID(ctx context.Context, companyID, entryID string) (*dto.ManagementHoursEntry, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, companyID, entryID)
	}
	return nil, repositories.ErrManagementHoursNotFound
}
func (m *mockMgmtRepo) ListForEmployee(ctx context.Context, companyID, employeeID string) ([]dto.ManagementHoursEntry, error) {
	if m.listForEmployeeFn != nil {
		return m.listForEmployeeFn(ctx, companyID, employeeID)
	}
	return []dto.ManagementHoursEntry{}, nil
}
func (m *mockMgmtRepo) ListForDirector(ctx context.Context, companyID, directorID string) ([]dto.ManagementHoursEntry, error) {
	if m.listForDirectorFn != nil {
		return m.listForDirectorFn(ctx, companyID, directorID)
	}
	return []dto.ManagementHoursEntry{}, nil
}
func (m *mockMgmtRepo) Update(ctx context.Context, companyID, ownerID, entryID string, h float64, notes *string) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, companyID, ownerID, entryID, h, notes)
	}
	return nil
}
func (m *mockMgmtRepo) Delete(ctx context.Context, companyID, ownerID, entryID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, companyID, ownerID, entryID)
	}
	return nil
}

// newMgmtSvc creates a service backed by the given mock.
func newMgmtSvc(r *mockMgmtRepo) *ManagementHoursService {
	return NewManagementHoursService(r)
}

// makeEntry builds a ManagementHoursEntry stub.
func makeEntry(employeeID, role string) *dto.ManagementHoursEntry {
	return &dto.ManagementHoursEntry{
		ID:           "entry-1",
		EmployeeID:   employeeID,
		EmployeeRole: role,
		WorkDate:     "2026-09-01",
		HoursWorked:  8,
	}
}

const (
	companyA = "company-a"
	companyB = "company-b"

	direktorA       = "direktor-a"
	direktorB       = "direktor-b"
	inzenjerA       = "inzenjer-a"
	inzenjerB       = "inzenjer-b"
	administracijaA = "admin-a"
	administracijaB = "admin-b"
)

// ── 1. Engineer creates own entry ─────────────────────────────────────────────

func TestMgmt_EngineerSubmit_OwnEntry_Allowed(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).Submit(context.Background(), companyA, inzenjerA, "user-1", "inzenjer",
		dto.SubmitManagementHoursRequest{WorkDate: "2026-09-01", HoursWorked: 8})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ── 2. Engineer reads own hours ───────────────────────────────────────────────

func TestMgmt_EngineerList_OwnHours_Allowed(t *testing.T) {
	called := false
	repo := &mockMgmtRepo{
		listForEmployeeFn: func(_ context.Context, companyID, empID string) ([]dto.ManagementHoursEntry, error) {
			called = true
			if companyID != companyA || empID != inzenjerA {
				t.Errorf("wrong args: company=%s emp=%s", companyID, empID)
			}
			return []dto.ManagementHoursEntry{*makeEntry(inzenjerA, "inzenjer")}, nil
		},
	}
	entries, err := newMgmtSvc(repo).List(context.Background(), companyA, inzenjerA, "inzenjer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("listForEmployee was not called")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

// ── 3. Engineer requests another engineer's hours ────────────────────────────

func TestMgmt_EngineerGetByID_AnotherEngineer_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(inzenjerB, "inzenjer"), nil // other engineer's entry
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, inzenjerA, "inzenjer", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ── 4. Engineer requests accounting hours ────────────────────────────────────

func TestMgmt_EngineerGetByID_Accounting_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(administracijaA, "administracija"), nil
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, inzenjerA, "inzenjer", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ── 5. Engineer requests director hours ──────────────────────────────────────

func TestMgmt_EngineerGetByID_Director_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(direktorA, "direktor"), nil
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, inzenjerA, "inzenjer", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ── 6. Accounting creates own entry ──────────────────────────────────────────

func TestMgmt_AccountingSubmit_OwnEntry_Allowed(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).Submit(context.Background(), companyA, administracijaA, "user-1", "administracija",
		dto.SubmitManagementHoursRequest{WorkDate: "2026-09-01", HoursWorked: 7.5})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ── 7. Accounting reads own hours ─────────────────────────────────────────────

func TestMgmt_AccountingList_OwnHours_Allowed(t *testing.T) {
	called := false
	repo := &mockMgmtRepo{
		listForEmployeeFn: func(_ context.Context, _, empID string) ([]dto.ManagementHoursEntry, error) {
			called = true
			return []dto.ManagementHoursEntry{*makeEntry(empID, "administracija")}, nil
		},
	}
	entries, err := newMgmtSvc(repo).List(context.Background(), companyA, administracijaA, "administracija")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("listForEmployee was not called")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

// ── 8. Accounting requests engineer hours ────────────────────────────────────

func TestMgmt_AccountingGetByID_Engineer_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(inzenjerA, "inzenjer"), nil
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, administracijaA, "administracija", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ── 9. Accounting requests another accounting employee ────────────────────────

func TestMgmt_AccountingGetByID_OtherAccounting_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(administracijaB, "administracija"), nil
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, administracijaA, "administracija", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ── 10. Accounting requests director hours ───────────────────────────────────

func TestMgmt_AccountingGetByID_Director_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(direktorA, "direktor"), nil
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, administracijaA, "administracija", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// ── 11. Director reads own hours ─────────────────────────────────────────────

func TestMgmt_DirectorGetByID_OwnEntry_Allowed(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(direktorA, "direktor"), nil
		},
	}
	entry, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, direktorA, "direktor", "entry-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.EmployeeID != direktorA {
		t.Errorf("wrong employee ID: %s", entry.EmployeeID)
	}
}

// ── 12. Director reads engineer hours ────────────────────────────────────────

func TestMgmt_DirectorGetByID_Engineer_Allowed(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(inzenjerA, "inzenjer"), nil
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, direktorA, "direktor", "entry-x")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ── 13. Director reads accounting hours ──────────────────────────────────────

func TestMgmt_DirectorGetByID_Accounting_Allowed(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(administracijaA, "administracija"), nil
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, direktorA, "direktor", "entry-x")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ── 14. Director reads another director's hours ──────────────────────────────

func TestMgmt_DirectorGetByID_AnotherDirector_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(direktorB, "direktor"), nil // different director
		},
	}
	_, err := newMgmtSvc(repo).GetByID(context.Background(), companyA, direktorA, "direktor", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden (director-to-director denied), got %v", err)
	}
}

// ── 15. Director cannot edit engineer/accounting hours ───────────────────────

func TestMgmt_DirectorUpdate_EngineerEntry_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(inzenjerA, "inzenjer"), nil
		},
	}
	err := newMgmtSvc(repo).Update(context.Background(), companyA, direktorA, "direktor", "entry-x",
		dto.UpdateManagementHoursRequest{HoursWorked: 8})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden (director may not edit engineer entry), got %v", err)
	}
}

func TestMgmt_DirectorDelete_AccountingEntry_Denied(t *testing.T) {
	repo := &mockMgmtRepo{
		getByIDFn: func(_ context.Context, _, _ string) (*dto.ManagementHoursEntry, error) {
			return makeEntry(administracijaA, "administracija"), nil
		},
	}
	err := newMgmtSvc(repo).Delete(context.Background(), companyA, direktorA, "direktor", "entry-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden (director may not delete accounting entry), got %v", err)
	}
}

// ── 16. Company scoping: director must not see another company's entries ──────

func TestMgmt_DirectorList_UsesCompanyScope(t *testing.T) {
	calledWith := ""
	repo := &mockMgmtRepo{
		listForDirectorFn: func(_ context.Context, companyID, _ string) ([]dto.ManagementHoursEntry, error) {
			calledWith = companyID
			return []dto.ManagementHoursEntry{}, nil
		},
	}
	_, err := newMgmtSvc(repo).List(context.Background(), companyA, direktorA, "direktor")
	if err != nil {
		t.Fatal(err)
	}
	if calledWith != companyA {
		t.Errorf("expected repo called with company=%s, got %s", companyA, calledWith)
	}
}

func TestMgmt_DirectorList_CannotSeeCompanyB(t *testing.T) {
	// Repository is scoped to companyA. Any entry returned belongs to companyA.
	// Verify the service does not accept a caller from companyB getting companyA entries.
	calledWithCompany := ""
	repo := &mockMgmtRepo{
		listForDirectorFn: func(_ context.Context, companyID, _ string) ([]dto.ManagementHoursEntry, error) {
			calledWithCompany = companyID
			return []dto.ManagementHoursEntry{}, nil
		},
	}
	_, err := newMgmtSvc(repo).List(context.Background(), companyB, direktorA, "direktor")
	if err != nil {
		t.Fatal(err)
	}
	if calledWithCompany != companyB {
		t.Errorf("service passed wrong company to repo: %s", calledWithCompany)
	}
}

// ── 17. radnik/poslovoda are blocked ─────────────────────────────────────────

func TestMgmt_RadnikSubmit_Blocked(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).Submit(context.Background(), companyA, "worker-1", "user-1", "radnik",
		dto.SubmitManagementHoursRequest{WorkDate: "2026-09-01", HoursWorked: 8})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for radnik, got %v", err)
	}
}

func TestMgmt_PoslovodaSubmit_Blocked(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).Submit(context.Background(), companyA, "plov-1", "user-1", "poslovoda",
		dto.SubmitManagementHoursRequest{WorkDate: "2026-09-01", HoursWorked: 8})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for poslovoda, got %v", err)
	}
}

func TestMgmt_RadnikList_Blocked(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).List(context.Background(), companyA, "worker-1", "radnik")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for radnik, got %v", err)
	}
}

// ── Input validation ──────────────────────────────────────────────────────────

func TestMgmt_Submit_InvalidDate_Rejected(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).Submit(context.Background(), companyA, inzenjerA, "user-1", "inzenjer",
		dto.SubmitManagementHoursRequest{WorkDate: "not-a-date", HoursWorked: 8})
	if err == nil {
		t.Fatal("expected validation error for bad date")
	}
}

func TestMgmt_Submit_HoursOver24_Rejected(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).Submit(context.Background(), companyA, inzenjerA, "user-1", "inzenjer",
		dto.SubmitManagementHoursRequest{WorkDate: "2026-09-01", HoursWorked: 25})
	if err == nil {
		t.Fatal("expected validation error for hours > 24")
	}
}

func TestMgmt_Submit_NegativeHours_Rejected(t *testing.T) {
	repo := &mockMgmtRepo{}
	_, err := newMgmtSvc(repo).Submit(context.Background(), companyA, inzenjerA, "user-1", "inzenjer",
		dto.SubmitManagementHoursRequest{WorkDate: "2026-09-01", HoursWorked: -1})
	if err == nil {
		t.Fatal("expected validation error for negative hours")
	}
}

// ── Director list scope ───────────────────────────────────────────────────────

func TestMgmt_DirectorList_CallsListForDirector_NotListForEmployee(t *testing.T) {
	listForEmployeeCalled := false
	listForDirectorCalled := false
	repo := &mockMgmtRepo{
		listForEmployeeFn: func(_ context.Context, _, _ string) ([]dto.ManagementHoursEntry, error) {
			listForEmployeeCalled = true
			return nil, nil
		},
		listForDirectorFn: func(_ context.Context, _, _ string) ([]dto.ManagementHoursEntry, error) {
			listForDirectorCalled = true
			return []dto.ManagementHoursEntry{}, nil
		},
	}
	newMgmtSvc(repo).List(context.Background(), companyA, direktorA, "direktor") //nolint:errcheck
	if !listForDirectorCalled {
		t.Error("expected ListForDirector to be called for direktor role")
	}
	if listForEmployeeCalled {
		t.Error("ListForEmployee must NOT be called for direktor role")
	}
}

func TestMgmt_InzenjerList_CallsListForEmployee_NotListForDirector(t *testing.T) {
	listForDirectorCalled := false
	repo := &mockMgmtRepo{
		listForDirectorFn: func(_ context.Context, _, _ string) ([]dto.ManagementHoursEntry, error) {
			listForDirectorCalled = true
			return nil, nil
		},
	}
	newMgmtSvc(repo).List(context.Background(), companyA, inzenjerA, "inzenjer") //nolint:errcheck
	if listForDirectorCalled {
		t.Error("ListForDirector must NOT be called for inzenjer role")
	}
}
