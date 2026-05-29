package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrEmailInUse = errors.New("email already in use")
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func validationErr(msg string) error { return &ValidationError{Message: msg} }

// ── Constants ─────────────────────────────────────────────────────────────────

// tempPasswordValue is the default password issued to all new login-capable employees.
const tempPasswordValue = "Temp1234!"

// ── Valid values ──────────────────────────────────────────────────────────────

var validEmployeeRoles = map[string]bool{
	"direktor": true, "inzenjer": true,
	"administracija": true, "poslovoda": true, "radnik": true,
}

// loginCapableRoles are the roles that get a users row and can log in.
var loginCapableRoles = map[string]bool{
	"direktor": true, "inzenjer": true,
	"administracija": true, "poslovoda": true,
}

var validAssetTypes = map[string]bool{
	"car": true, "tool": true, "equipment": true, "other": true,
}

// ── Service ───────────────────────────────────────────────────────────────────

type EmployeeService struct {
	db           *pgxpool.Pool
	empRepo      *repositories.EmployeeRepository
	assetRepo    *repositories.EmployeeAssetRepository
	auditRepo    *repositories.AuditRepository
	userRepo     *repositories.UserRepository
	hashPassword func(password string) (string, error)
}

func NewEmployeeService(
	db *pgxpool.Pool,
	empRepo *repositories.EmployeeRepository,
	assetRepo *repositories.EmployeeAssetRepository,
	auditRepo *repositories.AuditRepository,
	userRepo *repositories.UserRepository,
	hashPassword func(password string) (string, error),
) *EmployeeService {
	return &EmployeeService{
		db:           db,
		empRepo:      empRepo,
		assetRepo:    assetRepo,
		auditRepo:    auditRepo,
		userRepo:     userRepo,
		hashPassword: hashPassword,
	}
}

// ── CreateResult ──────────────────────────────────────────────────────────────

// CreateResult is returned by Create so the handler can build the full response.
type CreateResult struct {
	Employee           *dto.EmployeeDetail
	LoginCreated       bool
	TemporaryPassword  *string
	MustChangePassword bool
}

// ── Employee: List ────────────────────────────────────────────────────────────

type EmployeeFilter struct {
	Search string
	Role   *string
	Active *bool
}

func (s *EmployeeService) List(ctx context.Context, companyID, callerRole, callerEmpID string, f EmployeeFilter) ([]dto.EmployeeListItem, error) {
	repoFilter := repositories.EmployeeListFilter{
		Search: f.Search,
		Role:   f.Role,
		Active: f.Active,
	}
	if callerRole == "poslovoda" && callerEmpID != "" {
		repoFilter.ScopeEmpID = &callerEmpID
	}

	rows, err := s.empRepo.List(ctx, companyID, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}

	items := make([]dto.EmployeeListItem, 0, len(rows))
	for _, e := range rows {
		items = append(items, toListItem(e))
	}
	return items, nil
}

// ── Employee: GetByID ─────────────────────────────────────────────────────────

func (s *EmployeeService) GetByID(ctx context.Context, companyID, callerRole, callerEmpID, id string) (*dto.EmployeeDetail, error) {
	emp, err := s.empRepo.GetByID(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get employee: %w", err)
	}

	if callerRole == "poslovoda" && !poslovodaCanAccess(emp, callerEmpID) {
		return nil, ErrForbidden
	}

	return toDetail(emp), nil
}

// ── Employee: Create ──────────────────────────────────────────────────────────

func (s *EmployeeService) Create(ctx context.Context, companyID, callerUserID, ip, ua string, req dto.CreateEmployeeRequest) (*CreateResult, error) {
	email := emptyToNil(req.Email)

	if err := s.validateEmployeeFields(ctx, companyID, "", req.Role, email, req.SupervisorID); err != nil {
		return nil, err
	}

	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	phone := emptyToNil(req.Phone)

	if loginCapableRoles[req.Role] {
		return s.createWithLogin(ctx, companyID, callerUserID, ip, ua, firstName, lastName, req.Role, email, phone, req.SupervisorID)
	}

	return s.createRadnik(ctx, companyID, callerUserID, ip, ua, firstName, lastName, req.Role, email, phone, req.SupervisorID)
}

// createWithLogin creates an employee + user in a single transaction.
func (s *EmployeeService) createWithLogin(ctx context.Context, companyID, callerUserID, ip, ua, firstName, lastName, role string, email, phone, supervisorID *string) (*CreateResult, error) {
	hash, err := s.hashPassword(tempPasswordValue)
	if err != nil {
		return nil, fmt.Errorf("hash temp password: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { tx.Rollback(ctx) }() // no-op after Commit

	emp, err := s.empRepo.CreateWithTx(ctx, tx, companyID, firstName, lastName, role, email, phone, supervisorID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailInUse
		}
		return nil, fmt.Errorf("create employee: %w", err)
	}

	if err := s.userRepo.CreateWithTx(ctx, tx, companyID, emp.ID, *email, hash, role); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailInUse
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "employee.create",
		EntityType: "employee",
		EntityID:   &emp.ID,
		NewData:    emp,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	tmp := tempPasswordValue
	return &CreateResult{
		Employee:           toDetail(emp),
		LoginCreated:       true,
		TemporaryPassword:  &tmp,
		MustChangePassword: true,
	}, nil
}

// createRadnik creates an employee record only — no user/login account.
func (s *EmployeeService) createRadnik(ctx context.Context, companyID, callerUserID, ip, ua, firstName, lastName, role string, email, phone, supervisorID *string) (*CreateResult, error) {
	emp, err := s.empRepo.Create(ctx, companyID, firstName, lastName, role, email, phone, supervisorID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailInUse
		}
		return nil, fmt.Errorf("create employee: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "employee.create",
		EntityType: "employee",
		EntityID:   &emp.ID,
		NewData:    emp,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return &CreateResult{
		Employee:           toDetail(emp),
		LoginCreated:       false,
		TemporaryPassword:  nil,
		MustChangePassword: false,
	}, nil
}

// ── Employee: Update ──────────────────────────────────────────────────────────

func (s *EmployeeService) Update(ctx context.Context, companyID, callerUserID, ip, ua, id string, req dto.UpdateEmployeeRequest) (*dto.EmployeeDetail, error) {
	email := emptyToNil(req.Email)

	if err := s.validateEmployeeFields(ctx, companyID, id, req.Role, email, req.SupervisorID); err != nil {
		return nil, err
	}

	old, err := s.empRepo.GetByID(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update employee fetch: %w", err)
	}

	emp, err := s.empRepo.Update(ctx, companyID, id,
		strings.TrimSpace(req.FirstName),
		strings.TrimSpace(req.LastName),
		req.Role,
		email,
		emptyToNil(req.Phone),
		req.SupervisorID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailInUse
		}
		return nil, fmt.Errorf("update employee: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "employee.update",
		EntityType: "employee",
		EntityID:   &emp.ID,
		OldData:    old,
		NewData:    emp,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return toDetail(emp), nil
}

// ── Employee: SetActive ───────────────────────────────────────────────────────

func (s *EmployeeService) SetActive(ctx context.Context, companyID, callerUserID, ip, ua, id string, active bool) error {
	err := s.empRepo.SetActive(ctx, companyID, id, active)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("set employee active: %w", err)
	}

	action := "employee.activate"
	if !active {
		action = "employee.deactivate"
	}
	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     action,
		EntityType: "employee",
		EntityID:   &id,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return nil
}

// ── Asset: ListByEmployee ─────────────────────────────────────────────────────

func (s *EmployeeService) ListAssets(ctx context.Context, companyID, callerRole, callerEmpID, employeeID string) ([]dto.AssetListItem, error) {
	emp, err := s.empRepo.GetByID(ctx, companyID, employeeID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	if callerRole == "poslovoda" && !poslovodaCanAccess(emp, callerEmpID) {
		return nil, ErrForbidden
	}

	assets, err := s.assetRepo.ListByEmployee(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}

	items := make([]dto.AssetListItem, 0, len(assets))
	for _, a := range assets {
		items = append(items, toAssetItem(a))
	}
	return items, nil
}

// ── Asset: Create ─────────────────────────────────────────────────────────────

func (s *EmployeeService) CreateAsset(ctx context.Context, companyID, callerUserID, ip, ua, employeeID string, req dto.CreateAssetRequest) (*dto.AssetListItem, error) {
	if !validAssetTypes[req.AssetType] {
		return nil, validationErr("Asset type must be one of: car, tool, equipment, other")
	}
	if req.Quantity <= 0 {
		return nil, validationErr("Quantity must be greater than zero")
	}

	if _, err := s.empRepo.GetByID(ctx, companyID, employeeID); errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("create asset: %w", err)
	}

	asset, err := s.assetRepo.Create(ctx, companyID, employeeID,
		req.AssetType,
		strings.TrimSpace(req.Name),
		req.Quantity,
		emptyToNil(req.Unit),
		emptyToNil(req.SerialNumber),
		emptyToNil(req.Notes),
		&callerUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("create asset: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "employee_asset.create",
		EntityType: "employee_asset",
		EntityID:   &asset.ID,
		NewData:    asset,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	item := toAssetItem(*asset)
	return &item, nil
}

// ── Asset: Update ─────────────────────────────────────────────────────────────

func (s *EmployeeService) UpdateAsset(ctx context.Context, companyID, callerUserID, ip, ua, employeeID, assetID string, req dto.UpdateAssetRequest) (*dto.AssetListItem, error) {
	if !validAssetTypes[req.AssetType] {
		return nil, validationErr("Asset type must be one of: car, tool, equipment, other")
	}
	if req.Quantity <= 0 {
		return nil, validationErr("Quantity must be greater than zero")
	}

	existing, err := s.assetRepo.GetByID(ctx, companyID, assetID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
	}
	if existing.EmployeeID != employeeID {
		return nil, ErrNotFound
	}

	asset, err := s.assetRepo.Update(ctx, companyID, assetID,
		req.AssetType,
		strings.TrimSpace(req.Name),
		req.Quantity,
		emptyToNil(req.Unit),
		emptyToNil(req.SerialNumber),
		emptyToNil(req.Notes),
	)
	if err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "employee_asset.update",
		EntityType: "employee_asset",
		EntityID:   &asset.ID,
		OldData:    existing,
		NewData:    asset,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	item := toAssetItem(*asset)
	return &item, nil
}

// ── Asset: Deactivate ─────────────────────────────────────────────────────────

func (s *EmployeeService) DeactivateAsset(ctx context.Context, companyID, callerUserID, ip, ua, employeeID, assetID string) error {
	existing, err := s.assetRepo.GetByID(ctx, companyID, assetID)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("deactivate asset: %w", err)
	}
	if existing.EmployeeID != employeeID {
		return ErrNotFound
	}

	if err := s.assetRepo.SetActive(ctx, companyID, assetID, false); err != nil {
		return fmt.Errorf("deactivate asset: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "employee_asset.deactivate",
		EntityType: "employee_asset",
		EntityID:   &assetID,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (s *EmployeeService) validateEmployeeFields(ctx context.Context, companyID, selfID, role string, email *string, supervisorID *string) error {
	if !validEmployeeRoles[role] {
		return validationErr("Role must be one of: direktor, inzenjer, administracija, poslovoda, radnik")
	}

	// Login-capable roles require email because a user account will be created.
	if loginCapableRoles[role] && (email == nil || *email == "") {
		return validationErr("Email is required for " + role + " (a login account will be created)")
	}

	if role == "radnik" && supervisorID == nil {
		return validationErr("Workers (radnik) must have a supervisor assigned")
	}

	if supervisorID != nil {
		if selfID != "" && *supervisorID == selfID {
			return validationErr("An employee cannot be their own supervisor")
		}
		supRole, err := s.empRepo.GetRoleByID(ctx, companyID, *supervisorID)
		if errors.Is(err, repositories.ErrNotFound) {
			return validationErr("Supervisor not found in this company")
		}
		if err != nil {
			return fmt.Errorf("validate supervisor: %w", err)
		}
		if role == "radnik" && supRole != "poslovoda" {
			return validationErr("A worker's (radnik) supervisor must have the poslovoda role")
		}
	}

	return nil
}

func poslovodaCanAccess(emp *repositories.Employee, callerEmpID string) bool {
	if emp.ID == callerEmpID {
		return true
	}
	if emp.SupervisorID != nil && *emp.SupervisorID == callerEmpID {
		return true
	}
	return false
}

func emptyToNil(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// isUniqueViolation reports whether the error is a PostgreSQL unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	type pgError interface{ SQLState() string }
	var pge pgError
	if errors.As(err, &pge) {
		return pge.SQLState() == "23505"
	}
	return false
}

// ── Mapping helpers ───────────────────────────────────────────────────────────

func toListItem(e repositories.Employee) dto.EmployeeListItem {
	return dto.EmployeeListItem{
		ID:           e.ID,
		FirstName:    e.FirstName,
		LastName:     e.LastName,
		Role:         e.Role,
		Email:        e.Email,
		Phone:        e.Phone,
		Active:       e.Active,
		SupervisorID: e.SupervisorID,
		CreatedAt:    e.CreatedAt,
	}
}

func toDetail(e *repositories.Employee) *dto.EmployeeDetail {
	d := &dto.EmployeeDetail{
		ID:           e.ID,
		CompanyID:    e.CompanyID,
		FirstName:    e.FirstName,
		LastName:     e.LastName,
		Role:         e.Role,
		Email:        e.Email,
		Phone:        e.Phone,
		Active:       e.Active,
		SupervisorID: e.SupervisorID,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
	if e.SupervisorID != nil && e.SupervisorFirstName != nil {
		d.Supervisor = &dto.SupervisorRef{
			ID:        *e.SupervisorID,
			FirstName: *e.SupervisorFirstName,
			LastName:  *e.SupervisorLastName,
			Role:      *e.SupervisorRole,
		}
	}
	return d
}

func toAssetItem(a repositories.EmployeeAsset) dto.AssetListItem {
	return dto.AssetListItem{
		ID:           a.ID,
		EmployeeID:   a.EmployeeID,
		AssetType:    a.AssetType,
		Name:         a.Name,
		Quantity:     a.Quantity,
		Unit:         a.Unit,
		SerialNumber: a.SerialNumber,
		Notes:        a.Notes,
		Active:       a.Active,
		AssignedAt:   a.AssignedAt,
	}
}
