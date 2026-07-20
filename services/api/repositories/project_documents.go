package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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

// Create inserts a new document record at project root (no folder) and returns the new document ID.
func (r *ProjectDocumentsRepository) Create(
	ctx context.Context,
	companyID, projectID, uploadedByUserID, fileKey, originalName, contentType string,
	fileSize int64,
) (string, error) {
	return r.CreateWithFolder(ctx, companyID, projectID, uploadedByUserID, fileKey, originalName, contentType, fileSize, nil)
}

// CreateWithFolder inserts a document record in the given folder (nil = project root).
func (r *ProjectDocumentsRepository) CreateWithFolder(
	ctx context.Context,
	companyID, projectID, uploadedByUserID, fileKey, originalName, contentType string,
	fileSize int64,
	folderID *string,
) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO project_documents
		    (company_id, project_id, uploaded_by, file_key, original_name, content_type, file_size, folder_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8::uuid)
		RETURNING id::text
	`, companyID, projectID, uploadedByUserID, fileKey, originalName, contentType, fileSize, folderID).Scan(&id)
	return id, err
}

// BeginTx starts a transaction on the underlying pool.
func (r *ProjectDocumentsRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// List returns all non-deleted documents for the given project, newest first.
// Kept for backwards compatibility; new callers should prefer ListInFolder.
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
		       pd.created_at,
		       pd.folder_id::text
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
		if err := rows.Scan(&d.ID, &d.OriginalName, &d.ContentType, &d.FileSize, &d.UploadedByEmail, &d.CreatedAt, &d.FolderID); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// ListInFolder returns documents in the given folder (nil = project root).
func (r *ProjectDocumentsRepository) ListInFolder(
	ctx context.Context,
	companyID, projectID string,
	folderID *string,
) ([]dto.ProjectDocumentItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pd.id::text,
		       pd.original_name,
		       pd.content_type,
		       pd.file_size,
		       COALESCE(u.email, '') AS uploaded_by_email,
		       pd.created_at,
		       pd.folder_id::text
		FROM   project_documents pd
		LEFT JOIN users u ON u.id = pd.uploaded_by
		WHERE  pd.project_id = $1::uuid
		  AND  pd.company_id = $2::uuid
		  AND  pd.deleted_at IS NULL
		  AND  (pd.folder_id IS NOT DISTINCT FROM $3::uuid)
		ORDER BY lower(pd.original_name)
	`, projectID, companyID, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]dto.ProjectDocumentItem, 0)
	for rows.Next() {
		var d dto.ProjectDocumentItem
		if err := rows.Scan(&d.ID, &d.OriginalName, &d.ContentType, &d.FileSize, &d.UploadedByEmail, &d.CreatedAt, &d.FolderID); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// CheckDuplicate returns true if an active document with the same name (case-insensitive)
// already exists in the given folder (nil = project root).
func (r *ProjectDocumentsRepository) CheckDuplicate(
	ctx context.Context,
	companyID, projectID string,
	folderID *string,
	originalName string,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_documents
			WHERE company_id = $1::uuid
			  AND project_id = $2::uuid
			  AND (folder_id IS NOT DISTINCT FROM $3::uuid)
			  AND lower(original_name) = lower($4)
			  AND deleted_at IS NULL
		)
	`, companyID, projectID, folderID, originalName).Scan(&exists)
	return exists, err
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

// HasActiveDocuments reports whether the given folder contains any non-deleted documents.
// Must be called inside a transaction that has already locked the folder row.
func (r *ProjectDocumentsRepository) HasActiveDocuments(
	ctx context.Context,
	tx pgx.Tx,
	companyID, projectID, folderID string,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_documents
			WHERE folder_id = $1::uuid
			  AND company_id = $2::uuid
			  AND project_id = $3::uuid
			  AND deleted_at IS NULL
		)
	`, folderID, companyID, projectID).Scan(&exists)
	return exists, err
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
