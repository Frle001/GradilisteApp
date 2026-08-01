package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Fake in-memory storage ────────────────────────────────────────────────────

type fakeAttachStorage struct {
	files     map[string][]byte
	saveFn    func(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, reportID string) (string, string, int64, error)
	deleteFn  func(ctx context.Context, fileKey string) error
	deleteLog []string
}

func newFakeStorage() *fakeAttachStorage {
	s := &fakeAttachStorage{files: make(map[string][]byte)}
	s.saveFn = func(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, reportID string) (string, string, int64, error) {
		data, _ := io.ReadAll(file)
		key := "report-photos/" + companyID + "/" + reportID + "/fakehex.jpg"
		s.files[key] = data
		return key, "image/jpeg", int64(len(data)), nil
	}
	s.deleteFn = func(ctx context.Context, fileKey string) error {
		s.deleteLog = append(s.deleteLog, fileKey)
		delete(s.files, fileKey)
		return nil
	}
	return s
}

func (s *fakeAttachStorage) SaveReportPhotoFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, reportID string) (string, string, int64, error) {
	return s.saveFn(ctx, file, header, companyID, reportID)
}
func (s *fakeAttachStorage) ReadReceiptFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error) {
	data, ok := s.files[fileKey]
	if !ok {
		return nil, "", errors.New("not found in fake storage")
	}
	return io.NopCloser(bytes.NewReader(data)), "image/jpeg", nil
}
func (s *fakeAttachStorage) DeleteReceiptFile(ctx context.Context, fileKey string) error {
	return s.deleteFn(ctx, fileKey)
}

// ── Mock attachment repository ────────────────────────────────────────────────

type mockAttachRepo struct {
	belongsFn    func(ctx context.Context, companyID, reportID string) (bool, error)
	getMetaFn    func(ctx context.Context, companyID, reportID string) (string, string, error)
	countFn      func(ctx context.Context, companyID, reportID string) (int, error)
	listFn       func(ctx context.Context, companyID, reportID string) ([]dto.DailyReportAttachment, error)
	createFn     func(ctx context.Context, companyID, reportID, userID, fileKey, originalName, contentType string, fileSize int64) (string, error)
	getByIDFn    func(ctx context.Context, companyID, reportID, attachmentID string) (*repositories.DailyReportAttachmentRecord, error)
	softDeleteFn func(ctx context.Context, companyID, reportID, attachmentID string) error
	createCalls  int
}

func (m *mockAttachRepo) ReportBelongsToCompany(ctx context.Context, companyID, reportID string) (bool, error) {
	if m.belongsFn != nil {
		return m.belongsFn(ctx, companyID, reportID)
	}
	return true, nil
}
func (m *mockAttachRepo) GetReportMeta(ctx context.Context, companyID, reportID string) (string, string, error) {
	if m.getMetaFn != nil {
		return m.getMetaFn(ctx, companyID, reportID)
	}
	return "emp-poslovoda", "submitted", nil
}
func (m *mockAttachRepo) CountActive(ctx context.Context, companyID, reportID string) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx, companyID, reportID)
	}
	return 0, nil
}
func (m *mockAttachRepo) List(ctx context.Context, companyID, reportID string) ([]dto.DailyReportAttachment, error) {
	if m.listFn != nil {
		return m.listFn(ctx, companyID, reportID)
	}
	return []dto.DailyReportAttachment{}, nil
}
func (m *mockAttachRepo) Create(ctx context.Context, companyID, reportID, userID, fileKey, originalName, contentType string, fileSize int64) (string, error) {
	m.createCalls++
	if m.createFn != nil {
		return m.createFn(ctx, companyID, reportID, userID, fileKey, originalName, contentType, fileSize)
	}
	return "attach-id", nil
}
func (m *mockAttachRepo) GetByID(ctx context.Context, companyID, reportID, attachmentID string) (*repositories.DailyReportAttachmentRecord, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, companyID, reportID, attachmentID)
	}
	return &repositories.DailyReportAttachmentRecord{
		ID: attachmentID, FileKey: "report-photos/co-1/report-1/fakehex.jpg",
		OriginalName: "photo.jpg", ContentType: "image/jpeg", FileSize: 1024,
	}, nil
}
func (m *mockAttachRepo) SoftDelete(ctx context.Context, companyID, reportID, attachmentID string) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, companyID, reportID, attachmentID)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newAttachSvc(repo *mockAttachRepo, storage attachStorageIface) *DailyReportAttachmentsService {
	return &DailyReportAttachmentsService{attachRepo: repo, storage: storage}
}

type fakeFile struct{ *bytes.Reader }

func (f *fakeFile) Close() error { return nil }

func newFakePhoto(content []byte) multipart.File {
	return &fakeFile{bytes.NewReader(content)}
}

func makePhotoHeader(mime string, sizeBytes int64) *multipart.FileHeader {
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", mime)
	return &multipart.FileHeader{Filename: "photo.jpg", Header: h, Size: sizeBytes}
}

// ── Upload tests ──────────────────────────────────────────────────────────────

func TestAttachment_Upload_RejectsHEIC(t *testing.T) {
	svc := newAttachSvc(&mockAttachRepo{}, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/heic", 1000))
	if err == nil {
		t.Fatal("expected error for HEIC, got nil")
	}
	if !strings.Contains(err.Error(), "HEIC") {
		t.Errorf("expected HEIC in error, got: %v", err)
	}
}

func TestAttachment_Upload_RejectsOversizedFile(t *testing.T) {
	svc := newAttachSvc(&mockAttachRepo{}, newFakeStorage())
	oversize := int64(MaxReceiptFileSize + 1)
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", oversize))
	if err == nil {
		t.Fatal("expected error for oversized file, got nil")
	}
	if !strings.Contains(err.Error(), "prevelika") {
		t.Errorf("expected 'prevelika' in error, got: %v", err)
	}
}

func TestAttachment_Upload_RejectsUnsupportedMIME(t *testing.T) {
	svc := newAttachSvc(&mockAttachRepo{}, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("application/pdf", 1000))
	if err == nil {
		t.Fatal("expected error for unsupported MIME, got nil")
	}
	if !strings.Contains(err.Error(), "nepodržani") {
		t.Errorf("expected 'nepodržani' in error, got: %v", err)
	}
}

// TestAttachment_Upload_100thPhotoAccepted verifies the 100th upload succeeds
// (count=99 active, limit is 100).
func TestAttachment_Upload_100thPhotoAccepted(t *testing.T) {
	repo := &mockAttachRepo{
		countFn: func(_ context.Context, _, _ string) (int, error) { return 99, nil },
	}
	svc := newAttachSvc(repo, newFakeStorage())
	att, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if err != nil {
		t.Fatalf("100th photo should be accepted, got: %v", err)
	}
	if att == nil {
		t.Fatal("expected non-nil attachment for 100th photo")
	}
}

// TestAttachment_Upload_101stPhotoRejected verifies the 101st upload is rejected
// with ErrTooManyAttachments (count=100 active, limit is 100).
func TestAttachment_Upload_101stPhotoRejected(t *testing.T) {
	repo := &mockAttachRepo{
		countFn: func(_ context.Context, _, _ string) (int, error) { return 100, nil },
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if !errors.Is(err, ErrTooManyAttachments) {
		t.Errorf("expected ErrTooManyAttachments for 101st photo, got: %v", err)
	}
}

// TestAttachment_Upload_ErrorMessageContainsLimit verifies the ErrTooManyAttachments
// message mentions the 100-photo limit in Croatian.
func TestAttachment_Upload_ErrorMessageContainsLimit(t *testing.T) {
	repo := &mockAttachRepo{
		countFn: func(_ context.Context, _, _ string) (int, error) { return 100, nil },
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "100") {
		t.Errorf("error message should mention 100, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "fotografija") {
		t.Errorf("error message should contain 'fotografija', got: %s", err.Error())
	}
}

// TestAttachment_Upload_SoftDeletedNotCounted verifies that soft-deleted attachments
// are not counted toward the limit. CountActive returns 5 (95 were soft-deleted);
// the upload must succeed.
func TestAttachment_Upload_SoftDeletedNotCounted(t *testing.T) {
	repo := &mockAttachRepo{
		countFn: func(_ context.Context, _, _ string) (int, error) { return 5, nil },
	}
	svc := newAttachSvc(repo, newFakeStorage())
	att, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if err != nil {
		t.Fatalf("upload must succeed when only 5 active attachments exist (soft-deleted excluded), got: %v", err)
	}
	if att == nil {
		t.Fatal("expected non-nil attachment")
	}
}

func TestAttachment_Upload_CrossCompanyRejected(t *testing.T) {
	repo := &mockAttachRepo{
		getMetaFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", repositories.ErrDailyReportNotFound
		},
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-other", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound for cross-company upload, got: %v", err)
	}
}

func TestAttachment_Upload_ApprovedReportRejected(t *testing.T) {
	repo := &mockAttachRepo{
		getMetaFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "emp-poslovoda", "approved", nil
		},
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if !errors.Is(err, ErrDailyReportNotEditable) {
		t.Errorf("expected ErrDailyReportNotEditable for approved report, got: %v", err)
	}
}

func TestAttachment_Upload_StorageFailure_NoDatabaseRecord(t *testing.T) {
	storage := newFakeStorage()
	storage.saveFn = func(_ context.Context, _ multipart.File, _ *multipart.FileHeader, _, _ string) (string, string, int64, error) {
		return "", "", 0, errors.New("storage unavailable")
	}
	repo := &mockAttachRepo{}
	svc := newAttachSvc(repo, storage)

	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if err == nil {
		t.Fatal("expected storage error, got nil")
	}
	if repo.createCalls != 0 {
		t.Errorf("Create must not be called when storage fails, called %d times", repo.createCalls)
	}
}

func TestAttachment_Upload_DatabaseFailure_CleansUpStorageObject(t *testing.T) {
	storage := newFakeStorage()
	savedKey := ""
	storage.saveFn = func(_ context.Context, _ multipart.File, _ *multipart.FileHeader, companyID, reportID string) (string, string, int64, error) {
		savedKey = "report-photos/" + companyID + "/" + reportID + "/fakehex.jpg"
		storage.files[savedKey] = []byte("img")
		return savedKey, "image/jpeg", 3, nil
	}
	repo := &mockAttachRepo{
		createFn: func(_ context.Context, _, _, _, _, _, _ string, _ int64) (string, error) {
			return "", errors.New("db error")
		},
	}
	svc := newAttachSvc(repo, storage)

	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if err == nil {
		t.Fatal("expected DB error, got nil")
	}
	if len(storage.deleteLog) == 0 {
		t.Error("expected storage.DeleteReceiptFile to be called for orphan cleanup")
	}
	if storage.deleteLog[0] != savedKey {
		t.Errorf("expected cleanup of key %q, got %q", savedKey, storage.deleteLog[0])
	}
}

func TestAttachment_Upload_Success(t *testing.T) {
	svc := newAttachSvc(&mockAttachRepo{}, newFakeStorage())
	att, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "direktor", "", newFakePhoto([]byte("imgdata")), makePhotoHeader("image/jpeg", 1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if att == nil {
		t.Fatal("expected non-nil attachment")
	}
	if att.ID == "" {
		t.Error("expected non-empty attachment ID")
	}
	if att.ContentType != "image/jpeg" {
		t.Errorf("expected content_type image/jpeg, got %q", att.ContentType)
	}
}

func TestAttachment_Upload_StorageKeyContainsReportID(t *testing.T) {
	capturedKey := ""
	storage := newFakeStorage()
	storage.saveFn = func(_ context.Context, _ multipart.File, _ *multipart.FileHeader, companyID, reportID string) (string, string, int64, error) {
		capturedKey = "report-photos/" + companyID + "/" + reportID + "/fakehex.jpg"
		storage.files[capturedKey] = []byte("img")
		return capturedKey, "image/jpeg", 3, nil
	}
	svc := newAttachSvc(&mockAttachRepo{}, storage)
	_, err := svc.Upload(context.Background(), "co-1", "report-xyz", "user-1", "direktor", "", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedKey, "report-xyz") {
		t.Errorf("storage key %q does not contain reportID 'report-xyz'", capturedKey)
	}
}

func TestAttachment_Upload_PoslovodaCanUploadToOwnReport(t *testing.T) {
	repo := &mockAttachRepo{
		getMetaFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "emp-poslovoda", "submitted", nil
		},
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "poslovoda", "emp-poslovoda", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if err != nil {
		t.Fatalf("poslovoda should be able to upload to their own report, got: %v", err)
	}
}

func TestAttachment_Upload_PoslovodaCannotUploadToOtherReport(t *testing.T) {
	repo := &mockAttachRepo{
		getMetaFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "emp-other-poslovoda", "submitted", nil
		},
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Upload(context.Background(), "co-1", "report-1", "user-1", "poslovoda", "emp-poslovoda", newFakePhoto([]byte("img")), makePhotoHeader("image/jpeg", 1000))
	if !errors.Is(err, ErrAttachmentForbidden) {
		t.Errorf("expected ErrAttachmentForbidden for poslovoda accessing another's report, got: %v", err)
	}
}

// ── List tests ────────────────────────────────────────────────────────────────

func TestAttachment_List_Success(t *testing.T) {
	now := time.Now()
	repo := &mockAttachRepo{
		listFn: func(_ context.Context, _, _ string) ([]dto.DailyReportAttachment, error) {
			return []dto.DailyReportAttachment{
				{ID: "att-1", OriginalName: "a.jpg", ContentType: "image/jpeg", FileSize: 100, CreatedAt: now},
				{ID: "att-2", OriginalName: "b.png", ContentType: "image/png", FileSize: 200, CreatedAt: now},
			}, nil
		},
	}
	svc := newAttachSvc(repo, newFakeStorage())
	atts, err := svc.List(context.Background(), "co-1", "report-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(atts))
	}
}

func TestAttachment_List_CrossCompanyRejected(t *testing.T) {
	repo := &mockAttachRepo{
		belongsFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.List(context.Background(), "co-other", "report-1")
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound for cross-company list, got: %v", err)
	}
}

// ── Download tests ────────────────────────────────────────────────────────────

func TestAttachment_Download_Success(t *testing.T) {
	storage := newFakeStorage()
	storage.files["report-photos/co-1/report-1/fakehex.jpg"] = []byte("imgdata")
	repo := &mockAttachRepo{
		getByIDFn: func(_ context.Context, _, _, _ string) (*repositories.DailyReportAttachmentRecord, error) {
			return &repositories.DailyReportAttachmentRecord{
				ID: "att-1", FileKey: "report-photos/co-1/report-1/fakehex.jpg",
				OriginalName: "photo.jpg", ContentType: "image/jpeg", FileSize: 7,
			}, nil
		},
	}
	svc := newAttachSvc(repo, storage)
	dl, err := svc.Download(context.Background(), "co-1", "report-1", "att-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer dl.Reader.Close()
	if dl.ContentType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %q", dl.ContentType)
	}
}

func TestAttachment_Download_NotFound(t *testing.T) {
	repo := &mockAttachRepo{
		getByIDFn: func(_ context.Context, _, _, _ string) (*repositories.DailyReportAttachmentRecord, error) {
			return nil, repositories.ErrAttachmentNotFound
		},
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Download(context.Background(), "co-1", "report-1", "att-missing")
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound, got: %v", err)
	}
}

func TestAttachment_Download_CrossCompanyRejected(t *testing.T) {
	repo := &mockAttachRepo{
		belongsFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := newAttachSvc(repo, newFakeStorage())
	_, err := svc.Download(context.Background(), "co-other", "report-1", "att-1")
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound for cross-company download, got: %v", err)
	}
}

// ── Delete tests ──────────────────────────────────────────────────────────────

func TestAttachment_Delete_Success(t *testing.T) {
	storage := newFakeStorage()
	storage.files["report-photos/co-1/report-1/fakehex.jpg"] = []byte("imgdata")
	svc := newAttachSvc(&mockAttachRepo{}, storage)
	err := svc.Delete(context.Background(), "co-1", "report-1", "att-1", "direktor", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(storage.deleteLog) == 0 {
		t.Error("expected storage.DeleteReceiptFile to be called")
	}
}

func TestAttachment_Delete_StorageDeleteFailureLogsButDoesNotError(t *testing.T) {
	storage := newFakeStorage()
	storage.deleteFn = func(_ context.Context, fileKey string) error {
		storage.deleteLog = append(storage.deleteLog, fileKey)
		return errors.New("storage unavailable")
	}
	svc := newAttachSvc(&mockAttachRepo{}, storage)
	err := svc.Delete(context.Background(), "co-1", "report-1", "att-1", "direktor", "")
	if err != nil {
		t.Errorf("Delete should return nil even if storage delete fails, got: %v", err)
	}
	if len(storage.deleteLog) == 0 {
		t.Error("expected DeleteReceiptFile to be attempted")
	}
}

func TestAttachment_Delete_CrossCompanyRejected(t *testing.T) {
	repo := &mockAttachRepo{
		getMetaFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", repositories.ErrDailyReportNotFound
		},
	}
	svc := newAttachSvc(repo, newFakeStorage())
	err := svc.Delete(context.Background(), "co-other", "report-1", "att-1", "direktor", "")
	if !errors.Is(err, ErrAttachmentNotFound) {
		t.Errorf("expected ErrAttachmentNotFound for cross-company delete, got: %v", err)
	}
}
