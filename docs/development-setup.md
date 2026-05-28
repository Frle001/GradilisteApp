# Development Setup Guide

Complete instructions to get Gradilište App running locally.

## Prerequisites

### Required

- **Docker & Docker Compose** — For database and services
  - [Install Docker Desktop](https://www.docker.com/products/docker-desktop) (includes Compose)
  - Verify: `docker --version && docker-compose --version`

- **Node.js 18+** — For frontend
  - [Install Node.js](https://nodejs.org/)
  - Verify: `node --version && npm --version`

- **Go 1.21+** — For backend (if running outside Docker)
  - [Install Go](https://golang.org/dl/)
  - Verify: `go version`

### Optional

- **Git** — For version control
- **VS Code** — Recommended editor
- **PostgreSQL CLI tools** — For manual database access
- **Postman** — For API testing (later phases)

---

## Quick Start (5 minutes)

### 1. Start Full Stack with Docker Compose

```bash
cd /path/to/gradiliste-app

# Start all services (PostgreSQL, Go API, optional: Next.js)
docker-compose up

# Wait for output showing:
# postgres ... accepting connections
# api ... [GIN-debug] ...listening on :8080
```

Services are now running:
- **Frontend:** http://localhost:3000 (if enabled in docker-compose.yml)
- **Backend API:** http://localhost:8080
- **Database:** localhost:5432

### 2. Test Backend

```bash
curl http://localhost:8080/api/health
```

Expected response:
```json
{
  "status": "ok",
  "message": "Gradilište API is running"
}
```

### 3. Develop Frontend (in separate terminal)

```bash
cd apps/web
npm install
npm run dev
```

Open http://localhost:3000

### 4. Stop All Services

```bash
# In terminal running docker-compose
Ctrl+C

# Or forcefully stop
docker-compose down
```

---

## Detailed Setup

### Database Only

```bash
docker-compose up postgres

# The database is now running on localhost:5432
# All migrations and seeds are auto-applied
```

### Backend Only (Local Go Development)

```bash
# Terminal 1: Start database
docker-compose up postgres

# Terminal 2: Start backend
cd services/api

# Copy environment file
cp .env.example .env

# Download dependencies
go mod download

# Run server
go run main.go

# Output should show:
# Starting Gradilište API on port 8080
```

**Troubleshooting:**
- Port 8080 in use? Change `PORT=8081` in `.env`
- Database connection failed? Ensure `docker-compose up postgres` is running

### Frontend Only (Local Development)

```bash
cd apps/web

# Install dependencies
npm install

# Create .env.local (if needed)
echo "NEXT_PUBLIC_API_URL=http://localhost:8080/api" > .env.local

# Start dev server
npm run dev

# Output should show:
# ▲ Next.js 14.0
# - Local:        http://localhost:3000
```

Open http://localhost:3000 in browser.

---

## Project Structure Quick Reference

```
gradiliste-app/
├── apps/web/                    # Next.js frontend
│   ├── app/                     # App router (pages, layouts)
│   ├── components/              # React components
│   ├── lib/                     # Utilities (api-client.ts)
│   ├── package.json
│   ├── next.config.js
│   └── tsconfig.json
│
├── services/api/                # Go Gin backend
│   ├── main.go                  # Entry point
│   ├── database.go              # DB connection
│   ├── middleware.go            # CORS, logging
│   ├── handlers/                # HTTP handlers (future)
│   ├── services/                # Business logic (future)
│   ├── repositories/            # Data access (future)
│   ├── models/                  # Data structures
│   ├── go.mod
│   ├── .env.example
│   └── Dockerfile
│
├── database/
│   ├── migrations/              # SQL schema files
│   │   └── 001_initial_schema.sql
│   └── seeds/                   # Test data
│       └── seed_initial.sql
│
├── docs/                        # Documentation
│   ├── architecture.md
│   ├── roles-and-permissions.md
│   ├── api-plan.md
│   └── database-plan.md
│
└── docker-compose.yml           # Local dev stack
```

---

## Common Tasks

### View Database

```bash
# Connect to running database
docker-compose exec postgres psql -U gradiliste -d gradiliste

# Inside psql:
\dt              -- List tables
\d users         -- Show users table schema
SELECT * FROM users;  -- Query data
\q              -- Quit
```

### Check Backend Logs

```bash
# If running with docker-compose
docker-compose logs api

# If running locally
go run main.go  # Logs print to console
```

### Reset Database

```bash
# Delete volume (data is lost)
docker-compose down -v

# Start fresh
docker-compose up postgres
```

### Hot Reload Backend (Local)

During development, the backend **does not** auto-reload.

Two options:

1. **Manual restart** — Stop and `go run main.go` again
2. **Use air (watch tool)**:
   ```bash
   go install github.com/cosmtrek/air@latest
   cd services/api
   air  # Auto-restarts on file changes
   ```

### Hot Reload Frontend

Already enabled with Next.js:
- Changes to `.tsx` files instantly reload
- Changes to Tailwind config require restart

### Add npm Package to Frontend

```bash
cd apps/web
npm install <package-name>
```

### Add Go Package to Backend

```bash
cd services/api
go get github.com/package-author/package-name
go mod tidy
```

### Format Code

```bash
# Go
cd services/api
go fmt ./...

# Frontend (if configured)
cd apps/web
npm run lint
```

---

## Environment Variables

### Backend (.env)

```
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=gradiliste
DB_PASSWORD=gradiliste_dev_password
DB_NAME=gradiliste
ENV=development
```

See `services/api/.env.example`

### Frontend (.env.local)

```
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

### Docker Compose

Managed in `docker-compose.yml` — adjust as needed.

---

## Testing Endpoints

### Health Check

```bash
curl http://localhost:8080/api/health
```

### Via Browser

Visit http://localhost:3000 — homepage tests backend connection

### Via Frontend Code

The Next.js homepage (`apps/web/app/page.tsx`) calls `/api/health` on load and displays status.

---

## Troubleshooting

### Port Already in Use

```
Error: listen tcp :8080: bind: address already in use
```

**Solution:**
```bash
# Find process using port 8080
lsof -i :8080  # macOS/Linux
netstat -ano | findstr :8080  # Windows

# Kill process or use different port
PORT=8081 go run main.go
```

### Database Connection Failed

```
unable to parse database config: connection refused
```

**Solutions:**
1. Ensure `docker-compose up postgres` is running
2. Check `DB_HOST` in `.env` (should be `localhost` for local, `postgres` for Docker network)
3. Verify credentials in `.env` match docker-compose

### Node Modules Not Found

```bash
cd apps/web
rm -rf node_modules package-lock.json
npm install
```

### Go Modules Cache Issues

```bash
cd services/api
rm go.sum
go mod tidy
```

### Migrations Not Applied

```bash
# Manually apply
docker-compose exec postgres psql -U gradiliste -d gradiliste -f /docker-entrypoint-initdb.d/001_initial_schema.sql
```

---

## Next Steps

1. ✅ **Foundation** (you are here)
2. → Start Phase 2: Authentication & Authorization
3. → Add business logic for modules
4. → Connect frontend to backend endpoints
5. → Testing & deployment

---

## Useful Commands Reference

```bash
# Docker Compose
docker-compose up                     # Start all
docker-compose up postgres api        # Start specific services
docker-compose down                   # Stop and remove containers
docker-compose logs -f api            # Follow logs
docker-compose exec postgres bash     # Shell into container

# Database
docker-compose exec postgres psql -U gradiliste -d gradiliste

# Frontend
cd apps/web && npm run dev            # Dev server
cd apps/web && npm run build          # Production build
cd apps/web && npm install            # Install deps

# Backend
cd services/api && go run main.go     # Run locally
cd services/api && go mod tidy        # Clean dependencies
cd services/api && go fmt ./...       # Format code

# Testing
curl http://localhost:8080/api/health
curl http://localhost:3000
```

---

## IDE Setup

### VS Code Extensions (Recommended)

```
- Go (golang.go)
- REST Client (humao.rest-client)
- PostgreSQL (cweijan.vscode-postgresql-client2)
- Tailwind CSS IntelliSense (bradlc.vscode-tailwindcss)
- ESLint (dbaeumer.vscode-eslint)
```

Install via:
```bash
code --install-extension golang.go
code --install-extension humao.rest-client
# ... etc
```

### .vscode/settings.json (Optional)

```json
{
  "go.lintOnSave": "package",
  "[go]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "golang.go"
  },
  "[typescript]": {
    "editor.formatOnSave": true
  }
}
```

---

## Performance Tips

### Database

- Keep connection pool settings reasonable (25 max)
- Monitor slow queries (log queries > 500ms)
- Index frequently filtered columns

### Frontend

- Test with throttled network (DevTools)
- Check bundle size: `npm run build`
- Use React DevTools for component performance

### Backend

- Profile CPU/memory: `go test -cpuprofile=cpu.prof -memprofile=mem.prof`
- Use `pprof` for runtime analysis
- Load test with wrk/Apache Bench

---

## Resources

- **Go:** https://golang.org/doc
- **Gin:** https://gin-gonic.com/
- **PostgreSQL:** https://www.postgresql.org/docs/
- **pgx:** https://github.com/jackc/pgx
- **Next.js:** https://nextjs.org/docs
- **React:** https://react.dev
- **TypeScript:** https://www.typescriptlang.org/docs/
- **Tailwind CSS:** https://tailwindcss.com/docs

---

## Getting Help

1. Check [architecture.md](architecture.md) for design decisions
2. Check [api-plan.md](api-plan.md) for endpoint structure
3. Check [database-plan.md](database-plan.md) for schema
4. Check README files in each service folder

Got stuck? Review the error message, check logs, and work backward from the last successful step.
