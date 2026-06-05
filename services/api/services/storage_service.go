package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// StorageService abstracts receipt file storage. Swap local disk for S3/R2 without changing callers.
type StorageService interface {
	SaveReceiptFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, projectID string) (fileKey, originalFilename string, err error)
	ReadReceiptFile(ctx context.Context, fileKey string) (reader io.ReadCloser, contentType string, err error)
	DeleteReceiptFile(ctx context.Context, fileKey string) error
}

// AllowedReceiptMIMEs maps allowed MIME types to their canonical extension.
var AllowedReceiptMIMEs = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

const MaxReceiptFileSize = 10 << 20 // 10 MB

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

func mimeFromKey(fileKey string) string {
	switch strings.ToLower(filepath.Ext(fileKey)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
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
