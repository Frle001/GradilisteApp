package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
)

type ProjectDocumentsRepository struct {
	db *pgxpool.Pool
}

func NewProjectDocumentsRepository(db *pgxpool.Pool) *ProjectDocumentsRepository {
	return &ProjectDocumentsRepository{db: db}
}

// ProjectDocumentRecord is the internal struct used for download/delete operations.
type ProjectDocumentRecord struct {
	ID           string
	FileKey      string
	OriginalName string
	ContentType  string
	FileSize     int64
	DeletedAt    *time.Time
}

// Create inserts a new document record and returns the new document ID.
func (r *ProjectDocumentsRepository) Create(
	ctx context.Context,
	companyID, projectID, uploadedByUserID, fileKey, originalName, contentType string,
	fileSize int64,
) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO project_documents
		    (company_id, project_id, uploaded_by, file_key, original_name, content_type, file_size)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
		RETURNING id::text
	`, companyID, projectID, uploadedByUserID, fileKey, originalName, contentType, fileSize).Scan(&id)
	return id, err
}

// List returns all non-deleted documents for the given project, newest first.
func (r *ProjectDocumentsRepository) List(
	ctx context.Context,
	companyID, projectID string,
) ([]dto.ProjectDocumentItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pd.id::text,
		       pd.original_name,
		       pd.content_type,
		       pd.file_size,
		       COALESCE(u.email, '') AS uploaded_by_email,
		       pd.created_at
		FROM   project_documents pd
		LEFT JOIN users u ON u.id = pd.uploaded_by
		WHERE  pd.project_id = $1::uuid
		  AND  pd.company_id = $2::uuid
		  AND  pd.deleted_at IS NULL
		ORDER BY pd.created_at DESC
	`, projectID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]dto.ProjectDocumentItem, 0)
	for rows.Next() {
		var d dto.ProjectDocumentItem
		if err := rows.Scan(&d.ID, &d.OriginalName, &d.ContentType, &d.FileSize, &d.UploadedByEmail, &d.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// GetByID fetches a single document by ID, scoped to company and project. Returns ErrNotFound if missing or deleted.
func (r *ProjectDocumentsRepository) GetByID(
	ctx context.Context,
	companyID, projectID, docID string,
) (*ProjectDocumentRecord, error) {
	var rec ProjectDocumentRecord
	err := r.db.QueryRow(ctx, `
		SELECT id::text, file_key, original_name, content_type, file_size, deleted_at
		FROM   project_documents
		WHERE  id = $1::uuid
		  AND  company_id = $2::uuid
		  AND  project_id = $3::uuid
	`, docID, companyID, projectID).Scan(
		&rec.ID, &rec.FileKey, &rec.OriginalName, &rec.ContentType, &rec.FileSize, &rec.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	if rec.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return &rec, nil
}

// SoftDelete sets deleted_at on a document record. It does NOT remove the storage file.
func (r *ProjectDocumentsRepository) SoftDelete(ctx context.Context, companyID, projectID, docID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE project_documents
		SET    deleted_at = NOW()
		WHERE  id = $1::uuid
		  AND  company_id = $2::uuid
		  AND  project_id = $3::uuid
		  AND  deleted_at IS NULL
	`, docID, companyID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// IsProjectAssigned returns true if the given employee is actively assigned to the project.
func (r *ProjectDocumentsRepository) IsProjectAssigned(
	ctx context.Context,
	companyID, projectID, employeeID string,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_assignments
			WHERE  project_id  = $1::uuid
			  AND  company_id  = $2::uuid
			  AND  employee_id = $3::uuid
			  AND  active      = true
		)
	`, projectID, companyID, employeeID).Scan(&exists)
	return exists, err
}

// ProjectBelongsToCompany checks that the project exists and belongs to the company.
func (r *ProjectDocumentsRepository) ProjectBelongsToCompany(
	ctx context.Context,
	companyID, projectID string,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM projects
			WHERE id = $1::uuid AND company_id = $2::uuid
		)
	`, projectID, companyID).Scan(&exists)
	return exists, err
}
