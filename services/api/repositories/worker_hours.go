package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

type WorkerHoursRepository struct {
	db *pgxpool.Pool
}

func NewWorkerHoursRepository(db *pgxpool.Pool) *WorkerHoursRepository {
	return &WorkerHoursRepository{db: db}
}

// ListCompanyActiveProjects returns all active (non-archived) projects in the company.
// Used by radnik who may work on any active project.
func (r *WorkerHoursRepository) ListCompanyActiveProjects(ctx context.Context, companyID string) ([]dto.WorkerProject, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, name
		FROM projects
		WHERE company_id = $1::uuid
		  AND status     = 'active'
		ORDER BY name
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.ListCompanyActiveProjects: %w", err)
	}
	defer rows.Close()

	var result []dto.WorkerProject
	for rows.Next() {
		var p dto.WorkerProject
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, fmt.Errorf("worker_hours.ListCompanyActiveProjects scan: %w", err)
		}
		result = append(result, p)
	}
	if result == nil {
		result = []dto.WorkerProject{}
	}
	return result, rows.Err()
}

// GetOtherProjectsHoursForDate returns the sum of already-submitted hours for a worker
// on a given date, excluding the project being submitted (so the caller can check
// whether adding/updating that project would exceed the 24-hour daily limit).
func (r *WorkerHoursRepository) GetOtherProjectsHoursForDate(ctx context.Context, companyID, workerEmpID, projectID, workDate string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(hours_worked), 0)
		FROM worker_daily_hours
		WHERE company_id = $1::uuid
		  AND worker_id  = $2::uuid
		  AND work_date  = $3::date
		  AND project_id != $4::uuid
	`, companyID, workerEmpID, workDate, projectID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("worker_hours.GetOtherProjectsHoursForDate: %w", err)
	}
	return total, nil
}

// Upsert inserts or updates the worker's hours for a given project/date.
// Returns the full entry after the upsert.
func (r *WorkerHoursRepository) Upsert(ctx context.Context, companyID, workerEmpID, projectID, workDate string, hoursWorked float64, notes *string, submittedByUserID string) (*dto.WorkerHoursEntry, error) {
	var e dto.WorkerHoursEntry
	err := r.db.QueryRow(ctx, `
		INSERT INTO worker_daily_hours
			(company_id, worker_id, project_id, work_date, hours_worked, notes, submitted_by)
		VALUES
			($1::uuid, $2::uuid, $3::uuid, $4::date, $5, $6, $7::uuid)
		ON CONFLICT (company_id, worker_id, project_id, work_date)
		DO UPDATE SET
			hours_worked = EXCLUDED.hours_worked,
			notes        = EXCLUDED.notes,
			submitted_by = EXCLUDED.submitted_by
		RETURNING
			id::text,
			worker_id::text,
			project_id::text,
			(SELECT name FROM projects WHERE id = worker_daily_hours.project_id) AS project_name,
			work_date::text,
			hours_worked,
			notes,
			created_at,
			updated_at
	`, companyID, workerEmpID, projectID, workDate, hoursWorked, notes, submittedByUserID).Scan(
		&e.ID, &e.WorkerID, &e.ProjectID, &e.ProjectName,
		&e.WorkDate, &e.HoursWorked, &e.Notes,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.Upsert: %w", err)
	}
	return &e, nil
}

// ListForWorkerDate returns all hour entries for a worker on a given date.
func (r *WorkerHoursRepository) ListForWorkerDate(ctx context.Context, companyID, workerEmpID, workDate string) ([]dto.WorkerHoursEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			wdh.id::text,
			wdh.worker_id::text,
			wdh.project_id::text,
			p.name AS project_name,
			wdh.work_date::text,
			wdh.hours_worked,
			wdh.notes,
			wdh.created_at,
			wdh.updated_at
		FROM worker_daily_hours wdh
		JOIN projects p ON p.id = wdh.project_id
		WHERE wdh.company_id = $1::uuid
		  AND wdh.worker_id  = $2::uuid
		  AND wdh.work_date  = $3::date
		ORDER BY p.name
	`, companyID, workerEmpID, workDate)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.ListForWorkerDate: %w", err)
	}
	defer rows.Close()

	var result []dto.WorkerHoursEntry
	for rows.Next() {
		var e dto.WorkerHoursEntry
		if err := rows.Scan(
			&e.ID, &e.WorkerID, &e.ProjectID, &e.ProjectName,
			&e.WorkDate, &e.HoursWorked, &e.Notes,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("worker_hours.ListForWorkerDate scan: %w", err)
		}
		result = append(result, e)
	}
	if result == nil {
		result = []dto.WorkerHoursEntry{}
	}
	return result, rows.Err()
}

// ListBeforeDate returns all hour entries for a worker with work_date strictly
// before beforeDate (YYYY-MM-DD), ordered newest date first then by project name.
func (r *WorkerHoursRepository) ListBeforeDate(ctx context.Context, companyID, workerEmpID, beforeDate string) ([]dto.WorkerHoursEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			wdh.id::text,
			wdh.worker_id::text,
			wdh.project_id::text,
			p.name AS project_name,
			wdh.work_date::text,
			wdh.hours_worked,
			wdh.notes,
			wdh.created_at,
			wdh.updated_at
		FROM worker_daily_hours wdh
		JOIN projects p ON p.id = wdh.project_id
		WHERE wdh.company_id = $1::uuid
		  AND wdh.worker_id  = $2::uuid
		  AND wdh.work_date  < $3::date
		ORDER BY wdh.work_date DESC, p.name
	`, companyID, workerEmpID, beforeDate)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.ListBeforeDate: %w", err)
	}
	defer rows.Close()

	var result []dto.WorkerHoursEntry
	for rows.Next() {
		var e dto.WorkerHoursEntry
		if err := rows.Scan(
			&e.ID, &e.WorkerID, &e.ProjectID, &e.ProjectName,
			&e.WorkDate, &e.HoursWorked, &e.Notes,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("worker_hours.ListBeforeDate scan: %w", err)
		}
		result = append(result, e)
	}
	if result == nil {
		result = []dto.WorkerHoursEntry{}
	}
	return result, rows.Err()
}

// GetProjectStatus returns project status or empty string if not found.
func (r *WorkerHoursRepository) GetProjectStatus(ctx context.Context, companyID, projectID string) (string, error) {
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
