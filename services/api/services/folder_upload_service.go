package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Internal interfaces for dependency injection and testing ──────────────────

// uploadDocRepo is the document-repository surface used by FolderUploadService.
type uploadDocRepo interface {
	ProjectBelongsToCompany(ctx context.Context, companyID, projectID string) (bool, error)
	CheckDuplicate(ctx context.Context, companyID, projectID string, folderID *string, originalName string) (bool, error)
	CreateWithFolder(ctx context.Context, companyID, projectID, uploadedByUserID, fileKey, originalName, contentType string, fileSize int64, folderID *string) (string, error)
}

// uploadFolderRepo is the folder-repository surface used by FolderUploadService.
type uploadFolderRepo interface {
	BelongsToProject(ctx context.Context, companyID, projectID, folderID string) (bool, error)
	FindOrCreate(ctx context.Context, tx pgx.Tx, companyID, projectID string, parentID *string, name, createdBy string) (string, error)
}

// txBeginner allows starting a transaction; satisfied by *pgxpool.Pool.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

const (
	MaxBatchFiles  = 250
	MaxBatchBytes  = 500 << 20 // 500 MB total per batch
	MaxFolderDepth = 20
	MaxPathLen     = 1024
)

var systemFileNames = map[string]bool{
	".ds_store":   true,
	"thumbs.db":   true,
	"desktop.ini": true,
	"._.ds_store": true,
}

// BatchUploadInput pairs a file with its client-provided relative path.
type BatchUploadInput struct {
	File         multipart.File
	Header       *multipart.FileHeader
	RelativePath string
}

type FolderUploadService struct {
	db         txBeginner
	docRepo    uploadDocRepo
	folderRepo uploadFolderRepo
	storage    StorageService
}

func NewFolderUploadService(
	db *pgxpool.Pool,
	docRepo *repositories.ProjectDocumentsRepository,
	folderRepo *repositories.ProjectFoldersRepository,
	storage StorageService,
) *FolderUploadService {
	return &FolderUploadService{db: db, docRepo: docRepo, folderRepo: folderRepo, storage: storage}
}

// IsSystemFile reports whether the given base filename is a known hidden/system file.
func IsSystemFile(name string) bool {
	return systemFileNames[strings.ToLower(name)]
}

// NormalizeUploadPath validates and normalizes a client-supplied relative path.
// Returns the cleaned path or an error describing the violation.
func NormalizeUploadPath(rawPath string) (string, error) {
	// Normalize separators.
	p := strings.ReplaceAll(rawPath, "\\", "/")

	// Null bytes.
	if strings.ContainsRune(p, 0) {
		return "", fmt.Errorf("neispravan put (null byte)")
	}

	// Absolute paths (Unix or Windows drive).
	if strings.HasPrefix(p, "/") || (len(p) >= 2 && p[1] == ':') {
		return "", fmt.Errorf("apsolutni put nije dozvoljen")
	}

	// Total length.
	if len(p) > MaxPathLen {
		return "", fmt.Errorf("put je predugačak (max %d znakova)", MaxPathLen)
	}

	segments := strings.Split(p, "/")

	// Depth: path has len(segments) parts; last is the filename, rest are dirs.
	if len(segments)-1 > MaxFolderDepth {
		return "", fmt.Errorf("prevelika dubina mapa (max %d razina)", MaxFolderDepth)
	}

	for _, seg := range segments {
		if seg == "" {
			return "", fmt.Errorf("prazan segment u putu")
		}
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("neispravan segment u putu: %q", seg)
		}
	}

	return strings.Join(segments, "/"), nil
}

// Process validates and uploads a batch of files, preserving folder structure
// under rootFolderID (nil = project root).
func (s *FolderUploadService) Process(
	ctx context.Context,
	companyID, projectID, userID string,
	rootFolderID *string,
	inputs []BatchUploadInput,
) (*dto.FolderUploadResponse, error) {
	// Batch-level limits — check before any DB access.
	if len(inputs) > MaxBatchFiles {
		return nil, validationErr(fmt.Sprintf("previše datoteka (max %d, primljeno %d)", MaxBatchFiles, len(inputs)))
	}
	var totalBytes int64
	for _, inp := range inputs {
		totalBytes += inp.Header.Size
	}
	if totalBytes > MaxBatchBytes {
		return nil, validationErr(fmt.Sprintf("ukupna veličina premašuje %d MB", MaxBatchBytes>>20))
	}

	// Validate project belongs to company.
	exists, err := s.docRepo.ProjectBelongsToCompany(ctx, companyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check project: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	// Validate root folder belongs to project.
	if rootFolderID != nil && *rootFolderID != "" {
		ok, err := s.folderRepo.BelongsToProject(ctx, companyID, projectID, *rootFolderID)
		if err != nil {
			return nil, fmt.Errorf("check folder: %w", err)
		}
		if !ok {
			return nil, ErrForbidden
		}
	} else {
		rootFolderID = nil
	}

	resp := &dto.FolderUploadResponse{
		Results: make([]dto.FolderUploadResult, 0, len(inputs)),
	}
	// folderCache maps "rootID/seg1/seg2" -> *string folderID within this batch.
	folderCache := make(map[string]*string)

	for _, inp := range inputs {
		result := s.processOne(ctx, companyID, projectID, userID, rootFolderID, inp, folderCache)
		resp.Results = append(resp.Results, result)
		switch result.Status {
		case "uploaded":
			resp.Summary.Uploaded++
		case "skipped":
			resp.Summary.Skipped++
		case "rejected":
			resp.Summary.Rejected++
		case "failed":
			resp.Summary.Failed++
		}
	}
	resp.Summary.Total = len(inputs)
	return resp, nil
}

func (s *FolderUploadService) processOne(
	ctx context.Context,
	companyID, projectID, userID string,
	rootFolderID *string,
	inp BatchUploadInput,
	folderCache map[string]*string,
) dto.FolderUploadResult {
	result := dto.FolderUploadResult{RelativePath: inp.RelativePath}

	// System file check.
	if IsSystemFile(filepath.Base(inp.Header.Filename)) {
		result.Status = "skipped"
		result.Error = "sistemska datoteka preskočena"
		return result
	}

	// Path validation.
	normPath, err := NormalizeUploadPath(inp.RelativePath)
	if err != nil {
		result.Status = "rejected"
		result.Error = err.Error()
		return result
	}
	result.RelativePath = normPath

	// File validation (MIME + size).
	if err := ValidateDocumentFile(inp.Header); err != nil {
		result.Status = "rejected"
		result.Error = err.Error()
		return result
	}

	// Split path into folder segments + sanitized filename.
	parts := strings.Split(normPath, "/")
	filename := sanitizeDocumentFilename(parts[len(parts)-1])
	folderSegments := parts[:len(parts)-1]

	// Resolve (and create if needed) all folders in the path.
	folderID, err := s.resolveFolderPath(ctx, companyID, projectID, userID, rootFolderID, folderSegments, folderCache)
	if err != nil {
		result.Status = "failed"
		result.Error = "greška pri kreiranju mapa: " + err.Error()
		return result
	}

	// Duplicate check.
	isDup, err := s.docRepo.CheckDuplicate(ctx, companyID, projectID, folderID, filename)
	if err != nil {
		result.Status = "failed"
		result.Error = "greška pri provjeri duplikata"
		return result
	}
	if isDup {
		result.Status = "skipped"
		result.Error = "datoteka s istim nazivom već postoji"
		return result
	}

	// Upload to storage.
	fileKey, contentType, fileSize, err := s.storage.SaveDocumentFile(ctx, inp.File, inp.Header, companyID, projectID)
	if err != nil {
		result.Status = "failed"
		result.Error = "greška pri pohrani datoteke"
		return result
	}

	// Insert document record.
	docID, err := s.docRepo.CreateWithFolder(ctx, companyID, projectID, userID, fileKey, filename, contentType, fileSize, folderID)
	if err != nil {
		// Orphan cleanup: storage succeeded but DB failed — remove the file.
		_ = s.storage.DeleteReceiptFile(ctx, fileKey)
		result.Status = "failed"
		result.Error = "greška pri upisu zapisa"
		return result
	}

	result.Status = "uploaded"
	result.DocumentID = &docID
	return result
}

// resolveFolderPath walks segments creating folders as needed, caching results.
func (s *FolderUploadService) resolveFolderPath(
	ctx context.Context,
	companyID, projectID, userID string,
	root *string,
	segments []string,
	cache map[string]*string,
) (*string, error) {
	if len(segments) == 0 {
		return root, nil
	}

	rootKey := ""
	if root != nil {
		rootKey = *root
	}

	current := root
	cacheKey := rootKey

	for _, seg := range segments {
		cacheKey += "/" + seg
		if cached, ok := cache[cacheKey]; ok {
			current = cached
			continue
		}

		tx, err := s.db.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin tx: %w", err)
		}
		folderID, err := s.folderRepo.FindOrCreate(ctx, tx, companyID, projectID, current, seg, userID)
		if err != nil {
			tx.Rollback(ctx)
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit tx: %w", err)
		}
		id := folderID
		cache[cacheKey] = &id
		current = &id
	}
	return current, nil
}
