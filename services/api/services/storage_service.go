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

// StorageService abstracts receipt file storage. Swap local disk for S3/R2 without changing callers.
type StorageService interface {
	SaveReceiptFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, companyID, projectID string) (fileKey, originalFilename string, err error)
	ReadReceiptFile(ctx context.Context, fileKey string) (reader io.ReadCloser, contentType string, err error)
	DeleteReceiptFile(ctx context.Context, fileKey string) error
	CheckHealth(ctx context.Context) error
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

// ── helpers ───────────────────────────────────────────────────────────────────

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
