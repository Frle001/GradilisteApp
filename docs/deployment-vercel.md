# Vercel Deployment (Frontend)

## Prerequisites

- Vercel account linked to GitHub
- GitHub repository pushed to `main` (and optionally `staging`)
- Backend deployed on Render with a known public URL

## First-time setup

### 1. Create Vercel project

1. Go to vercel.com → New Project → Import Git Repository
2. Select this repository
3. **Root directory**: set to `apps/web`
4. **Framework preset**: Next.js (auto-detected)
5. **Build command**: `npm run build` (default)
6. **Output directory**: `.next` (default)

### 2. Set environment variables

In Vercel → Project → Settings → Environment Variables:

| Variable | Value | Environment |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `https://<render-service>.onrender.com/api` | Production |
| `NEXT_PUBLIC_API_URL` | `https://<render-staging-service>.onrender.com/api` | Preview |
| `NEXT_PUBLIC_APP_VERSION` | `$VERCEL_GIT_COMMIT_SHA` | All |

> **Note:** `NEXT_PUBLIC_*` variables are embedded at build time by Next.js. If you change a value, you must redeploy for it to take effect.

### 3. Deploy

Push to `main` — Vercel auto-deploys. Each PR also gets a preview URL.

## Monorepo configuration

Vercel should auto-detect the Next.js app when `Root Directory` is set to `apps/web`.
If not, add a `vercel.json` at the repo root:

```json
{
  "buildCommand": "cd apps/web && npm run build",
  "outputDirectory": "apps/web/.next",
  "installCommand": "cd apps/web && npm ci"
}
```

## Custom domain

1. Vercel → Project → Settings → Domains → Add
2. Add your domain (e.g. `app.gradiliste.hr`)
3. Add DNS records as instructed by Vercel
4. Update `CORS_ALLOWED_ORIGINS` on Render to include the new domain

## Preview deployments

Every PR gets an isolated preview URL. These use the **Preview** environment variables.
Set `NEXT_PUBLIC_API_URL` for Preview to point at the staging Render service.

## Rollback

Vercel keeps a full deploy history. To roll back:
1. Vercel → Project → Deployments
2. Find the last good deployment
3. Click the three-dot menu → Redeploy → Promote to Production
