package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

// ── Models ────────────────────────────────────────────────────────────────────

type Project struct {
	ID                   string
	CompanyID            string
	Name                 string
	Address              *string
	Description          *string
	Status               string
	StartDate            *string // "YYYY-MM-DD" from start_date::text
	EndDate              *string // "YYYY-MM-DD" from end_date::text
	ClosedAt             *time.Time
	ClosedBy             *string
	CreatedBy            *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	PrimaryPoslovodaID   *string
	PrimaryPoslovodaName *string
	AssignmentCount      int
	MaterialsCount       int
	DailyReportsCount    int
}

type ProjectAssignment struct {
	ID            string
	CompanyID     string
	ProjectID     string
	EmployeeID    string
	EmployeeName  string
	EmployeeRole  string
	RoleOnProject string
	Active        bool
	AssignedAt    time.Time
}

// ── Filter ────────────────────────────────────────────────────────────────────

type ProjectListFilter struct {
	Search     string
	Status     *string
	ScopeEmpID *string // non-nil → poslovoda scope: active + assigned only
}

// ── Repository ────────────────────────────────────────────────────────────────

type ProjectRepository struct {
	db *pgxpool.Pool
}

func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// ── List ──────────────────────────────────────────────────────────────────────

const projectSelectCols = `
	p.id::text,
	p.company_id::text,
	p.name,
	p.address,
	p.description,
	p.status,
	p.start_date::text,
	p.end_date::text,
	p.closed_at,
	p.created_by::text,
	p.created_at,
	p.updated_at,
	pa.employee_id::text AS primary_poslovoda_id,
	CASE WHEN e.id IS NOT NULL THEN (e.first_name || ' ' || e.last_name) END AS primary_poslovoda_name,
	(SELECT COUNT(*) FROM project_assignments pac
	 WHERE pac.project_id = p.id AND pac.active = true AND pac.company_id = p.company_id) AS assignment_count
`

const projectJoins = `
	FROM projects p
	LEFT JOIN project_assignments pa ON
		pa.project_id = p.id
		AND pa.role_on_project = 'poslovoda'
		AND pa.active = true
		AND pa.company_id = p.company_id
	LEFT JOIN employees e ON e.id = pa.employee_id
`

func (r *ProjectRepository) List(ctx context.Context, companyID string, filter ProjectListFilter) ([]Project, error) {
	args := []interface{}{companyID}
	conditions := []string{"p.company_id = $1::uuid"}

	if filter.ScopeEmpID != nil {
		// Poslovoda: only active projects where they are assigned
		conditions = append(conditions, "p.status = 'active'")
		args = append(args, *filter.ScopeEmpID)
		n := len(args)
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM project_assignments pa_scope
			WHERE pa_scope.project_id = p.id
			AND pa_scope.employee_id = $%d::uuid
			AND pa_scope.role_on_project = 'poslovoda'
			AND pa_scope.active = true
			AND pa_scope.company_id = p.company_id
		)`, n))
	}

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		n := len(args)
		conditions = append(conditions, fmt.Sprintf("(p.name ILIKE $%d OR p.address ILIKE $%d)", n, n))
	}

	if filter.Status != nil && filter.ScopeEmpID == nil {
		// Status filter only for non-poslovoda (poslovoda is always scoped to active)
		args = append(args, *filter.Status)
		conditions = append(conditions, fmt.Sprintf("p.status = $%d", len(args)))
	}

	where := strings.Join(conditions, " AND ")
	query := fmt.Sprintf("SELECT %s %s WHERE %s ORDER BY p.created_at DESC",
		projectSelectCols, projectJoins, where)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("projects.List: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var assignmentCount int64
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.Name, &p.Address, &p.Description,
			&p.Status, &p.StartDate, &p.EndDate, &p.ClosedAt,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
			&p.PrimaryPoslovodaID, &p.PrimaryPoslovodaName,
			&assignmentCount,
		); err != nil {
			return nil, fmt.Errorf("projects.List scan: %w", err)
		}
		p.AssignmentCount = int(assignmentCount)
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (r *ProjectRepository) GetByID(ctx context.Context, companyID, id string) (*Project, error) {
	var p Project
	var assignmentCount, materialsCount, dailyReportsCount int64

	err := r.db.QueryRow(ctx, `
		SELECT
			p.id::text,
			p.company_id::text,
			p.name,
			p.address,
			p.description,
			p.status,
			p.start_date::text,
			p.end_date::text,
			p.closed_at,
			p.created_by::text,
			p.closed_by::text,
			p.created_at,
			p.updated_at,
			pa.employee_id::text AS primary_poslovoda_id,
			CASE WHEN e.id IS NOT NULL THEN (e.first_name || ' ' || e.last_name) END AS primary_poslovoda_name,
			(SELECT COUNT(*) FROM project_assignments pac
			 WHERE pac.project_id = p.id AND pac.active = true AND pac.company_id = p.company_id) AS assignment_count,
			(SELECT COUNT(*) FROM project_materials pm
			 WHERE pm.project_id = p.id AND pm.company_id = p.company_id AND pm.active = true) AS materials_count,
			(SELECT COUNT(*) FROM daily_reports dr
			 WHERE dr.project_id = p.id AND dr.company_id = p.company_id) AS daily_reports_count
		FROM projects p
		LEFT JOIN project_assignments pa ON
			pa.project_id = p.id
			AND pa.role_on_project = 'poslovoda'
			AND pa.active = true
			AND pa.company_id = p.company_id
		LEFT JOIN employees e ON e.id = pa.employee_id
		WHERE p.id = $1::uuid AND p.company_id = $2::uuid
	`, id, companyID).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.Address, &p.Description,
		&p.Status, &p.StartDate, &p.EndDate, &p.ClosedAt,
		&p.CreatedBy, &p.ClosedBy, &p.CreatedAt, &p.UpdatedAt,
		&p.PrimaryPoslovodaID, &p.PrimaryPoslovodaName,
		&assignmentCount, &materialsCount, &dailyReportsCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("projects.GetByID: %w", err)
	}

	p.AssignmentCount = int(assignmentCount)
	p.MaterialsCount = int(materialsCount)
	p.DailyReportsCount = int(dailyReportsCount)
	return &p, nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func (r *ProjectRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, companyID, createdBy, name string, address, description, startDate, endDate *string) (*Project, error) {
	var p Project
	err := tx.QueryRow(ctx, `
		INSERT INTO projects (company_id, name, address, description, status, start_date, end_date, created_by)
		VALUES ($1::uuid, $2, $3, $4, 'active', $5::date, $6::date, $7::uuid)
		RETURNING
			id::text, company_id::text, name, address, description, status,
			start_date::text, end_date::text, closed_at, created_by::text, closed_by::text,
			created_at, updated_at
	`, companyID, name, address, description, startDate, endDate, createdBy).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.Address, &p.Description, &p.Status,
		&p.StartDate, &p.EndDate, &p.ClosedAt, &p.CreatedBy, &p.ClosedBy,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("projects.CreateWithTx: %w", err)
	}
	return &p, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (r *ProjectRepository) Update(ctx context.Context, companyID, id, name string, address, description, startDate, endDate *string) (*Project, error) {
	var p Project
	err := r.db.QueryRow(ctx, `
		UPDATE projects
		SET name = $1, address = $2, description = $3, start_date = $4::date, end_date = $5::date
		WHERE id = $6::uuid AND company_id = $7::uuid
		RETURNING
			id::text, company_id::text, name, address, description, status,
			start_date::text, end_date::text, closed_at, created_by::text, closed_by::text,
			created_at, updated_at
	`, name, address, description, startDate, endDate, id, companyID).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.Address, &p.Description, &p.Status,
		&p.StartDate, &p.EndDate, &p.ClosedAt, &p.CreatedBy, &p.ClosedBy,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("projects.Update: %w", err)
	}
	return &p, nil
}

// ── Status transitions ────────────────────────────────────────────────────────

func (r *ProjectRepository) Close(ctx context.Context, companyID, id, closedByUserID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE projects
		SET status = 'closed', closed_at = NOW(), closed_by = $1::uuid
		WHERE id = $2::uuid AND company_id = $3::uuid AND status != 'closed'
	`, closedByUserID, id, companyID)
	if err != nil {
		return fmt.Errorf("projects.Close: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProjectRepository) Archive(ctx context.Context, companyID, id, closedByUserID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE projects
		SET status = 'archived',
		    closed_at = COALESCE(closed_at, NOW()),
		    closed_by = COALESCE(closed_by, $1::uuid)
		WHERE id = $2::uuid AND company_id = $3::uuid AND status != 'archived'
	`, closedByUserID, id, companyID)
	if err != nil {
		return fmt.Errorf("projects.Archive: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProjectRepository) Reactivate(ctx context.Context, companyID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE projects
		SET status = 'active', closed_at = NULL, closed_by = NULL
		WHERE id = $1::uuid AND company_id = $2::uuid AND status != 'active'
	`, id, companyID)
	if err != nil {
		return fmt.Errorf("projects.Reactivate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Assignments ───────────────────────────────────────────────────────────────

func (r *ProjectRepository) ListAssignments(ctx context.Context, companyID, projectID string) ([]ProjectAssignment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			pa.id::text,
			pa.company_id::text,
			pa.project_id::text,
			pa.employee_id::text,
			(e.first_name || ' ' || e.last_name) AS employee_name,
			e.role AS employee_role,
			pa.role_on_project,
			pa.active,
			pa.assigned_at
		FROM project_assignments pa
		JOIN employees e ON e.id = pa.employee_id
		WHERE pa.project_id = $1::uuid AND pa.company_id = $2::uuid
		ORDER BY pa.assigned_at DESC
	`, projectID, companyID)
	if err != nil {
		return nil, fmt.Errorf("projects.ListAssignments: %w", err)
	}
	defer rows.Close()

	var assignments []ProjectAssignment
	for rows.Next() {
		var a ProjectAssignment
		if err := rows.Scan(
			&a.ID, &a.CompanyID, &a.ProjectID, &a.EmployeeID,
			&a.EmployeeName, &a.EmployeeRole, &a.RoleOnProject,
			&a.Active, &a.AssignedAt,
		); err != nil {
			return nil, fmt.Errorf("projects.ListAssignments scan: %w", err)
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

func (r *ProjectRepository) DeactivatePoslovodaAssignments(ctx context.Context, tx pgx.Tx, companyID, projectID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE project_assignments
		SET active = false
		WHERE project_id = $1::uuid AND company_id = $2::uuid AND role_on_project = 'poslovoda' AND active = true
	`, projectID, companyID)
	if err != nil {
		return fmt.Errorf("projects.DeactivatePoslovodaAssignments: %w", err)
	}
	return nil
}

func (r *ProjectRepository) UpsertPoslovodaAssignment(ctx context.Context, tx pgx.Tx, companyID, projectID, employeeID, assignedByUserID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO project_assignments (company_id, project_id, employee_id, role_on_project, active, assigned_by, assigned_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'poslovoda', true, $4::uuid, NOW())
		ON CONFLICT (project_id, employee_id, company_id)
		DO UPDATE SET active = true, assigned_by = EXCLUDED.assigned_by, assigned_at = NOW()
	`, companyID, projectID, employeeID, assignedByUserID)
	if err != nil {
		return fmt.Errorf("projects.UpsertPoslovodaAssignment: %w", err)
	}
	return nil
}

func (r *ProjectRepository) IsAssigned(ctx context.Context, companyID, projectID, employeeID, roleOnProject string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM project_assignments
			WHERE project_id = $1::uuid
			AND company_id = $2::uuid
			AND employee_id = $3::uuid
			AND role_on_project = $4
			AND active = true
		)
	`, projectID, companyID, employeeID, roleOnProject).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("projects.IsAssigned: %w", err)
	}
	return exists, nil
}

// ── Archive ───────────────────────────────────────────────────────────────────

// GetStatus returns the current status of a project scoped to a company.
func (r *ProjectRepository) GetStatus(ctx context.Context, companyID, id string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx,
		`SELECT status FROM projects WHERE id = $1::uuid AND company_id = $2::uuid`,
		id, companyID,
	).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("projects.GetStatus: %w", err)
	}
	return status, nil
}

// ArchiveListFilter is the input filter for ListArchive.
type ArchiveListFilter struct {
	Search      string
	Status      string  // "" = both closed+archived, else "closed" or "archived"
	DateFrom    *string // YYYY-MM-DD, filter on closed_at
	DateTo      *string // YYYY-MM-DD, filter on closed_at
	PoslovodaID string  // filter by any assignment (ever assigned)
	Limit       int
	Offset      int
}

// ListArchive returns closed/archived projects with pagination.
func (r *ProjectRepository) ListArchive(ctx context.Context, companyID string, filter ArchiveListFilter) ([]dto.ArchiveListItem, int, error) {
	args := []interface{}{companyID}
	conditions := []string{"p.company_id = $1::uuid"}

	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf("p.status = $%d", len(args)))
	} else {
		conditions = append(conditions, "p.status IN ('closed', 'archived')")
	}

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		n := len(args)
		conditions = append(conditions, fmt.Sprintf("(p.name ILIKE $%d OR p.address ILIKE $%d)", n, n))
	}

	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("p.closed_at::date >= $%d::date", len(args)))
	}

	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("p.closed_at::date <= $%d::date", len(args)))
	}

	if filter.PoslovodaID != "" {
		args = append(args, filter.PoslovodaID)
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM project_assignments pa_f
			WHERE pa_f.project_id = p.id
			  AND pa_f.company_id = p.company_id
			  AND pa_f.employee_id = $%d::uuid
			  AND pa_f.role_on_project = 'poslovoda'
		)`, len(args)))
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQ := "SELECT COUNT(*)::int FROM projects p WHERE " + where
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("projects.ListArchive count: %w", err)
	}

	// Data query
	args = append(args, filter.Limit, filter.Offset)
	limitN := len(args) - 1
	offsetN := len(args)

	dataQ := fmt.Sprintf(`
		SELECT
			p.id::text,
			p.name,
			p.address,
			p.status,
			p.start_date::text,
			p.end_date::text,
			p.closed_at,
			COALESCE(ecb.first_name || ' ' || ecb.last_name, ucb.email) AS closed_by_name,
			CASE WHEN e.id IS NOT NULL THEN (e.first_name || ' ' || e.last_name) END AS primary_poslovoda_name,
			(SELECT COUNT(*)::int FROM daily_reports dr
			 WHERE dr.project_id = p.id AND dr.company_id = p.company_id) AS daily_reports_count,
			(SELECT COUNT(*)::int FROM material_purchase_sessions mps
			 WHERE mps.project_id = p.id AND mps.company_id = p.company_id) AS material_purchases_count,
			(SELECT COUNT(*)::int FROM project_materials pm
			 WHERE pm.project_id = p.id AND pm.company_id = p.company_id) AS project_materials_count
		FROM projects p
		LEFT JOIN project_assignments pa ON
			pa.project_id = p.id
			AND pa.role_on_project = 'poslovoda'
			AND pa.active = true
			AND pa.company_id = p.company_id
		LEFT JOIN employees e ON e.id = pa.employee_id
		LEFT JOIN users ucb ON ucb.id = p.closed_by
		LEFT JOIN employees ecb ON ecb.id = ucb.employee_id
		WHERE %s
		ORDER BY p.closed_at DESC NULLS LAST
		LIMIT $%d OFFSET $%d
	`, where, limitN, offsetN)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("projects.ListArchive query: %w", err)
	}
	defer rows.Close()

	var items []dto.ArchiveListItem
	for rows.Next() {
		var it dto.ArchiveListItem
		if err := rows.Scan(
			&it.ID, &it.Name, &it.Address, &it.Status,
			&it.StartDate, &it.EndDate, &it.ClosedAt,
			&it.ClosedByName, &it.PrimaryPoslovodaName,
			&it.DailyReportsCount, &it.MaterialPurchasesCount, &it.ProjectMaterialsCount,
		); err != nil {
			return nil, 0, fmt.Errorf("projects.ListArchive scan: %w", err)
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

// GetArchiveSummary returns a detailed summary of counts for a project.
func (r *ProjectRepository) GetArchiveSummary(ctx context.Context, companyID, id string) (*dto.ArchiveSummary, error) {
	var s dto.ArchiveSummary
	var closedByName *string

	err := r.db.QueryRow(ctx, `
		SELECT
			p.id::text,
			p.name,
			p.address,
			p.status,
			p.start_date::text,
			p.end_date::text,
			p.closed_at,
			CASE WHEN ecb.id IS NOT NULL
				THEN (ecb.first_name || ' ' || ecb.last_name)
				ELSE ucb.email
			END AS closed_by_name,
			(SELECT COUNT(*)::int FROM daily_reports dr
			 WHERE dr.project_id = p.id AND dr.company_id = p.company_id),
			(SELECT COALESCE(SUM(dwh.hours_worked), 0)
			 FROM daily_report_worker_hours dwh
			 JOIN daily_reports dr ON dr.id = dwh.daily_report_id
			 WHERE dr.project_id = p.id AND dr.company_id = p.company_id),
			(SELECT COUNT(*)::int FROM daily_report_activities dra
			 JOIN daily_reports dr ON dr.id = dra.daily_report_id
			 WHERE dr.project_id = p.id AND dr.company_id = p.company_id),
			(SELECT COUNT(*)::int FROM daily_report_activities dra
			 JOIN daily_reports dr ON dr.id = dra.daily_report_id
			 WHERE dr.project_id = p.id AND dr.company_id = p.company_id AND dra.is_vtk = true),
			(SELECT COUNT(*)::int FROM material_purchase_sessions mps
			 WHERE mps.project_id = p.id AND mps.company_id = p.company_id),
			(SELECT COUNT(*)::int FROM material_purchase_items mpi
			 JOIN material_purchase_sessions mps ON mps.id = mpi.purchase_session_id
			 WHERE mps.project_id = p.id AND mps.company_id = p.company_id),
			(SELECT COUNT(*)::int FROM project_materials pm
			 WHERE pm.project_id = p.id AND pm.company_id = p.company_id),
			(SELECT COUNT(*)::int FROM employee_material_responsibility emr
			 JOIN project_materials pm ON pm.id = emr.project_material_id
			 WHERE pm.project_id = p.id AND pm.company_id = p.company_id),
			(SELECT COUNT(*)::int FROM asset_transfers atr
			 WHERE atr.project_id = p.id AND atr.company_id = p.company_id)
		FROM projects p
		LEFT JOIN users ucb ON ucb.id = p.closed_by
		LEFT JOIN employees ecb ON ecb.id = ucb.employee_id
		WHERE p.id = $1::uuid AND p.company_id = $2::uuid
	`, id, companyID).Scan(
		&s.ID, &s.Name, &s.Address, &s.Status,
		&s.StartDate, &s.EndDate, &s.ClosedAt,
		&closedByName,
		&s.DailyReportsCount, &s.WorkerHoursTotal,
		&s.ActivitiesCount, &s.VtkCount,
		&s.PurchaseSessionsCount, &s.PurchaseItemsCount,
		&s.ProjectMaterialsCount, &s.MaterialResponsibilityCount,
		&s.TransfersCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("projects.GetArchiveSummary: %w", err)
	}
	s.ClosedByName = closedByName

	// Fetch assigned poslovode (all, including inactive)
	rows, err := r.db.Query(ctx, `
		SELECT e.id::text, (e.first_name || ' ' || e.last_name), pa.active
		FROM project_assignments pa
		JOIN employees e ON e.id = pa.employee_id
		WHERE pa.project_id = $1::uuid AND pa.company_id = $2::uuid AND pa.role_on_project = 'poslovoda'
		ORDER BY pa.assigned_at DESC
	`, id, companyID)
	if err != nil {
		return nil, fmt.Errorf("projects.GetArchiveSummary assignees: %w", err)
	}
	defer rows.Close()

	s.AssignedPoslovode = []dto.ArchiveSummaryAssignee{}
	for rows.Next() {
		var a dto.ArchiveSummaryAssignee
		if err := rows.Scan(&a.ID, &a.Name, &a.Active); err != nil {
			return nil, fmt.Errorf("projects.GetArchiveSummary assignee scan: %w", err)
		}
		s.AssignedPoslovode = append(s.AssignedPoslovode, a)
	}
	return &s, rows.Err()
}
