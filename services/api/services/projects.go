package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Sentinel errors ───────────────────────────────────────────────────────────

var ErrProjectNotDeletable = errors.New("projects are not permanently deleted")

// ── Filters ───────────────────────────────────────────────────────────────────

type ProjectFilter struct {
	Search string
	Status *string
}

// ── Service ───────────────────────────────────────────────────────────────────

type ProjectService struct {
	db        *pgxpool.Pool
	projRepo  *repositories.ProjectRepository
	auditRepo *repositories.AuditRepository
}

func NewProjectService(
	db *pgxpool.Pool,
	projRepo *repositories.ProjectRepository,
	auditRepo *repositories.AuditRepository,
) *ProjectService {
	return &ProjectService{db: db, projRepo: projRepo, auditRepo: auditRepo}
}

// ── List ──────────────────────────────────────────────────────────────────────

func (s *ProjectService) List(ctx context.Context, companyID, callerRole, callerEmpID string, f ProjectFilter) ([]dto.ProjectListItem, error) {
	repoFilter := repositories.ProjectListFilter{
		Search: f.Search,
		Status: f.Status,
	}
	if callerRole == "poslovoda" && callerEmpID != "" {
		repoFilter.ScopeEmpID = &callerEmpID
	}

	rows, err := s.projRepo.List(ctx, companyID, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	items := make([]dto.ProjectListItem, 0, len(rows))
	for _, p := range rows {
		items = append(items, projToListItem(p))
	}
	return items, nil
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (s *ProjectService) GetByID(ctx context.Context, companyID, callerRole, callerEmpID, id string) (*dto.ProjectDetail, error) {
	proj, err := s.projRepo.GetByID(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	if callerRole == "poslovoda" {
		if proj.Status != "active" {
			return nil, ErrForbidden
		}
		assigned, err := s.projRepo.IsAssigned(ctx, companyID, id, callerEmpID, "poslovoda")
		if err != nil {
			return nil, fmt.Errorf("check assignment: %w", err)
		}
		if !assigned {
			return nil, ErrForbidden
		}
	}

	assignments, err := s.projRepo.ListAssignments(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}

	return projToDetail(proj, assignments), nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func (s *ProjectService) Create(ctx context.Context, companyID, callerUserID, ip, ua string, req dto.CreateProjectRequest) (*dto.ProjectDetail, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, validationErr("Project name is required")
	}

	if err := validateDates(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { tx.Rollback(ctx) }()

	proj, err := s.projRepo.CreateWithTx(ctx, tx, companyID, callerUserID, name, req.Address, req.Description, req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	if req.PrimaryPoslovodaID != nil && *req.PrimaryPoslovodaID != "" {
		if err := s.validatePoslovoda(ctx, companyID, *req.PrimaryPoslovodaID); err != nil {
			return nil, err
		}
		if err := s.projRepo.UpsertPoslovodaAssignment(ctx, tx, companyID, proj.ID, *req.PrimaryPoslovodaID, callerUserID); err != nil {
			return nil, fmt.Errorf("assign poslovoda: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project.create",
		EntityType: "project",
		EntityID:   &proj.ID,
		NewData:    proj,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	// Load fresh with joined data
	full, err := s.projRepo.GetByID(ctx, companyID, proj.ID)
	if err != nil {
		return nil, fmt.Errorf("reload project: %w", err)
	}
	assignments, _ := s.projRepo.ListAssignments(ctx, companyID, proj.ID)
	return projToDetail(full, assignments), nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (s *ProjectService) Update(ctx context.Context, companyID, callerUserID, ip, ua, id string, req dto.UpdateProjectRequest) (*dto.ProjectDetail, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, validationErr("Project name is required")
	}

	if err := validateDates(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	old, err := s.projRepo.GetByID(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update project fetch: %w", err)
	}

	proj, err := s.projRepo.Update(ctx, companyID, id, name, req.Address, req.Description, req.StartDate, req.EndDate)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project.update",
		EntityType: "project",
		EntityID:   &proj.ID,
		OldData:    old,
		NewData:    proj,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	full, err := s.projRepo.GetByID(ctx, companyID, proj.ID)
	if err != nil {
		return nil, fmt.Errorf("reload project: %w", err)
	}
	assignments, _ := s.projRepo.ListAssignments(ctx, companyID, proj.ID)
	return projToDetail(full, assignments), nil
}

// ── AssignPoslovoda ───────────────────────────────────────────────────────────

func (s *ProjectService) AssignPoslovoda(ctx context.Context, companyID, callerUserID, ip, ua, projectID, poslovodaID string) error {
	if _, err := s.projRepo.GetByID(ctx, companyID, projectID); errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("assign poslovoda: %w", err)
	}

	if err := s.validatePoslovoda(ctx, companyID, poslovodaID); err != nil {
		return err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { tx.Rollback(ctx) }()

	if err := s.projRepo.DeactivatePoslovodaAssignments(ctx, tx, companyID, projectID); err != nil {
		return fmt.Errorf("deactivate old assignment: %w", err)
	}

	if err := s.projRepo.UpsertPoslovodaAssignment(ctx, tx, companyID, projectID, poslovodaID, callerUserID); err != nil {
		return fmt.Errorf("create assignment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project.assign_poslovoda",
		EntityType: "project",
		EntityID:   &projectID,
		NewData:    map[string]string{"poslovoda_id": poslovodaID},
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return nil
}

// ── Close ─────────────────────────────────────────────────────────────────────

func (s *ProjectService) Close(ctx context.Context, companyID, callerUserID, ip, ua, id string) error {
	old, err := s.projRepo.GetByID(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("close project fetch: %w", err)
	}
	if old.Status == "closed" {
		return validationErr("Project is already closed")
	}

	if err := s.projRepo.Close(ctx, companyID, id, callerUserID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("close project: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project.close",
		EntityType: "project",
		EntityID:   &id,
		OldData:    map[string]string{"status": old.Status},
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return nil
}

// ── Archive ───────────────────────────────────────────────────────────────────

func (s *ProjectService) Archive(ctx context.Context, companyID, callerUserID, ip, ua, id string) error {
	old, err := s.projRepo.GetByID(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("archive project fetch: %w", err)
	}
	if old.Status == "archived" {
		return validationErr("Project is already archived")
	}

	if err := s.projRepo.Archive(ctx, companyID, id, callerUserID); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("archive project: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project.archive",
		EntityType: "project",
		EntityID:   &id,
		OldData:    map[string]string{"status": old.Status},
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return nil
}

// ── Reactivate ────────────────────────────────────────────────────────────────

func (s *ProjectService) Reactivate(ctx context.Context, companyID, callerUserID, ip, ua, id string) error {
	old, err := s.projRepo.GetByID(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("reactivate project fetch: %w", err)
	}
	if old.Status == "active" {
		return validationErr("Project is already active")
	}

	if err := s.projRepo.Reactivate(ctx, companyID, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("reactivate project: %w", err)
	}

	go s.auditRepo.Log(context.Background(), repositories.AuditParams{
		CompanyID:  companyID,
		UserID:     &callerUserID,
		Action:     "project.reactivate",
		EntityType: "project",
		EntityID:   &id,
		OldData:    map[string]string{"status": old.Status},
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return nil
}

// ── ListAssignments ───────────────────────────────────────────────────────────

func (s *ProjectService) ListAssignments(ctx context.Context, companyID, callerRole, callerEmpID, projectID string) ([]dto.AssignmentItem, error) {
	proj, err := s.projRepo.GetByID(ctx, companyID, projectID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}

	if callerRole == "poslovoda" {
		if proj.Status != "active" {
			return nil, ErrForbidden
		}
		assigned, err := s.projRepo.IsAssigned(ctx, companyID, projectID, callerEmpID, "poslovoda")
		if err != nil || !assigned {
			return nil, ErrForbidden
		}
	}

	assignments, err := s.projRepo.ListAssignments(ctx, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}

	items := make([]dto.AssignmentItem, 0, len(assignments))
	for _, a := range assignments {
		items = append(items, toAssignmentItem(a))
	}
	return items, nil
}

// ── EnsureProjectIsActive ─────────────────────────────────────────────────────

// EnsureProjectIsActive returns ErrProjectNotActive if the project is not active.
// Returns ErrNotFound if the project doesn't exist in the company.
func (s *ProjectService) EnsureProjectIsActive(ctx context.Context, companyID, projectID string) error {
	status, err := s.projRepo.GetStatus(ctx, companyID, projectID)
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check project status: %w", err)
	}
	if status != "active" {
		return ErrProjectNotActive
	}
	return nil
}

// ── Archive list ──────────────────────────────────────────────────────────────

type ArchiveFilter struct {
	Search      string
	Status      string
	DateFrom    *string
	DateTo      *string
	PoslovodaID string
	Page        int
	PerPage     int
}

func (s *ProjectService) ListArchive(ctx context.Context, companyID string, f ArchiveFilter) (*dto.ArchiveListResponse, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 25
	}

	filter := repositories.ArchiveListFilter{
		Search:      f.Search,
		Status:      f.Status,
		DateFrom:    f.DateFrom,
		DateTo:      f.DateTo,
		PoslovodaID: f.PoslovodaID,
		Limit:       f.PerPage,
		Offset:      (f.Page - 1) * f.PerPage,
	}

	items, total, err := s.projRepo.ListArchive(ctx, companyID, filter)
	if err != nil {
		return nil, fmt.Errorf("list archive: %w", err)
	}
	if items == nil {
		items = []dto.ArchiveListItem{}
	}

	totalPages := (total + f.PerPage - 1) / f.PerPage
	return &dto.ArchiveListResponse{
		Projects:   items,
		Total:      total,
		Page:       f.Page,
		PerPage:    f.PerPage,
		TotalPages: totalPages,
	}, nil
}

// ── Archive summary ───────────────────────────────────────────────────────────

func (s *ProjectService) GetArchiveSummary(ctx context.Context, companyID, callerRole, callerEmpID, id string) (*dto.ArchiveSummary, error) {
	summary, err := s.projRepo.GetArchiveSummary(ctx, companyID, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get archive summary: %w", err)
	}

	// Poslovoda may only see projects they were ever assigned to
	if callerRole == "poslovoda" {
		assigned := false
		for _, a := range summary.AssignedPoslovode {
			if a.ID == callerEmpID {
				assigned = true
				break
			}
		}
		if !assigned {
			return nil, ErrForbidden
		}
	}

	return summary, nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// validatePoslovoda checks that the employee exists, belongs to the company,
// has role=poslovoda, and is active — all in one query.
func (s *ProjectService) validatePoslovoda(ctx context.Context, companyID, poslovodaID string) error {
	var role string
	var active bool
	err := s.db.QueryRow(ctx, `
		SELECT role, active FROM employees
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, poslovodaID, companyID).Scan(&role, &active)
	if errors.Is(err, pgxv5.ErrNoRows) {
		return validationErr("Poslovođa not found in this company")
	}
	if err != nil {
		return fmt.Errorf("validate poslovoda: %w", err)
	}
	if role != "poslovoda" {
		return validationErr("Assigned employee must have the poslovoda role")
	}
	if !active {
		return validationErr("Assigned poslovoda must be active")
	}
	return nil
}

func validateDates(startDate, endDate *string) error {
	if startDate == nil || endDate == nil {
		return nil
	}
	start, errS := time.Parse("2006-01-02", *startDate)
	end, errE := time.Parse("2006-01-02", *endDate)
	if errS != nil {
		return validationErr("start_date must be in YYYY-MM-DD format")
	}
	if errE != nil {
		return validationErr("end_date must be in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return validationErr("end_date must not be before start_date")
	}
	return nil
}

// ── Mapping helpers ───────────────────────────────────────────────────────────

func projToListItem(p repositories.Project) dto.ProjectListItem {
	return dto.ProjectListItem{
		ID:                   p.ID,
		Name:                 p.Name,
		Address:              p.Address,
		Description:          p.Description,
		Status:               p.Status,
		StartDate:            p.StartDate,
		EndDate:              p.EndDate,
		ClosedAt:             p.ClosedAt,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
		PrimaryPoslovodaID:   p.PrimaryPoslovodaID,
		PrimaryPoslovodaName: p.PrimaryPoslovodaName,
		AssignmentCount:      p.AssignmentCount,
	}
}

func projToDetail(p *repositories.Project, assignments []repositories.ProjectAssignment) *dto.ProjectDetail {
	items := make([]dto.AssignmentItem, 0, len(assignments))
	for _, a := range assignments {
		items = append(items, toAssignmentItem(a))
	}
	return &dto.ProjectDetail{
		ID:                   p.ID,
		CompanyID:            p.CompanyID,
		Name:                 p.Name,
		Address:              p.Address,
		Description:          p.Description,
		Status:               p.Status,
		StartDate:            p.StartDate,
		EndDate:              p.EndDate,
		ClosedAt:             p.ClosedAt,
		CreatedBy:            p.CreatedBy,
		ClosedBy:             p.ClosedBy,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
		PrimaryPoslovodaID:   p.PrimaryPoslovodaID,
		PrimaryPoslovodaName: p.PrimaryPoslovodaName,
		MaterialsCount:       p.MaterialsCount,
		DailyReportsCount:    p.DailyReportsCount,
		Assignments:          items,
	}
}

func toAssignmentItem(a repositories.ProjectAssignment) dto.AssignmentItem {
	return dto.AssignmentItem{
		ID:            a.ID,
		EmployeeID:    a.EmployeeID,
		EmployeeName:  a.EmployeeName,
		EmployeeRole:  a.EmployeeRole,
		RoleOnProject: a.RoleOnProject,
		Active:        a.Active,
		AssignedAt:    a.AssignedAt,
	}
}
