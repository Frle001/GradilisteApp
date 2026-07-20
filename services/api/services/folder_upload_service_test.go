package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func makeFileHeader(name string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{Filename: name, Size: size}
}

// makeFileHeaderWithCT creates a FileHeader with an explicit Content-Type so
// ValidateDocumentFile accepts it without a live multipart parse.
func makeFileHeaderWithCT(name, contentType string, size int64) *multipart.FileHeader {
	h := &multipart.FileHeader{Filename: name, Size: size}
	h.Header = map[string][]string{"Content-Type": {contentType}}
	return h
}

func flatInputPDF(filename string) BatchUploadInput {
	return BatchUploadInput{
		File:         mockFile{},
		Header:       makeFileHeaderWithCT(filename, "application/pdf", 100),
		RelativePath: filename,
	}
}

// ── Mocks ─────────────────────────────────────────────────────────────────────

// mockTx is a minimal pgx.Tx for tests. Only Commit and Rollback are called by
// FolderUploadService; the embedded nil interface panics on any other method.
type mockTx struct {
	pgx.Tx
	committed bool
	commitErr error
}

func (m *mockTx) Commit(_ context.Context) error   { m.committed = true; return m.commitErr }
func (m *mockTx) Rollback(_ context.Context) error { return nil }

// mockTxBeginner implements txBeginner for delete tests.
type mockTxBeginner struct {
	tx  pgx.Tx
	err error
}

func (m *mockTxBeginner) Begin(_ context.Context) (pgx.Tx, error) { return m.tx, m.err }

// mockDocRepo implements uploadDocRepo for tests.
type mockDocRepo struct {
	belongsFn      func(ctx context.Context, companyID, projectID string) (bool, error)
	dupFn          func(ctx context.Context, companyID, projectID string, folderID *string, name string) (bool, error)
	createFn       func(ctx context.Context, companyID, projectID, userID, fileKey, name, ct string, size int64, folderID *string) (string, error)
	hasActiveDocFn func(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) (bool, error)
}

func (m *mockDocRepo) ProjectBelongsToCompany(ctx context.Context, companyID, projectID string) (bool, error) {
	if m.belongsFn != nil {
		return m.belongsFn(ctx, companyID, projectID)
	}
	return true, nil
}
func (m *mockDocRepo) CheckDuplicate(ctx context.Context, companyID, projectID string, folderID *string, name string) (bool, error) {
	if m.dupFn != nil {
		return m.dupFn(ctx, companyID, projectID, folderID, name)
	}
	return false, nil
}
func (m *mockDocRepo) CreateWithFolder(ctx context.Context, companyID, projectID, userID, fileKey, name, ct string, size int64, folderID *string) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, companyID, projectID, userID, fileKey, name, ct, size, folderID)
	}
	return "doc-id-1", nil
}
func (m *mockDocRepo) HasActiveDocuments(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) (bool, error) {
	if m.hasActiveDocFn != nil {
		return m.hasActiveDocFn(ctx, tx, companyID, projectID, folderID)
	}
	return false, nil
}

// mockFolderRepo implements uploadFolderRepo for tests.
type mockFolderRepo struct {
	belongsFn            func(ctx context.Context, companyID, projectID, folderID string) (bool, error)
	findOrCreateFn       func(ctx context.Context, tx pgx.Tx, companyID, projectID string, parentID *string, name, createdBy string) (string, error)
	getFolderForUpdateFn func(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) (bool, error)
	hasChildFoldersFn    func(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) (bool, error)
	deleteFolderFn       func(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) error
	deletedFolderIDs     []string
}

func (m *mockFolderRepo) BelongsToProject(ctx context.Context, companyID, projectID, folderID string) (bool, error) {
	if m.belongsFn != nil {
		return m.belongsFn(ctx, companyID, projectID, folderID)
	}
	return true, nil
}
func (m *mockFolderRepo) FindOrCreate(ctx context.Context, tx pgx.Tx, companyID, projectID string, parentID *string, name, createdBy string) (string, error) {
	if m.findOrCreateFn != nil {
		return m.findOrCreateFn(ctx, tx, companyID, projectID, parentID, name, createdBy)
	}
	return "folder-id-1", nil
}
func (m *mockFolderRepo) GetFolderForUpdate(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) (bool, error) {
	if m.getFolderForUpdateFn != nil {
		return m.getFolderForUpdateFn(ctx, tx, companyID, projectID, folderID)
	}
	return true, nil
}
func (m *mockFolderRepo) HasChildFolders(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) (bool, error) {
	if m.hasChildFoldersFn != nil {
		return m.hasChildFoldersFn(ctx, tx, companyID, projectID, folderID)
	}
	return false, nil
}
func (m *mockFolderRepo) DeleteFolder(ctx context.Context, tx pgx.Tx, companyID, projectID, folderID string) error {
	m.deletedFolderIDs = append(m.deletedFolderIDs, folderID)
	if m.deleteFolderFn != nil {
		return m.deleteFolderFn(ctx, tx, companyID, projectID, folderID)
	}
	return nil
}

// mockStorage implements StorageService for tests.
type mockStorage struct {
	saveFn   func() (string, string, int64, error)
	deleteFn func(key string) error

	savedKeys   []string
	deletedKeys []string
}

func (m *mockStorage) SaveDocumentFile(_ context.Context, _ multipart.File, h *multipart.FileHeader, _, _ string) (string, string, int64, error) {
	if m.saveFn != nil {
		key, ct, size, err := m.saveFn()
		if err == nil {
			m.savedKeys = append(m.savedKeys, key)
		}
		return key, ct, size, err
	}
	key := "key/" + h.Filename
	m.savedKeys = append(m.savedKeys, key)
	return key, "application/pdf", h.Size, nil
}
func (m *mockStorage) DeleteReceiptFile(_ context.Context, key string) error {
	m.deletedKeys = append(m.deletedKeys, key)
	if m.deleteFn != nil {
		return m.deleteFn(key)
	}
	return nil
}
func (m *mockStorage) SaveReceiptFile(_ context.Context, _ multipart.File, _ *multipart.FileHeader, _, _ string) (string, string, error) {
	return "", "", nil
}
func (m *mockStorage) ReadReceiptFile(_ context.Context, _ string) (io.ReadCloser, string, error) {
	return nil, "", errors.New("not implemented")
}
func (m *mockStorage) CheckHealth(_ context.Context) error { return nil }

// mockFile is a minimal multipart.File that returns empty reads.
type mockFile struct{}

func (mockFile) Read(_ []byte) (int, error)            { return 0, nil }
func (mockFile) ReadAt(_ []byte, _ int64) (int, error) { return 0, nil }
func (mockFile) Seek(_ int64, _ int) (int64, error)    { return 0, nil }
func (mockFile) Close() error                          { return nil }

func flatInput(filename string, size int64) BatchUploadInput {
	return BatchUploadInput{
		File:         mockFile{},
		Header:       makeFileHeaderWithCT(filename, "application/pdf", size),
		RelativePath: filename,
	}
}

func newTestSvc(docRepo uploadDocRepo, folderRepo uploadFolderRepo, storage StorageService) *FolderUploadService {
	return &FolderUploadService{
		db:         nil, // nil is safe for flat-file tests (no resolveFolderPath DB calls)
		docRepo:    docRepo,
		folderRepo: folderRepo,
		storage:    storage,
	}
}

// ── IsSystemFile ─────────────────────────────────────────────────────────────

func TestIsSystemFile_KnownFiles(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{".DS_Store", true},
		{".ds_store", true},
		{"Thumbs.db", true},
		{"thumbs.db", true},
		{"desktop.ini", true},
		{"DESKTOP.INI", true},
		{"._.DS_Store", true},
		{"report.pdf", false},
		{"drawing.dwg", false},
		{"", false},
		{"document.docx", false},
	}
	for _, tc := range cases {
		got := IsSystemFile(tc.name)
		if got != tc.want {
			t.Errorf("IsSystemFile(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// ── NormalizeUploadPath ───────────────────────────────────────────────────────

func TestNormalizeUploadPath_ValidPaths(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"report.pdf", "report.pdf"},
		{"docs/report.pdf", "docs/report.pdf"},
		{"a/b/c/file.txt", "a/b/c/file.txt"},
		{"docs\\sub\\file.pdf", "docs/sub/file.pdf"},
		{"docs/sub\\file.pdf", "docs/sub/file.pdf"},
	}
	for _, tc := range cases {
		got, err := NormalizeUploadPath(tc.raw)
		if err != nil {
			t.Errorf("NormalizeUploadPath(%q) unexpected error: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizeUploadPath(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestNormalizeUploadPath_Rejections(t *testing.T) {
	cases := []struct {
		raw    string
		reason string
	}{
		{"/etc/passwd", "absolute unix path"},
		{"C:/Windows/file.txt", "windows drive path"},
		{string([]byte{'a', 0, 'b'}), "null byte"},
		{"a//b", "empty segment"},
		{"./file.txt", "dot segment"},
		{"../file.txt", "dotdot segment"},
		{"a/../b", "dotdot segment mid-path"},
		{strings.Repeat("a/", MaxFolderDepth+1) + "file.txt", "depth exceeded"},
		{strings.Repeat("a", MaxPathLen+1), "path too long"},
	}
	for _, tc := range cases {
		_, err := NormalizeUploadPath(tc.raw)
		if err == nil {
			t.Errorf("NormalizeUploadPath(%q) should have rejected (%s)", tc.raw, tc.reason)
		}
	}
}

func TestNormalizeUploadPath_MaxDepthExactlyAllowed(t *testing.T) {
	segments := make([]string, MaxFolderDepth+1)
	for i := range segments {
		segments[i] = "d"
	}
	path := strings.Join(segments, "/")
	if _, err := NormalizeUploadPath(path); err != nil {
		t.Errorf("path at exactly MaxFolderDepth should be allowed, got: %v", err)
	}
}

func TestNormalizeUploadPath_DepthOneBeyondLimit(t *testing.T) {
	segments := make([]string, MaxFolderDepth+2)
	for i := range segments {
		segments[i] = "d"
	}
	path := strings.Join(segments, "/")
	if _, err := NormalizeUploadPath(path); err == nil {
		t.Errorf("path exceeding MaxFolderDepth should be rejected")
	}
}

// ── Batch-level limits (no DB) ────────────────────────────────────────────────

func TestProcess_TooManyFiles(t *testing.T) {
	svc := &FolderUploadService{}
	inputs := make([]BatchUploadInput, MaxBatchFiles+1)
	_, err := svc.Process(context.Background(), "company", "project", "user", nil, inputs)
	if err == nil {
		t.Fatal("expected error for too many files")
	}
	ve := AsValidationError(err)
	if ve == nil {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(ve.Message, "previše datoteka") {
		t.Errorf("unexpected message: %q", ve.Message)
	}
}

func TestProcess_TotalSizeTooLarge(t *testing.T) {
	svc := &FolderUploadService{}
	inputs := []BatchUploadInput{
		{Header: makeFileHeader("big.pdf", MaxBatchBytes+1)},
	}
	_, err := svc.Process(context.Background(), "company", "project", "user", nil, inputs)
	if err == nil {
		t.Fatal("expected error for oversized batch")
	}
	ve := AsValidationError(err)
	if ve == nil {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(ve.Message, "ukupna veličina") {
		t.Errorf("unexpected message: %q", ve.Message)
	}
}

// ── Ownership / access checks ─────────────────────────────────────────────────

func TestProcess_ProjectNotFound_ReturnsErrNotFound(t *testing.T) {
	docRepo := &mockDocRepo{
		belongsFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := newTestSvc(docRepo, &mockFolderRepo{}, &mockStorage{})
	_, err := svc.Process(context.Background(), "company-a", "project-x", "user", nil, []BatchUploadInput{flatInput("file.pdf", 100)})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProcess_RootFolderFromAnotherProject_Rejected(t *testing.T) {
	docRepo := &mockDocRepo{}
	folderRepo := &mockFolderRepo{
		belongsFn: func(_ context.Context, _, _, _ string) (bool, error) { return false, nil },
	}
	svc := newTestSvc(docRepo, folderRepo, &mockStorage{})
	foreignFolder := "folder-from-other-project"
	_, err := svc.Process(context.Background(), "company", "project", "user", &foreignFolder, []BatchUploadInput{flatInput("file.pdf", 100)})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProcess_RootFolderFromAnotherCompany_Rejected(t *testing.T) {
	docRepo := &mockDocRepo{}
	folderRepo := &mockFolderRepo{
		// BelongsToProject checks company_id+project_id in the query; returns false for wrong company.
		belongsFn: func(_ context.Context, companyID, _, _ string) (bool, error) {
			if companyID == "company-a" {
				return false, nil
			}
			return true, nil
		},
	}
	svc := newTestSvc(docRepo, folderRepo, &mockStorage{})
	fid := "folder-id"
	_, err := svc.Process(context.Background(), "company-a", "project", "user", &fid, []BatchUploadInput{flatInput("file.pdf", 100)})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ── Per-file processing ───────────────────────────────────────────────────────

func TestProcess_SystemFile_Skipped(t *testing.T) {
	svc := newTestSvc(&mockDocRepo{}, &mockFolderRepo{}, &mockStorage{})
	inp := BatchUploadInput{
		File:         mockFile{},
		Header:       makeFileHeaderWithCT(".DS_Store", "application/octet-stream", 100),
		RelativePath: ".DS_Store",
	}
	resp, err := svc.Process(context.Background(), "co", "proj", "user", nil, []BatchUploadInput{inp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Results[0].Status != "skipped" {
		t.Errorf("expected skipped, got %q", resp.Results[0].Status)
	}
	if resp.Summary.Skipped != 1 {
		t.Errorf("expected Skipped=1, got %d", resp.Summary.Skipped)
	}
}

func TestProcess_InvalidPath_Rejected(t *testing.T) {
	svc := newTestSvc(&mockDocRepo{}, &mockFolderRepo{}, &mockStorage{})
	inp := BatchUploadInput{
		File:         mockFile{},
		Header:       makeFileHeader("file.pdf", 100),
		RelativePath: "../etc/passwd",
	}
	resp, err := svc.Process(context.Background(), "co", "proj", "user", nil, []BatchUploadInput{inp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Results[0].Status != "rejected" {
		t.Errorf("expected rejected, got %q", resp.Results[0].Status)
	}
}

func TestProcess_DuplicateDocument_Skipped(t *testing.T) {
	docRepo := &mockDocRepo{
		dupFn: func(_ context.Context, _, _ string, _ *string, _ string) (bool, error) {
			return true, nil // always a duplicate
		},
	}
	svc := newTestSvc(docRepo, &mockFolderRepo{}, &mockStorage{})
	resp, err := svc.Process(context.Background(), "co", "proj", "user", nil, []BatchUploadInput{flatInput("report.pdf", 100)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Results[0].Status != "skipped" {
		t.Errorf("expected skipped for duplicate, got %q", resp.Results[0].Status)
	}
}

func TestProcess_StorageFailure_StatusFailed_NoDocumentID(t *testing.T) {
	storage := &mockStorage{
		saveFn: func() (string, string, int64, error) {
			return "", "", 0, fmt.Errorf("disk full")
		},
	}
	svc := newTestSvc(&mockDocRepo{}, &mockFolderRepo{}, storage)
	resp, err := svc.Process(context.Background(), "co", "proj", "user", nil, []BatchUploadInput{flatInput("report.pdf", 100)})
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	r := resp.Results[0]
	if r.Status != "failed" {
		t.Errorf("expected failed, got %q", r.Status)
	}
	if r.DocumentID != nil {
		t.Errorf("expected nil DocumentID on storage failure, got %v", *r.DocumentID)
	}
	// Storage never returned a key, so delete should NOT have been called.
	if len(storage.deletedKeys) != 0 {
		t.Errorf("DeleteReceiptFile should not be called on save failure, called with %v", storage.deletedKeys)
	}
}

func TestProcess_DBFailure_CleansStorageObject(t *testing.T) {
	const uploadedKey = "documents/co/proj/abc.pdf"
	storage := &mockStorage{
		saveFn: func() (string, string, int64, error) {
			return uploadedKey, "application/pdf", 100, nil
		},
	}
	docRepo := &mockDocRepo{
		createFn: func(_ context.Context, _, _, _, _, _, _ string, _ int64, _ *string) (string, error) {
			return "", fmt.Errorf("unique constraint violation")
		},
	}
	svc := newTestSvc(docRepo, &mockFolderRepo{}, storage)
	resp, err := svc.Process(context.Background(), "co", "proj", "user", nil, []BatchUploadInput{flatInput("report.pdf", 100)})
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if resp.Results[0].Status != "failed" {
		t.Errorf("expected failed, got %q", resp.Results[0].Status)
	}
	if len(storage.deletedKeys) != 1 || storage.deletedKeys[0] != uploadedKey {
		t.Errorf("expected DeleteReceiptFile(%q), got %v", uploadedKey, storage.deletedKeys)
	}
}

func TestProcess_SuccessfulUpload_SummaryCorrect(t *testing.T) {
	svc := newTestSvc(&mockDocRepo{}, &mockFolderRepo{}, &mockStorage{})
	inputs := []BatchUploadInput{
		flatInput("a.pdf", 100),
		{File: mockFile{}, Header: makeFileHeader(".DS_Store", 50), RelativePath: ".DS_Store"}, // skipped
		flatInput("b.pdf", 200),
	}
	resp, err := svc.Process(context.Background(), "co", "proj", "user", nil, inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Summary.Total != 3 {
		t.Errorf("Total want 3, got %d", resp.Summary.Total)
	}
	if resp.Summary.Uploaded != 2 {
		t.Errorf("Uploaded want 2, got %d", resp.Summary.Uploaded)
	}
	if resp.Summary.Skipped != 1 {
		t.Errorf("Skipped want 1, got %d", resp.Summary.Skipped)
	}
}

// ── DeleteEmptyFolder ─────────────────────────────────────────────────────────

func newDeleteSvc(docRepo uploadDocRepo, folderRepo uploadFolderRepo) *FolderUploadService {
	tx := &mockTx{}
	return &FolderUploadService{
		db:         &mockTxBeginner{tx: tx},
		docRepo:    docRepo,
		folderRepo: folderRepo,
		storage:    &mockStorage{},
	}
}

func TestDeleteEmptyFolder_Success(t *testing.T) {
	tx := &mockTx{}
	folderRepo := &mockFolderRepo{}
	svc := &FolderUploadService{
		db:         &mockTxBeginner{tx: tx},
		docRepo:    &mockDocRepo{},
		folderRepo: folderRepo,
		storage:    &mockStorage{},
	}
	err := svc.DeleteEmptyFolder(context.Background(), "co", "proj", "folder-1")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !tx.committed {
		t.Error("expected transaction to be committed")
	}
	if len(folderRepo.deletedFolderIDs) != 1 || folderRepo.deletedFolderIDs[0] != "folder-1" {
		t.Errorf("expected DeleteFolder(folder-1), got %v", folderRepo.deletedFolderIDs)
	}
}

func TestDeleteEmptyFolder_HasDocuments_Returns409(t *testing.T) {
	tx := &mockTx{}
	docRepo := &mockDocRepo{
		hasActiveDocFn: func(_ context.Context, _ pgx.Tx, _, _, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := &FolderUploadService{
		db:         &mockTxBeginner{tx: tx},
		docRepo:    docRepo,
		folderRepo: &mockFolderRepo{},
		storage:    &mockStorage{},
	}
	err := svc.DeleteEmptyFolder(context.Background(), "co", "proj", "folder-1")
	if !errors.Is(err, ErrFolderHasDocs) {
		t.Errorf("expected ErrFolderHasDocs, got %v", err)
	}
}

func TestDeleteEmptyFolder_HasChildFolders_Returns409(t *testing.T) {
	tx := &mockTx{}
	folderRepo := &mockFolderRepo{
		hasChildFoldersFn: func(_ context.Context, _ pgx.Tx, _, _, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := &FolderUploadService{
		db:         &mockTxBeginner{tx: tx},
		docRepo:    &mockDocRepo{},
		folderRepo: folderRepo,
		storage:    &mockStorage{},
	}
	err := svc.DeleteEmptyFolder(context.Background(), "co", "proj", "folder-1")
	if !errors.Is(err, ErrFolderHasChildren) {
		t.Errorf("expected ErrFolderHasChildren, got %v", err)
	}
}

func TestDeleteEmptyFolder_ForeignProject_ReturnsNotFound(t *testing.T) {
	folderRepo := &mockFolderRepo{
		belongsFn: func(_ context.Context, _, _, _ string) (bool, error) { return false, nil },
	}
	svc := newDeleteSvc(&mockDocRepo{}, folderRepo)
	err := svc.DeleteEmptyFolder(context.Background(), "co", "other-proj", "folder-1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEmptyFolder_ForeignCompany_ReturnsNotFound(t *testing.T) {
	folderRepo := &mockFolderRepo{
		belongsFn: func(_ context.Context, companyID, _, _ string) (bool, error) {
			return companyID == "correct-co", nil
		},
	}
	svc := newDeleteSvc(&mockDocRepo{}, folderRepo)
	err := svc.DeleteEmptyFolder(context.Background(), "wrong-co", "proj", "folder-1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEmptyFolder_LockedButGone_ReturnsNotFound(t *testing.T) {
	// BelongsToProject passes but GetFolderForUpdate finds nothing (race condition simulation).
	tx := &mockTx{}
	folderRepo := &mockFolderRepo{
		getFolderForUpdateFn: func(_ context.Context, _ pgx.Tx, _, _, _ string) (bool, error) {
			return false, nil
		},
	}
	svc := &FolderUploadService{
		db:         &mockTxBeginner{tx: tx},
		docRepo:    &mockDocRepo{},
		folderRepo: folderRepo,
		storage:    &mockStorage{},
	}
	err := svc.DeleteEmptyFolder(context.Background(), "co", "proj", "folder-1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEmptyFolder_OnlyTargetFolderDeleted(t *testing.T) {
	tx := &mockTx{}
	folderRepo := &mockFolderRepo{}
	svc := &FolderUploadService{
		db:         &mockTxBeginner{tx: tx},
		docRepo:    &mockDocRepo{},
		folderRepo: folderRepo,
		storage:    &mockStorage{},
	}
	_ = svc.DeleteEmptyFolder(context.Background(), "co", "proj", "target-folder")
	if len(folderRepo.deletedFolderIDs) != 1 {
		t.Fatalf("expected exactly 1 deletion, got %d", len(folderRepo.deletedFolderIDs))
	}
	if folderRepo.deletedFolderIDs[0] != "target-folder" {
		t.Errorf("expected target-folder to be deleted, got %q", folderRepo.deletedFolderIDs[0])
	}
}
