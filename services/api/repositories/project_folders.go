package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectFolder struct {
	ID        string
	CompanyID string
	ProjectID string
	ParentID  *string
	Name      string
	CreatedAt time.Time
}

type ProjectFoldersRepository struct {
	db *pgxpool.Pool
}

func NewProjectFoldersRepository(db *pgxpool.Pool) *ProjectFoldersRepository {
	return &ProjectFoldersRepository{db: db}
}

// ListByParent returns immediate children of the given parent (nil = project root).
func (r *ProjectFoldersRepository) ListByParent(
	ctx context.Context,
	companyID, projectID string,
	parentID *string,
) ([]ProjectFolder, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if parentID == nil {
		rows, err = r.db.Query(ctx, `
			SELECT id::text, company_id::text, project_id::text, parent_id::text, name, created_at
			FROM project_folders
			WHERE company_id = $1::uuid AND project_id = $2::uuid AND parent_id IS NULL
			ORDER BY lower(name)
		`, companyID, projectID)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT id::text, company_id::text, project_id::text, parent_id::text, name, created_at
			FROM project_folders
			WHERE company_id = $1::uuid AND project_id = $2::uuid AND parent_id = $3::uuid
			ORDER BY lower(name)
		`, companyID, projectID, *parentID)
	}
	if err != nil {
		return nil, fmt.Errorf("folders.ListByParent: %w", err)
	}
	defer rows.Close()

	var folders []ProjectFolder
	for rows.Next() {
		var f ProjectFolder
		if err := rows.Scan(&f.ID, &f.CompanyID, &f.ProjectID, &f.ParentID, &f.Name, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("folders.ListByParent scan: %w", err)
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// GetByID returns a folder scoped to company + project.
func (r *ProjectFoldersRepository) GetByID(
	ctx context.Context,
	companyID, projectID, id string,
) (*ProjectFolder, error) {
	var f ProjectFolder
	err := r.db.QueryRow(ctx, `
		SELECT id::text, company_id::text, project_id::text, parent_id::text, name, created_at
		FROM project_folders
		WHERE id = $1::uuid AND company_id = $2::uuid AND project_id = $3::uuid
	`, id, companyID, projectID).Scan(
		&f.ID, &f.CompanyID, &f.ProjectID, &f.ParentID, &f.Name, &f.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("folders.GetByID: %w", err)
	}
	return &f, nil
}

// BelongsToProject returns true if the folder exists and belongs to the project.
func (r *ProjectFoldersRepository) BelongsToProject(
	ctx context.Context,
	companyID, projectID, folderID string,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_folders
			WHERE id = $1::uuid AND company_id = $2::uuid AND project_id = $3::uuid
		)
	`, folderID, companyID, projectID).Scan(&exists)
	return exists, err
}

// GetFolderForUpdate locks the folder row within tx.
// Returns (false, nil) if the folder does not exist under the given company + project.
func (r *ProjectFoldersRepository) GetFolderForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	companyID, projectID, folderID string,
) (bool, error) {
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id::text FROM project_folders
		WHERE id = $1::uuid AND company_id = $2::uuid AND project_id = $3::uuid
		FOR UPDATE
	`, folderID, companyID, projectID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// HasChildFolders reports whether the folder has any direct child sub-folders.
func (r *ProjectFoldersRepository) HasChildFolders(
	ctx context.Context,
	tx pgx.Tx,
	companyID, projectID, folderID string,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_folders
			WHERE parent_id = $1::uuid AND company_id = $2::uuid AND project_id = $3::uuid
		)
	`, folderID, companyID, projectID).Scan(&exists)
	return exists, err
}

// DeleteFolder hard-deletes a folder row scoped to company and project.
func (r *ProjectFoldersRepository) DeleteFolder(
	ctx context.Context,
	tx pgx.Tx,
	companyID, projectID, folderID string,
) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM project_folders
		WHERE id = $1::uuid AND company_id = $2::uuid AND project_id = $3::uuid
	`, folderID, companyID, projectID)
	return err
}

// FindOrCreate returns an existing matching folder's ID or creates it.
// Safe under concurrent inserts: uses INSERT…ON CONFLICT DO NOTHING then re-SELECT.
func (r *ProjectFoldersRepository) FindOrCreate(
	ctx context.Context,
	tx pgx.Tx,
	companyID, projectID string,
	parentID *string,
	name, createdBy string,
) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("folder name cannot be empty")
	}

	var id string

	if parentID == nil {
		// Root-level folder — partial index: WHERE parent_id IS NULL
		err := tx.QueryRow(ctx, `
			INSERT INTO project_folders (company_id, project_id, parent_id, name, created_by)
			VALUES ($1::uuid, $2::uuid, NULL, $3, $4::uuid)
			ON CONFLICT (company_id, project_id, (lower(name))) WHERE parent_id IS NULL
			DO NOTHING
			RETURNING id::text
		`, companyID, projectID, name, createdBy).Scan(&id)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("folders.FindOrCreate insert root: %w", err)
		}
		if id == "" {
			// Conflict: fetch the existing row
			if err := tx.QueryRow(ctx, `
				SELECT id::text FROM project_folders
				WHERE company_id = $1::uuid AND project_id = $2::uuid
				  AND parent_id IS NULL AND lower(name) = lower($3)
			`, companyID, projectID, name).Scan(&id); err != nil {
				return "", fmt.Errorf("folders.FindOrCreate select root: %w", err)
			}
		}
	} else {
		// Child folder — partial index: WHERE parent_id IS NOT NULL
		err := tx.QueryRow(ctx, `
			INSERT INTO project_folders (company_id, project_id, parent_id, name, created_by)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5::uuid)
			ON CONFLICT (company_id, project_id, parent_id, (lower(name))) WHERE parent_id IS NOT NULL
			DO NOTHING
			RETURNING id::text
		`, companyID, projectID, *parentID, name, createdBy).Scan(&id)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("folders.FindOrCreate insert child: %w", err)
		}
		if id == "" {
			if err := tx.QueryRow(ctx, `
				SELECT id::text FROM project_folders
				WHERE company_id = $1::uuid AND project_id = $2::uuid
				  AND parent_id = $3::uuid AND lower(name) = lower($4)
			`, companyID, projectID, *parentID, name).Scan(&id); err != nil {
				return "", fmt.Errorf("folders.FindOrCreate select child: %w", err)
			}
		}
	}

	return id, nil
}
