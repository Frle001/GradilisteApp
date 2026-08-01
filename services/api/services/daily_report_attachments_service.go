package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

const maxReportAttachments = 100

var ErrAttachmentNotFound = errors.New("attachment not found")
var ErrAttachmentForbidden = errors.New("access denied to this attachment")
var ErrTooManyAttachments = fmt.Errorf("Možete dodati najviše %d fotografija.", maxReportAttachments)

// drAttachRepoIface is the repository interface used by DailyReportAttachmentsService.
type drAttachRepoIface interface {
	ReportBelongsToCompany(ctx context.Context, companyID, reportID string) (bool, error)
	GetReportMeta(ctx context.Context, companyID, reportID string) (poslovodaEmpID, status string, err error)
	CountActive(ctx context.Context, companyID, reportID string) (int, error)
	List(ctx context.Context, companyID, reportID string) ([]dto.DailyReportAttachment, error)
	Create(ctx context.Context, companyID, reportID, userID, fileKey, originalName, contentType string, fileSize int64) (string, error)
	GetByID(ctx context.Context, companyID, reportID, attachmentID string) (*repositories.DailyReportAttachmentRecord, error)
	SoftDelete(ctx context.Context, companyID, reportID, attachmentID string) error
}

// attachStorageIface is the minimal storage subset used by DailyReportAttachmentsService.
// StorageService is a superset, so any StorageService value satisfies this interface.
type attachStorageIface interface {
	SaveReportPhotoFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, reportID string) (fileKey, contentType string, fileSize int64, err error)
	ReadReceiptFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error)
	DeleteReceiptFile(ctx context.Context, fileKey string) error
}

// AttachmentDownload bundles the file stream and metadata for streaming to the client.
type AttachmentDownload struct {
	Reader      io.ReadCloser
	ContentType string
	OrigName    string
	FileSize    int64
}

type DailyReportAttachmentsService struct {
	attachRepo drAttachRepoIface
	storage    attachStorageIface
}

func NewDailyReportAttachmentsService(
	attachRepo *repositories.DailyReportAttachmentsRepository,
	storage StorageService,
) *DailyReportAttachmentsService {
	return &DailyReportAttachmentsService{attachRepo: attachRepo, storage: storage}
}

// List returns all active attachments for a report. Company isolation enforced.
func (s *DailyReportAttachmentsService) List(ctx context.Context, companyID, reportID string) ([]dto.DailyReportAttachment, error) {
	ok, err := s.attachRepo.ReportBelongsToCompany(ctx, companyID, reportID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAttachmentNotFound
	}
	return s.attachRepo.List(ctx, companyID, reportID)
}

// Upload validates and stores a photo, then creates the DB record.
// If storage write succeeds but DB insert fails, the orphaned object is deleted from storage.
func (s *DailyReportAttachmentsService) Upload(
	ctx context.Context,
	companyID, reportID, callerUserID, callerRole, callerEmpID string,
	file multipart.File,
	header *multipart.FileHeader,
) (*dto.DailyReportAttachment, error) {
	poslovodaEmpID, status, err := s.attachRepo.GetReportMeta(ctx, companyID, reportID)
	if err != nil {
		if errors.Is(err, repositories.ErrDailyReportNotFound) {
			return nil, ErrAttachmentNotFound
		}
		return nil, err
	}

	if err := s.checkWritePermission(callerRole, callerEmpID, poslovodaEmpID, status); err != nil {
		return nil, err
	}

	if err := ValidateReportPhotoFile(header); err != nil {
		return nil, err
	}

	count, err := s.attachRepo.CountActive(ctx, companyID, reportID)
	if err != nil {
		return nil, err
	}
	if count >= maxReportAttachments {
		return nil, ErrTooManyAttachments
	}

	origName := sanitizeDocumentFilename(header.Filename)

	fileKey, contentType, fileSize, err := s.storage.SaveReportPhotoFile(ctx, file, header, companyID, reportID)
	if err != nil {
		return nil, fmt.Errorf("save photo: %w", err)
	}

	attachID, err := s.attachRepo.Create(ctx, companyID, reportID, callerUserID, fileKey, origName, contentType, fileSize)
	if err != nil {
		if delErr := s.storage.DeleteReceiptFile(ctx, fileKey); delErr != nil {
			log.Printf("[report_attachments] orphan cleanup failed for key %q: %v", fileKey, delErr)
		}
		return nil, fmt.Errorf("create attachment record: %w", err)
	}

	return &dto.DailyReportAttachment{
		ID:           attachID,
		OriginalName: origName,
		ContentType:  contentType,
		FileSize:     fileSize,
	}, nil
}

// Download returns a readable stream for an attachment. Company isolation enforced.
func (s *DailyReportAttachmentsService) Download(
	ctx context.Context,
	companyID, reportID, attachmentID string,
) (*AttachmentDownload, error) {
	ok, err := s.attachRepo.ReportBelongsToCompany(ctx, companyID, reportID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAttachmentNotFound
	}

	rec, err := s.attachRepo.GetByID(ctx, companyID, reportID, attachmentID)
	if err != nil {
		return nil, ErrAttachmentNotFound
	}

	reader, _, err := s.storage.ReadReceiptFile(ctx, rec.FileKey)
	if err != nil {
		return nil, fmt.Errorf("read attachment file: %w", err)
	}

	return &AttachmentDownload{
		Reader:      reader,
		ContentType: rec.ContentType,
		OrigName:    rec.OriginalName,
		FileSize:    rec.FileSize,
	}, nil
}

// Delete soft-deletes the DB record. If the subsequent storage delete fails, it is
// logged but does NOT cause Delete to return an error — the record is already hidden
// from users and can be cleaned up separately.
func (s *DailyReportAttachmentsService) Delete(
	ctx context.Context,
	companyID, reportID, attachmentID, callerRole, callerEmpID string,
) error {
	poslovodaEmpID, status, err := s.attachRepo.GetReportMeta(ctx, companyID, reportID)
	if err != nil {
		if errors.Is(err, repositories.ErrDailyReportNotFound) {
			return ErrAttachmentNotFound
		}
		return err
	}

	if err := s.checkWritePermission(callerRole, callerEmpID, poslovodaEmpID, status); err != nil {
		return err
	}

	rec, err := s.attachRepo.GetByID(ctx, companyID, reportID, attachmentID)
	if err != nil {
		return ErrAttachmentNotFound
	}

	if err := s.attachRepo.SoftDelete(ctx, companyID, reportID, attachmentID); err != nil {
		if errors.Is(err, repositories.ErrAttachmentNotFound) {
			return ErrAttachmentNotFound
		}
		return err
	}

	if err := s.storage.DeleteReceiptFile(ctx, rec.FileKey); err != nil {
		log.Printf("[report_attachments] storage delete failed for key %q (DB soft-delete already succeeded): %v", rec.FileKey, err)
	}

	return nil
}

// checkWritePermission enforces that approved reports cannot be modified and that a
// poslovoda can only modify their own report.
func (s *DailyReportAttachmentsService) checkWritePermission(callerRole, callerEmpID, poslovodaEmpID, status string) error {
	if status == "approved" {
		return ErrDailyReportNotEditable
	}
	if callerRole == "poslovoda" && callerEmpID != poslovodaEmpID {
		return ErrAttachmentForbidden
	}
	return nil
}
