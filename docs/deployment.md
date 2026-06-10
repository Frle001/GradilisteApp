# Deployment Guide

## Option A — Managed cloud (recommended)

Deploy the Go API to a managed platform and the Next.js frontend to Vercel. No servers to manage.

### Backend on Fly.io (or Railway / Render)

1. Install Fly CLI: `brew install flyctl` (or see fly.io/docs)
2. From `services/api/`: `fly launch` — choose region, decline Postgres (use managed below)
3. Set secrets — **never put real values in compose files or the repo**:
   ```bash
   fly secrets set \
     JWT_SECRET="$(openssl rand -base64 48)" \
     DATABASE_URL="postgres://user:pass@host/dbname?sslmode=require" \
     CORS_ALLOWED_ORIGINS="https://your-app.vercel.app" \
     APP_ENV="production"
   ```
4. Deploy: `fly deploy`
5. Run migrations once after first deploy:
   ```bash
   fly ssh console -C "psql $DATABASE_URL -f /path/to/009_refresh_tokens.sql"
   ```
   Or use a one-off process if your platform supports it.

**Do NOT run seed files in production.**

### Managed PostgreSQL

- **Fly.io**: `fly postgres create` + `fly postgres attach`
- **Railway**: add PostgreSQL service in the dashboard
- **Render**: add PostgreSQL, copy the external connection string to `DATABASE_URL`
- **Neon / Supabase**: create DB, copy connection string

Always use `DATABASE_URL` on managed platforms. `sslmode=require` is typical.

### Frontend on Vercel

1. Import the repo in Vercel, set **Root Directory** to `apps/web`.
2. Set environment variable: `NEXT_PUBLIC_API_URL=https://your-api.fly.dev/api`
3. Vercel builds and deploys on every push to `main`.

### CORS setup

`CORS_ALLOWED_ORIGINS` must match the **exact origin** the browser sends, including scheme and domain. No trailing slash.

```
# Single Vercel deployment
CORS_ALLOWED_ORIGINS=https://gradiliste.vercel.app

# Custom domain + preview
CORS_ALLOWED_ORIGINS=https://app.example.com,https://gradiliste.vercel.app
```

---

## Option B — VPS / Docker (self-hosted)

**Prerequisites**: VPS with Docker + Docker Compose, a domain with DNS pointed at the server.

### Steps

1. SSH into server, clone the repo:
   ```bash
   git clone https://github.com/your-org/gradiliste.git
   cd gradiliste
   ```

2. Create the production compose file:
   ```bash
   cp docker-compose.prod.example.yml docker-compose.prod.yml
   ```
   Edit `docker-compose.prod.yml` — fill in all `CHANGE_ME` placeholders. **Do not commit this file.**

3. Run migrations on first deploy:
   ```bash
   docker compose -f docker-compose.prod.yml run --rm api \
     /bin/sh -c 'psql "$DATABASE_URL" < /migrations/009_refresh_tokens.sql'
   ```
   Or connect to the postgres container directly:
   ```bash
   docker compose -f docker-compose.prod.yml exec postgres \
     psql -U gradiliste -d gradiliste -f /path/to/009_refresh_tokens.sql
   ```

4. Start services:
   ```bash
   docker compose -f docker-compose.prod.yml up -d
   ```

5. Set up a reverse proxy (Caddy example):
   ```
   # /etc/caddy/Caddyfile
   api.example.com {
     reverse_proxy localhost:8080
   }
   app.example.com {
     reverse_proxy localhost:3000
   }
   ```
   Caddy provisions TLS automatically via Let's Encrypt.

### Updates

```bash
git pull
docker compose -f docker-compose.prod.yml build api
docker compose -f docker-compose.prod.yml up -d --no-deps api
```

Run new migration files manually before restarting the API.

### Environment variable reference

See `services/api/.env.example` for all variables with descriptions.

Key production requirements:

| Variable | Requirement |
|---|---|
| `APP_ENV=production` | Enables secure cookie flags, strict CORS |
| `JWT_SECRET` | Required — server exits on startup if missing |
| `CORS_ALLOWED_ORIGINS` | Required — server exits on startup if missing |
| `DATABASE_URL` or `DB_*` | Required |

---

## Health checks

| Endpoint | Purpose |
|---|---|
| `GET /health` | Liveness — returns 200 if process is alive |
| `GET /api/health` | Same, under /api prefix |
| `GET /api/ready` | Readiness — checks DB + storage; returns 503 if not ready |

Use `GET /health` for load-balancer health checks (no auth required, fast).
Use `GET /api/ready` for orchestrator readiness probes (Fly.io `[checks]`, K8s readinessProbe).
