package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

type ReportsRepository struct {
	db *pgxpool.Pool
}

func NewReportsRepository(db *pgxpool.Pool) *ReportsRepository {
	return &ReportsRepository{db: db}
}

// ── WHERE builder ─────────────────────────────────────────────────────────────

type reportWhere struct {
	conds []string
	args  []interface{}
	idx   int
}

func newReportWhere(companyIDArg interface{}, dnevnikTable bool) *reportWhere {
	// companyID is always arg $1, scoped via the child-table's company_id
	w := &reportWhere{idx: 2}
	if dnevnikTable {
		w.conds = []string{"drwh.company_id = $1::uuid"}
	} else {
		w.conds = []string{"dra.company_id = $1::uuid"}
	}
	w.args = []interface{}{companyIDArg}
	return w
}

func (w *reportWhere) add(cond string, arg interface{}) {
	w.conds = append(w.conds, fmt.Sprintf(cond, w.idx))
	w.args = append(w.args, arg)
	w.idx++
}

// addRaw adds a condition with an explicit placeholder already included (for multi-ref of same arg)
func (w *reportWhere) addRaw(cond string, arg interface{}) {
	placeholder := fmt.Sprintf("$%d", w.idx)
	w.conds = append(w.conds, strings.ReplaceAll(cond, "?", placeholder))
	w.args = append(w.args, arg)
	w.idx++
}

func (w *reportWhere) sql() string {
	return "WHERE " + strings.Join(w.conds, " AND ")
}

func (w *reportWhere) clone() *reportWhere {
	clone := &reportWhere{
		conds: make([]string, len(w.conds)),
		args:  make([]interface{}, len(w.args)),
		idx:   w.idx,
	}
	copy(clone.conds, w.conds)
	copy(clone.args, w.args)
	return clone
}

// ── Dnevnik helpers ───────────────────────────────────────────────────────────

func buildDnevnikWhere(companyID string, f dto.ReportFilter) *reportWhere {
	w := newReportWhere(companyID, true)

	if f.PoslovodaID != nil {
		w.add("dr.poslovoda_id = $%d::uuid", *f.PoslovodaID)
	}
	if f.ProjectID != nil {
		w.add("dr.project_id = $%d::uuid", *f.ProjectID)
	}
	if f.WorkerID != nil {
		w.add("drwh.worker_id = $%d::uuid", *f.WorkerID)
	}
	if f.Status != nil {
		w.add("dr.status = $%d", *f.Status)
	}
	if f.DateFrom != nil {
		w.add("dr.report_date >= $%d::date", *f.DateFrom)
	}
	if f.DateTo != nil {
		w.add("dr.report_date <= $%d::date", *f.DateTo)
	}
	if f.Search != nil && strings.TrimSpace(*f.Search) != "" {
		like := "%" + *f.Search + "%"
		// PostgreSQL allows referencing the same $N multiple times
		n := w.idx
		w.conds = append(w.conds, fmt.Sprintf(
			"(p.name ILIKE $%d OR (pe.first_name||' '||pe.last_name) ILIKE $%d OR (we.first_name||' '||we.last_name) ILIKE $%d)",
			n, n, n,
		))
		w.args = append(w.args, like)
		w.idx++
	}

	return w
}

// ── Knjiga helpers ────────────────────────────────────────────────────────────

func buildKnjigaWhere(companyID string, f dto.ReportFilter) *reportWhere {
	w := newReportWhere(companyID, false)

	if f.PoslovodaID != nil {
		w.add("dr.poslovoda_id = $%d::uuid", *f.PoslovodaID)
	}
	if f.ProjectID != nil {
		w.add("dr.project_id = $%d::uuid", *f.ProjectID)
	}
	if f.MaterialID != nil {
		w.add("dra.project_material_id = $%d::uuid", *f.MaterialID)
	}
	if f.ActivityType != nil {
		w.add("dra.activity_type = $%d", *f.ActivityType)
	}
	if f.IsVTK != nil {
		w.add("dra.is_vtk = $%d", *f.IsVTK)
	}
	if f.Status != nil {
		w.add("dr.status = $%d", *f.Status)
	}
	if f.DateFrom != nil {
		w.add("dr.report_date >= $%d::date", *f.DateFrom)
	}
	if f.DateTo != nil {
		w.add("dr.report_date <= $%d::date", *f.DateTo)
	}
	if f.Search != nil && strings.TrimSpace(*f.Search) != "" {
		like := "%" + *f.Search + "%"
		n := w.idx
		w.conds = append(w.conds, fmt.Sprintf(
			"(p.name ILIKE $%d OR (pe.first_name||' '||pe.last_name) ILIKE $%d OR "+
				"CASE WHEN dra.is_vtk THEN dra.custom_material_name ELSE pm.material_name END ILIKE $%d)",
			n, n, n,
		))
		w.args = append(w.args, like)
		w.idx++
	}

	return w
}

const dnevnikJoins = `
FROM daily_report_worker_hours drwh
JOIN daily_reports dr ON dr.id = drwh.daily_report_id
JOIN projects p ON p.id = dr.project_id
JOIN employees pe ON pe.id = dr.poslovoda_id
JOIN employees we ON we.id = drwh.worker_id`

const knjigaJoins = `
FROM daily_report_activities dra
JOIN daily_reports dr ON dr.id = dra.daily_report_id
JOIN projects p ON p.id = dr.project_id
JOIN employees pe ON pe.id = dr.poslovoda_id
LEFT JOIN project_materials pm ON pm.id = dra.project_material_id`

// ── Građevinski dnevnik ───────────────────────────────────────────────────────

func (r *ReportsRepository) CountDnevnik(ctx context.Context, companyID string, f dto.ReportFilter) (int, error) {
	w := buildDnevnikWhere(companyID, f)
	q := fmt.Sprintf(`SELECT COUNT(*) %s %s`, dnevnikJoins, w.sql())
	var total int
	err := r.db.QueryRow(ctx, q, w.args...).Scan(&total)
	return total, err
}

func (r *ReportsRepository) ListDnevnik(ctx context.Context, companyID string, f dto.ReportFilter, page, perPage int) ([]dto.GradevinskiDnevnikRow, error) {
	w := buildDnevnikWhere(companyID, f)
	offset := (page - 1) * perPage
	q := fmt.Sprintf(`
		SELECT
			dr.id::text,
			dr.report_date::text,
			dr.project_id::text,
			p.name,
			p.address,
			dr.poslovoda_id::text,
			pe.first_name||' '||pe.last_name,
			drwh.worker_id::text,
			we.first_name||' '||we.last_name,
			drwh.hours_worked,
			drwh.notes,
			dr.status
		%s %s
		ORDER BY dr.report_date DESC, p.name, we.last_name, we.first_name
		LIMIT $%d OFFSET $%d
	`, dnevnikJoins, w.sql(), w.idx, w.idx+1)

	args := append(w.args, perPage, offset)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.GradevinskiDnevnikRow
	for rows.Next() {
		var row dto.GradevinskiDnevnikRow
		if err := rows.Scan(
			&row.ReportID, &row.ReportDate,
			&row.ProjectID, &row.ProjectName, &row.ProjectAddress,
			&row.PoslovodaID, &row.PoslovodaName,
			&row.WorkerID, &row.WorkerName,
			&row.HoursWorked, &row.Notes, &row.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	if items == nil {
		items = []dto.GradevinskiDnevnikRow{}
	}
	return items, rows.Err()
}

func (r *ReportsRepository) SummaryDnevnik(ctx context.Context, companyID string, f dto.ReportFilter) (dto.GradevinskiDnevnikSummary, error) {
	w := buildDnevnikWhere(companyID, f)
	q := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(drwh.hours_worked), 0),
			COUNT(DISTINCT drwh.worker_id),
			COUNT(DISTINCT dr.project_id)
		%s %s
	`, dnevnikJoins, w.sql())

	var s dto.GradevinskiDnevnikSummary
	err := r.db.QueryRow(ctx, q, w.args...).Scan(&s.TotalHours, &s.WorkersCount, &s.ProjectsCount)
	return s, err
}

func (r *ReportsRepository) ExportDnevnik(ctx context.Context, companyID string, f dto.ReportFilter, limit int) ([]dto.GradevinskiDnevnikRow, error) {
	w := buildDnevnikWhere(companyID, f)
	q := fmt.Sprintf(`
		SELECT
			dr.id::text,
			dr.report_date::text,
			dr.project_id::text,
			p.name,
			p.address,
			dr.poslovoda_id::text,
			pe.first_name||' '||pe.last_name,
			drwh.worker_id::text,
			we.first_name||' '||we.last_name,
			drwh.hours_worked,
			drwh.notes,
			dr.status
		%s %s
		ORDER BY dr.report_date DESC, p.name, we.last_name, we.first_name
		LIMIT $%d
	`, dnevnikJoins, w.sql(), w.idx)

	args := append(w.args, limit+1) // fetch limit+1 to detect overflow
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.GradevinskiDnevnikRow
	for rows.Next() {
		var row dto.GradevinskiDnevnikRow
		if err := rows.Scan(
			&row.ReportID, &row.ReportDate,
			&row.ProjectID, &row.ProjectName, &row.ProjectAddress,
			&row.PoslovodaID, &row.PoslovodaName,
			&row.WorkerID, &row.WorkerName,
			&row.HoursWorked, &row.Notes, &row.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

// ── Građevinska knjiga ────────────────────────────────────────────────────────

func (r *ReportsRepository) CountKnjiga(ctx context.Context, companyID string, f dto.ReportFilter) (int, error) {
	w := buildKnjigaWhere(companyID, f)
	q := fmt.Sprintf(`SELECT COUNT(*) %s %s`, knjigaJoins, w.sql())
	var total int
	err := r.db.QueryRow(ctx, q, w.args...).Scan(&total)
	return total, err
}

func (r *ReportsRepository) ListKnjiga(ctx context.Context, companyID string, f dto.ReportFilter, page, perPage int) ([]dto.GradevinskaKnjigaRow, error) {
	w := buildKnjigaWhere(companyID, f)
	offset := (page - 1) * perPage
	q := fmt.Sprintf(`
		SELECT
			dr.id::text,
			dr.report_date::text,
			dr.project_id::text,
			p.name,
			p.address,
			dr.poslovoda_id::text,
			pe.first_name||' '||pe.last_name,
			dra.id::text,
			dra.project_material_id::text,
			COALESCE(CASE WHEN dra.is_vtk THEN dra.custom_material_name ELSE pm.material_name END, '—'),
			pm.material_code,
			dra.custom_material_name,
			dra.quantity,
			dra.unit,
			dra.activity_type,
			dra.is_vtk,
			dra.notes,
			dr.status
		%s %s
		ORDER BY dr.report_date DESC, p.name
		LIMIT $%d OFFSET $%d
	`, knjigaJoins, w.sql(), w.idx, w.idx+1)

	args := append(w.args, perPage, offset)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.GradevinskaKnjigaRow
	for rows.Next() {
		var row dto.GradevinskaKnjigaRow
		if err := rows.Scan(
			&row.ReportID, &row.ReportDate,
			&row.ProjectID, &row.ProjectName, &row.ProjectAddress,
			&row.PoslovodaID, &row.PoslovodaName,
			&row.ActivityID, &row.MaterialID, &row.MaterialName,
			&row.MaterialCode, &row.CustomMaterialName,
			&row.Quantity, &row.Unit,
			&row.ActivityType, &row.IsVTK, &row.Notes, &row.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	if items == nil {
		items = []dto.GradevinskaKnjigaRow{}
	}
	return items, rows.Err()
}

func (r *ReportsRepository) SummaryKnjiga(ctx context.Context, companyID string, f dto.ReportFilter) (dto.GradevinskaKnjigaSummary, error) {
	w := buildKnjigaWhere(companyID, f)
	q := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(SUM(dra.quantity), 0),
			COUNT(CASE WHEN dra.is_vtk THEN 1 END),
			COUNT(DISTINCT dr.project_id)
		%s %s
	`, knjigaJoins, w.sql())

	var s dto.GradevinskaKnjigaSummary
	err := r.db.QueryRow(ctx, q, w.args...).Scan(&s.ActivitiesCount, &s.TotalQuantity, &s.VTKCount, &s.ProjectsCount)
	return s, err
}

func (r *ReportsRepository) ExportKnjiga(ctx context.Context, companyID string, f dto.ReportFilter, limit int) ([]dto.GradevinskaKnjigaRow, error) {
	w := buildKnjigaWhere(companyID, f)
	q := fmt.Sprintf(`
		SELECT
			dr.id::text,
			dr.report_date::text,
			dr.project_id::text,
			p.name,
			p.address,
			dr.poslovoda_id::text,
			pe.first_name||' '||pe.last_name,
			dra.id::text,
			dra.project_material_id::text,
			COALESCE(CASE WHEN dra.is_vtk THEN dra.custom_material_name ELSE pm.material_name END, '—'),
			pm.material_code,
			dra.custom_material_name,
			dra.quantity,
			dra.unit,
			dra.activity_type,
			dra.is_vtk,
			dra.notes,
			dr.status
		%s %s
		ORDER BY dr.report_date DESC, p.name
		LIMIT $%d
	`, knjigaJoins, w.sql(), w.idx)

	args := append(w.args, limit+1)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.GradevinskaKnjigaRow
	for rows.Next() {
		var row dto.GradevinskaKnjigaRow
		if err := rows.Scan(
			&row.ReportID, &row.ReportDate,
			&row.ProjectID, &row.ProjectName, &row.ProjectAddress,
			&row.PoslovodaID, &row.PoslovodaName,
			&row.ActivityID, &row.MaterialID, &row.MaterialName,
			&row.MaterialCode, &row.CustomMaterialName,
			&row.Quantity, &row.Unit,
			&row.ActivityType, &row.IsVTK, &row.Notes, &row.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

// ── Filter options ────────────────────────────────────────────────────────────

func (r *ReportsRepository) FilterOptionsForAll(ctx context.Context, companyID string) (dto.ReportFilterOptions, error) {
	var opts dto.ReportFilterOptions
	var err error

	// Projects
	rows, err := r.db.Query(ctx, `
		SELECT id::text, name, status FROM projects
		WHERE company_id = $1::uuid
		ORDER BY name
	`, companyID)
	if err != nil {
		return opts, err
	}
	defer rows.Close()
	for rows.Next() {
		var p dto.FilterOptionProject
		if err := rows.Scan(&p.ID, &p.Name, &p.Status); err != nil {
			return opts, err
		}
		opts.Projects = append(opts.Projects, p)
	}
	if err := rows.Err(); err != nil {
		return opts, err
	}

	// Poslovode
	rows2, err := r.db.Query(ctx, `
		SELECT id::text, first_name||' '||last_name FROM employees
		WHERE company_id = $1::uuid AND role = 'poslovoda' AND active = true
		ORDER BY last_name, first_name
	`, companyID)
	if err != nil {
		return opts, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var p dto.FilterOptionPerson
		if err := rows2.Scan(&p.ID, &p.FullName); err != nil {
			return opts, err
		}
		opts.Poslovode = append(opts.Poslovode, p)
	}
	if err := rows2.Err(); err != nil {
		return opts, err
	}

	// Workers
	rows3, err := r.db.Query(ctx, `
		SELECT id::text, first_name||' '||last_name, supervisor_id::text FROM employees
		WHERE company_id = $1::uuid AND role = 'radnik' AND active = true
		ORDER BY last_name, first_name
	`, companyID)
	if err != nil {
		return opts, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var p dto.FilterOptionPerson
		if err := rows3.Scan(&p.ID, &p.FullName, &p.SupervisorID); err != nil {
			return opts, err
		}
		opts.Workers = append(opts.Workers, p)
	}
	if err := rows3.Err(); err != nil {
		return opts, err
	}

	// Materials
	rows4, err := r.db.Query(ctx, `
		SELECT id::text, material_name, project_id::text FROM project_materials
		WHERE company_id = $1::uuid AND active = true
		ORDER BY material_name
	`, companyID)
	if err != nil {
		return opts, err
	}
	defer rows4.Close()
	for rows4.Next() {
		var m dto.FilterOptionMaterial
		if err := rows4.Scan(&m.ID, &m.MaterialName, &m.ProjectID); err != nil {
			return opts, err
		}
		opts.Materials = append(opts.Materials, m)
	}
	if err := rows4.Err(); err != nil {
		return opts, err
	}

	opts.ActivityTypes = []string{"montaza", "demontaza", "other"}
	opts.Statuses = []string{"draft", "submitted", "approved", "rejected"}
	return ensureSlices(opts), nil
}

func (r *ReportsRepository) FilterOptionsForPoslovoda(ctx context.Context, companyID, empID string) (dto.ReportFilterOptions, error) {
	var opts dto.ReportFilterOptions

	// Only their own assigned active projects
	rows, err := r.db.Query(ctx, `
		SELECT p.id::text, p.name, p.status FROM projects p
		JOIN project_assignments pa ON pa.project_id = p.id
		WHERE pa.employee_id = $1::uuid AND pa.role_on_project = 'poslovoda'
		  AND pa.active = true AND p.company_id = $2::uuid
		ORDER BY p.name
	`, empID, companyID)
	if err != nil {
		return opts, err
	}
	defer rows.Close()
	for rows.Next() {
		var p dto.FilterOptionProject
		if err := rows.Scan(&p.ID, &p.Name, &p.Status); err != nil {
			return opts, err
		}
		opts.Projects = append(opts.Projects, p)
	}
	if err := rows.Err(); err != nil {
		return opts, err
	}

	// Only themselves as poslovoda
	var self dto.FilterOptionPerson
	err = r.db.QueryRow(ctx, `
		SELECT id::text, first_name||' '||last_name FROM employees
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, empID, companyID).Scan(&self.ID, &self.FullName)
	if err == nil {
		opts.Poslovode = append(opts.Poslovode, self)
	}

	// Workers under them
	rows2, err := r.db.Query(ctx, `
		SELECT id::text, first_name||' '||last_name, supervisor_id::text FROM employees
		WHERE supervisor_id = $1::uuid AND company_id = $2::uuid AND role = 'radnik' AND active = true
		ORDER BY last_name, first_name
	`, empID, companyID)
	if err != nil {
		return opts, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var p dto.FilterOptionPerson
		if err := rows2.Scan(&p.ID, &p.FullName, &p.SupervisorID); err != nil {
			return opts, err
		}
		opts.Workers = append(opts.Workers, p)
	}
	if err := rows2.Err(); err != nil {
		return opts, err
	}

	// Materials from their assigned projects
	rows3, err := r.db.Query(ctx, `
		SELECT pm.id::text, pm.material_name, pm.project_id::text FROM project_materials pm
		JOIN project_assignments pa ON pa.project_id = pm.project_id
		WHERE pa.employee_id = $1::uuid AND pa.role_on_project = 'poslovoda'
		  AND pa.active = true AND pm.company_id = $2::uuid AND pm.active = true
		ORDER BY pm.material_name
	`, empID, companyID)
	if err != nil {
		return opts, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var m dto.FilterOptionMaterial
		if err := rows3.Scan(&m.ID, &m.MaterialName, &m.ProjectID); err != nil {
			return opts, err
		}
		opts.Materials = append(opts.Materials, m)
	}
	if err := rows3.Err(); err != nil {
		return opts, err
	}

	opts.ActivityTypes = []string{"montaza", "demontaza", "other"}
	opts.Statuses = []string{"draft", "submitted", "approved", "rejected"}
	return ensureSlices(opts), nil
}

func ensureSlices(opts dto.ReportFilterOptions) dto.ReportFilterOptions {
	if opts.Projects == nil {
		opts.Projects = []dto.FilterOptionProject{}
	}
	if opts.Poslovode == nil {
		opts.Poslovode = []dto.FilterOptionPerson{}
	}
	if opts.Workers == nil {
		opts.Workers = []dto.FilterOptionPerson{}
	}
	if opts.Materials == nil {
		opts.Materials = []dto.FilterOptionMaterial{}
	}
	return opts
}
