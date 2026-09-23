package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

var ErrManagementHoursNotFound = errors.New("management hours entry not found")

type ManagementHoursRepository struct {
	db *pgxpool.Pool
}

func NewManagementHoursRepository(db *pgxpool.Pool) *ManagementHoursRepository {
	return &ManagementHoursRepository{db: db}
}

// Upsert inserts or updates a management employee's hours for a given date.
// project_id is optional (nil = no project). One entry per employee per day.
func (r *ManagementHoursRepository) Upsert(
	ctx context.Context,
	companyID, employeeID, submittedByUserID, workDate string,
	hoursWorked float64,
	notes, projectID *string,
) (*dto.ManagementHoursEntry, error) {
	var e dto.ManagementHoursEntry
	err := r.db.QueryRow(ctx, `
		INSERT INTO worker_daily_hours
			(company_id, worker_id, project_id, work_date, hours_worked, notes, submitted_by)
		VALUES
			($1::uuid, $2::uuid, $3::uuid, $4::date, $5, $6, $7::uuid)
		ON CONFLICT (company_id, worker_id, work_date) WHERE project_id IS NULL
		DO UPDATE SET
			hours_worked = EXCLUDED.hours_worked,
			notes        = EXCLUDED.notes,
			submitted_by = EXCLUDED.submitted_by
		RETURNING
			id::text,
			worker_id::text,
			project_id::text,
			(SELECT name FROM projects WHERE id = worker_daily_hours.project_id),
			work_date::text,
			hours_worked,
			notes,
			created_at,
			updated_at
	`, companyID, employeeID, projectID, workDate, hoursWorked, notes, submittedByUserID,
	).Scan(
		&e.ID, &e.EmployeeID, &e.ProjectID, &e.ProjectName,
		&e.WorkDate, &e.HoursWorked, &e.Notes,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("management_hours.Upsert: %w", err)
	}
	e.EmployeeRole = "" // caller fills in from JWT
	return &e, nil
}

// GetByID returns one management entry (project_id IS NULL) scoped to the company.
// Returns ErrManagementHoursNotFound when absent.
func (r *ManagementHoursRepository) GetByID(ctx context.Context, companyID, entryID string) (*dto.ManagementHoursEntry, error) {
	var e dto.ManagementHoursEntry
	err := r.db.QueryRow(ctx, `
		SELECT
			wdh.id::text,
			wdh.worker_id::text,
			CONCAT(emp.first_name, ' ', emp.last_name),
			emp.role,
			wdh.project_id::text,
			(SELECT name FROM projects p WHERE p.id = wdh.project_id),
			wdh.work_date::text,
			wdh.hours_worked,
			wdh.notes,
			wdh.created_at,
			wdh.updated_at
		FROM worker_daily_hours wdh
		JOIN employees emp ON emp.id = wdh.worker_id
		WHERE wdh.id         = $1::uuid
		  AND wdh.company_id = $2::uuid
		  AND wdh.project_id IS NULL
	`, entryID, companyID).Scan(
		&e.ID, &e.EmployeeID, &e.EmployeeName, &e.EmployeeRole,
		&e.ProjectID, &e.ProjectName,
		&e.WorkDate, &e.HoursWorked, &e.Notes,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManagementHoursNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("management_hours.GetByID: %w", err)
	}
	return &e, nil
}

// ListForEmployee returns all management entries (project_id IS NULL) for one employee.
func (r *ManagementHoursRepository) ListForEmployee(ctx context.Context, companyID, employeeID string) ([]dto.ManagementHoursEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			wdh.id::text,
			wdh.worker_id::text,
			CONCAT(emp.first_name, ' ', emp.last_name),
			emp.role,
			wdh.project_id::text,
			(SELECT name FROM projects p WHERE p.id = wdh.project_id),
			wdh.work_date::text,
			wdh.hours_worked,
			wdh.notes,
			wdh.created_at,
			wdh.updated_at
		FROM worker_daily_hours wdh
		JOIN employees emp ON emp.id = wdh.worker_id
		WHERE wdh.company_id = $1::uuid
		  AND wdh.worker_id  = $2::uuid
		  AND wdh.project_id IS NULL
		ORDER BY wdh.work_date DESC
	`, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("management_hours.ListForEmployee: %w", err)
	}
	defer rows.Close()
	return scanManagementRows(rows)
}

// ListForDirector returns management entries visible to a director:
//   - director's own entries
//   - all inzenjer entries in the company
//   - all administracija entries in the company
//
// Other directors' entries are explicitly excluded.
func (r *ManagementHoursRepository) ListForDirector(ctx context.Context, companyID, directorEmployeeID string) ([]dto.ManagementHoursEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			wdh.id::text,
			wdh.worker_id::text,
			CONCAT(emp.first_name, ' ', emp.last_name),
			emp.role,
			wdh.project_id::text,
			(SELECT name FROM projects p WHERE p.id = wdh.project_id),
			wdh.work_date::text,
			wdh.hours_worked,
			wdh.notes,
			wdh.created_at,
			wdh.updated_at
		FROM worker_daily_hours wdh
		JOIN employees emp ON emp.id = wdh.worker_id AND emp.company_id = $1::uuid
		WHERE wdh.company_id = $1::uuid
		  AND wdh.project_id IS NULL
		  AND (
		        wdh.worker_id = $2::uuid
		        OR emp.role IN ('inzenjer', 'administracija')
		      )
		ORDER BY wdh.work_date DESC, emp.last_name, emp.first_name
	`, companyID, directorEmployeeID)
	if err != nil {
		return nil, fmt.Errorf("management_hours.ListForDirector: %w", err)
	}
	defer rows.Close()
	return scanManagementRows(rows)
}

// Update changes hours_worked and notes on an entry the employee owns.
// Returns ErrManagementHoursNotFound when the entry does not exist or is not
// owned by ownerEmployeeID within the company.
func (r *ManagementHoursRepository) Update(
	ctx context.Context,
	companyID, ownerEmployeeID, entryID string,
	hoursWorked float64,
	notes *string,
) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE worker_daily_hours
		SET hours_worked = $1,
		    notes        = $2
		WHERE id         = $3::uuid
		  AND company_id = $4::uuid
		  AND worker_id  = $5::uuid
		  AND project_id IS NULL
	`, hoursWorked, notes, entryID, companyID, ownerEmployeeID)
	if err != nil {
		return fmt.Errorf("management_hours.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrManagementHoursNotFound
	}
	return nil
}

// Delete removes an entry that the employee owns.
// Returns ErrManagementHoursNotFound when no matching row exists.
func (r *ManagementHoursRepository) Delete(ctx context.Context, companyID, ownerEmployeeID, entryID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM worker_daily_hours
		WHERE id         = $1::uuid
		  AND company_id = $2::uuid
		  AND worker_id  = $3::uuid
		  AND project_id IS NULL
	`, entryID, companyID, ownerEmployeeID)
	if err != nil {
		return fmt.Errorf("management_hours.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrManagementHoursNotFound
	}
	return nil
}

func scanManagementRows(rows pgx.Rows) ([]dto.ManagementHoursEntry, error) {
	var result []dto.ManagementHoursEntry
	for rows.Next() {
		var e dto.ManagementHoursEntry
		if err := rows.Scan(
			&e.ID, &e.EmployeeID, &e.EmployeeName, &e.EmployeeRole,
			&e.ProjectID, &e.ProjectName,
			&e.WorkDate, &e.HoursWorked, &e.Notes,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("management_hours scan: %w", err)
		}
		result = append(result, e)
	}
	if result == nil {
		result = []dto.ManagementHoursEntry{}
	}
	return result, rows.Err()
}
