package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageService abstracts file storage. Swap local disk for S3/R2 without changing callers.
type StorageService interface {
	SaveReceiptFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, projectID string) (fileKey, originalFilename string, err error)
	ReadReceiptFile(ctx context.Context, fileKey string) (reader io.ReadCloser, contentType string, err error)
	DeleteReceiptFile(ctx context.Context, fileKey string) error
	SaveDocumentFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, projectID string) (fileKey, contentType string, fileSize int64, err error)
	SaveReportPhotoFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, reportID string) (fileKey, contentType string, fileSize int64, err error)
	// SaveEmployeeDocFile stores an HR/compliance document for an employee.
	// Key path: employee-docs/<companyID>/<employeeID>/<randomID><ext>
	SaveEmployeeDocFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, employeeID string) (fileKey, contentType string, fileSize int64, err error)
	// SaveFinanceDocFile stores a finance document (invoice or R1 receipt).
	// Key path: finance-docs/<companyID>/<subdir>/<randomID><ext>  (subdir: "racuni" or "r1")
	SaveFinanceDocFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, subdir string) (fileKey, contentType string, fileSize int64, err error)
	ReadFinanceDocFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error)
	ReadEmployeeDocFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error)
	CheckHealth(ctx context.Context) error
}

// AllowedReceiptMIMEs maps allowed MIME types to their canonical extension.
var AllowedReceiptMIMEs = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

const MaxReceiptFileSize = 10 << 20  // 10 MB
const MaxDocumentFileSize = 25 << 20 // 25 MB

// AllowedDocumentMIMEs maps allowed MIME types to their canonical extension for project documents.
var AllowedDocumentMIMEs = map[string]string{
	"application/pdf":    ".pdf",
	"application/msword": ".doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	"application/vnd.ms-excel": ".xls",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
	"application/vnd.ms-powerpoint":                                             ".ppt",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
	"image/jpeg":                   ".jpg",
	"image/png":                    ".png",
	"image/webp":                   ".webp",
	"image/gif":                    ".gif",
	"image/tiff":                   ".tiff",
	"text/plain":                   ".txt",
	"text/csv":                     ".csv",
	"application/zip":              ".zip",
	"application/x-zip-compressed": ".zip",
}

// dwgMIMEValues is the set of MIME types that browsers and tools may report for AutoCAD
// DWG files. Includes application/octet-stream because many browsers send it for
// unrecognised formats; it is only accepted when the filename ends in .dwg.
var dwgMIMEValues = map[string]bool{
	"application/acad":         true,
	"application/x-acad":       true,
	"application/autocad":      true,
	"application/x-autocad":    true,
	"application/dwg":          true,
	"application/x-dwg":        true,
	"image/vnd.dwg":            true,
	"application/octet-stream": true,
}

// canonicalDWGMIME is the content type stored in the DB and returned on download
// for all DWG files, regardless of what the browser reported.
const canonicalDWGMIME = "application/dwg"

// NormalizeDocumentMIME returns the canonical MIME for storage. For files whose
// sanitized name ends in .dwg it always returns canonicalDWGMIME so that
// downloads have a useful Content-Type even when the browser sent octet-stream.
func NormalizeDocumentMIME(ct, filename string) string {
	if strings.ToLower(filepath.Ext(filepath.Base(filename))) == ".dwg" {
		return canonicalDWGMIME
	}
	return ct
}

// ValidateDocumentFile checks MIME type and file size for project documents.
// DWG files are accepted when the filename ends in .dwg (case-insensitive) AND
// the MIME type is a known DWG value or application/octet-stream. All other files
// must have a MIME type listed in AllowedDocumentMIMEs.
func ValidateDocumentFile(header *multipart.FileHeader) error {
	if header.Size > MaxDocumentFileSize {
		return fmt.Errorf("datoteka je prevelika (maksimalno 25 MB, primljeno %d bajtova)", header.Size)
	}
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	ext := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	if ext == ".dwg" {
		if dwgMIMEValues[ct] {
			return nil
		}
		return fmt.Errorf("nepodržani tip datoteke %q za DWG datoteku; očekivani su DWG MIME tipovi ili application/octet-stream", ct)
	}

	if _, ok := AllowedDocumentMIMEs[ct]; !ok {
		return fmt.Errorf("nepodržani tip datoteke %q; dopušteni su: PDF, Word, Excel, PowerPoint, slike, CSV, TXT, ZIP, DWG", ct)
	}
	return nil
}

// AllowedReportPhotoMIMEs maps accepted image MIME types for daily-report attachments.
var AllowedReportPhotoMIMEs = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ValidateReportPhotoFile checks MIME type and file size for daily-report photo attachments.
// HEIC/HEIF files are explicitly rejected with a user-facing Croatian message.
func ValidateReportPhotoFile(header *multipart.FileHeader) error {
	if header.Size > MaxReceiptFileSize {
		return fmt.Errorf("slika je prevelika (maksimalno 10 MB, primljeno %d bajtova)", header.Size)
	}
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	lower := strings.ToLower(ct)
	if lower == "image/heic" || lower == "image/heif" {
		return fmt.Errorf("HEIC/HEIF format nije podržan; koristite JPEG, PNG ili WebP")
	}
	if _, ok := AllowedReportPhotoMIMEs[ct]; !ok {
		return fmt.Errorf("nepodržani tip datoteke %q; dopušteni su: JPEG, PNG, WebP", ct)
	}
	return nil
}

// ValidateReceiptFile checks MIME type and file size. Call before saving.
func ValidateReceiptFile(header *multipart.FileHeader) error {
	if header.Size > MaxReceiptFileSize {
		return fmt.Errorf("receipt file too large (max 10 MB, got %d bytes)", header.Size)
	}
	ct := header.Header.Get("Content-Type")
	// Strip parameters like "; charset=utf-8"
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	if _, ok := AllowedReceiptMIMEs[ct]; !ok {
		return fmt.Errorf("invalid receipt file type %q; allowed: JPEG, PNG, WEBP, PDF", ct)
	}
	return nil
}

// ── Local disk implementation ─────────────────────────────────────────────────

type LocalStorageService struct {
	basePath string
}

func NewLocalStorageService(basePath string) *LocalStorageService {
	return &LocalStorageService{basePath: basePath}
}

func (s *LocalStorageService) CheckHealth(ctx context.Context) error {
	if _, err := os.Stat(s.basePath); err != nil {
		return fmt.Errorf("local upload dir unavailable: %w", err)
	}
	return nil
}

func (s *LocalStorageService) SaveReceiptFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, projectID string,
) (string, string, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ext := AllowedReceiptMIMEs[ct]
	if ext == "" {
		ext = filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".bin"
		}
	}

	id, err := randomHex(16)
	if err != nil {
		return "", "", fmt.Errorf("generate file id: %w", err)
	}

	relPath := filepath.Join("receipts", companyID, projectID, id+ext)
	absPath := filepath.Join(s.basePath, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", "", fmt.Errorf("create upload dir: %w", err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return "", "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", "", fmt.Errorf("write file: %w", err)
	}

	return relPath, header.Filename, nil
}

func (s *LocalStorageService) ReadReceiptFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error) {
	absPath := filepath.Join(s.basePath, fileKey)
	f, err := os.Open(absPath)
	if err != nil {
		return nil, "", err
	}
	return f, mimeFromKey(fileKey), nil
}

func (s *LocalStorageService) DeleteReceiptFile(ctx context.Context, fileKey string) error {
	return os.Remove(filepath.Join(s.basePath, fileKey))
}

func (s *LocalStorageService) SaveDocumentFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, projectID string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = NormalizeDocumentMIME(ct, header.Filename)
	ext := AllowedDocumentMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".bin"
		}
	}

	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}

	relPath := filepath.Join("documents", companyID, projectID, id+ext)
	absPath := filepath.Join(s.basePath, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", "", 0, fmt.Errorf("create upload dir: %w", err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, file)
	if err != nil {
		return "", "", 0, fmt.Errorf("write file: %w", err)
	}

	return relPath, ct, n, nil
}

func (s *LocalStorageService) SaveEmployeeDocFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, employeeID string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = NormalizeDocumentMIME(ct, header.Filename)
	ext := AllowedDocumentMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".bin"
		}
	}
	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}
	relPath := filepath.Join("employee-docs", companyID, employeeID, id+ext)
	absPath := filepath.Join(s.basePath, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", "", 0, fmt.Errorf("create upload dir: %w", err)
	}
	dst, err := os.Create(absPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()
	n, err := io.Copy(dst, file)
	if err != nil {
		return "", "", 0, fmt.Errorf("write file: %w", err)
	}
	return relPath, ct, n, nil
}

func (s *LocalStorageService) SaveFinanceDocFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, subdir string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = NormalizeDocumentMIME(ct, header.Filename)
	ext := AllowedDocumentMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".bin"
		}
	}
	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}
	relPath := filepath.Join("finance-docs", companyID, subdir, id+ext)
	absPath := filepath.Join(s.basePath, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", "", 0, fmt.Errorf("create upload dir: %w", err)
	}
	dst, err := os.Create(absPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()
	n, err := io.Copy(dst, file)
	if err != nil {
		return "", "", 0, fmt.Errorf("write file: %w", err)
	}
	return relPath, ct, n, nil
}

func (s *LocalStorageService) ReadFinanceDocFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error) {
	return s.ReadReceiptFile(ctx, fileKey)
}

func (s *LocalStorageService) ReadEmployeeDocFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error) {
	return s.ReadReceiptFile(ctx, fileKey)
}

func (s *LocalStorageService) SaveReportPhotoFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, reportID string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ext := AllowedReportPhotoMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".jpg"
		}
	}

	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}

	relPath := filepath.Join("report-photos", companyID, reportID, id+ext)
	absPath := filepath.Join(s.basePath, relPath)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", "", 0, fmt.Errorf("create upload dir: %w", err)
	}

	dst, err := os.Create(absPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, file)
	if err != nil {
		return "", "", 0, fmt.Errorf("write file: %w", err)
	}

	return relPath, ct, n, nil
}

// ── S3 / Cloudflare R2 implementation ────────────────────────────────────────

type S3StorageService struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

// NewS3StorageService creates an S3StorageService configured for the given endpoint.
// Set endpoint to the Cloudflare R2 account URL (e.g. https://<id>.r2.cloudflarestorage.com).
// Set usePathStyle=true for Cloudflare R2 (required).
func NewS3StorageService(endpoint, accessKeyID, secretKey, region, bucket, publicBaseURL string, usePathStyle bool) (*S3StorageService, error) {
	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create S3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = usePathStyle
	})

	return &S3StorageService{
		client:        client,
		bucket:        bucket,
		publicBaseURL: publicBaseURL,
	}, nil
}

func (s *S3StorageService) CheckHealth(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("S3 bucket %q unreachable: %w", s.bucket, err)
	}
	return nil
}

func (s *S3StorageService) SaveReceiptFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, projectID string,
) (string, string, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ext := AllowedReceiptMIMEs[ct]
	if ext == "" {
		ext = filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".bin"
		}
	}

	id, err := randomHex(16)
	if err != nil {
		return "", "", fmt.Errorf("generate file id: %w", err)
	}

	// Use forward slashes for S3 keys (never filepath.Join — OS-specific separator)
	key := "receipts/" + companyID + "/" + projectID + "/" + id + ext

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("read upload: %w", err)
	}

	size := int64(len(data))
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(ct),
		ContentLength: &size,
	})
	if err != nil {
		return "", "", fmt.Errorf("upload to S3: %w", err)
	}

	return key, header.Filename, nil
}

func (s *S3StorageService) ReadReceiptFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return nil, "", fmt.Errorf("get S3 object: %w", err)
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	if ct == "" {
		ct = mimeFromKey(fileKey)
	}
	return out.Body, ct, nil
}

func (s *S3StorageService) DeleteReceiptFile(ctx context.Context, fileKey string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileKey),
	})
	return err
}

func (s *S3StorageService) SaveDocumentFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, projectID string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = NormalizeDocumentMIME(ct, header.Filename)
	ext := AllowedDocumentMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".bin"
		}
	}

	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}

	key := "documents/" + companyID + "/" + projectID + "/" + id + ext

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", 0, fmt.Errorf("read upload: %w", err)
	}

	size := int64(len(data))
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(ct),
		ContentLength: &size,
	})
	if err != nil {
		return "", "", 0, fmt.Errorf("upload to S3: %w", err)
	}

	return key, ct, size, nil
}

func (s *S3StorageService) SaveEmployeeDocFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, employeeID string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = NormalizeDocumentMIME(ct, header.Filename)
	ext := AllowedDocumentMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".bin"
		}
	}
	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}
	key := "employee-docs/" + companyID + "/" + employeeID + "/" + id + ext
	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", 0, fmt.Errorf("read upload: %w", err)
	}
	size := int64(len(data))
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(ct),
		ContentLength: &size,
	})
	if err != nil {
		return "", "", 0, fmt.Errorf("upload to S3: %w", err)
	}
	return key, ct, size, nil
}

func (s *S3StorageService) SaveFinanceDocFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, subdir string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = NormalizeDocumentMIME(ct, header.Filename)
	ext := AllowedDocumentMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".bin"
		}
	}
	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}
	key := "finance-docs/" + companyID + "/" + subdir + "/" + id + ext
	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", 0, fmt.Errorf("read upload: %w", err)
	}
	size := int64(len(data))
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(ct),
		ContentLength: &size,
	})
	if err != nil {
		return "", "", 0, fmt.Errorf("upload to S3: %w", err)
	}
	return key, ct, size, nil
}

func (s *S3StorageService) ReadFinanceDocFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error) {
	return s.ReadReceiptFile(ctx, fileKey)
}

func (s *S3StorageService) ReadEmployeeDocFile(ctx context.Context, fileKey string) (io.ReadCloser, string, error) {
	return s.ReadReceiptFile(ctx, fileKey)
}

func (s *S3StorageService) SaveReportPhotoFile(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	companyID, reportID string,
) (string, string, int64, error) {
	ct := header.Header.Get("Content-Type")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ext := AllowedReportPhotoMIMEs[ct]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".jpg"
		}
	}

	id, err := randomHex(16)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate file id: %w", err)
	}

	key := "report-photos/" + companyID + "/" + reportID + "/" + id + ext

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", 0, fmt.Errorf("read upload: %w", err)
	}

	size := int64(len(data))
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(ct),
		ContentLength: &size,
	})
	if err != nil {
		return "", "", 0, fmt.Errorf("upload to S3: %w", err)
	}

	return key, ct, size, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mimeFromKey(fileKey string) string {
	switch strings.ToLower(filepath.Ext(fileKey)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".tiff":
		return "image/tiff"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	case ".zip":
		return "application/zip"
	case ".dwg":
		return canonicalDWGMIME
	default:
		return "application/octet-stream"
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
