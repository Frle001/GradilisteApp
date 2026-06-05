package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

type ImportJobRepository struct {
	db *pgxpool.Pool
}

func NewImportJobRepository(db *pgxpool.Pool) *ImportJobRepository {
	return &ImportJobRepository{db: db}
}

var ErrImportJobNotFound = errors.New("import job not found")

type ImportJob struct {
	ID              string
	CompanyID       string
	ProjectID       string
	OriginalFilename string
	Status          string
	TotalRows       int
	ValidRows       int
	InvalidRows     int
}

// CreateJob inserts a new import_jobs row and returns its ID.
func (r *ImportJobRepository) CreateJob(ctx context.Context, tx pgx.Tx, companyID, projectID, filename, createdByUserID string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO import_jobs (company_id, project_id, import_type, original_filename, status, created_by)
		VALUES ($1::uuid, $2::uuid, 'project_materials', $3, 'uploaded', $4::uuid)
		RETURNING id
	`, companyID, projectID, filename, createdByUserID).Scan(&id)
	return id, err
}

// CreateRow inserts a single import_job_rows record.
// normalizedData may be nil for invalid rows.
func (r *ImportJobRepository) CreateRow(ctx context.Context, tx pgx.Tx, companyID, jobID string, rowNum int, rawData interface{}, normalizedData interface{}, status, errMsg string) error {
	rawJSON, err := json.Marshal(rawData)
	if err != nil {
		return err
	}

	var normJSON []byte
	if normalizedData != nil {
		normJSON, err = json.Marshal(normalizedData)
		if err != nil {
			return err
		}
	}

	var errMsgPtr *string
	if errMsg != "" {
		errMsgPtr = &errMsg
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO import_job_rows (company_id, import_job_id, row_number, raw_data, normalized_data, status, error_message)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
	`, companyID, jobID, rowNum, rawJSON, normJSON, status, errMsgPtr)
	return err
}

// UpdateJobCounts sets total/valid/invalid row counts and advances status to 'parsed'.
func (r *ImportJobRepository) UpdateJobCounts(ctx context.Context, tx pgx.Tx, jobID string, total, valid, invalid int) error {
	_, err := tx.Exec(ctx, `
		UPDATE import_jobs
		SET status = 'parsed', total_rows = $1, valid_rows = $2, invalid_rows = $3
		WHERE id = $4::uuid
	`, total, valid, invalid, jobID)
	return err
}

// GetJob retrieves an import job by ID, scoped to company and project.
func (r *ImportJobRepository) GetJob(ctx context.Context, jobID, projectID, companyID string) (*ImportJob, error) {
	var j ImportJob
	err := r.db.QueryRow(ctx, `
		SELECT id, company_id, project_id, original_filename, status, total_rows, valid_rows, invalid_rows
		FROM import_jobs
		WHERE id = $1::uuid AND project_id = $2::uuid AND company_id = $3::uuid
	`, jobID, projectID, companyID).Scan(
		&j.ID, &j.CompanyID, &j.ProjectID, &j.OriginalFilename,
		&j.Status, &j.TotalRows, &j.ValidRows, &j.InvalidRows,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrImportJobNotFound
	}
	return &j, err
}

// GetValidRows returns all valid rows for a job as ImportPreviewRows.
func (r *ImportJobRepository) GetValidRows(ctx context.Context, tx pgx.Tx, jobID, companyID string) ([]dto.ImportPreviewRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT row_number, normalized_data
		FROM import_job_rows
		WHERE import_job_id = $1::uuid AND company_id = $2::uuid AND status = 'valid'
		ORDER BY row_number ASC
	`, jobID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dto.ImportPreviewRow
	for rows.Next() {
		var rowNum int
		var normData []byte
		if err := rows.Scan(&rowNum, &normData); err != nil {
			return nil, err
		}
		var row dto.ImportPreviewRow
		if err := json.Unmarshal(normData, &row); err != nil {
			return nil, err
		}
		row.RowNumber = rowNum
		result = append(result, row)
	}
	return result, rows.Err()
}

// MarkJobConfirmed advances status to 'confirmed'.
func (r *ImportJobRepository) MarkJobConfirmed(ctx context.Context, tx pgx.Tx, jobID string) error {
	_, err := tx.Exec(ctx, `UPDATE import_jobs SET status = 'confirmed' WHERE id = $1::uuid`, jobID)
	return err
}

// MarkRowsImported sets status='imported' for all valid rows of a job.
func (r *ImportJobRepository) MarkRowsImported(ctx context.Context, tx pgx.Tx, jobID, companyID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE import_job_rows SET status = 'imported'
		WHERE import_job_id = $1::uuid AND company_id = $2::uuid AND status = 'valid'
	`, jobID, companyID)
	return err
}

// ── Phase 6.5 wizard import helpers ──────────────────────────────────────────

// RawImportRow holds a single row of raw cell data from the analyze step.
type RawImportRow struct {
	RowNumber int
	Cells     map[string]string // {originalColHeader: cellValue}
}

// SetJobTotalRows updates total_rows without changing status (used at analyze time).
func (r *ImportJobRepository) SetJobTotalRows(ctx context.Context, tx pgx.Tx, jobID string, totalRows int) error {
	_, err := tx.Exec(ctx, `
		UPDATE import_jobs SET total_rows = $1 WHERE id = $2::uuid
	`, totalRows, jobID)
	return err
}

// StoreAnalyzeRows inserts raw cell rows as import_job_rows with status='invalid'
// (placeholder — not yet validated). dataRowStart is the 1-based Excel row number of
// the first data row (header row number + 1), used to store correct row numbers.
func (r *ImportJobRepository) StoreAnalyzeRows(ctx context.Context, tx pgx.Tx, companyID, jobID string, rows []map[string]string, dataRowStart int) error {
	for i, cells := range rows {
		rawJSON, err := json.Marshal(cells)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO import_job_rows (company_id, import_job_id, row_number, raw_data, normalized_data, status)
			VALUES ($1::uuid, $2::uuid, $3, $4, NULL, 'invalid')
		`, companyID, jobID, dataRowStart+i, rawJSON)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAllRawRows returns all rows for a job (regardless of status), with raw_data.
func (r *ImportJobRepository) GetAllRawRows(ctx context.Context, jobID, companyID string) ([]RawImportRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT row_number, raw_data
		FROM import_job_rows
		WHERE import_job_id = $1::uuid AND company_id = $2::uuid
		ORDER BY row_number ASC
	`, jobID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RawImportRow
	for rows.Next() {
		var rowNum int
		var rawJSON []byte
		if err := rows.Scan(&rowNum, &rawJSON); err != nil {
			return nil, err
		}
		var cells map[string]string
		if err := json.Unmarshal(rawJSON, &cells); err != nil {
			return nil, err
		}
		result = append(result, RawImportRow{RowNumber: rowNum, Cells: cells})
	}
	return result, rows.Err()
}

// DeleteAllRows removes all import_job_rows for a job (called before re-inserting after mapping).
func (r *ImportJobRepository) DeleteAllRows(ctx context.Context, tx pgx.Tx, jobID, companyID string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM import_job_rows WHERE import_job_id = $1::uuid AND company_id = $2::uuid
	`, jobID, companyID)
	return err
}

// StoreWizardRows inserts validated wizard preview rows into import_job_rows.
func (r *ImportJobRepository) StoreWizardRows(ctx context.Context, tx pgx.Tx, companyID, jobID string, rows []dto.WizardPreviewRow) error {
	for _, row := range rows {
		var normJSON []byte
		var err error
		if row.Status == "valid" {
			normJSON, err = json.Marshal(row)
			if err != nil {
				return err
			}
		}

		errMsg := ""
		if len(row.Errors) > 0 {
			errMsg = strings.Join(row.Errors, "; ")
		}
		var errMsgPtr *string
		if errMsg != "" {
			errMsgPtr = &errMsg
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO import_job_rows (company_id, import_job_id, row_number, raw_data, normalized_data, status, error_message)
			VALUES ($1::uuid, $2::uuid, $3, '{}'::jsonb, $4, $5, $6)
		`, companyID, jobID, row.RowNumber, normJSON, row.Status, errMsgPtr)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetValidWizardRows returns valid rows for a job as WizardPreviewRows, for use during confirm.
func (r *ImportJobRepository) GetValidWizardRows(ctx context.Context, tx pgx.Tx, jobID, companyID string) ([]dto.WizardPreviewRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT row_number, normalized_data
		FROM import_job_rows
		WHERE import_job_id = $1::uuid AND company_id = $2::uuid AND status = 'valid'
		ORDER BY row_number ASC
	`, jobID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dto.WizardPreviewRow
	for rows.Next() {
		var rowNum int
		var normData []byte
		if err := rows.Scan(&rowNum, &normData); err != nil {
			return nil, err
		}
		var row dto.WizardPreviewRow
		if err := json.Unmarshal(normData, &row); err != nil {
			return nil, err
		}
		row.RowNumber = rowNum
		result = append(result, row)
	}
	return result, rows.Err()
}
