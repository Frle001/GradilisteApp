# Database Schema Quick Reference

## Table Relationships (Entity Diagram)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           COMPANIES (Multi-tenant)                       │
└──────────────────┬────────────────────────────────────────────────────────┘
                   │ company_id (all tables)
        ┌──────────┼──────────┬───────────────┬──────────────┐
        │          │          │               │              │
        ▼          ▼          ▼               ▼              ▼
      USERS   EMPLOYEES   PROJECTS    IMPORT_JOBS      AUDIT_LOGS
        │         │ │        │ │
        │   [PK]  │ │        │ │
        │    FK───┘ │        │ │
        │       │   │        │ │
        │       │   │ ┌──────┼─┴──────┐
        │       │   │ │      │        │
        │       │   │ ▼      ▼        ▼
        │       │   └─ PROJECT_ASSIGNMENTS
        │       │        │
        │       │    [FK to employees]
        │       │
        │       ├─────── DAILY_REPORTS ─────┐
        │       │              │             │
        │       │              ▼             ▼
        │       │      DAILY_REPORT_    DAILY_REPORT_
        │       │      WORKER_HOURS     ACTIVITIES
        │       │              │             │
        │       │              │             ├─┐
        │       │              │             │ │
        │       └──────────────┴─────────────┤ └─ PROJECT_MATERIALS
        │
        ├────── EMPLOYEE_ASSETS
        │        │
        │        └─── ASSET_TRANSFERS ◄──┐
        │                    │            │
        │                    └────────────┤
        │                                 │
        ├────── MATERIAL_PURCHASE_        │
        │       SESSIONS                  │
        │        │                        │
        │        ├─ PROJECT_MATERIALS     │
        │        │   │                    │
        │        └─► MATERIAL_PURCHASE_   │
        │            ITEMS                │
        │                                 │
        └──────► EMPLOYEE_MATERIAL_───────┘
                 RESPONSIBILITY
```

## Quick Table Reference

### Core

| Table | Rows | Purpose | Key Fields |
|-------|------|---------|-----------|
| **companies** | 1+ | Organization/tenant | name, oib, address |
| **users** | 4+ | Login accounts | email, role, password_hash |
| **employees** | 7+ | Staff records | role, supervisor_id |

### Projects

| Table | Rows | Purpose | Key Fields |
|-------|------|---------|-----------|
| **projects** | 3+ | Gradilišta | name, status, closed_at |
| **project_assignments** | 10+ | Team on projects | employee_id, role_on_project |
| **project_materials** | 15+ | Project inventory | material_name, planned_quantity |

### Work

| Table | Rows | Purpose | Key Fields |
|-------|------|---------|-----------|
| **daily_reports** | 0+ | Work reports | report_date, status |
| **daily_report_worker_hours** | 0+ | Hours logged | worker_id, hours_worked |
| **daily_report_activities** | 0+ | Work activities | activity_type, is_vtk |

### Materials

| Table | Rows | Purpose | Key Fields |
|-------|------|---------|-----------|
| **material_purchase_sessions** | 0+ | Purchase batches | buyer_id, receipt_file_url |
| **material_purchase_items** | 0+ | Purchased items | quantity, unit |
| **employee_material_responsibility** | 0+ | Assigned materials | employee_id, quantity |

### Assets

| Table | Rows | Purpose | Key Fields |
|-------|------|---------|-----------|
| **employee_assets** | 3+ | Cars, tools | asset_type, serial_number |
| **asset_transfers** | 0+ | Transfer history | from_employee_id, to_employee_id |

### Admin

| Table | Rows | Purpose | Key Fields |
|-------|------|---------|-----------|
| **import_jobs** | 0+ | Excel uploads | import_type, status |
| **import_job_rows** | 0+ | Import rows | raw_data, status |
| **audit_logs** | 10+ | Activity log | action, entity_type, old_data |

## Field Patterns

### IDs
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `company_id UUID NOT NULL REFERENCES companies(id)`

### Timestamps
- `created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP` ← AUTO-UPDATED

### Enums (Check Constraints)
- `role TEXT CHECK (role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik'))`
- `status TEXT CHECK (status IN ('draft', 'submitted', 'approved', 'rejected'))`

### Foreign Keys
- `REFERENCES company(id) ON DELETE CASCADE`
- `REFERENCES users(id) ON DELETE SET NULL`
- `UNIQUE (project_id, employee_id, company_id)`

### Soft Deletes
- `active BOOLEAN NOT NULL DEFAULT true`
- Query: `SELECT * FROM employees WHERE active = true`

## Indexes

### By Type

**Foreign Keys:**
- employees(company_id, supervisor_id)
- project_assignments(project_id, employee_id)
- daily_reports(project_id, poslovoda_id)

**Filters:**
- employees(role, active)
- projects(status, active)
- daily_reports(status)
- daily_report_activities(is_vtk)

**Text Search:**
- employees(LOWER(email))
- project_materials(LOWER(material_name))

**Temporal:**
- daily_reports(report_date)
- asset_transfers(transferred_at)
- audit_logs(created_at)

## Constraints

### Domain

| Constraint | Enforced | Example |
|-----------|----------|---------|
| Role values | CHECK | users.role IN ('direktor', 'inzenjer', ...) |
| Project status | CHECK | projects.status IN ('active', 'closed', ...) |
| Hours range | CHECK | daily_report_worker_hours >= 0 AND <= 24 |
| Quantities | CHECK | quantities >= 0 |

### Uniqueness

| Constraint | Tables |
|-----------|--------|
| Global | users.email |
| Per company+project | project_assignments |
| Per report+worker | daily_report_worker_hours |
| Per report+date | daily_reports (project_id + poslovoda_id + date) |

### Referential

| Parent | Child | Delete Strategy |
|--------|-------|------------------|
| companies | all | CASCADE (orphaned data deleted) |
| employees | assignments, reports | CASCADE |
| projects | materials, reports | CASCADE |
| daily_reports | worker_hours, activities | CASCADE |
| material_purchase_sessions | items | CASCADE |

## Common Queries

### Get project team
```sql
SELECT e.first_name, e.last_name, pa.role_on_project
FROM project_assignments pa
JOIN employees e ON pa.employee_id = e.id
WHERE pa.project_id = $1 AND pa.active = true;
```

### Get employee's projects
```sql
SELECT p.name, pa.role_on_project, pa.assigned_at
FROM project_assignments pa
JOIN projects p ON pa.project_id = p.id
WHERE pa.employee_id = $1 AND pa.active = true;
```

### Get daily report details
```sql
SELECT 
  dr.report_date,
  COALESCE(SUM(drwh.hours_worked), 0) as total_hours,
  COUNT(DISTINCT drwh.worker_id) as workers,
  COUNT(dra.id) as activities
FROM daily_reports dr
LEFT JOIN daily_report_worker_hours drwh ON dr.id = drwh.daily_report_id
LEFT JOIN daily_report_activities dra ON dr.id = dra.daily_report_id
WHERE dr.project_id = $1
GROUP BY dr.id, dr.report_date
ORDER BY dr.report_date DESC;
```

### Get material accountability
```sql
SELECT 
  em.quantity,
  pm.unit,
  e.first_name,
  e.last_name,
  em.active
FROM employee_material_responsibility em
JOIN project_materials pm ON em.project_material_id = pm.id
JOIN employees e ON em.employee_id = e.id
WHERE em.project_id = $1 AND pm.id = $2;
```

### Audit trail
```sql
SELECT action, entity_type, old_data, new_data, created_at
FROM audit_logs
WHERE entity_id = $1 AND entity_type = 'project'
ORDER BY created_at DESC;
```

## Connection Info

**Local Development**
```
Host: localhost
Port: 5432
User: gradiliste
Password: gradiliste_dev_password
Database: gradiliste
```

**Docker Compose**
```
Host: postgres
Port: 5432
(same credentials)
```

**pgx Connection String**
```go
connStr := "postgres://gradiliste:gradiliste_dev_password@localhost:5432/gradiliste?sslmode=disable"
```

## Performance Tuning

### For High Volume

```sql
-- Analyze query plans
EXPLAIN ANALYZE SELECT * FROM daily_reports WHERE project_id = $1;

-- Add partitioning (daily_reports by date)
CREATE TABLE daily_reports_2026_q2 PARTITION OF daily_reports
  FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');

-- Archive old data
UPDATE employee_material_responsibility SET active = false
WHERE created_at < CURRENT_DATE - INTERVAL '1 year';
```

### Connection Pool Tuning

```go
config.MaxConns = 25  // Adjust based on load
config.MinConns = 5
config.MaxConnLifetime = 10 * time.Minute
```

## Testing

### Seed Data UUIDs

```
Companies:   10000000-0000-0000-0000-000000000001
Employees:   20000000-0000-0000-0000-00000000000X (1-7)
Users:       30000000-0000-0000-0000-00000000000X (1-4)
Projects:    40000000-0000-0000-0000-00000000000X (1-3)
Assignments: 50000000-0000-0000-0000-00000000000X (1-12)
Materials:   60000000-0000-0000-0000-00000000000X (1-3)
Assets:      70000000-0000-0000-0000-00000000000X (1-3)
```

### Quick Test Queries

```bash
# Connect
psql -h localhost -U gradiliste -d gradiliste

# Count tables
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';

# Count records
SELECT 
  (SELECT COUNT(*) FROM companies) as companies,
  (SELECT COUNT(*) FROM users) as users,
  (SELECT COUNT(*) FROM employees) as employees,
  (SELECT COUNT(*) FROM projects) as projects;
```

## Migration Checklist

- [x] Extensions (uuid, pgcrypto)
- [x] Base tables (companies, users, employees)
- [x] Projects (projects, assignments, materials)
- [x] Daily reports (reports, hours, activities)
- [x] Materials & assets (purchases, transfers, responsibility)
- [x] Import & audit (import jobs, audit logs)
- [x] Triggers (updated_at)
- [x] Indexes (FKs, filters, search)
- [x] Constraints (checks, unique)
- [x] Seed data
- [x] Documentation
