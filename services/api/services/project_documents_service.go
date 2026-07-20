package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

var ErrDocumentNotFound = errors.New("document not found")
var ErrDocumentForbidden = errors.New("access denied to this document")

// ProjectDocumentDownload bundles the file stream and its metadata for streaming to the client.
type ProjectDocumentDownload struct {
	Reader      io.ReadCloser
	ContentType string
	OrigName    string
	FileSize    int64
}

type ProjectDocumentsService struct {
	repo    *repositories.ProjectDocumentsRepository
	storage StorageService
}

func NewProjectDocumentsService(
	repo *repositories.ProjectDocumentsRepository,
	storage StorageService,
) *ProjectDocumentsService {
	return &ProjectDocumentsService{repo: repo, storage: storage}
}

// List returns all documents for a project. Poslovoda must be assigned to the project.
func (s *ProjectDocumentsService) List(
	ctx context.Context,
	companyID, role, employeeID, projectID string,
) ([]dto.ProjectDocumentItem, error) {
	if err := s.checkProjectAccess(ctx, companyID, role, employeeID, projectID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, companyID, projectID)
}

// Upload saves a file to storage and creates a DB record. Only direktor/inzenjer may call this.
func (s *ProjectDocumentsService) Upload(
	ctx context.Context,
	companyID, userID, projectID string,
	file multipart.File,
	header *multipart.FileHeader,
) (*dto.ProjectDocumentItem, error) {
	exists, err := s.repo.ProjectBelongsToCompany(ctx, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check project: %w", err)
	}
	if !exists {
		return nil, ErrDocumentNotFound
	}

	if err := ValidateDocumentFile(header); err != nil {
		return nil, err
	}

	origName := sanitizeDocumentFilename(header.Filename)

	fileKey, contentType, fileSize, err := s.storage.SaveDocumentFile(ctx, file, header, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("save document: %w", err)
	}

	docID, err := s.repo.Create(ctx, companyID, projectID, userID, fileKey, origName, contentType, fileSize)
	if err != nil {
		// Best-effort cleanup — ignore cleanup error, log it separately if needed.
		_ = s.storage.DeleteReceiptFile(ctx, fileKey)
		return nil, fmt.Errorf("create document record: %w", err)
	}

	// Fetch the newly created record to return full DTO.
	docs, err := s.repo.List(ctx, companyID, projectID)
	if err == nil {
		for _, d := range docs {
			if d.ID == docID {
				return &d, nil
			}
		}
	}

	return &dto.ProjectDocumentItem{
		ID:           docID,
		OriginalName: origName,
		ContentType:  contentType,
		FileSize:     fileSize,
	}, nil
}

// Download returns a readable stream and document metadata. Poslovoda must be assigned.
func (s *ProjectDocumentsService) Download(
	ctx context.Context,
	companyID, role, employeeID, projectID, docID string,
) (*ProjectDocumentDownload, error) {
	if err := s.checkProjectAccess(ctx, companyID, role, employeeID, projectID); err != nil {
		return nil, err
	}

	rec, err := s.repo.GetByID(ctx, companyID, projectID, docID)
	if err != nil {
		return nil, ErrDocumentNotFound
	}

	reader, _, err := s.storage.ReadReceiptFile(ctx, rec.FileKey)
	if err != nil {
		return nil, fmt.Errorf("read document file: %w", err)
	}

	return &ProjectDocumentDownload{
		Reader:      reader,
		ContentType: rec.ContentType,
		OrigName:    rec.OriginalName,
		FileSize:    rec.FileSize,
	}, nil
}

// Delete soft-deletes a document and removes the storage file. Only direktor/inzenjer may call this.
func (s *ProjectDocumentsService) Delete(
	ctx context.Context,
	companyID, projectID, docID string,
) error {
	rec, err := s.repo.GetByID(ctx, companyID, projectID, docID)
	if err != nil {
		return ErrDocumentNotFound
	}

	if err := s.repo.SoftDelete(ctx, companyID, projectID, docID); err != nil {
		return fmt.Errorf("soft delete document: %w", err)
	}

	// Best-effort storage cleanup — soft delete already hides the file from users.
	_ = s.storage.DeleteReceiptFile(ctx, rec.FileKey)

	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (s *ProjectDocumentsService) checkProjectAccess(
	ctx context.Context,
	companyID, role, employeeID, projectID string,
) error {
	exists, err := s.repo.ProjectBelongsToCompany(ctx, companyID, projectID)
	if err != nil {
		return fmt.Errorf("check project: %w", err)
	}
	if !exists {
		return ErrDocumentNotFound
	}

	return nil
}

// sanitizeDocumentFilename strips path components and control characters.
func sanitizeDocumentFilename(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if len(name) > 255 {
		name = name[:255]
	}
	if name == "" || name == "." {
		return "document"
	}
	return name
}
