package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

type WorkerHoursService struct {
	repo *repositories.WorkerHoursRepository
}

func NewWorkerHoursService(repo *repositories.WorkerHoursRepository) *WorkerHoursService {
	return &WorkerHoursService{repo: repo}
}

// ListCompanyProjects returns all active (non-archived) projects in the company.
// Radnik can choose any active project for the day they worked on.
func (s *WorkerHoursService) ListCompanyProjects(ctx context.Context, companyID string) ([]dto.WorkerProject, error) {
	return s.repo.ListCompanyActiveProjects(ctx, companyID)
}

// ListForDate returns the radnik's hour entries for a given date.
// If date is empty, today (Europe/Zagreb) is used.
func (s *WorkerHoursService) ListForDate(ctx context.Context, companyID, workerEmpID, date string) ([]dto.WorkerHoursEntry, error) {
	if date == "" {
		loc, _ := time.LoadLocation("Europe/Zagreb")
		date = time.Now().In(loc).Format("2006-01-02")
	}
	return s.repo.ListForWorkerDate(ctx, companyID, workerEmpID, date)
}

// Submit validates and upserts the radnik's hours for today.
func (s *WorkerHoursService) Submit(ctx context.Context, companyID, workerEmpID, callerUserID string, req dto.SubmitWorkerHoursRequest) (*dto.WorkerHoursEntry, error) {
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

	return s.repo.Upsert(ctx, companyID, workerEmpID, req.ProjectID, req.WorkDate, req.HoursWorked, req.Notes, callerUserID)
}
