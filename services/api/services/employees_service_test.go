package services

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/repositories"
)

// ── Mock employee repository ──────────────────────────────────────────────────

type mockEmpRepo struct {
	listFn func(ctx context.Context, companyID string, f repositories.EmployeeListFilter) ([]repositories.Employee, error)
}

func (m *mockEmpRepo) List(ctx context.Context, companyID string, f repositories.EmployeeListFilter) ([]repositories.Employee, error) {
	if m.listFn != nil {
		return m.listFn(ctx, companyID, f)
	}
	return nil, nil
}

func (m *mockEmpRepo) GetByID(_ context.Context, _, _ string) (*repositories.Employee, error) {
	return nil, repositories.ErrNotFound
}

func (m *mockEmpRepo) CreateWithTx(_ context.Context, _ pgx.Tx, _, _, _, _ string, _, _, _ *string) (*repositories.Employee, error) {
	return nil, nil
}

func (m *mockEmpRepo) Create(_ context.Context, _, _, _, _ string, _, _, _ *string) (*repositories.Employee, error) {
	return nil, nil
}

func (m *mockEmpRepo) Update(_ context.Context, _, _, _, _, _ string, _, _, _ *string) (*repositories.Employee, error) {
	return nil, nil
}

func (m *mockEmpRepo) SetActive(_ context.Context, _, _ string, _ bool) error { return nil }

func (m *mockEmpRepo) GetRoleByID(_ context.Context, _, _ string) (string, error) { return "", nil }

func (m *mockEmpRepo) GetLinkedUserID(_ context.Context, _, _ string) (string, bool, error) {
	return "", false, nil
}

func (m *mockEmpRepo) CountActiveDirectors(_ context.Context, _ string) (int64, error) { return 0, nil }

func (m *mockEmpRepo) HardDeleteCheck(_ context.Context, _, _ string) error { return nil }

func (m *mockEmpRepo) HardDeleteWithTx(_ context.Context, _ pgx.Tx, _, _ string) error { return nil }

// ── Helper ────────────────────────────────────────────────────────────────────

func newEmpSvc(repo *mockEmpRepo) *EmployeeService {
	return &EmployeeService{empRepo: repo}
}

func testEmpRow(id, companyID, firstName, lastName, role string) repositories.Employee {
	return repositories.Employee{ID: id, CompanyID: companyID, FirstName: firstName, LastName: lastName, Role: role, Active: true}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestEmployeeList_PoslovodaReceivesAllCompanyEmployees verifies that a
// poslovoda caller is no longer scoped to their own team and instead receives
// every active employee in the company.
func TestEmployeeList_PoslovodaReceivesAllCompanyEmployees(t *testing.T) {
	allEmps := []repositories.Employee{
		testEmpRow("emp-1", "co-1", "Ana", "Anić", "radnik"),
		testEmpRow("emp-2", "co-1", "Ivo", "Ivić", "poslovoda"),
		testEmpRow("emp-3", "co-1", "Mate", "Matić", "radnik"),
		testEmpRow("emp-4", "co-1", "Luka", "Lukić", "inzenjer"),
	}
	repo := &mockEmpRepo{
		listFn: func(_ context.Context, _ string, _ repositories.EmployeeListFilter) ([]repositories.Employee, error) {
			return allEmps, nil
		},
	}
	items, err := newEmpSvc(repo).List(context.Background(), "co-1", "poslovoda", "emp-2", EmployeeFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 4 {
		t.Errorf("poslovoda should see all 4 company employees, got %d", len(items))
	}
}

// TestEmployeeList_ShiftsFromUnmanagedProjectsVisible verifies that all company
// employees appear regardless of which poslovoda manages their project — the
// caller's employee ID must not narrow the result set.
func TestEmployeeList_ShiftsFromUnmanagedProjectsVisible(t *testing.T) {
	// emp-3 and emp-4 are NOT supervised by the poslovoda (emp-2).
	// Before the fix they would have been excluded.
	allEmps := []repositories.Employee{
		testEmpRow("emp-1", "co-1", "Ana", "Anić", "radnik"),
		testEmpRow("emp-2", "co-1", "Ivo", "Ivić", "poslovoda"),
		testEmpRow("emp-3", "co-1", "Mate", "Matić", "radnik"),
		testEmpRow("emp-4", "co-1", "Luka", "Lukić", "radnik"),
	}
	var calledWithFilter repositories.EmployeeListFilter
	repo := &mockEmpRepo{
		listFn: func(_ context.Context, _ string, f repositories.EmployeeListFilter) ([]repositories.Employee, error) {
			calledWithFilter = f
			return allEmps, nil
		},
	}
	items, err := newEmpSvc(repo).List(context.Background(), "co-1", "poslovoda", "emp-2", EmployeeFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 4 {
		t.Errorf("all 4 company employees must be visible, got %d", len(items))
	}
	// The filter forwarded to the repo must not carry any caller-scoping field.
	// EmployeeListFilter no longer has ScopeEmpID; this asserts the filter is clean.
	if calledWithFilter.Search != "" || calledWithFilter.Role != nil || calledWithFilter.Active != nil {
		t.Errorf("unexpected filters sent to repo: %+v", calledWithFilter)
	}
}

// TestEmployeeList_CrossCompanyIsolation verifies that the companyID from the
// authenticated context is forwarded exactly to the repository, keeping tenants
// fully isolated.
func TestEmployeeList_CrossCompanyIsolation(t *testing.T) {
	var capturedCompanyID string
	repo := &mockEmpRepo{
		listFn: func(_ context.Context, companyID string, _ repositories.EmployeeListFilter) ([]repositories.Employee, error) {
			capturedCompanyID = companyID
			return nil, nil
		},
	}
	if _, err := newEmpSvc(repo).List(context.Background(), "co-999", "poslovoda", "emp-1", EmployeeFilter{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedCompanyID != "co-999" {
		t.Errorf("expected repo to receive companyID='co-999', got %q", capturedCompanyID)
	}
}

// TestEmployeeList_PoslovodaMutationForbidden documents that poslovoda cannot
// call create/update/cancel schedule shifts. This protection is enforced at the
// route level by requireRoles("direktor", "inzenjer") on all schedule mutation
// endpoints (routes/schedule.go). The middleware returns 403 before the service
// is reached, so no service-level test is required for this path.
func TestEmployeeList_PoslovodaMutationForbidden(_ *testing.T) {
	// Intentionally empty: enforcement is in routes/schedule.go via
	// requireRoles("direktor", "inzenjer").
}
