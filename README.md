# Gradilište App

Production-style construction company management application.

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Node.js 18+ (for local frontend development)
- Go 1.21+ (for local backend development)

### Run the full stack
```bash
docker-compose up
```

Frontend: `http://localhost:3000`
Backend API: `http://localhost:8080`
Database: `localhost:5432`

### Development Setup (separate services)

**Backend:**
```bash
cd services/api
cp .env.example .env
go run main.go
```

**Frontend:**
```bash
cd apps/web
npm install
npm run dev
```

**Database migrations:**
```bash
# Phase 1 (initial setup - now replaced by Phase 2)
# psql -h localhost -U gradiliste -d gradiliste -f migrations/001_initial_schema.sql

# Phase 2 (complete schema with migrations)
cd database
for f in migrations/*.sql; do
  psql -h localhost -U gradiliste -d gradiliste -f "$f"
done

# Apply seed data (optional - for test data)
psql -h localhost -U gradiliste -d gradiliste -f seeds/seed_phase2.sql
```

## Project Structure

```
apps/web/              # Next.js frontend
services/api/          # Go Gin backend
database/              # PostgreSQL migrations and seeds
docs/                  # Architecture and planning documentation
docker-compose.yml     # Local development stack
```

## Documentation

- [Architecture](docs/architecture.md) — System design and decisions
- [Roles & Permissions](docs/roles-and-permissions.md) — User role definitions
- [API Plan](docs/api-plan.md) — Endpoint structure and design
- [Database Plan](docs/database-plan.md) — Schema and data model

## Tech Stack

- **Frontend:** Next.js, TypeScript, Tailwind CSS, shadcn/ui
- **Backend:** Go, Gin, pgx
- **Database:** PostgreSQL
- **DevOps:** Docker Compose

## Development Workflow

1. Make changes in `apps/web/` or `services/api/`
2. Changes auto-reload with hot reload (frontend) or go run watch
3. Database schema changes go in `database/migrations/`
4. New migrations need docker-compose restart or psql apply

### Testing Database Connection

```bash
# Check if API can reach database
curl http://localhost:8080/api/db-health

# Get database summary (development only)
curl http://localhost:8080/api/debug/db-summary
```

⚠️ **Note:** Debug endpoints are for local development only. Disable them in production.

## Team Roles

- **Direktor** — Full access, strategic decisions
- **Inženjer** — Full access, technical decisions
- **Administracija** — Employee management
- **Poslovođa** — Project management, daily reports, material tracking
- **Radnik** — Employee record (may not have login yet)

See [roles-and-permissions.md](docs/roles-and-permissions.md) for detailed permissions.
