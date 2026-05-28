# Gradilište API - Backend

Go Gin backend for the construction company management application.

## Stack

- **Go 1.21** — Programming language
- **Gin** — Web framework
- **pgx** — PostgreSQL driver with connection pooling
- **godotenv** — Environment variable management

## Setup

```bash
# Copy environment variables
cp .env.example .env

# Install dependencies
go mod download

# Run server
go run main.go
```

Server runs on `http://localhost:8080`

## Folder Structure

```
├── main.go            # Entry point
├── database.go        # DB initialization and connection pool
├── middleware.go      # HTTP middleware (CORS, logging, etc.)
├── models/            # Data models and constants
├── handlers/          # HTTP request handlers
├── services/          # Business logic
├── repositories/      # Data access layer
├── utils/             # Utility functions
└── Dockerfile         # Container image
```

## Architecture

### Request Flow

```
HTTP Request
    ↓
Middleware (CORS, logging)
    ↓
Router → Handlers
    ↓
Services (business logic)
    ↓
Repositories (data access)
    ↓
PostgreSQL Database
```

### Patterns

**Handlers** — Thin layer that:
- Parses HTTP requests
- Validates input
- Calls services
- Returns HTTP responses

**Services** — Contains business logic:
- Orchestrates repositories
- Implements domain rules
- No HTTP concerns

**Repositories** — Data access layer:
- Direct pgx queries
- Prepared statements (sqlc-ready)
- Returns data models
- No business logic

**Models** — Data structures:
- Database models (match schema)
- Constants (roles, statuses)
- API DTOs (future)

## Database

Connection configured via environment variables:
- `DB_HOST` — Database host
- `DB_PORT` — Database port (default: 5432)
- `DB_USER` — Database user
- `DB_PASSWORD` — Database password
- `DB_NAME` — Database name

Uses pgx connection pool with:
- 25 max connections
- 5 min connections
- Connection pooling and reuse

## API Endpoints

### Health Check
```
GET /api/health
```

Returns:
```json
{
  "status": "ok",
  "message": "Gradilište API is running"
}
```

### Future Modules

- **Auth** — `/api/auth/login`, `/api/auth/register`
- **Employees** — `/api/employees`
- **Projects** — `/api/projects`
- **Daily Reports** — `/api/reports`
- **Materials** — `/api/materials`
- **Inventory** — `/api/inventory`

## Environment Variables

See `.env.example` for all available options.

## Development

### Adding a New Module

1. Create repository in `repositories/`
2. Create service in `services/`
3. Create handlers in `handlers/`
4. Add routes in `main.go`
5. Add models in `models/` if needed

### Database Migrations

Migrations are in `database/migrations/`

Apply manually:
```bash
psql -h localhost -U gradiliste -d gradiliste -f migrations/001_initial_schema.sql
```

Or with docker-compose (auto-applied on startup)

## Docker

Build image:
```bash
docker build -t gradiliste-api .
```

Run container:
```bash
docker run --env-file .env -p 8080:8080 gradiliste-api
```

With docker-compose (recommended):
```bash
docker-compose up api
```

## Future Enhancements

- [ ] Authentication & JWT tokens
- [ ] Input validation & sanitization
- [ ] Error handling & logging
- [ ] Rate limiting
- [ ] Database transactions
- [ ] Unit tests
- [ ] Integration tests
- [ ] API documentation (Swagger/OpenAPI)
- [ ] sqlc code generation
- [ ] Caching layer (Redis)
