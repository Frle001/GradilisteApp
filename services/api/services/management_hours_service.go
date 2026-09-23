package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/models"
	"github.com/gradiliste/api/repositories"
)

// managementHoursRepoIface is the repository surface used by ManagementHoursService.
type managementHoursRepoIface interface {
	Upsert(ctx context.Context, companyID, employeeID, submittedByUserID, workDate string, hoursWorked float64, notes, projectID *string) (*dto.ManagementHoursEntry, error)
	GetByID(ctx context.Context, companyID, entryID string) (*dto.ManagementHoursEntry, error)
	ListForEmployee(ctx context.Context, companyID, employeeID string) ([]dto.ManagementHoursEntry, error)
	ListForDirector(ctx context.Context, companyID, directorEmployeeID string) ([]dto.ManagementHoursEntry, error)
	Update(ctx context.Context, companyID, ownerEmployeeID, entryID string, hoursWorked float64, notes *string) error
	Delete(ctx context.Context, companyID, ownerEmployeeID, entryID string) error
}

// ManagementHoursService handles hour entry for direktor, inženjer, and administracija.
type ManagementHoursService struct {
	repo managementHoursRepoIface
}

func NewManagementHoursService(repo managementHoursRepoIface) *ManagementHoursService {
	return &ManagementHoursService{repo: repo}
}

// isManagementRole returns true for the three roles that may use this service.
func isManagementRole(role string) bool {
	return role == models.RoleDirektor ||
		role == models.RoleInzenjer ||
		role == models.RoleAdministracija
}

// Submit creates or updates the authenticated employee's hours for a given date.
// The employee ID is always taken from the JWT — callers cannot submit on behalf of others.
func (s *ManagementHoursService) Submit(
	ctx context.Context,
	companyID, employeeID, callerUserID, callerRole string,
	req dto.SubmitManagementHoursRequest,
) (*dto.ManagementHoursEntry, error) {
	if !isManagementRole(callerRole) {
		return nil, ErrForbidden
	}
	if err := validateDate(req.WorkDate); err != nil {
		return nil, err
	}
	if req.HoursWorked < 0 || req.HoursWorked > 24 {
		return nil, validationErr("Broj sati mora biti između 0 i 24")
	}

	entry, err := s.repo.Upsert(ctx, companyID, employeeID, callerUserID, req.WorkDate, req.HoursWorked, req.Notes, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("management_hours.Submit: %w", err)
	}
	entry.EmployeeRole = callerRole
	return entry, nil
}

// List returns entries visible to the authenticated employee based on their role:
//
//   - direktor: own entries + all inženjer entries + all administracija entries
//     (other directors' entries are excluded).
//   - inženjer: only own entries.
//   - administracija: only own entries.
func (s *ManagementHoursService) List(
	ctx context.Context,
	companyID, callerEmployeeID, callerRole string,
) ([]dto.ManagementHoursEntry, error) {
	if !isManagementRole(callerRole) {
		return nil, ErrForbidden
	}
	if callerRole == models.RoleDirektor {
		return s.repo.ListForDirector(ctx, companyID, callerEmployeeID)
	}
	return s.repo.ListForEmployee(ctx, companyID, callerEmployeeID)
}

// GetByID returns one entry if it is visible to the caller.
// Visibility follows the same rules as List.
// Returns ErrForbidden when the entry exists but the caller cannot see it.
func (s *ManagementHoursService) GetByID(
	ctx context.Context,
	companyID, callerEmployeeID, callerRole, entryID string,
) (*dto.ManagementHoursEntry, error) {
	if !isManagementRole(callerRole) {
		return nil, ErrForbidden
	}
	entry, err := s.repo.GetByID(ctx, companyID, entryID)
	if err != nil {
		return nil, err // ErrManagementHoursNotFound or internal
	}
	if !s.canView(callerEmployeeID, callerRole, entry) {
		return nil, ErrForbidden
	}
	return entry, nil
}

// Update changes hours_worked/notes on an entry the caller owns.
// Directors, engineers, and accounting staff may only edit their own entries.
func (s *ManagementHoursService) Update(
	ctx context.Context,
	companyID, callerEmployeeID, callerRole, entryID string,
	req dto.UpdateManagementHoursRequest,
) error {
	if !isManagementRole(callerRole) {
		return ErrForbidden
	}
	// Fetch to confirm ownership before mutating.
	entry, err := s.repo.GetByID(ctx, companyID, entryID)
	if err != nil {
		return err
	}
	if entry.EmployeeID != callerEmployeeID {
		return ErrForbidden
	}
	if req.HoursWorked < 0 || req.HoursWorked > 24 {
		return validationErr("Broj sati mora biti između 0 i 24")
	}
	return s.repo.Update(ctx, companyID, callerEmployeeID, entryID, req.HoursWorked, req.Notes)
}

// Delete removes an entry the caller owns.
func (s *ManagementHoursService) Delete(
	ctx context.Context,
	companyID, callerEmployeeID, callerRole, entryID string,
) error {
	if !isManagementRole(callerRole) {
		return ErrForbidden
	}
	entry, err := s.repo.GetByID(ctx, companyID, entryID)
	if err != nil {
		return err
	}
	if entry.EmployeeID != callerEmployeeID {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, companyID, callerEmployeeID, entryID)
}

// ── Visibility helpers ────────────────────────────────────────────────────────

// canView returns true when callerRole/callerEmployeeID may read the given entry.
//
// Rules:
//   - Own entry: always allowed for all management roles.
//   - direktor seeing inzenjer or administracija: allowed.
//   - direktor seeing another direktor: denied.
//   - inzenjer/administracija seeing anyone else: denied.
func (s *ManagementHoursService) canView(callerEmployeeID, callerRole string, entry *dto.ManagementHoursEntry) bool {
	if entry.EmployeeID == callerEmployeeID {
		return true
	}
	if callerRole == models.RoleDirektor {
		return entry.EmployeeRole == models.RoleInzenjer ||
			entry.EmployeeRole == models.RoleAdministracija
	}
	return false
}

// ── Date validation ───────────────────────────────────────────────────────────

func validateDate(s string) error {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return validationErr("Datum mora biti u formatu YYYY-MM-DD")
	}
	return nil
}

// Ensure the concrete repository satisfies the interface at compile time.
var _ managementHoursRepoIface = (*repositories.ManagementHoursRepository)(nil)
