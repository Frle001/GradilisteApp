# Storage Service

## Overview

Receipt files are stored through the `StorageService` interface defined in `services/api/services/storage_service.go`. The abstraction allows swapping local disk for S3/R2/Supabase Storage without changing any calling code.

## Interface

```go
type StorageService interface {
    SaveReceiptFile(ctx, file, header, companyID, projectID) (fileKey, originalFilename string, err error)
    ReadReceiptFile(ctx, fileKey) (reader io.ReadCloser, contentType string, err error)
    DeleteReceiptFile(ctx, fileKey) error
}
```

## Local disk implementation (development)

Files are stored at:
```
$UPLOADS_DIR/receipts/{company_id}/{project_id}/{random_hex}.{ext}
```

Default `UPLOADS_DIR = ./uploads` (relative to the Go binary, i.e. inside the API container).

### Configuration

Set `UPLOADS_DIR` in `.env` or environment:
```
UPLOADS_DIR=/data/uploads
```

### Docker volume (required for persistence)

Add a volume mount in `docker-compose.yml` so files survive container restarts:
```yaml
services:
  api:
    volumes:
      - ./uploads:/app/uploads
```

Without this, uploaded files are lost when the container restarts.

## Receipt access

Receipt files are **never served publicly**. They are accessed only through:
```
GET /api/material-purchases/:id/receipt
```

This endpoint:
1. Verifies JWT authentication
2. Checks the caller has permission to view the purchase
3. Reads the file from the storage service
4. Streams it with the correct `Content-Type`

## Validation

Before saving, `ValidateReceiptFile()` checks:
- **Allowed MIME types**: `image/jpeg`, `image/png`, `image/webp`, `application/pdf`
- **Maximum size**: 10 MB

The original filename is stored but never used to construct the file path (safe unique names are generated server-side).

## Swapping to S3/R2

1. Implement the `StorageService` interface for your provider
2. Change the instantiation in `main.go`:
   ```go
   storageSvc := services.NewS3StorageService(bucket, region, ...)
   ```
3. Migrate existing `receipt_file_key` values (currently local paths) to S3 keys

The `receipt_file_key` column in `material_purchase_sessions` stores the internal storage key. The public URL is always computed as `/api/material-purchases/{id}/receipt` and never stored in the database.
