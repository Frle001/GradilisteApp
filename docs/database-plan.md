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

### Phase 2 Tables (Implemented)

All tables use **UUID primary keys** with `gen_random_uuid()` for generation.
All include **timestamptz** for proper timezone handling and audit trails.
All include **company_id** on every business table for multi-tenant data isolation.

#### companies

Root table for multi-tenancy. Every other business table references `companies.id`.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | `gen_random_uuid()` |
| name | TEXT NOT NULL | |
| oib | TEXT | Croatian business ID |
| address | TEXT | |
| created_at / updated_at | TIMESTAMPTZ | Auto-managed by trigger |

---

#### employees

All staff records, including radnik (workers who have no login account).

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | |
| company_id | UUID FK → companies | |
| first_name / last_name | TEXT NOT NULL | |
| role | TEXT NOT NULL | `CHECK (role IN ('direktor','inzenjer','administracija','poslovoda','radnik'))` |
| supervisor_id | UUID FK → employees | Self-reference; poslovoda supervises radnik |
| email / phone | TEXT | |
| active | BOOLEAN DEFAULT true | Soft delete |

**Indexes:** company_id, role, active, supervisor_id, unique(company_id, email)

---

#### users

Login accounts only. Radnik does not get a user record.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | |
| company_id | UUID FK → companies | |
| employee_id | UUID FK → employees | Optional link |
| email | TEXT UNIQUE NOT NULL | |
| password_hash | TEXT NOT NULL | bcrypt; placeholder in seeds |
| role | TEXT NOT NULL | `CHECK (role IN ('direktor','inzenjer','administracija','poslovoda'))` |
| active | BOOLEAN DEFAULT true | |
| email_verified | BOOLEAN DEFAULT false | |
| last_login_at | TIMESTAMPTZ | |

---

#### projects

Construction sites (gradilišta). Status lifecycle: `active → closed → archived`.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | |
| company_id | UUID FK | |
| name / address / description | TEXT | |
| status | TEXT | `CHECK (status IN ('active','closed','archived'))` |
| start_date / end_date | DATE | |
| closed_at | TIMESTAMPTZ | |
| closed_by / created_by | UUID FK → users | Audit trail |

---

#### project_assignments

Who works on which project and in what capacity.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | |
| company_id / project_id / employee_id | UUID FKs | |
| role_on_project | TEXT | `CHECK (role_on_project IN ('poslovoda','worker','engineer'))` |
| active | BOOLEAN | Soft remove from project |
| assigned_by | UUID FK → users | |
| UNIQUE | (project_id, employee_id, company_id) | One assignment per person per project |

---

#### project_materials

Project-specific material inventory list, typically imported from Excel.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | |
| company_id / project_id | UUID FKs | |
| material_name / material_code | TEXT | |
| planned_quantity / used_quantity / available_quantity | NUMERIC(12,2) | |
| unit | TEXT NOT NULL | |
| source / source_excel_row | TEXT / INT | Excel import traceability |
| UNIQUE | (project_id, company_id, LOWER(material_name), unit) | |

---

#### daily_reports

Daily work reports submitted by poslovođa.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | |
| company_id / project_id | UUID FKs | |
| poslovoda_id | UUID FK → employees | |
| report_date | DATE NOT NULL | |
| status | TEXT | `CHECK (status IN ('draft','submitted','approved','rejected'))` |
| submitted_by | UUID FK → users | |
| UNIQUE | (project_id, poslovoda_id, report_date, company_id) | |

---

#### daily_report_worker_hours

Hours each worker put in on a specific daily report.

| Column | Type | Notes |
|--------|------|-------|
| daily_report_id | UUID FK ON DELETE CASCADE | |
| worker_id | UUID FK → employees | |
| hours_worked | NUMERIC(5,2) | `CHECK (hours_worked >= 0 AND hours_worked <= 24)` |
| UNIQUE | (daily_report_id, worker_id, company_id) | |

---

#### daily_report_activities

Materials used or work done within a daily report. Supports both project material list and VTK (custom manual) entries.

| Column | Type | Notes |
|--------|------|-------|
| daily_report_id | UUID FK ON DELETE CASCADE | |
| project_material_id | UUID FK NULLABLE | NULL when is_vtk=true |
| custom_material_name | TEXT NULLABLE | NULL when is_vtk=false |
| is_vtk | BOOLEAN | VTK = custom entry not from project list |
| activity_type | TEXT | `CHECK (activity_type IN ('montaza','demontaza','other'))` |
| CHECK | `(is_vtk=false AND project_material_id IS NOT NULL) OR (is_vtk=true AND custom_material_name IS NOT NULL)` | |

---

#### employee_assets

Physical assets (cars, tools, equipment) assigned to individuals.

| Column | Type | Notes |
|--------|------|-------|
| employee_id | UUID FK → employees | |
| asset_type | TEXT | `CHECK (asset_type IN ('car','tool','equipment','other'))` |
| name / quantity / unit / serial_number | various | |
| assigned_by | UUID FK → users | |
| active | BOOLEAN | Soft delete |

---

#### material_purchase_sessions

Groups a shopping trip: buyer, project, optional receipt file.

| Column | Type | Notes |
|--------|------|-------|
| project_id / buyer_id | UUID FKs | |
| receipt_file_url / receipt_file_key | TEXT | S3/storage for future upload |
| receipt_original_filename | TEXT | |
| purchased_at | TIMESTAMPTZ | |

---

#### material_purchase_items

Individual line items within a purchase session.

| Column | Type | Notes |
|--------|------|-------|
| purchase_session_id | UUID FK ON DELETE CASCADE | |
| project_material_id | UUID FK → project_materials | |
| quantity / unit | NUMERIC / TEXT | |

---

#### employee_material_responsibility

Tracks which employee is responsible for which batch of project material.

| Column | Type | Notes |
|--------|------|-------|
| employee_id / project_id / project_material_id | UUID FKs | |
| quantity / unit | NUMERIC / TEXT | |
| source_purchase_session_id | UUID FK NULLABLE | Where it came from |
| active | BOOLEAN | False when transferred away |

**Indexes:** employee_id, project_id, project_material_id, company_id

---

#### asset_transfers

Immutable audit trail of asset/material transfers between employees.

| Column | Type | Notes |
|--------|------|-------|
| from_employee_id / to_employee_id | UUID FKs → employees | |
| asset_type | TEXT | `CHECK (asset_type IN ('car','tool','equipment','material','other'))` |
| employee_asset_id | UUID FK NULLABLE | Required when asset_type != 'material' |
| employee_material_responsibility_id | UUID FK NULLABLE | Required when asset_type = 'material' |
| CHECK | `(asset_type='material' AND emr_id IS NOT NULL) OR (asset_type IN (...) AND asset_id IS NOT NULL)` | |

---

#### import_jobs

Tracks Excel file uploads and their processing status.

| Column | Type | Notes |
|--------|------|-------|
| project_id | UUID FK NULLABLE | |
| import_type | TEXT | `CHECK (import_type IN ('project_materials','employees','other'))` |
| status | TEXT | `CHECK (status IN ('uploaded','parsed','confirmed','failed'))` |
| total_rows / valid_rows / invalid_rows | INTEGER | |

---

#### import_job_rows

Individual rows from an import, with validation details.

| Column | Type | Notes |
|--------|------|-------|
| import_job_id | UUID FK ON DELETE CASCADE | |
| row_number | INTEGER | |
| raw_data | JSONB | Original data |
| normalized_data | JSONB | Cleaned/transformed |
| status | TEXT | `CHECK (status IN ('valid','invalid','imported'))` |

---

#### audit_logs

Comprehensive audit trail. Append-only — no `updated_at`.

| Column | Type | Notes |
|--------|------|-------|
| user_id / employee_id | UUID FKs NULLABLE | Who acted |
| action | TEXT | create, update, delete, approve, reject, etc. |
| entity_type / entity_id | TEXT / UUID | What was affected |
| old_data / new_data | JSONB | Before/after state |
| ip_address / user_agent | TEXT | |

**Indexes:** company_id, user_id, (entity_type, entity_id), created_at, action

---

## Key Design Decisions

1. **UUID Primary Keys** — Better for distributed systems and privacy
2. **Multi-tenancy** — `company_id` on all business tables; never leak data across companies
3. **Timestamps** — All using `timestamptz` with automatic `updated_at` triggers
4. **Soft Deletes** — `active` boolean flags preserve history; nothing is hard-deleted
5. **Audit Trail** — Complete `audit_logs` table with `old_data`/`new_data` JSON
6. **Check Constraints** — Enum values enforced at the database level
7. **Indexing** — Strategic indexes on FK columns, common filters (active, status, date)
8. **Referential Integrity** — Foreign keys with `ON DELETE CASCADE` for child records, `ON DELETE SET NULL` for optional audit references
9. **VTK constraint** — Database-level CHECK ensures VTK activities have custom names and non-VTK activities reference project materials
10. **Employee ≠ User** — Radnik exists as an employee record but has no login credentials

## Migration Files

```
database/migrations/
  001_extensions.sql              — uuid-ossp, pgcrypto, update_updated_at trigger
  002_base_tables.sql             — companies, employees, users
  003_projects.sql                — projects, project_assignments, project_materials
  004_daily_reports.sql           — daily_reports, worker_hours, activities
  005_materials_and_assets.sql    — employee_assets, purchase_sessions, purchase_items,
                                    employee_material_responsibility, asset_transfers
  006_import_and_audit.sql        — import_jobs, import_job_rows, audit_logs
  archive/
    001_initial_schema.sql        — Phase 1 (superseded, kept for reference)
```

Total: **17 tables**

## Applying Migrations

### Docker Compose (automatic)

```bash
docker-compose up
# PostgreSQL runs all .sql files in database/migrations/ on first startup
```

### Manually

```bash
cd database
for f in migrations/00*.sql; do
  psql -h localhost -U gradiliste -d gradiliste -f "$f"
done

# Apply seed data (dev only)
psql -h localhost -U gradiliste -d gradiliste -f seeds/seed_phase2.sql
```

### Connect to running container

```bash
docker-compose exec postgres psql -U gradiliste -d gradiliste
```

## Seeds

### seed_phase2.sql

Populates test data for local development:
- 1 company (Test Construction Company)
- 4 users (direktor, inzenjer, administracija, poslovoda) — passwords are placeholder hashes
- 7 employees (direktor, inzenjer, administracija, poslovoda, 3× radnik)
- Radnik employees have `supervisor_id` pointing to the poslovoda
- 3 projects (all active)
- 3 project assignments (poslovoda on all projects)
- 3 worker project assignments (all 3 radniks on project 1)
- 3 project materials (cement, steel rebar, formwork)
- 3 employee assets (van, tools, safety harness)

**Safe to re-run** — Uses `INSERT ... ON CONFLICT DO NOTHING`

⚠️ **Seed passwords are placeholders** — Replace in production with real bcrypt hashes.

## Debug Endpoint

```
GET /api/debug/db-summary
```

Returns row counts for all main tables. Only registered when `ENV=development`.
Set via `.env` or docker-compose environment variables.

```json
{
  "status": "ok",
  "data": {
    "companies": 1,
    "users": 4,
    "employees": 7,
    "projects": 3,
    "project_assignments": 6,
    "project_materials": 3,
    "daily_reports": 0,
    "employee_assets": 3,
    "material_purchase_sessions": 0,
    "audit_logs": 1
  }
}
```

## Performance Considerations

### Connection Pool (Backend)

```go
config.MaxConns = 25  // Max concurrent connections
config.MinConns = 5   // Min idle connections
```

### Prepared Statements

Always use parameterized queries (pgx does this by default):

```go
// Good
err := db.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&id)

// Bad — SQL injection risk
err := db.QueryRow(ctx, fmt.Sprintf("SELECT id FROM users WHERE email = '%s'", email)).Scan(&id)
```

## Future: sqlc

Ready for integration with **sqlc** (SQL Compiler for Go):

1. Create `sqlc.yaml` in project root
2. Create `queries/` folder with named SQL queries
3. Run `sqlc generate` to produce type-safe Go code
4. Use generated types in repositories

## Backups

### Development

Not critical — recreate from migrations + seeds.

### Production (Future Phase)

- Automated daily backups via `pg_dump`
- Point-in-time recovery (PITR)
- Off-site backup storage
- Regular restore testing

## Access Control

### Development

Single user (`gradiliste`) with full access.

### Production (Future Phase)

```sql
CREATE ROLE app_user LOGIN PASSWORD '...';
GRANT CONNECT ON DATABASE gradiliste TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
```

## Tools & Commands

```bash
# Connect
psql -h localhost -U gradiliste -d gradiliste

# List tables
\dt

# Show table schema
\d employees

# Count rows
SELECT COUNT(*) FROM employees;

# Manual backup
pg_dump -h localhost -U gradiliste gradiliste > backup.sql

# Restore
psql -h localhost -U gradiliste gradiliste < backup.sql
```
