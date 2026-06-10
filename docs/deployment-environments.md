# Deployment Environments

## Two isolated environments

The Gradilište app runs two fully separate environments. They share no data, no database, and no storage bucket.

| | Staging | Production |
|---|---|---|
| **Purpose** | Pre-release verification | Live company data |
| **Database** | Render managed PostgreSQL (staging) | Render managed PostgreSQL (production) |
| **Storage** | Cloudflare R2 bucket: `gradiliste-receipts-staging` | Cloudflare R2 bucket: `gradiliste-receipts` |
| **Frontend** | Vercel preview / separate Vercel project | Vercel production deployment |
| **Backend** | Render service: `gradiliste-api-staging` | Render service: `gradiliste-api` |
| **Admin user** | Created with `create-admin` tool; test accounts only | Created with `create-admin` tool; real pilot users |

## Environment variables

Each environment has its own set of secrets. Set them in the Render dashboard and Vercel dashboard respectively. Never share `JWT_SECRET`, `DATABASE_URL`, or R2 credentials between staging and production.

Key variables that differ between environments:

| Variable | Staging | Production |
|---|---|---|
| `DATABASE_URL` | Render staging DB internal URL | Render prod DB internal URL |
| `JWT_SECRET` | Unique random secret | Unique random secret |
| `S3_BUCKET` | `gradiliste-receipts-staging` | `gradiliste-receipts` |
| `CORS_ALLOWED_ORIGINS` | Vercel staging URL | Vercel production URL |
| `APP_ENV` | `production` | `production` |
| `LOG_LEVEL` | `debug` | `info` |

## Git branching strategy

- `main` — production branch. Merges to main trigger Vercel + Render production deploys.
- `staging` — staging branch. Merges to staging trigger staging deploys.
- Feature branches are developed against `staging`, reviewed, then promoted to `main`.

Never commit secrets. Never push directly to `main` without a passing CI run and smoke test on staging.

## Database isolation rules

- **Never point staging at the production database.**
- **Never run seed scripts or `DROP TABLE` in production.**
- **Never copy production data to staging** without anonymizing it first (GDPR / privacy).

Staging may use truncated, anonymized, or synthetic data only.
