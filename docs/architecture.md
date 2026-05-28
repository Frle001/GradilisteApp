# Gradilište App - Architecture

## Overview

Gradilište App is a production-style construction company management system built with a **monorepo structure** using modern web technologies.

The system is split into **three main layers**:

1. **Frontend** — Next.js web application (TypeScript, Tailwind, shadcn/ui)
2. **Backend** — Go Gin REST API with pgx database driver
3. **Database** — PostgreSQL with clean migrations and seeds

## Why This Architecture?

### Monorepo Structure

**Benefit:** Single repository, unified versioning, easier dependency management

```
gradiliste-app/
├── apps/web              # Frontend
├── services/api          # Backend
├── database/             # Migrations & seeds
└── docs/                 # Shared documentation
```

### Separate Frontend & Backend

**Benefit:** Independently scalable, independently deployable, clear API contract

- **Frontend** → REST API calls → **Backend** → Database queries → **Database**
- Frontend can be cached/served from CDN
- Backend can scale independently
- Easy to swap frontend framework in future (mobile app, etc.)

### Go + Gin Backend

**Benefits:**
- Fast compilation and execution
- Low memory footprint
- Excellent concurrency model (goroutines)
- Simple to deploy (single binary)
- Built-in HTTP/2 support
- Strong standard library

### PostgreSQL + pgx

**Benefits:**
- pgx is lightweight, fast, and works directly with prepared statements
- No heavy ORM overhead
- Clear control over queries (sqlc-ready for code generation)
- Strong data integrity (constraints, foreign keys)
- ACID transactions

### Next.js Frontend

**Benefits:**
- React with TypeScript
- File-based routing (App Router)
- Built-in optimization (code splitting, image optimization)
- API routes (if needed in future)
- Excellent developer experience (hot reload)

## Request Flow

```
User Browser
    ↓
Next.js Frontend (Port 3000)
    ↓
HTTP/REST API calls to /api/*
    ↓
Go Gin Backend (Port 8080)
    ↓
Middleware (CORS, logging, auth)
    ↓
Router → Handlers
    ↓
Services (business logic)
    ↓
Repositories (data access)
    ↓
PostgreSQL (Port 5432)
    ↓
Response back through the chain
```

## Backend Layering (Separation of Concerns)

```
┌─────────────────────────────────────┐
│   HTTP Layer (Gin Router)           │
│   - Routes, middleware              │
└────────────────┬────────────────────┘
                 ↓
┌─────────────────────────────────────┐
│   Handlers (handlers/)              │
│   - Parse HTTP requests             │
│   - Validate input                  │
│   - Call services                   │
│   - Return responses                │
└────────────────┬────────────────────┘
                 ↓
┌─────────────────────────────────────┐
│   Services (services/)              │
│   - Business logic                  │
│   - Domain rules                    │
│   - Orchestrate repositories        │
│   - No HTTP concerns                │
└────────────────┬────────────────────┘
                 ↓
┌─────────────────────────────────────┐
│   Repositories (repositories/)       │
│   - Data access (pgx queries)       │
│   - Return data models              │
│   - No business logic               │
└────────────────┬────────────────────┘
                 ↓
┌─────────────────────────────────────┐
│   Models (models/)                  │
│   - Data structures                 │
│   - Role/status constants           │
└─────────────────────────────────────┘
```

**Why this structure?**

1. **Testability** — Each layer can be tested independently with mocks
2. **Maintainability** — Clear responsibility boundaries
3. **Reusability** — Services can be called from different sources (CLI, events, etc.)
4. **Scalability** — Easy to add caching layer, change DB, etc.

## Data Models

### Core Entities

- **User** — System user with login credentials (future)
- **Employee** — Employee record (may or may not have User account)
- **Project** — Construction project (gradilište)
- **ProjectAssignment** — Links employees (Poslovođa) to projects

### Roles

- **Direktor** (Director) — Full system access
- **Inženjer** (Engineer) — Full system access (separate from Direktor)
- **Administracija** (Admin) — Employee management
- **Poslovođa** (Site Manager) — Project, reports, materials
- **Radnik** (Worker) — Employee record, may not have login

See [roles-and-permissions.md](roles-and-permissions.md) for detailed permissions.

## Environment & Deployment

### Local Development

```bash
docker-compose up
```

- Frontend: http://localhost:3000
- Backend: http://localhost:8080
- Database: localhost:5432

### Configuration

All services use **environment variables** for configuration:

```
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=gradiliste
DB_PASSWORD=gradiliste_dev_password
DB_NAME=gradiliste
ENV=development
```

See `services/api/.env.example` and `apps/web/.env.example`.

### Docker Compose

- **PostgreSQL 16** — Alpine image, persistent volume
- **Go API** — Auto-reloaded on file changes (development mode)
- **Next.js** — Optional service (usually run locally)

## Database Schema

### Migrations

Located in `database/migrations/`

- **001_initial_schema.sql** — Core tables, constraints, indexes

Each migration is **idempotent** (safe to re-run).

### Migrations Strategy

1. Create migration: `002_description.sql`
2. Add SQL to the file
3. Run: `psql ... -f migrations/002_description.sql`
4. With docker-compose: Auto-applied on startup
5. Test thoroughly before committing

**Never modify existing migrations** — create new ones for changes.

### Seeds

Located in `database/seeds/`

- **seed_initial.sql** — Test data for development

Safe to re-run (uses `INSERT ... ON CONFLICT`).

## Future Enhancements

### Phase 2 (Modules)

- Authentication & JWT tokens
- Authorization & permission checking
- Logging & monitoring
- Error handling & validation
- API documentation (Swagger/OpenAPI)

### Phase 3 (Features)

- Daily reports
- Material tracking
- Inventory management
- File uploads (S3-compatible storage)
- Excel import/export
- Document management

### Tech Additions

- **Cache Layer** — Redis for sessions, frequently accessed data
- **Search** — Elasticsearch for document searching
- **Async Jobs** — Background task processing
- **Real-time** — WebSockets for live updates
- **Monitoring** — Prometheus metrics, Grafana dashboards
- **Logging** — Structured logging (JSON logs)
- **Rate Limiting** — Protect API from abuse

## Key Files & Their Purpose

| File/Folder | Purpose |
|---|---|
| `docker-compose.yml` | Local dev stack definition |
| `apps/web/` | Next.js frontend source |
| `services/api/main.go` | Backend entry point |
| `services/api/database.go` | DB connection pool setup |
| `services/api/middleware.go` | HTTP middleware (CORS, etc.) |
| `services/api/models/` | Data structures, constants |
| `services/api/handlers/` | HTTP request handlers |
| `services/api/services/` | Business logic |
| `services/api/repositories/` | Data access (pgx queries) |
| `database/migrations/` | Schema versioning |
| `database/seeds/` | Initial test data |
| `docs/` | Architecture, API, DB planning |

## Performance Considerations

### Database

- Connection pooling (25 max, 5 min)
- Indexes on frequently queried columns
- Prepared statements (sqlc-ready)
- Pagination for large result sets

### API

- Request middleware logging
- Error recovery with gin.Recovery()
- CORS properly configured
- Content compression (todo)

### Frontend

- Next.js automatic code splitting
- Image optimization
- API result caching (todo)
- Lazy loading (todo)

## Security Notes

This is a **foundation** — security features are NOT implemented yet:

- ✗ No authentication
- ✗ No authorization checks
- ✗ No input validation
- ✗ No password hashing (yet)
- ✗ No HTTPS (dev only)
- ✗ No CSRF protection
- ✗ No rate limiting

These will be added in Phase 2.

## Next Steps

1. ✅ **Phase 1** — Foundation (you are here)
2. → **Phase 2** — Authentication & Authorization
3. → **Phase 3** — Core Features (projects, employees, daily reports)
4. → **Phase 4** — Advanced Features (materials, inventory, documents)

See [api-plan.md](api-plan.md) and [database-plan.md](database-plan.md) for detailed endpoint and schema planning.
