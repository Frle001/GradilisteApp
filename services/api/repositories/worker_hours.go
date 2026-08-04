package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

var ErrWorkerHoursNotFound = errors.New("worker hours entry not found")

// ErrSubmissionConflict is returned by Upsert when a client_submission_id has
// already been stored with a different (project, date, hours, notes, work_description) payload.
var ErrSubmissionConflict = errors.New("worker_hours: client_submission_id already used with different payload")

type WorkerHoursRepository struct {
	db *pgxpool.Pool
}

func NewWorkerHoursRepository(db *pgxpool.Pool) *WorkerHoursRepository {
	return &WorkerHoursRepository{db: db}
}

// ListCompanyActiveProjects returns all active (non-archived) projects in the company.
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
// on a given date, excluding the project being submitted.
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
//
// When clientSubmissionID is non-nil, the function first checks for an existing
// row that was committed under that ID:
//   - Same payload → returns the existing row idempotently (retry safe).
//   - Different payload → returns ErrSubmissionConflict (409 territory).
//   - No row yet → proceeds with the upsert and stores the ID on the row.
//
// Rows created without a submission ID (legacy / no-network-guard path) are not
// touched by the ID index and behave as before.
func (r *WorkerHoursRepository) Upsert(
	ctx context.Context,
	companyID, workerEmpID, projectID, workDate string,
	hoursWorked float64,
	notes, workDescription *string,
	submittedByUserID string,
	clientSubmissionID *string,
) (*dto.WorkerHoursEntry, error) {
	// ── Idempotency check ────────────────────────────────────────────────────
	if clientSubmissionID != nil {
		var e dto.WorkerHoursEntry
		var payloadMatches bool
		err := r.db.QueryRow(ctx, `
			SELECT
				id::text,
				worker_id::text,
				project_id::text,
				(SELECT name FROM projects WHERE id = wdh.project_id),
				work_date::text,
				hours_worked,
				notes,
				work_description,
				created_at,
				updated_at,
				(project_id             = $3::uuid
				 AND work_date          = $4::date
				 AND hours_worked       = $5
				 AND notes              IS NOT DISTINCT FROM $6
				 AND work_description   IS NOT DISTINCT FROM $7) AS payload_matches
			FROM worker_daily_hours wdh
			WHERE company_id           = $1::uuid
			  AND client_submission_id  = $2::uuid
		`, companyID, *clientSubmissionID, projectID, workDate, hoursWorked, notes, workDescription,
		).Scan(
			&e.ID, &e.WorkerID, &e.ProjectID, &e.ProjectName,
			&e.WorkDate, &e.HoursWorked, &e.Notes, &e.WorkDescription,
			&e.CreatedAt, &e.UpdatedAt,
			&payloadMatches,
		)
		if err == nil {
			if payloadMatches {
				return &e, nil
			}
			return nil, ErrSubmissionConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("worker_hours.Upsert idempotency check: %w", err)
		}
		// No row for this submission ID — fall through to the upsert.
	}

	// ── Upsert ──────────────────────────────────────────────────────────────
	var e dto.WorkerHoursEntry

	if clientSubmissionID != nil {
		err := r.db.QueryRow(ctx, `
			INSERT INTO worker_daily_hours
				(company_id, worker_id, project_id, work_date, hours_worked, notes, work_description, submitted_by, client_submission_id)
			VALUES
				($1::uuid, $2::uuid, $3::uuid, $4::date, $5, $6, $7, $8::uuid, $9::uuid)
			ON CONFLICT (company_id, worker_id, project_id, work_date)
			DO UPDATE SET
				hours_worked         = EXCLUDED.hours_worked,
				notes                = EXCLUDED.notes,
				work_description     = EXCLUDED.work_description,
				submitted_by         = EXCLUDED.submitted_by,
				client_submission_id = EXCLUDED.client_submission_id
			RETURNING
				id::text,
				worker_id::text,
				project_id::text,
				(SELECT name FROM projects WHERE id = worker_daily_hours.project_id),
				work_date::text,
				hours_worked,
				notes,
				work_description,
				created_at,
				updated_at
		`, companyID, workerEmpID, projectID, workDate, hoursWorked, notes, workDescription, submittedByUserID, *clientSubmissionID,
		).Scan(
			&e.ID, &e.WorkerID, &e.ProjectID, &e.ProjectName,
			&e.WorkDate, &e.HoursWorked, &e.Notes, &e.WorkDescription,
			&e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("worker_hours.Upsert: %w", err)
		}
		return &e, nil
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO worker_daily_hours
			(company_id, worker_id, project_id, work_date, hours_worked, notes, work_description, submitted_by)
		VALUES
			($1::uuid, $2::uuid, $3::uuid, $4::date, $5, $6, $7, $8::uuid)
		ON CONFLICT (company_id, worker_id, project_id, work_date)
		DO UPDATE SET
			hours_worked     = EXCLUDED.hours_worked,
			notes            = EXCLUDED.notes,
			work_description = EXCLUDED.work_description,
			submitted_by     = EXCLUDED.submitted_by
		RETURNING
			id::text,
			worker_id::text,
			project_id::text,
			(SELECT name FROM projects WHERE id = worker_daily_hours.project_id),
			work_date::text,
			hours_worked,
			notes,
			work_description,
			created_at,
			updated_at
	`, companyID, workerEmpID, projectID, workDate, hoursWorked, notes, workDescription, submittedByUserID,
	).Scan(
		&e.ID, &e.WorkerID, &e.ProjectID, &e.ProjectName,
		&e.WorkDate, &e.HoursWorked, &e.Notes, &e.WorkDescription,
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
			wdh.work_description,
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
			&e.WorkDate, &e.HoursWorked, &e.Notes, &e.WorkDescription,
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

// ListBeforeDate returns all hour entries for a worker with work_date strictly before
// beforeDate (YYYY-MM-DD), ordered newest date first then by project name.
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
			wdh.work_description,
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
			&e.WorkDate, &e.HoursWorked, &e.Notes, &e.WorkDescription,
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

// ListForManager returns all worker-hours entries for a company with optional filters.
// projectID and workDate may be empty to omit those filters.
func (r *WorkerHoursRepository) ListForManager(ctx context.Context, companyID, projectID, workDate string) ([]dto.ManagerWorkerHoursEntry, error) {
	args := []any{companyID}
	filter := ""
	if projectID != "" {
		args = append(args, projectID)
		filter += fmt.Sprintf(" AND wdh.project_id = $%d::uuid", len(args))
	}
	if workDate != "" {
		args = append(args, workDate)
		filter += fmt.Sprintf(" AND wdh.work_date = $%d::date", len(args))
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			wdh.id::text,
			wdh.worker_id::text,
			CONCAT(e.first_name, ' ', e.last_name) AS worker_name,
			e.role                                  AS worker_role,
			wdh.project_id::text,
			p.name                                  AS project_name,
			wdh.work_date::text,
			wdh.hours_worked,
			wdh.notes,
			wdh.work_description,
			EXISTS(SELECT 1 FROM worker_daily_hours_revisions rev WHERE rev.worker_daily_hours_id = wdh.id) AS has_revisions,
			EXISTS(SELECT 1 FROM worker_daily_hours_comments  cmt WHERE cmt.worker_daily_hours_id = wdh.id) AS has_comments,
			wdh.updated_at
		FROM worker_daily_hours wdh
		JOIN employees e ON e.id = wdh.worker_id
		JOIN projects   p ON p.id = wdh.project_id
		WHERE wdh.company_id = $1::uuid`+filter+`
		ORDER BY wdh.work_date DESC, e.last_name, e.first_name, p.name
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.ListForManager: %w", err)
	}
	defer rows.Close()

	var result []dto.ManagerWorkerHoursEntry
	for rows.Next() {
		var m dto.ManagerWorkerHoursEntry
		if err := rows.Scan(
			&m.ID, &m.WorkerID, &m.WorkerName, &m.WorkerRole,
			&m.ProjectID, &m.ProjectName,
			&m.WorkDate, &m.HoursWorked, &m.Notes, &m.WorkDescription,
			&m.HasRevisions, &m.HasComments, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("worker_hours.ListForManager scan: %w", err)
		}
		result = append(result, m)
	}
	if result == nil {
		result = []dto.ManagerWorkerHoursEntry{}
	}
	return result, rows.Err()
}

// GetByID returns a full detail view of a single entry including revisions and comments.
// Returns ErrWorkerHoursNotFound if the entry does not belong to the given company.
func (r *WorkerHoursRepository) GetByID(ctx context.Context, companyID, entryID string) (*dto.WorkerHoursDetail, error) {
	var d dto.WorkerHoursDetail
	err := r.db.QueryRow(ctx, `
		SELECT
			wdh.id::text,
			wdh.worker_id::text,
			CONCAT(e.first_name, ' ', e.last_name) AS worker_name,
			e.role                                  AS worker_role,
			wdh.project_id::text,
			p.name                                  AS project_name,
			wdh.work_date::text,
			wdh.hours_worked,
			wdh.notes,
			wdh.work_description,
			wdh.created_at,
			wdh.updated_at
		FROM worker_daily_hours wdh
		JOIN employees e ON e.id = wdh.worker_id
		JOIN projects   p ON p.id = wdh.project_id
		WHERE wdh.id = $1::uuid AND wdh.company_id = $2::uuid
	`, entryID, companyID).Scan(
		&d.ID, &d.WorkerID, &d.WorkerName, &d.WorkerRole,
		&d.ProjectID, &d.ProjectName,
		&d.WorkDate, &d.HoursWorked, &d.Notes, &d.WorkDescription,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkerHoursNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("worker_hours.GetByID: %w", err)
	}

	revisions, err := r.listRevisions(ctx, companyID, entryID)
	if err != nil {
		return nil, err
	}
	d.Revisions = revisions

	comments, err := r.listComments(ctx, companyID, entryID)
	if err != nil {
		return nil, err
	}
	d.Comments = comments

	return &d, nil
}

func (r *WorkerHoursRepository) listRevisions(ctx context.Context, companyID, entryID string) ([]dto.WorkerHoursRevision, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			rev.id::text,
			rev.previous_hours,
			rev.new_hours,
			rev.reason,
			(SELECT CONCAT(e.first_name, ' ', e.last_name)
			 FROM users u LEFT JOIN employees e ON e.id = u.employee_id
			 WHERE u.id = rev.changed_by) AS changed_by_name,
			rev.created_at
		FROM worker_daily_hours_revisions rev
		WHERE rev.worker_daily_hours_id = $1::uuid
		  AND rev.company_id            = $2::uuid
		ORDER BY rev.created_at
	`, entryID, companyID)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.listRevisions: %w", err)
	}
	defer rows.Close()

	var result []dto.WorkerHoursRevision
	for rows.Next() {
		var rev dto.WorkerHoursRevision
		if err := rows.Scan(&rev.ID, &rev.PreviousHours, &rev.NewHours, &rev.Reason, &rev.ChangedByName, &rev.CreatedAt); err != nil {
			return nil, fmt.Errorf("worker_hours.listRevisions scan: %w", err)
		}
		result = append(result, rev)
	}
	if result == nil {
		result = []dto.WorkerHoursRevision{}
	}
	return result, rows.Err()
}

func (r *WorkerHoursRepository) listComments(ctx context.Context, companyID, entryID string) ([]dto.WorkerHoursComment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			cmt.id::text,
			(SELECT CONCAT(e.first_name, ' ', e.last_name)
			 FROM users u LEFT JOIN employees e ON e.id = u.employee_id
			 WHERE u.id = cmt.author_user_id) AS author_name,
			cmt.comment_text,
			cmt.created_at
		FROM worker_daily_hours_comments cmt
		WHERE cmt.worker_daily_hours_id = $1::uuid
		  AND cmt.company_id            = $2::uuid
		ORDER BY cmt.created_at
	`, entryID, companyID)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.listComments: %w", err)
	}
	defer rows.Close()

	var result []dto.WorkerHoursComment
	for rows.Next() {
		var c dto.WorkerHoursComment
		if err := rows.Scan(&c.ID, &c.AuthorName, &c.CommentText, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("worker_hours.listComments scan: %w", err)
		}
		result = append(result, c)
	}
	if result == nil {
		result = []dto.WorkerHoursComment{}
	}
	return result, rows.Err()
}

// CorrectHours updates the hours for an entry and records the change in the revision table.
// Executed in a single transaction to keep the entry and revision consistent.
func (r *WorkerHoursRepository) CorrectHours(
	ctx context.Context,
	companyID, entryID string,
	newHours float64,
	reason, changedByUserID string,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("worker_hours.CorrectHours: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var previousHours float64
	err = tx.QueryRow(ctx, `
		SELECT hours_worked
		FROM worker_daily_hours
		WHERE id = $1::uuid AND company_id = $2::uuid
		FOR UPDATE
	`, entryID, companyID).Scan(&previousHours)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkerHoursNotFound
	}
	if err != nil {
		return fmt.Errorf("worker_hours.CorrectHours: lock: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		UPDATE worker_daily_hours
		SET hours_worked = $1, updated_at = NOW()
		WHERE id = $2::uuid AND company_id = $3::uuid
	`, newHours, entryID, companyID); err != nil {
		return fmt.Errorf("worker_hours.CorrectHours: update: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO worker_daily_hours_revisions
			(company_id, worker_daily_hours_id, previous_hours, new_hours, reason, changed_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid)
	`, companyID, entryID, previousHours, newHours, reason, changedByUserID); err != nil {
		return fmt.Errorf("worker_hours.CorrectHours: insert revision: %w", err)
	}

	return tx.Commit(ctx)
}

// AddComment inserts a private manager comment on an entry.
func (r *WorkerHoursRepository) AddComment(
	ctx context.Context,
	companyID, entryID, authorUserID, text string,
) (*dto.WorkerHoursComment, error) {
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM worker_daily_hours WHERE id = $1::uuid AND company_id = $2::uuid)`,
		entryID, companyID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("worker_hours.AddComment: check entry: %w", err)
	}
	if !exists {
		return nil, ErrWorkerHoursNotFound
	}

	var c dto.WorkerHoursComment
	err := r.db.QueryRow(ctx, `
		INSERT INTO worker_daily_hours_comments
			(company_id, worker_daily_hours_id, author_user_id, comment_text)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		RETURNING
			id::text,
			(SELECT CONCAT(e.first_name, ' ', e.last_name)
			 FROM users u LEFT JOIN employees e ON e.id = u.employee_id
			 WHERE u.id = $3::uuid) AS author_name,
			comment_text,
			created_at
	`, companyID, entryID, authorUserID, text).Scan(
		&c.ID, &c.AuthorName, &c.CommentText, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("worker_hours.AddComment: %w", err)
	}
	return &c, nil
}
