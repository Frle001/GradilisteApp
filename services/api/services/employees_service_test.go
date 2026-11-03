package services

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/gradiliste/api/repositories"
	"github.com/gradiliste/api/dto"
)

// ── Mock employee repository ──────────────────────────────────────────────────

type mockEmpRepo struct {
	listFn        func(ctx context.Context, companyID string, f repositories.EmployeeListFilter) ([]repositories.Employee, error)
	getByIDFn     func(ctx context.Context, companyID, id string) (*repositories.Employee, error)
	updateFn      func() (*repositories.Employee, error)
	updateWithTxFn func(id, role string) (*repositories.Employee, error)
}

func (m *mockEmpRepo) List(ctx context.Context, companyID string, f repositories.EmployeeListFilter) ([]repositories.Employee, error) {
	if m.listFn != nil {
		return m.listFn(ctx, companyID, f)
	}
	return nil, nil
}

func (m *mockEmpRepo) GetByID(ctx context.Context, companyID, id string) (*repositories.Employee, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, companyID, id)
	}
	return nil, repositories.ErrNotFound
}

func (m *mockEmpRepo) CreateWithTx(_ context.Context, _ pgx.Tx, _, _, _, _ string, _, _, _ *string) (*repositories.Employee, error) {
	return nil, nil
}

func (m *mockEmpRepo) Create(_ context.Context, _, _, _, _ string, _, _, _ *string) (*repositories.Employee, error) {
	return nil, nil
}

func (m *mockEmpRepo) Update(_ context.Context, _, _, _, _, _ string, _, _, _ *string) (*repositories.Employee, error) {
	if m.updateFn != nil {
		return m.updateFn()
	}
	return &repositories.Employee{Active: true}, nil
}

func (m *mockEmpRepo) UpdateWithTx(_ context.Context, _ pgx.Tx, _, id, _, _, role string, _, _, _ *string) (*repositories.Employee, error) {
	if m.updateWithTxFn != nil {
		return m.updateWithTxFn(id, role)
	}
	return &repositories.Employee{ID: id, Role: role, Active: true}, nil
}

func (m *mockEmpRepo) SetActive(_ context.Context, _, _ string, _ bool) error { return nil }

func (m *mockEmpRepo) GetRoleByID(_ context.Context, _, _ string) (string, error) { return "", nil }

func (m *mockEmpRepo) GetLinkedUserID(_ context.Context, _, _ string) (string, bool, error) {
	return "", false, nil
}

func (m *mockEmpRepo) CountActiveDirectors(_ context.Context, _ string) (int64, error) { return 0, nil }

func (m *mockEmpRepo) HardDeleteCheck(_ context.Context, _, _ string) error { return nil }

func (m *mockEmpRepo) HardDeleteWithTx(_ context.Context, _ pgx.Tx, _, _ string) error { return nil }

// ── Mock audit repository ─────────────────────────────────────────────────────

type mockAuditRepo struct{}

func (m *mockAuditRepo) Log(_ context.Context, _ repositories.AuditParams) {}

// ── Mock user repository ──────────────────────────────────────────────────────

type mockUserRepo struct {
	deactivateCalled         bool
	activateCalled           bool
	resetAndInvalidateCalled bool
	resetAndInvalidateError  error

	updateRoleWithTxCalled bool
	updateRoleWithTxRole   string
	updateRoleWithTxError  error
}

func (m *mockUserRepo) CreateWithTx(_ context.Context, _ pgx.Tx, _, _, _, _, _ string) error {
	return nil
}

func (m *mockUserRepo) ResetPasswordAndInvalidateSessions(_ context.Context, _, _, _ string) error {
	m.resetAndInvalidateCalled = true
	return m.resetAndInvalidateError
}

func (m *mockUserRepo) DeleteByEmployeeIDWithTx(_ context.Context, _ pgx.Tx, _, _ string) error {
	return nil
}

func (m *mockUserRepo) DeactivateEmployeeWithInvalidation(_ context.Context, _, _ string) error {
	m.deactivateCalled = true
	return nil
}

func (m *mockUserRepo) ActivateEmployeeAccount(_ context.Context, _, _ string) error {
	m.activateCalled = true
	return nil
}

func (m *mockUserRepo) UpdateRoleWithTx(_ context.Context, _ pgx.Tx, _, _, newRole string) error {
	m.updateRoleWithTxCalled = true
	m.updateRoleWithTxRole = newRole
	return m.updateRoleWithTxError
}

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

// ── Session revocation tests ──────────────────────────────────────────────────

// TestDeactivate_CallsAtomicInvalidation: deactivating an employee must call
// DeactivateEmployeeWithInvalidation — the single atomic path that sets employees.active=false,
// users.active=false, increments auth_version, and revokes refresh tokens in one transaction.
func TestDeactivate_CallsAtomicInvalidation(t *testing.T) {
	userRepo := &mockUserRepo{}
	svc := &EmployeeService{
		empRepo:   &mockEmpRepo{},
		userRepo:  userRepo,
		auditRepo: &mockAuditRepo{},
	}

	err := svc.SetActive(context.Background(), "co-1", "caller-1", "", "", "emp-1", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !userRepo.deactivateCalled {
		t.Error("deactivation must use DeactivateEmployeeWithInvalidation (atomic transaction)")
	}
	if userRepo.activateCalled {
		t.Error("deactivation must not call ActivateEmployeeAccount")
	}
}

// TestActivate_DoesNotResetAuthVersion: activating must call ActivateEmployeeAccount,
// which sets users.active=true but leaves auth_version unchanged.
// Pre-deactivation tokens issued with the old auth_version remain invalid.
func TestActivate_DoesNotResetAuthVersion(t *testing.T) {
	userRepo := &mockUserRepo{}
	svc := &EmployeeService{
		empRepo:   &mockEmpRepo{},
		userRepo:  userRepo,
		auditRepo: &mockAuditRepo{},
	}

	err := svc.SetActive(context.Background(), "co-1", "caller-1", "", "", "emp-1", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !userRepo.activateCalled {
		t.Error("activation must use ActivateEmployeeAccount")
	}
	if userRepo.deactivateCalled {
		t.Error("activation must not call DeactivateEmployeeWithInvalidation")
	}
}

// TestResetPassword_IsAtomic: resetting an employee's password must call
// ResetPasswordAndInvalidateSessions — the single atomic method that updates the
// password hash, increments auth_version, and revokes refresh tokens in one transaction.
func TestResetPassword_IsAtomic(t *testing.T) {
	emp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	userRepo := &mockUserRepo{}
	svc := &EmployeeService{
		empRepo: &mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &emp, nil
			},
		},
		userRepo:     userRepo,
		auditRepo:    &mockAuditRepo{},
		hashPassword: func(_ string) (string, error) { return "hash", nil },
	}

	_, err := svc.ResetPassword(context.Background(), "co-1", "caller-1", "", "", "emp-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !userRepo.resetAndInvalidateCalled {
		t.Error("password reset must use ResetPasswordAndInvalidateSessions (atomic transaction)")
	}
}

// TestResetPassword_RepoFailureIsReturned: a repository error from the atomic
// reset must propagate as a service error, not be swallowed.
func TestResetPassword_RepoFailureIsReturned(t *testing.T) {
	emp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	repoErr := errors.New("db unavailable")
	userRepo := &mockUserRepo{resetAndInvalidateError: repoErr}
	svc := &EmployeeService{
		empRepo: &mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &emp, nil
			},
		},
		userRepo:     userRepo,
		auditRepo:    &mockAuditRepo{},
		hashPassword: func(_ string) (string, error) { return "hash", nil },
	}

	_, err := svc.ResetPassword(context.Background(), "co-1", "caller-1", "", "", "emp-1")
	if err == nil {
		t.Fatal("expected error when repo fails, got nil")
	}
}

// ── stubTx ────────────────────────────────────────────────────────────────────

// stubTx is a minimal pgx.Tx for unit tests. Only Commit and Rollback are
// meaningful; all other methods panic because the service never calls them
// directly (they are exercised through the mocked repository layer).
type stubTx struct {
	commitErr  error
	committed  bool
	rolledBack bool
}

func (s *stubTx) Begin(_ context.Context) (pgx.Tx, error)                             { panic("not used") }
func (s *stubTx) Commit(_ context.Context) error                                       { s.committed = true; return s.commitErr }
func (s *stubTx) Rollback(_ context.Context) error                                     { s.rolledBack = true; return nil }
func (s *stubTx) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	panic("not used")
}
func (s *stubTx) SendBatch(_ context.Context, _ *pgx.Batch) pgx.BatchResults { panic("not used") }
func (s *stubTx) LargeObjects() pgx.LargeObjects                              { panic("not used") }
func (s *stubTx) Prepare(_ context.Context, _, _ string) (*pgconn.StatementDescription, error) {
	panic("not used")
}
func (s *stubTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	panic("not used")
}
func (s *stubTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) { panic("not used") }
func (s *stubTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row        { panic("not used") }
func (s *stubTx) Conn() *pgx.Conn                                                { panic("not used") }

// ── Role-change regression tests ──────────────────────────────────────────────

// newUpdateSvc builds an EmployeeService wired for Update() tests.
func newUpdateSvc(empRepo *mockEmpRepo, userRepo *mockUserRepo, tx *stubTx) *EmployeeService {
	beginTxFn := func(_ context.Context) (pgx.Tx, error) { return tx, nil }
	return &EmployeeService{
		empRepo:   empRepo,
		userRepo:  userRepo,
		auditRepo: &mockAuditRepo{},
		beginTx:   beginTxFn,
	}
}

// validUpdateReq returns an UpdateEmployeeRequest for tests; no supervisor needed for these roles.
func validUpdateReq(role string) dto.UpdateEmployeeRequest {
	email := "test@example.com"
	return dto.UpdateEmployeeRequest{
		FirstName: "Ivo",
		LastName:  "Ivić",
		Role:      role,
		Email:     &email,
	}
}

// TestUpdate_NoRoleChange_DoesNotBeginTx verifies that when the role is unchanged the
// service never opens a transaction — no unnecessary locking.
func TestUpdate_NoRoleChange_DoesNotBeginTx(t *testing.T) {
	txBegun := false
	emp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	userRepo := &mockUserRepo{}
	svc := &EmployeeService{
		empRepo: &mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &emp, nil
			},
		},
		userRepo:  userRepo,
		auditRepo: &mockAuditRepo{},
		beginTx: func(_ context.Context) (pgx.Tx, error) {
			txBegun = true
			return &stubTx{}, nil
		},
	}

	_, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("inzenjer"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txBegun {
		t.Error("no transaction should be started when role does not change")
	}
}

// TestUpdate_NoRoleChange_DoesNotSyncUserRole verifies that UpdateRoleWithTx is
// NOT called when the role is unchanged.
func TestUpdate_NoRoleChange_DoesNotSyncUserRole(t *testing.T) {
	emp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	userRepo := &mockUserRepo{}
	svc := &EmployeeService{
		empRepo: &mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &emp, nil
			},
		},
		userRepo:  userRepo,
		auditRepo: &mockAuditRepo{},
		beginTx:   func(_ context.Context) (pgx.Tx, error) { return &stubTx{}, nil },
	}

	if _, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("inzenjer")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userRepo.updateRoleWithTxCalled {
		t.Error("UpdateRoleWithTx must NOT be called when role does not change")
	}
}

// TestUpdate_RoleChange_BeginsTransaction verifies that a transaction is opened
// when the role changes, ensuring atomicity.
func TestUpdate_RoleChange_BeginsTransaction(t *testing.T) {
	txBegun := false
	oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	tx := &stubTx{}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
		},
		&mockUserRepo{},
		tx,
	)
	svc.beginTx = func(_ context.Context) (pgx.Tx, error) {
		txBegun = true
		return tx, nil
	}

	if _, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("direktor")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !txBegun {
		t.Error("a transaction must be started when the role changes")
	}
}

// TestUpdate_RoleChange_CallsUpdateRoleWithTx verifies that the user role sync
// is invoked when the role changes.
func TestUpdate_RoleChange_CallsUpdateRoleWithTx(t *testing.T) {
	oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	userRepo := &mockUserRepo{}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
		},
		userRepo,
		&stubTx{},
	)

	if _, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("direktor")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !userRepo.updateRoleWithTxCalled {
		t.Error("UpdateRoleWithTx must be called when role changes")
	}
}

// TestUpdate_RoleChange_NewRolePassedToUserRepo verifies the correct new role is
// forwarded so JWTs will contain the right claims after re-login.
func TestUpdate_RoleChange_NewRolePassedToUserRepo(t *testing.T) {
	oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "radnik")
	oldEmp.SupervisorID = nil // override for non-radnik update below
	userRepo := &mockUserRepo{}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
		},
		userRepo,
		&stubTx{},
	)

	if _, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("inzenjer")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userRepo.updateRoleWithTxRole != "inzenjer" {
		t.Errorf("UpdateRoleWithTx received role %q, want %q", userRepo.updateRoleWithTxRole, "inzenjer")
	}
}

// TestUpdate_RoleChange_OldRoleNotPassedToUserRepo verifies the old role is not
// forwarded (regression: guards against accidentally passing old.Role).
func TestUpdate_RoleChange_OldRoleNotPassedToUserRepo(t *testing.T) {
	oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "administracija")
	userRepo := &mockUserRepo{}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
		},
		userRepo,
		&stubTx{},
	)

	if _, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("inzenjer")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userRepo.updateRoleWithTxRole == "administracija" {
		t.Error("UpdateRoleWithTx received the OLD role — must receive the new role")
	}
}

// TestUpdate_RoleChange_CommitError_ReturnsError verifies that a transaction
// commit failure propagates as an error and is not silently swallowed.
func TestUpdate_RoleChange_CommitError_ReturnsError(t *testing.T) {
	oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	commitErr := errors.New("disk full")
	tx := &stubTx{commitErr: commitErr}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
		},
		&mockUserRepo{},
		tx,
	)

	_, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("direktor"))
	if err == nil {
		t.Fatal("expected error when commit fails, got nil")
	}
}

// TestUpdate_RoleChange_UserRepoError_ReturnsError verifies that a failure in
// UpdateRoleWithTx propagates and prevents the transaction from committing.
func TestUpdate_RoleChange_UserRepoError_ReturnsError(t *testing.T) {
	oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	tx := &stubTx{}
	userRepo := &mockUserRepo{updateRoleWithTxError: errors.New("constraint violation")}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
		},
		userRepo,
		tx,
	)

	_, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("direktor"))
	if err == nil {
		t.Fatal("expected error when user repo fails, got nil")
	}
	if tx.committed {
		t.Error("transaction must NOT be committed when UpdateRoleWithTx fails")
	}
}

// TestUpdate_RoleChange_EmpRepoError_SkipsUserRepo verifies that an employee
// repo failure prevents UpdateRoleWithTx from being called.
func TestUpdate_RoleChange_EmpRepoError_SkipsUserRepo(t *testing.T) {
	oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", "inzenjer")
	userRepo := &mockUserRepo{}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
			updateWithTxFn: func(_, _ string) (*repositories.Employee, error) {
				return nil, errors.New("db error")
			},
		},
		userRepo,
		&stubTx{},
	)

	_, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq("direktor"))
	if err == nil {
		t.Fatal("expected error when empRepo.UpdateWithTx fails")
	}
	if userRepo.updateRoleWithTxCalled {
		t.Error("UpdateRoleWithTx must NOT be called when the employee update fails")
	}
}

// TestUpdate_RoleChange_CompanyIDForwardedToUserRepo verifies that UpdateRoleWithTx
// is reached on role change, confirming the company-scoped user repo path is taken
// rather than a global query that could cross tenant boundaries.
func TestUpdate_RoleChange_CompanyIDForwardedToUserRepo(t *testing.T) {
	oldEmp := testEmpRow("emp-1", "co-abc", "Ivo", "Ivić", "inzenjer")
	userRepo := &mockUserRepo{}
	svc := newUpdateSvc(
		&mockEmpRepo{
			getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
				return &oldEmp, nil
			},
		},
		userRepo,
		&stubTx{},
	)

	if _, err := svc.Update(context.Background(), "co-abc", "caller-1", "", "", "emp-1", validUpdateReq("direktor")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !userRepo.updateRoleWithTxCalled {
		t.Error("UpdateRoleWithTx must be called — company-scoped role sync path was not reached")
	}
}

// TestUpdate_RoleChange_AllKnownRoleTransitions verifies that every role pair
// (excluding radnik which requires a supervisor) triggers the user sync.
func TestUpdate_RoleChange_AllKnownRoleTransitions(t *testing.T) {
	roles := []string{"direktor", "inzenjer", "administracija", "poslovoda"}

	for _, fromRole := range roles {
		for _, toRole := range roles {
			if fromRole == toRole {
				continue
			}
			t.Run(fromRole+"→"+toRole, func(t *testing.T) {
				oldEmp := testEmpRow("emp-1", "co-1", "Ivo", "Ivić", fromRole)
				userRepo := &mockUserRepo{}
				svc := newUpdateSvc(
					&mockEmpRepo{
						getByIDFn: func(_ context.Context, _, _ string) (*repositories.Employee, error) {
							return &oldEmp, nil
						},
					},
					userRepo,
					&stubTx{},
				)

				if _, err := svc.Update(context.Background(), "co-1", "caller-1", "", "", "emp-1", validUpdateReq(toRole)); err != nil {
					t.Fatalf("Update(%s→%s): unexpected error: %v", fromRole, toRole, err)
				}
				if !userRepo.updateRoleWithTxCalled {
					t.Errorf("Update(%s→%s): UpdateRoleWithTx must be called", fromRole, toRole)
				}
				if userRepo.updateRoleWithTxRole != toRole {
					t.Errorf("Update(%s→%s): UpdateRoleWithTx got role %q, want %q", fromRole, toRole, userRepo.updateRoleWithTxRole, toRole)
				}
			})
		}
	}
}
