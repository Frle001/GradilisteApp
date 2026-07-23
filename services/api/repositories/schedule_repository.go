package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Row types ─────────────────────────────────────────────────────────────────

type ShiftRow struct {
	ID          string
	CompanyID   string
	ProjectID   string
	ProjectName string
	ShiftDate   string  // "YYYY-MM-DD"
	StartTime   *string // "HH:MM" or nil
	EndTime     *string // "HH:MM" or nil
	Notes       *string
	Status      string
	CancelledAt *time.Time
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ShiftAssignmentRow struct {
	ID                string
	ShiftID           string
	EmployeeID        string
	EmployeeName      string
	AssignedBy        string
	OverlapOverridden bool
	CreatedAt         time.Time
}

// ShiftConflictRow is returned by FindDailyConflicts: employees who already have
// an active shift on the given date (excluding the current shift).
type ShiftConflictRow struct {
	EmployeeID   string
	EmployeeName string
	ShiftID      string
	ProjectName  string
}

// EmployeeForDateRow is returned by EmployeesForDate.
type EmployeeForDateRow struct {
	ID          string
	Name        string
	Role        string
	Assigned    bool
	ShiftID     *string
	ProjectName *string
}

// ── Repository ────────────────────────────────────────────────────────────────

type ScheduleRepository struct {
	db *pgxpool.Pool
}

func NewScheduleRepository(db *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) ProjectBelongsToCompany(ctx context.Context, companyID, projectID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM projects
			WHERE id = $1::uuid
  				AND company_id = $2::uuid
		)
	`, projectID, companyID).Scan(&exists)
	return exists, err
}

// scanShiftRow reuses a single scan call for both QueryRow and Rows.Next loops.
func scanShiftRow(s *ShiftRow, scan func(...any) error) error {
	return scan(
		&s.ID, &s.CompanyID, &s.ProjectID, &s.ProjectName,
		&s.ShiftDate, &s.StartTime, &s.EndTime,
		&s.Notes, &s.Status, &s.CancelledAt,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
}

const shiftSelectCols = `
	ps.id::text,
	ps.company_id::text,
	ps.project_id::text,
	p.name  AS project_name,
	ps.shift_date::text,
	to_char(ps.start_time, 'HH24:MI') AS start_time,
	to_char(ps.end_time,   'HH24:MI') AS end_time,
	ps.notes,
	ps.status,
	ps.cancelled_at,
	ps.created_by::text,
	ps.created_at,
	ps.updated_at
`

func (r *ScheduleRepository) CreateShift(
	ctx context.Context, tx pgx.Tx,
	companyID, projectID, shiftDate string,
	startTime, endTime *string,
	notes *string, createdBy string,
) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO project_shifts
			(company_id, project_id, shift_date, start_time, end_time, notes, created_by)
		VALUES ($1::uuid, $2::uuid, $3::date, $4::time, $5::time, $6, $7::uuid)
		RETURNING id::text
	`, companyID, projectID, shiftDate, startTime, endTime, notes, createdBy).Scan(&id)
	return id, err
}

func (r *ScheduleRepository) GetShift(ctx context.Context, companyID, shiftID string) (*ShiftRow, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+shiftSelectCols+`
		FROM project_shifts ps
		JOIN projects p ON p.id = ps.project_id
		WHERE ps.id = $1::uuid AND ps.company_id = $2::uuid
	`, shiftID, companyID)
	var s ShiftRow
	if err := scanShiftRow(&s, row.Scan); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *ScheduleRepository) GetShiftForUpdate(ctx context.Context, tx pgx.Tx, companyID, shiftID string) (*ShiftRow, error) {
	row := tx.QueryRow(ctx, `
		SELECT `+shiftSelectCols+`
		FROM project_shifts ps
		JOIN projects p ON p.id = ps.project_id
		WHERE ps.id = $1::uuid AND ps.company_id = $2::uuid
		FOR UPDATE OF ps
	`, shiftID, companyID)
	var s ShiftRow
	if err := scanShiftRow(&s, row.Scan); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *ScheduleRepository) ListShifts(
	ctx context.Context,
	companyID, dateFrom, dateTo string,
	projectID *string,
) ([]ShiftRow, error) {
	query := `
		SELECT ` + shiftSelectCols + `
		FROM project_shifts ps
		JOIN projects p ON p.id = ps.project_id
		WHERE ps.company_id = $1::uuid
		  AND ps.shift_date BETWEEN $2::date AND $3::date
		  AND ($4::uuid IS NULL OR ps.project_id = $4::uuid)
		ORDER BY ps.shift_date, ps.created_at
	`
	rows, err := r.db.Query(ctx, query, companyID, dateFrom, dateTo, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ShiftRow
	for rows.Next() {
		var s ShiftRow
		if err := scanShiftRow(&s, rows.Scan); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *ScheduleRepository) UpdateShift(
	ctx context.Context, tx pgx.Tx,
	shiftID, shiftDate string,
	startTime, endTime *string,
	notes *string,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE project_shifts
		SET shift_date = $2::date,
		    start_time = $3::time,
		    end_time   = $4::time,
		    notes      = $5
		WHERE id = $1::uuid
	`, shiftID, shiftDate, startTime, endTime, notes)
	return err
}

func (r *ScheduleRepository) CancelShift(
	ctx context.Context, tx pgx.Tx,
	companyID, shiftID, cancelledBy string,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE project_shifts
		SET status       = 'cancelled',
		    cancelled_at = NOW(),
		    cancelled_by = $3::uuid
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, shiftID, companyID, cancelledBy)
	return err
}

func (r *ScheduleRepository) ListAssignments(
	ctx context.Context, companyID, shiftID string,
) ([]ShiftAssignmentRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			psa.id::text,
			psa.shift_id::text,
			psa.employee_id::text,
			CONCAT_WS(' ', e.first_name, e.last_name) AS employee_name,
			psa.assigned_by::text,
			psa.overlap_overridden,
			psa.created_at
		FROM project_shift_assignments psa
		JOIN employees e ON e.id = psa.employee_id
		WHERE psa.shift_id = $1::uuid AND psa.company_id = $2::uuid
		ORDER BY e.last_name, e.first_name
	`, shiftID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ShiftAssignmentRow
	for rows.Next() {
		var a ShiftAssignmentRow
		if err := rows.Scan(&a.ID, &a.ShiftID, &a.EmployeeID, &a.EmployeeName,
			&a.AssignedBy, &a.OverlapOverridden, &a.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// FindDailyConflicts returns employees from employeeIDs who already have an
// active shift on shiftDate in this company, excluding excludeShiftID (use
// "00000000-0000-0000-0000-000000000000" when there is no shift to exclude).
func (r *ScheduleRepository) FindDailyConflicts(
	ctx context.Context,
	companyID, shiftDate, excludeShiftID string,
	employeeIDs []string,
) ([]ShiftConflictRow, error) {
	if len(employeeIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			psa.employee_id::text,
			CONCAT_WS(' ', e.first_name, e.last_name) AS employee_name,
			ps.id::text AS shift_id,
			p.name      AS project_name
		FROM project_shift_assignments psa
		JOIN project_shifts ps ON ps.id = psa.shift_id
		JOIN employees      e  ON e.id  = psa.employee_id
		JOIN projects       p  ON p.id  = ps.project_id
		WHERE psa.company_id    = $1::uuid
		  AND ps.shift_date     = $2::date
		  AND ps.status         = 'active'
		  AND ps.id            != $3::uuid
		  AND psa.employee_id IN (SELECT unnest($4::text[])::uuid)
		ORDER BY e.last_name, e.first_name
	`, companyID, shiftDate, excludeShiftID, employeeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ShiftConflictRow
	for rows.Next() {
		var o ShiftConflictRow
		if err := rows.Scan(&o.EmployeeID, &o.EmployeeName, &o.ShiftID, &o.ProjectName); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

func (r *ScheduleRepository) DeleteAssignments(ctx context.Context, tx pgx.Tx, shiftID string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM project_shift_assignments WHERE shift_id = $1::uuid
	`, shiftID)
	return err
}

func (r *ScheduleRepository) CreateAssignment(
	ctx context.Context, tx pgx.Tx,
	companyID, shiftID, employeeID, assignedBy string,
	overlapOverridden bool, overriddenBy *string,
) error {
	if !overlapOverridden {
		overriddenBy = nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO project_shift_assignments
			(company_id, shift_id, employee_id, assigned_by,
			 overlap_overridden, overlap_overridden_by, overlap_overridden_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid,
		        $5,
		        $6::uuid,
		        CASE WHEN $5 THEN NOW() ELSE NULL END)
	`, companyID, shiftID, employeeID, assignedBy, overlapOverridden, overriddenBy)
	return err
}

// EmployeesForDate returns all active company employees with their assignment
// status on shiftDate. excludeShiftID is the shift being edited (those
// assignments are not counted as conflicts). Pass the nil-UUID when not editing.
func (r *ScheduleRepository) EmployeesForDate(
	ctx context.Context,
	companyID, shiftDate, excludeShiftID string,
) ([]EmployeeForDateRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			e.id::text,
			CONCAT_WS(' ', e.first_name, e.last_name) AS name,
			e.role,
			(asgn.shift_id IS NOT NULL)               AS assigned,
			asgn.shift_id::text,
			p.name AS project_name
		FROM employees e
		LEFT JOIN LATERAL (
			SELECT psa.shift_id
			FROM project_shift_assignments psa
			JOIN project_shifts ps2 ON ps2.id = psa.shift_id
			WHERE psa.employee_id   = e.id
			  AND psa.company_id    = $1::uuid
			  AND ps2.shift_date    = $2::date
			  AND ps2.status        = 'active'
			  AND ps2.id           != $3::uuid
			LIMIT 1
		) asgn ON true
		LEFT JOIN project_shifts ps ON ps.id = asgn.shift_id
		LEFT JOIN projects        p  ON p.id  = ps.project_id
		WHERE e.company_id = $1::uuid
		  AND e.active     = true
		ORDER BY e.last_name, e.first_name
	`, companyID, shiftDate, excludeShiftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []EmployeeForDateRow
	for rows.Next() {
		var row EmployeeForDateRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Role, &row.Assigned,
			&row.ShiftID, &row.ProjectName); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
