# Database Plan

## Overview

PostgreSQL 16 database with clean schema, migrations, and seeds.

## Connection Details

### Development

```
Host: localhost
Port: 5432
User: gradiliste
Password: gradiliste_dev_password
Database: gradiliste
```

### Docker Compose

```
Host: postgres
Port: 5432
(same credentials)
```

### Connection String (pgx)

```
postgres://gradiliste:gradiliste_dev_password@localhost:5432/gradiliste?sslmode=disable
```

## Schema Overview

### Current Tables (001_initial_schema.sql)

#### users

System users for authentication.

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    password_hash VARCHAR(255),
    role VARCHAR(50) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

**Indexes:**
- `idx_users_email` — Fast lookups by email
- `idx_users_role` — Filter users by role
- `idx_users_active` — Find active users

**Constraints:**
- `role` must be one of: 'direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik'

---

#### employees

Employee records (may or may not have User accounts).

```sql
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

**Indexes:**
- `idx_employees_role` — Filter employees by role
- `idx_employees_active` — Find active employees

**Notes:**
- Independent of `users` table (employee ≠ user)
- Can represent contractors or workers without login
- Administracija manages this table

---

#### projects

Construction projects (gradilišta).

```sql
CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

**Indexes:**
- `idx_projects_active` — Find active projects

**Notes:**
- Basic project info
- Extended details added in future phases (budget, timeline, location, etc.)

---

#### project_assignments

Junction table linking employees to projects.

```sql
CREATE TABLE project_assignments (
    id SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    employee_id INTEGER NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'poslovoda',
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, employee_id)
)
```

**Indexes:**
- `idx_project_assignments_project` — Find assignments for a project
- `idx_project_assignments_employee` — Find projects for an employee

**Constraints:**
- Foreign keys ensure referential integrity
- Unique constraint prevents duplicate assignments
- Cascade delete removes assignments if project/employee deleted

**Notes:**
- Represents Poslovođa or team members assigned to projects
- Currently only stores assignment (future: add end_date, status, etc.)

---

## Future Phases

### Phase 2 (Auth Enhancement)

```sql
ALTER TABLE users ADD COLUMN last_login TIMESTAMP;
ALTER TABLE users ADD COLUMN failed_login_attempts INT DEFAULT 0;
ALTER TABLE users ADD COLUMN locked_until TIMESTAMP;

-- Add foreign key linking user to employee (optional)
ALTER TABLE users ADD COLUMN employee_id INT REFERENCES employees(id);
```

### Phase 3 (Daily Reports)

```sql
CREATE TABLE daily_reports (
    id SERIAL PRIMARY KEY,
    project_id INT NOT NULL REFERENCES projects(id),
    created_by INT NOT NULL REFERENCES employees(id),
    date DATE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE daily_report_workers (
    id SERIAL PRIMARY KEY,
    report_id INT NOT NULL REFERENCES daily_reports(id) ON DELETE CASCADE,
    employee_id INT NOT NULL REFERENCES employees(id),
    hours DECIMAL(5,2) NOT NULL,
    PRIMARY KEY (report_id, employee_id)
);
```

### Phase 3 (Material Tracking)

```sql
CREATE TABLE materials (
    id SERIAL PRIMARY KEY,
    project_id INT NOT NULL REFERENCES projects(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity DECIMAL(10,2),
    unit VARCHAR(50),
    cost DECIMAL(10,2),
    date DATE NOT NULL,
    created_by INT REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE material_receipts (
    id SERIAL PRIMARY KEY,
    material_id INT NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    file_url VARCHAR(500),
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Phase 4 (Inventory)

```sql
CREATE TABLE inventory_items (
    id SERIAL PRIMARY KEY,
    employee_id INT NOT NULL REFERENCES employees(id),
    name VARCHAR(255) NOT NULL,
    quantity INT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE inventory_transfers (
    id SERIAL PRIMARY KEY,
    item_id INT NOT NULL REFERENCES inventory_items(id),
    from_employee_id INT REFERENCES employees(id),
    to_employee_id INT REFERENCES employees(id),
    quantity INT NOT NULL,
    date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    notes TEXT
);
```

### Phase 4 (Documents)

```sql
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    project_id INT REFERENCES projects(id),
    document_type VARCHAR(50),
    file_url VARCHAR(500),
    uploaded_by INT REFERENCES employees(id),
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Migrations

Stored in `database/migrations/`

### File Structure

- **001_initial_schema.sql** — Core tables (users, employees, projects, assignments)
- **002_add_daily_reports.sql** — Daily report tables
- **003_add_materials.sql** — Material tracking
- **004_add_inventory.sql** — Inventory management
- **005_add_documents.sql** — Document storage

### Migration Rules

1. **Idempotent** — Use `IF NOT EXISTS` / `IF EXISTS` clauses
2. **Reversible** — Can be rolled back if needed
3. **Tested** — Apply in dev environment first
4. **Minimal** — One logical change per migration
5. **Transactional** — Wrap in `BEGIN ... COMMIT` where appropriate

### Applying Migrations

#### With Docker Compose

Files in `database/migrations/` are automatically applied on startup via `docker-entrypoint-initdb.d`:

```bash
docker-compose up
# Postgres initializes with all migrations
```

#### Manually

```bash
psql -h localhost -U gradiliste -d gradiliste -f migrations/001_initial_schema.sql
```

Or:

```bash
cd database
for f in migrations/*.sql; do
  psql -h localhost -U gradiliste -d gradiliste -f "$f"
done
```

#### Connect to Running Container

```bash
docker-compose exec postgres psql -U gradiliste -d gradiliste
```

## Seeds

Stored in `database/seeds/`

### seed_initial.sql

Populates initial test data:
- 4 test users (Direktor, Inženjer, Admin, Poslovođa)
- 7 test employees
- 3 test projects
- Project assignments (Poslovođa → Projects)

**Safe to re-run** — Uses `INSERT ... ON CONFLICT DO NOTHING`

### Applying Seeds

```bash
psql -h localhost -U gradiliste -d gradiliste -f seeds/seed_initial.sql
```

Or with Docker:

```bash
docker-compose exec postgres psql -U gradiliste -d gradiliste -f /docker-entrypoint-initdb.d/seed_initial.sql
```

## Data Types

### IDs

- `SERIAL PRIMARY KEY` — Auto-incrementing integers
- Future: Consider `BIGSERIAL` for very large tables or `UUID` for distributed systems

### Dates & Times

- `DATE` — For specific dates (daily reports, material dates)
- `TIMESTAMP` — With timezone info (created_at, updated_at)
- `TIMESTAMP DEFAULT CURRENT_TIMESTAMP` — Auto-set on insert

### Enums

Currently stored as VARCHAR with CHECK constraints:

```sql
role VARCHAR(50) NOT NULL CHECK (role IN ('direktor', 'inzenjer', ...))
```

**Future:** Could migrate to PostgreSQL ENUM type for better type safety:

```sql
CREATE TYPE role_type AS ENUM ('direktor', 'inzenjer', ...);
ALTER TABLE users ADD COLUMN role role_type;
```

### Soft Deletes (Future)

Currently using `active BOOLEAN` flag for soft deletes.

Future: Consider `deleted_at TIMESTAMP` for audit trails:

```sql
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP;
```

## Indexes

Strategic indexes on frequently queried columns:

- **Primary keys** — Automatic
- **Foreign keys** — On `*_id` columns (faster joins)
- **Active status** — Users/employees/projects filtered by active
- **Roles** — Filter by role
- **Dates** — Daily reports filtered by date range

**Rule:** Index before profiling shows need, but don't over-index (write penalty).

## Performance Considerations

### Connection Pool (Backend)

```go
config.MaxConns = 25  // Max concurrent connections
config.MinConns = 5   // Min idle connections
```

Adjust based on load testing.

### Prepared Statements

Always use parameterized queries (pgx does this):

```go
// Good
var id int
err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&id)

// Bad (vulnerable to SQL injection)
err := db.QueryRow(fmt.Sprintf("SELECT id FROM users WHERE email = '%s'", email)).Scan(&id)
```

### Query Optimization (Phase 3+)

- Use `EXPLAIN` to analyze slow queries
- Add indexes for frequently filtered/joined columns
- Consider denormalization for complex reports

### Monitoring (Phase 3+)

- Log slow queries (> 500ms)
- Monitor connection pool usage
- Track lock contention

## Backups

### Development

Not critical (easily recreate from migrations + seeds).

### Production

**Not yet implemented** — Phase 3+ task

- Automated daily backups
- Point-in-time recovery (PITR)
- Regular restore testing
- Off-site backup storage

Example:

```bash
# Manual backup
pg_dump -h localhost -U gradiliste gradiliste > backup.sql

# Restore
psql -h localhost -U gradiliste gradiliste < backup.sql
```

## Access Control

### Development

Single user (`gradiliste`) with full access.

### Production

**Future:** Create separate roles:

```sql
CREATE ROLE app_user LOGIN PASSWORD '...';
GRANT CONNECT ON DATABASE gradiliste TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
```

## Disaster Recovery

### Phase 3+

- Automated backups
- Standby replicas
- Failover procedures
- RTO/RPO targets

## Schema Versioning

Tracked via migrations folder. Each migration is numbered sequentially.

Version = highest migration number applied.

Example:
- Applied: 001, 002, 003
- Version: 003
- Next: 004

## Tools & Commands

### Connection

```bash
psql -h localhost -U gradiliste -d gradiliste
```

### List Tables

```sql
\dt
```

### Show Table Schema

```sql
\d users
```

### Show Indexes

```sql
\di
```

### Run Query

```bash
psql -h localhost -U gradiliste -d gradiliste -c "SELECT * FROM users;"
```

### Export Data

```bash
pg_dump -h localhost -U gradiliste gradiliste > backup.sql
```

### Import Data

```bash
psql -h localhost -U gradiliste gradiliste < backup.sql
```

## Future: sqlc

Prepared for integration with **sqlc** (SQL Compiler for Go):

1. Create `sqlc.yaml` in project root
2. Create `queries/` folder with named queries
3. Run `sqlc generate` to create Go code
4. Use generated types and functions in repositories

This eliminates manual SQL-to-Go type mapping.

Example workflow:

```sql
-- queries/users.sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users WHERE active = true ORDER BY created_at DESC;
```

Then in Go:

```go
user, err := q.GetUserByEmail(ctx, email)
users, err := q.ListUsers(ctx)
```
