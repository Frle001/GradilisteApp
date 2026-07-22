package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gradiliste/api/dto"
)

// workerHoursRepoIface is the repository surface used by WorkerHoursService.
type workerHoursRepoIface interface {
	ListActiveProjectsByAssignment(ctx context.Context, companyID, empID, roleOnProject string) ([]dto.WorkerProject, error)
	HasActiveAssignment(ctx context.Context, empID, projectID, roleOnProject string) (bool, error)
	GetProjectStatus(ctx context.Context, companyID, projectID string) (string, error)
	GetOtherProjectsHoursForDate(ctx context.Context, companyID, workerEmpID, projectID, workDate string) (float64, error)
	Upsert(ctx context.Context, companyID, workerEmpID, projectID, workDate string, hoursWorked float64, notes *string, submittedByUserID string) (*dto.WorkerHoursEntry, error)
	ListForWorkerDate(ctx context.Context, companyID, workerEmpID, workDate string) ([]dto.WorkerHoursEntry, error)
	ListBeforeDate(ctx context.Context, companyID, workerEmpID, beforeDate string) ([]dto.WorkerHoursEntry, error)
}

type WorkerHoursService struct {
	repo workerHoursRepoIface
}

func NewWorkerHoursService(repo workerHoursRepoIface) *WorkerHoursService {
	return &WorkerHoursService{repo: repo}
}

// appRoleToProjectRole maps the application role to the role_on_project value
// used in project_assignments. Returns "" for unrecognized roles.
func appRoleToProjectRole(appRole string) string {
	switch appRole {
	case "radnik":
		return "worker"
	case "poslovoda":
		return "poslovoda"
	default:
		return ""
	}
}

// ListCompanyProjects returns active projects the caller is assigned to, filtered
// by the caller's application role (radnik → role_on_project='worker';
// poslovoda → role_on_project='poslovoda').
func (s *WorkerHoursService) ListCompanyProjects(ctx context.Context, companyID, empID, callerRole string) ([]dto.WorkerProject, error) {
	roleOnProject := appRoleToProjectRole(callerRole)
	if roleOnProject == "" {
		return nil, ErrForbidden
	}
	return s.repo.ListActiveProjectsByAssignment(ctx, companyID, empID, roleOnProject)
}

// ListForDate returns the employee's hour entries for a given date.
// If date is empty, today (Europe/Zagreb) is used.
func (s *WorkerHoursService) ListForDate(ctx context.Context, companyID, workerEmpID, date string) ([]dto.WorkerHoursEntry, error) {
	if date == "" {
		loc, _ := time.LoadLocation("Europe/Zagreb")
		date = time.Now().In(loc).Format("2006-01-02")
	}
	return s.repo.ListForWorkerDate(ctx, companyID, workerEmpID, date)
}

// ListHistory returns the employee's hour entries for all days strictly before
// today (Europe/Zagreb), newest date first.
func (s *WorkerHoursService) ListHistory(ctx context.Context, companyID, workerEmpID string) ([]dto.WorkerHoursEntry, error) {
	loc, _ := time.LoadLocation("Europe/Zagreb")
	today := time.Now().In(loc).Format("2006-01-02")
	return s.repo.ListBeforeDate(ctx, companyID, workerEmpID, today)
}

// Submit validates and upserts the employee's hours for today.
// callerRole determines which role_on_project assignment is required.
func (s *WorkerHoursService) Submit(ctx context.Context, companyID, workerEmpID, callerUserID, callerRole string, req dto.SubmitWorkerHoursRequest) (*dto.WorkerHoursEntry, error) {
	// Enforce today-only in Europe/Zagreb timezone
	loc, _ := time.LoadLocation("Europe/Zagreb")
	today := time.Now().In(loc).Format("2006-01-02")
	if req.WorkDate != today {
		return nil, validationErr(fmt.Sprintf("Sati se mogu unositi samo za današnji datum (%s)", today))
	}

	if req.HoursWorked < 0 || req.HoursWorked > 24 {
		return nil, validationErr("Broj sati mora biti između 0 i 24")
	}

	// Validate project belongs to company and is active
	status, err := s.repo.GetProjectStatus(ctx, companyID, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("submit worker hours: %w", err)
	}
	if status == "" {
		return nil, validationErr("Projekt nije pronađen")
	}
	if status != "active" {
		return nil, validationErr("Sati se mogu unositi samo za aktivne projekte")
	}

	// Validate the employee has an active assignment with the matching role
	roleOnProject := appRoleToProjectRole(callerRole)
	if roleOnProject == "" {
		return nil, ErrForbidden
	}
	ok, err := s.repo.HasActiveAssignment(ctx, workerEmpID, req.ProjectID, roleOnProject)
	if err != nil {
		return nil, fmt.Errorf("submit worker hours: %w", err)
	}
	if !ok {
		return nil, ErrForbidden
	}

	// Enforce daily total ≤ 24 across all projects
	otherHours, err := s.repo.GetOtherProjectsHoursForDate(ctx, companyID, workerEmpID, req.ProjectID, req.WorkDate)
	if err != nil {
		return nil, fmt.Errorf("submit worker hours: %w", err)
	}
	if otherHours+req.HoursWorked > 24 {
		return nil, validationErr(fmt.Sprintf("Ukupni sati za danas bi premašili 24 sata (već upisano: %.1f h na ostalim projektima)", otherHours))
	}

	return s.repo.Upsert(ctx, companyID, workerEmpID, req.ProjectID, req.WorkDate, req.HoursWorked, req.Notes, callerUserID)
}
