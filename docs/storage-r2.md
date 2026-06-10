# Cloudflare R2 Storage

Receipt files (images, PDFs) are stored in Cloudflare R2. R2 is S3-compatible, has zero egress fees, and files survive container restarts.

## Create R2 buckets

Create two separate buckets — one per environment:

1. Cloudflare Dashboard → R2 → Create Bucket
2. Name: `gradiliste-receipts` (production) and `gradiliste-receipts-staging` (staging)
3. Location: choose **Europe** (Frankfurt) to match Render region
4. **Do not** enable public access — receipts are private company documents served through the API

## Create an R2 API token

R2 API tokens are separate from your Cloudflare account API key.

1. Cloudflare Dashboard → R2 → Manage R2 API Tokens → Create API Token
2. Permissions: **Object Read & Write** (do not use Admin)
3. Bucket scope: restrict to your specific bucket(s)
4. Copy the **Access Key ID** and **Secret Access Key** — you will not see the secret again

Create one token per environment (staging token can only access the staging bucket).

## Object key structure

Files are stored with the key pattern:

```
receipts/{company_id}/{project_id}/{random_hex}.{ext}
```

This ensures company data isolation at the key level and allows future per-company lifecycle policies.

## Environment variables

```
S3_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
S3_BUCKET=gradiliste-receipts
S3_ACCESS_KEY_ID=<token key id>
S3_SECRET_ACCESS_KEY=<token secret>
S3_REGION=auto
S3_USE_PATH_STYLE=true
```

`S3_REGION=auto` is required for Cloudflare R2 — it does not use AWS regions.
`S3_USE_PATH_STYLE=true` is required for Cloudflare R2 — virtual-hosted style is not supported.

## Access pattern

Receipts are **never served directly from a public R2 URL**. Clients request them via the API:

```
GET /api/material-purchases/{id}/receipt
```

The API authenticates the request, verifies company isolation, then proxies the file from R2 to the client. This ensures that:
- Unauthenticated users cannot access any receipt
- Users from company A cannot access receipts from company B
- No pre-signed URLs leak file paths

## Retention policy

R2 does not enforce object expiry by default. Receipts should be retained as long as the company is active. Consider enabling versioning or object lock for audit compliance requirements.

## Disaster recovery

If R2 becomes unavailable, the API returns HTTP 503 on receipt requests. The database records remain intact. Files are not recoverable from the database.

To back up R2 objects, use rclone with the S3 provider or the Cloudflare R2 API to copy to another bucket or region.
