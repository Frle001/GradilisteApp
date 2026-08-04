package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// workerHoursRepoIface is the repository surface used by WorkerHoursService.
type workerHoursRepoIface interface {
	ListCompanyActiveProjects(ctx context.Context, companyID string) ([]dto.WorkerProject, error)
	GetProjectStatus(ctx context.Context, companyID, projectID string) (string, error)
	GetOtherProjectsHoursForDate(ctx context.Context, companyID, workerEmpID, projectID, workDate string) (float64, error)
	Upsert(ctx context.Context, companyID, workerEmpID, projectID, workDate string, hoursWorked float64, notes, workDescription *string, submittedByUserID string, clientSubmissionID *string) (*dto.WorkerHoursEntry, error)
	ListForWorkerDate(ctx context.Context, companyID, workerEmpID, workDate string) ([]dto.WorkerHoursEntry, error)
	ListBeforeDate(ctx context.Context, companyID, workerEmpID, beforeDate string) ([]dto.WorkerHoursEntry, error)

	// Manager methods
	ListForManager(ctx context.Context, companyID, projectID, workDate string) ([]dto.ManagerWorkerHoursEntry, error)
	GetByID(ctx context.Context, companyID, entryID string) (*dto.WorkerHoursDetail, error)
	CorrectHours(ctx context.Context, companyID, entryID string, newHours float64, reason, changedByUserID string) error
	AddComment(ctx context.Context, companyID, entryID, authorUserID, text string) (*dto.WorkerHoursComment, error)
}

type WorkerHoursService struct {
	repo workerHoursRepoIface
}

func NewWorkerHoursService(repo workerHoursRepoIface) *WorkerHoursService {
	return &WorkerHoursService{repo: repo}
}

// ListCompanyProjects returns all active projects in the company.
// Both radnik and poslovoda may submit hours to any active company project.
func (s *WorkerHoursService) ListCompanyProjects(ctx context.Context, companyID, callerRole string) ([]dto.WorkerProject, error) {
	switch callerRole {
	case "radnik", "poslovoda":
		return s.repo.ListCompanyActiveProjects(ctx, companyID)
	default:
		return nil, ErrForbidden
	}
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
// Both radnik and poslovoda use the same validation and upsert flow.
// work_description is accepted from any role but only surfaced in the poslovoda UI.
func (s *WorkerHoursService) Submit(ctx context.Context, companyID, workerEmpID, callerUserID, callerRole string, req dto.SubmitWorkerHoursRequest) (*dto.WorkerHoursEntry, error) {
	switch callerRole {
	case "radnik", "poslovoda":
	default:
		return nil, ErrForbidden
	}

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

	// Enforce daily total ≤ 24 across all projects
	otherHours, err := s.repo.GetOtherProjectsHoursForDate(ctx, companyID, workerEmpID, req.ProjectID, req.WorkDate)
	if err != nil {
		return nil, fmt.Errorf("submit worker hours: %w", err)
	}
	if otherHours+req.HoursWorked > 24 {
		return nil, validationErr(fmt.Sprintf("Ukupni sati za danas bi premašili 24 sata (već upisano: %.1f h na ostalim projektima)", otherHours))
	}

	entry, err := s.repo.Upsert(ctx, companyID, workerEmpID, req.ProjectID, req.WorkDate, req.HoursWorked, req.Notes, req.WorkDescription, callerUserID, req.ClientSubmissionID)
	if errors.Is(err, repositories.ErrSubmissionConflict) {
		return nil, ErrSubmissionConflict
	}
	return entry, err
}

// ── Manager methods ───────────────────────────────────────────────────────────

func (s *WorkerHoursService) isManager(role string) bool {
	return role == "direktor" || role == "inzenjer"
}

// ListManagerEntries returns all worker-hours entries visible to a manager.
// projectID and workDate are optional filters (empty string = no filter).
func (s *WorkerHoursService) ListManagerEntries(ctx context.Context, companyID, callerRole, projectID, workDate string) ([]dto.ManagerWorkerHoursEntry, error) {
	if !s.isManager(callerRole) {
		return nil, ErrForbidden
	}
	return s.repo.ListForManager(ctx, companyID, projectID, workDate)
}

// GetDetail returns the full entry detail including revision history and comments.
func (s *WorkerHoursService) GetDetail(ctx context.Context, companyID, callerRole, entryID string) (*dto.WorkerHoursDetail, error) {
	if !s.isManager(callerRole) {
		return nil, ErrForbidden
	}
	return s.repo.GetByID(ctx, companyID, entryID)
}

// CorrectEntry validates and applies a manager correction to worker hours.
// The correction is recorded in the revisions table within a single transaction.
func (s *WorkerHoursService) CorrectEntry(ctx context.Context, companyID, callerRole, callerUserID, entryID string, req dto.CorrectWorkerHoursRequest) error {
	if !s.isManager(callerRole) {
		return ErrForbidden
	}
	if req.NewHours < 0 || req.NewHours > 24 {
		return validationErr("Ispravljeni broj sati mora biti između 0 i 24")
	}
	if req.Reason == "" {
		return validationErr("Razlog ispravka je obavezan")
	}
	return s.repo.CorrectHours(ctx, companyID, entryID, req.NewHours, req.Reason, callerUserID)
}

// AddComment adds a private internal comment on a worker-hours entry.
// Comments are only visible to direktor and inženjer.
func (s *WorkerHoursService) AddComment(ctx context.Context, companyID, callerRole, callerUserID, entryID, text string) (*dto.WorkerHoursComment, error) {
	if !s.isManager(callerRole) {
		return nil, ErrForbidden
	}
	if text == "" {
		return nil, validationErr("Komentar ne smije biti prazan")
	}
	return s.repo.AddComment(ctx, companyID, entryID, callerUserID, text)
}
