# Render Deployment (Backend API)

## Prerequisites

- Render account linked to GitHub
- Managed PostgreSQL instance created in Render (see below)
- Cloudflare R2 bucket and API token created (see [storage-r2.md](storage-r2.md))

## Create managed PostgreSQL

1. Render Dashboard → New → PostgreSQL
2. Name: `gradiliste-db` (prod) or `gradiliste-db-staging` (staging)
3. Plan: Starter (upgrade to Standard for production SLAs)
4. Region: Frankfurt (EU)
5. After creation, copy the **Internal Database URL** — used as `DATABASE_URL` in the API service

> Use the **internal** URL for zero-latency, free-egress connection between the database and API service on Render.

## Create the API web service

1. Render Dashboard → New → Web Service
2. Connect your GitHub repository
3. **Name**: `gradiliste-api` (or `gradiliste-api-staging`)
4. **Root Directory**: `services/api`
5. **Runtime**: Docker
6. **Dockerfile Path**: `./Dockerfile`
7. **Region**: Frankfurt (same as database)
8. **Plan**: Starter

### Health check

Set the health check path to `/health`. Render will poll this every 30 seconds and restart the container if it returns a non-200 response.

## Environment variables

Set these in Render → Service → Environment:

```
APP_ENV=production
APP_VERSION=<leave blank — Render sets RENDER_GIT_COMMIT>
DATABASE_URL=<internal PostgreSQL URL from above>
JWT_SECRET=<generate: openssl rand -base64 48>
CORS_ALLOWED_ORIGINS=https://<your-vercel-domain>.vercel.app
COOKIE_SECURE=true
COOKIE_SAME_SITE=none
UPLOAD_STORAGE_DRIVER=s3
S3_ENDPOINT=https://<account_id>.r2.cloudflarestorage.com
S3_BUCKET=gradiliste-receipts
S3_ACCESS_KEY_ID=<R2 API token key ID>
S3_SECRET_ACCESS_KEY=<R2 API token secret>
S3_REGION=auto
S3_USE_PATH_STYLE=true
LOG_LEVEL=info
BCRYPT_COST=12
```

> Never set `UPLOAD_STORAGE_DRIVER=local` in production. Container filesystems on Render are ephemeral — all files are lost on restart.

## Running database migrations

Render does not run migrations automatically. Options:

**Option A — Render Shell (recommended for first deploy):**
1. Service → Shell tab
2. Run: `psql $DATABASE_URL -f /path/to/migrations/xxx.sql`

**Option B — Pre-deploy command (add to render.yaml):**
```yaml
buildCommand: go build -o api .
preDeployCommand: psql $DATABASE_URL < /migrations/latest.sql
```

Apply migrations before switching traffic to a new deploy.

## Bootstrap first admin user

After the first successful deploy and after running all migrations:

```bash
# From your local machine with DATABASE_URL pointing at the Render database
# (use the External Database URL for external access)
DATABASE_URL=postgres://... go run ./services/api/cmd/create-admin
```

The CLI will prompt for company name, admin name, email, role, and a temporary password.
The user will be forced to change the password on first login.

## Auto-deploy

Render auto-deploys when `main` (or your configured branch) is pushed to GitHub.
Each deploy runs a new container build. Zero-downtime is achieved via Render's rolling deploy.

## Rollback

Render → Service → Deploys → find the last good deploy → Redeploy.
If the deploy involved a destructive migration, restore the database from backup first.

## Logs

Render → Service → Logs tab. The API logs structured JSON (via `log/slog`).
Filter for `"level":"error"` to see only errors.
