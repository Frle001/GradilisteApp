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

var ErrDailyReportNotFound = errors.New("daily report not found")
var ErrDailyReportDuplicate = errors.New("report already exists for this project, poslovoda, and date")

type DailyReportRepository struct {
	db *pgxpool.Pool
}

func NewDailyReportRepository(db *pgxpool.Pool) *DailyReportRepository {
	return &DailyReportRepository{db: db}
}

// ── Validation helpers ────────────────────────────────────────────────────────

// GetProjectStatus returns the project status or empty string if not found.
func (r *DailyReportRepository) GetProjectStatus(ctx context.Context, projectID, companyID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx,
		`SELECT status FROM projects WHERE id = $1::uuid AND company_id = $2::uuid`,
		projectID, companyID,
	).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return status, err
}

// GetWorkerInfo returns (role, supervisorID) for a worker. supervisorID is nil if unset.
func (r *DailyReportRepository) GetWorkerInfo(ctx context.Context, workerID, companyID string) (role string, supervisorID *string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT role, supervisor_id::text FROM employees WHERE id = $1::uuid AND company_id = $2::uuid AND active = true`,
		workerID, companyID,
	).Scan(&role, &supervisorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, nil
	}
	return role, supervisorID, err
}

// IsMaterialInProject checks if a material belongs to the given project and is active.
func (r *DailyReportRepository) IsMaterialInProject(ctx context.Context, materialID, projectID, companyID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_materials
			WHERE id = $1::uuid AND project_id = $2::uuid AND company_id = $3::uuid AND active = true
		)
	`, materialID, projectID, companyID).Scan(&exists)
	return exists, err
}

// GetByIDForEdit returns (status, poslovodaID, projectID) for edit-permission checks.
func (r *DailyReportRepository) GetByIDForEdit(ctx context.Context, id, companyID string) (status, poslovodaID, projectID string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT status, poslovoda_id::text, project_id::text FROM daily_reports WHERE id = $1::uuid AND company_id = $2::uuid`,
		id, companyID,
	).Scan(&status, &poslovodaID, &projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", ErrDailyReportNotFound
	}
	return status, poslovodaID, projectID, err
}

// ActivityForApproval holds data for one activity needed during report approval.
type ActivityForApproval struct {
	ID                 string
	ProjectMaterialID  *string
	CustomMaterialName *string
	Quantity           float64
	Unit               string
	ActivityType       string
	IsVTK              bool
}

// GetActivitiesForApproval returns montaža and demontaža activities for a report.
func (r *DailyReportRepository) GetActivitiesForApproval(ctx context.Context, reportID, companyID string) ([]ActivityForApproval, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id::text,
			project_material_id::text,
			custom_material_name,
			quantity,
			unit,
			activity_type,
			is_vtk
		FROM daily_report_activities
		WHERE daily_report_id = $1::uuid
		  AND company_id = $2::uuid
		  AND activity_type IN ('montaza', 'demontaza')
		ORDER BY created_at
	`, reportID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var acts []ActivityForApproval
	for rows.Next() {
		var a ActivityForApproval
		if err := rows.Scan(
			&a.ID,
			&a.ProjectMaterialID,
			&a.CustomMaterialName,
			&a.Quantity,
			&a.Unit,
			&a.ActivityType,
			&a.IsVTK,
		); err != nil {
			return nil, err
		}
		acts = append(acts, a)
	}
	return acts, rows.Err()
}

// ── List ──────────────────────────────────────────────────────────────────────

func (r *DailyReportRepository) List(ctx context.Context, companyID string, f dto.DailyReportFilter) ([]dto.DailyReportListItem, error) {
	args := []interface{}{companyID}
	conditions := []string{"dr.company_id = $1::uuid"}
	idx := 2

	if f.PoslovodaID != nil {
		args = append(args, *f.PoslovodaID)
		conditions = append(conditions, fmt.Sprintf("dr.poslovoda_id = $%d::uuid", idx))
		idx++
	}
	if f.ProjectID != nil {
		args = append(args, *f.ProjectID)
		conditions = append(conditions, fmt.Sprintf("dr.project_id = $%d::uuid", idx))
		idx++
	}
	if f.Status != nil {
		args = append(args, *f.Status)
		conditions = append(conditions, fmt.Sprintf("dr.status = $%d", idx))
		idx++
	}
	if f.DateFrom != nil {
		args = append(args, *f.DateFrom)
		conditions = append(conditions, fmt.Sprintf("dr.report_date >= $%d::date", idx))
		idx++
	}
	if f.DateTo != nil {
		args = append(args, *f.DateTo)
		conditions = append(conditions, fmt.Sprintf("dr.report_date <= $%d::date", idx))
		idx++
	}
	_ = idx

	where := "WHERE " + strings.Join(conditions, " AND ")

	q := fmt.Sprintf(`
		SELECT
			dr.id::text,
			dr.project_id::text,
			p.name AS project_name,
			dr.poslovoda_id::text,
			e.first_name || ' ' || e.last_name AS poslovoda_name,
			dr.report_date::text,
			dr.status,
			COUNT(DISTINCT drwh.id) AS worker_hours_count,
			COALESCE(SUM(drwh.hours_worked), 0) AS total_hours,
			COUNT(DISTINCT dra.id) AS activities_count,
			dr.submitted_at,
			dr.created_at,
			dr.updated_at
		FROM daily_reports dr
		JOIN projects p ON p.id = dr.project_id
		JOIN employees e ON e.id = dr.poslovoda_id
		LEFT JOIN daily_report_worker_hours drwh ON drwh.daily_report_id = dr.id AND drwh.company_id = dr.company_id
		LEFT JOIN daily_report_activities dra ON dra.daily_report_id = dr.id AND dra.company_id = dr.company_id
		%s
		GROUP BY dr.id, p.name, e.first_name, e.last_name
		ORDER BY dr.report_date DESC, dr.created_at DESC
	`, where)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.DailyReportListItem
	for rows.Next() {
		var item dto.DailyReportListItem
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.ProjectName,
			&item.PoslovodaID, &item.PoslovodaName,
			&item.ReportDate, &item.Status,
			&item.WorkerHoursCount, &item.TotalHours, &item.ActivitiesCount,
			&item.SubmittedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []dto.DailyReportListItem{}
	}
	return items, rows.Err()
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func (r *DailyReportRepository) GetByID(ctx context.Context, id, companyID string) (*dto.DailyReportDetail, error) {
	var d dto.DailyReportDetail

	err := r.db.QueryRow(ctx, `
		SELECT
			dr.id::text,
			dr.project_id::text, p.name, p.status,
			dr.poslovoda_id::text, e.first_name, e.last_name,
			dr.report_date::text,
			dr.status,
			dr.notes,
			dr.submitted_at,
			dr.created_at,
			dr.updated_at
		FROM daily_reports dr
		JOIN projects p ON p.id = dr.project_id
		JOIN employees e ON e.id = dr.poslovoda_id
		WHERE dr.id = $1::uuid AND dr.company_id = $2::uuid
	`, id, companyID).Scan(
		&d.ID,
		&d.Project.ID, &d.Project.Name, &d.Project.Status,
		&d.Poslovoda.ID, &d.Poslovoda.FirstName, &d.Poslovoda.LastName,
		&d.ReportDate, &d.Status, &d.Notes,
		&d.SubmittedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDailyReportNotFound
	}
	if err != nil {
		return nil, err
	}
	d.Poslovoda.FullName = d.Poslovoda.FirstName + " " + d.Poslovoda.LastName

	// Worker hours
	whRows, err := r.db.Query(ctx, `
		SELECT
			drwh.id::text,
			drwh.worker_id::text,
			e.first_name || ' ' || e.last_name AS worker_name,
			drwh.hours_worked,
			drwh.notes
		FROM daily_report_worker_hours drwh
		JOIN employees e ON e.id = drwh.worker_id
		WHERE drwh.daily_report_id = $1::uuid AND drwh.company_id = $2::uuid
		ORDER BY e.last_name, e.first_name
	`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer whRows.Close()
	d.WorkerHours = make([]dto.DailyReportWorkerHours, 0)
	for whRows.Next() {
		var wh dto.DailyReportWorkerHours
		if err := whRows.Scan(&wh.ID, &wh.WorkerID, &wh.WorkerName, &wh.HoursWorked, &wh.Notes); err != nil {
			return nil, err
		}
		d.WorkerHours = append(d.WorkerHours, wh)
	}
	if err := whRows.Err(); err != nil {
		return nil, err
	}

	// Activities
	actRows, err := r.db.Query(ctx, `
		SELECT
			dra.id::text,
			dra.project_material_id::text,
			CASE WHEN dra.is_vtk THEN dra.custom_material_name ELSE pm.material_name END AS material_name,
			dra.custom_material_name,
			dra.quantity,
			dra.unit,
			dra.activity_type,
			dra.is_vtk,
			dra.notes
		FROM daily_report_activities dra
		LEFT JOIN project_materials pm ON pm.id = dra.project_material_id
		WHERE dra.daily_report_id = $1::uuid AND dra.company_id = $2::uuid
		ORDER BY dra.created_at
	`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer actRows.Close()
	d.Activities = make([]dto.DailyReportActivity, 0)
	for actRows.Next() {
		var a dto.DailyReportActivity
		if err := actRows.Scan(
			&a.ID, &a.ProjectMaterialID, &a.MaterialName,
			&a.CustomMaterialName, &a.Quantity, &a.Unit,
			&a.ActivityType, &a.IsVTK, &a.Notes,
		); err != nil {
			return nil, err
		}
		d.Activities = append(d.Activities, a)
	}
	if err := actRows.Err(); err != nil {
		return nil, err
	}

	return &d, nil
}

// ── Create (transactional) ────────────────────────────────────────────────────

func (r *DailyReportRepository) Create(ctx context.Context, tx pgx.Tx, companyID, projectID, poslovodaEmpID, userID string, req dto.CreateDailyReportRequest) (string, error) {
	var reportID string
	err := tx.QueryRow(ctx, `
		INSERT INTO daily_reports
			(company_id, project_id, poslovoda_id, report_date, status, notes, submitted_by, submitted_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::date, 'submitted', $5, $6::uuid, NOW())
		RETURNING id::text
	`, companyID, projectID, poslovodaEmpID, req.ReportDate, req.Notes, userID).Scan(&reportID)
	if err != nil {
		if isDuplicateKey(err) {
			return "", ErrDailyReportDuplicate
		}
		return "", err
	}

	if err := r.insertWorkerHours(ctx, tx, companyID, reportID, req.WorkerHours); err != nil {
		return "", err
	}
	if err := r.insertActivities(ctx, tx, companyID, reportID, req.Activities); err != nil {
		return "", err
	}
	return reportID, nil
}

// ── Update (transactional) ────────────────────────────────────────────────────

func (r *DailyReportRepository) Update(ctx context.Context, tx pgx.Tx, id, companyID string, req dto.CreateDailyReportRequest) error {
	tag, err := tx.Exec(ctx, `
		UPDATE daily_reports SET notes = $1, updated_at = NOW()
		WHERE id = $2::uuid AND company_id = $3::uuid
	`, req.Notes, id, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDailyReportNotFound
	}

	// Replace child rows
	if _, err := tx.Exec(ctx,
		`DELETE FROM daily_report_worker_hours WHERE daily_report_id = $1::uuid AND company_id = $2::uuid`,
		id, companyID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM daily_report_activities WHERE daily_report_id = $1::uuid AND company_id = $2::uuid`,
		id, companyID,
	); err != nil {
		return err
	}

	if err := r.insertWorkerHours(ctx, tx, companyID, id, req.WorkerHours); err != nil {
		return err
	}
	return r.insertActivities(ctx, tx, companyID, id, req.Activities)
}

// ── Approve / Reject ──────────────────────────────────────────────────────────

func (r *DailyReportRepository) SetStatus(ctx context.Context, tx pgx.Tx, id, companyID, status string, notes *string) error {
	var err error
	if notes != nil {
		_, err = tx.Exec(ctx, `
			UPDATE daily_reports SET status = $1, notes = $2, updated_at = NOW()
			WHERE id = $3::uuid AND company_id = $4::uuid
		`, status, notes, id, companyID)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE daily_reports SET status = $1, updated_at = NOW()
			WHERE id = $2::uuid AND company_id = $3::uuid
		`, status, id, companyID)
	}
	return err
}

// ── Form data ─────────────────────────────────────────────────────────────────

func (r *DailyReportRepository) GetActiveProjects(ctx context.Context, companyID string) ([]dto.FormDataProject, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, name FROM projects
		WHERE company_id = $1::uuid AND status = 'active'
		ORDER BY name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFormDataProjects(rows)
}

func (r *DailyReportRepository) GetWorkersForPoslovoda(ctx context.Context, poslovodaEmpID, companyID string) ([]dto.FormDataWorker, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, first_name, last_name
		FROM employees
		WHERE supervisor_id = $1::uuid
		  AND company_id = $2::uuid
		  AND role = 'radnik'
		  AND active = true
		ORDER BY last_name, first_name
	`, poslovodaEmpID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFormDataWorkers(rows)
}

func (r *DailyReportRepository) GetAllActiveWorkers(ctx context.Context, companyID string) ([]dto.FormDataWorker, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, first_name, last_name
		FROM employees
		WHERE company_id = $1::uuid AND role = 'radnik' AND active = true
		ORDER BY last_name, first_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFormDataWorkers(rows)
}

func (r *DailyReportRepository) GetMaterialsForProject(ctx context.Context, projectID, companyID string) ([]dto.FormDataMaterial, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, material_name, material_code, available_quantity, unit
		FROM project_materials
		WHERE project_id = $1::uuid AND company_id = $2::uuid AND active = true
		ORDER BY material_name
	`, projectID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.FormDataMaterial
	for rows.Next() {
		var m dto.FormDataMaterial
		if err := rows.Scan(&m.ID, &m.MaterialName, &m.MaterialCode, &m.AvailableQuantity, &m.Unit); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	if items == nil {
		items = []dto.FormDataMaterial{}
	}
	return items, rows.Err()
}

// ── Private helpers ───────────────────────────────────────────────────────────

func (r *DailyReportRepository) insertWorkerHours(ctx context.Context, tx pgx.Tx, companyID, reportID string, entries []dto.WorkerHoursInput) error {
	for _, wh := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO daily_report_worker_hours (company_id, daily_report_id, worker_id, hours_worked, notes)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
		`, companyID, reportID, wh.WorkerID, wh.HoursWorked, wh.Notes); err != nil {
			return fmt.Errorf("insert worker hours for worker %s: %w", wh.WorkerID, err)
		}
	}
	return nil
}

func (r *DailyReportRepository) insertActivities(ctx context.Context, tx pgx.Tx, companyID, reportID string, entries []dto.ActivityInput) error {
	for _, a := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO daily_report_activities
				(company_id, daily_report_id, project_material_id, custom_material_name, quantity, unit, activity_type, is_vtk, notes)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9)
		`, companyID, reportID, a.ProjectMaterialID, a.CustomMaterialName,
			a.Quantity, a.Unit, a.ActivityType, a.IsVTK, a.Notes,
		); err != nil {
			return fmt.Errorf("insert activity: %w", err)
		}
	}
	return nil
}

func scanFormDataProjects(rows pgx.Rows) ([]dto.FormDataProject, error) {
	defer rows.Close()
	var items []dto.FormDataProject
	for rows.Next() {
		var p dto.FormDataProject
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if items == nil {
		items = []dto.FormDataProject{}
	}
	return items, rows.Err()
}

func scanFormDataWorkers(rows pgx.Rows) ([]dto.FormDataWorker, error) {
	defer rows.Close()
	var items []dto.FormDataWorker
	for rows.Next() {
		var w dto.FormDataWorker
		if err := rows.Scan(&w.ID, &w.FirstName, &w.LastName); err != nil {
			return nil, err
		}
		w.FullName = w.FirstName + " " + w.LastName
		items = append(items, w)
	}
	if items == nil {
		items = []dto.FormDataWorker{}
	}
	return items, rows.Err()
}

// GetReportTimestamp is used for stale-time checks (future use).
func (r *DailyReportRepository) GetReportTimestamp(ctx context.Context, id, companyID string) (*time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx,
		`SELECT created_at FROM daily_reports WHERE id = $1::uuid AND company_id = $2::uuid`,
		id, companyID,
	).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDailyReportNotFound
	}
	return &t, err
}
