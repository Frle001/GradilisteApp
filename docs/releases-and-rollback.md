# Releases and Rollback

## Release process

1. **Develop on a feature branch** against `staging`
2. **Open a PR into `staging`** — CI runs (lint, type-check, build, go vet/test)
3. **Merge to `staging`** — staging auto-deploys; run smoke test (see [post-deployment-smoke-test.md](post-deployment-smoke-test.md))
4. **Open a PR from `staging` into `main`** — final review
5. **Merge to `main`** — production auto-deploys
6. **Smoke test production** within 15 minutes of deploy

## Migrations

- All migrations live in `database/migrations/` and are numbered sequentially
- Migrations are applied manually before the code deploy (see [deployment-render.md](deployment-render.md))
- **Never apply a migration to production without first applying and verifying it on staging**
- Prefer additive migrations (add column, add table) over destructive ones
- For a destructive migration: take a manual backup first, apply, verify immediately

## Frontend rollback (Vercel)

No database changes needed — just redeploy a previous build.

1. Vercel → Project → Deployments
2. Find the last known-good deployment
3. Three-dot menu → Redeploy → Promote to Production

Takes effect in ~30 seconds.

## Backend rollback (Render)

1. Render → Service (`gradiliste-api`) → Deploys
2. Find the last known-good deploy
3. Click **Redeploy**

If the failing deploy included a backwards-incompatible migration, you must also:
1. Stop inbound traffic (Render → Service → Suspend)
2. Restore the database from the pre-migration backup
3. Redeploy the previous API version
4. Verify, then resume traffic

## Determining the deployed version

The `/api/health` endpoint returns the current version:

```bash
curl https://api.example.com/api/health
# {"status":"ok","env":"production","version":"abc1234"}
```

`APP_VERSION` is set to the git commit SHA via CI/CD (`RENDER_GIT_COMMIT` on Render).

## Hotfix procedure

For urgent production fixes that cannot wait for the normal staging cycle:

1. Branch from `main` (not `staging`)
2. Apply the minimum fix
3. Open PR directly into `main`
4. After merge and deploy, back-merge into `staging` to keep branches in sync

## No-downtime deploys

Render performs rolling container replacement. During a deploy there is a brief window where both old and new containers run simultaneously. Design API changes to be backwards-compatible with the previous frontend version during this window.
