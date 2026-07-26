package repositories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

var ErrAttachmentNotFound = errors.New("attachment not found")

// DailyReportAttachmentRecord is the full DB record used for download and delete.
type DailyReportAttachmentRecord struct {
	ID           string
	FileKey      string
	OriginalName string
	ContentType  string
	FileSize     int64
}

type DailyReportAttachmentsRepository struct {
	db *pgxpool.Pool
}

func NewDailyReportAttachmentsRepository(db *pgxpool.Pool) *DailyReportAttachmentsRepository {
	return &DailyReportAttachmentsRepository{db: db}
}

// ReportBelongsToCompany returns true if the report exists and belongs to the company.
func (r *DailyReportAttachmentsRepository) ReportBelongsToCompany(ctx context.Context, companyID, reportID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM daily_reports WHERE id = $1::uuid AND company_id = $2::uuid)`,
		reportID, companyID,
	).Scan(&exists)
	return exists, err
}

// GetReportMeta returns the report's poslovoda_id and status for write-permission checks.
// Returns ErrDailyReportNotFound if the report does not belong to the company.
func (r *DailyReportAttachmentsRepository) GetReportMeta(ctx context.Context, companyID, reportID string) (poslovodaEmpID, status string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT poslovoda_id::text, status FROM daily_reports WHERE id = $1::uuid AND company_id = $2::uuid`,
		reportID, companyID,
	).Scan(&poslovodaEmpID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrDailyReportNotFound
	}
	return poslovodaEmpID, status, err
}

// CountActive returns the number of non-deleted attachments for a report.
func (r *DailyReportAttachmentsRepository) CountActive(ctx context.Context, companyID, reportID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM daily_report_attachments
		 WHERE daily_report_id = $1::uuid AND company_id = $2::uuid AND deleted_at IS NULL`,
		reportID, companyID,
	).Scan(&count)
	return count, err
}

// List returns all active attachments for a report ordered by upload time.
func (r *DailyReportAttachmentsRepository) List(ctx context.Context, companyID, reportID string) ([]dto.DailyReportAttachment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, original_name, content_type, file_size, created_at
		FROM daily_report_attachments
		WHERE daily_report_id = $1::uuid AND company_id = $2::uuid AND deleted_at IS NULL
		ORDER BY created_at
	`, reportID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]dto.DailyReportAttachment, 0)
	for rows.Next() {
		var a dto.DailyReportAttachment
		if err := rows.Scan(&a.ID, &a.OriginalName, &a.ContentType, &a.FileSize, &a.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

// Create inserts a new attachment record and returns its ID.
func (r *DailyReportAttachmentsRepository) Create(
	ctx context.Context,
	companyID, reportID, userID, fileKey, originalName, contentType string,
	fileSize int64,
) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO daily_report_attachments
			(company_id, daily_report_id, uploaded_by, file_key, original_name, content_type, file_size)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
		RETURNING id::text
	`, companyID, reportID, userID, fileKey, originalName, contentType, fileSize).Scan(&id)
	return id, err
}

// GetByID returns the full record for a non-deleted attachment.
func (r *DailyReportAttachmentsRepository) GetByID(ctx context.Context, companyID, reportID, attachmentID string) (*DailyReportAttachmentRecord, error) {
	var rec DailyReportAttachmentRecord
	err := r.db.QueryRow(ctx, `
		SELECT id::text, file_key, original_name, content_type, file_size
		FROM daily_report_attachments
		WHERE id = $1::uuid AND daily_report_id = $2::uuid AND company_id = $3::uuid AND deleted_at IS NULL
	`, attachmentID, reportID, companyID).Scan(
		&rec.ID, &rec.FileKey, &rec.OriginalName, &rec.ContentType, &rec.FileSize,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAttachmentNotFound
	}
	return &rec, err
}

// SoftDelete marks an attachment as deleted. Returns ErrAttachmentNotFound if already deleted or absent.
func (r *DailyReportAttachmentsRepository) SoftDelete(ctx context.Context, companyID, reportID, attachmentID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE daily_report_attachments SET deleted_at = NOW()
		WHERE id = $1::uuid AND daily_report_id = $2::uuid AND company_id = $3::uuid AND deleted_at IS NULL
	`, attachmentID, reportID, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}
