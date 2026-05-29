# Phase 2 Verification Checklist

Use this checklist to verify the database setup is working correctly.

## 1. Start Services

```bash
cd c:/Projects/ProjektGradiliste
docker-compose up
```

- [ ] Postgres starts: `accepting connections`
- [ ] API starts: `listening on :8080`
- [ ] No errors in logs

## 2. Verify Database Connection

```bash
curl http://localhost:8080/api/health
```

Expected:
```json
{
  "status": "ok",
  "message": "Gradiliste API is running"
}
```

- [ ] Returns 200 status

```bash
curl http://localhost:8080/api/db-health
```

Expected:
```json
{
  "status": "ok",
  "message": "Database connection is working"
}
```

- [ ] Returns 200 status

## 3. Check Database Summary

```bash
curl http://localhost:8080/api/debug/db-summary
```

Expected (with seed data):
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

- [ ] Returns 200 status
- [ ] Row counts match expected values
- [ ] All tables exist (no errors)

## 4. Connect via psql

```bash
docker-compose exec postgres psql -U gradiliste -d gradiliste
```

### Inside psql:

List all tables:
```sql
\dt
```

- [ ] Shows 18 tables (including audit_logs, daily_reports, etc.)

Check companies:
```sql
SELECT * FROM companies;
```

- [ ] Shows 1 test company with name "Test Construction Company"

Check users:
```sql
SELECT email, role FROM users;
```

- [ ] Shows 4 users: direktor, inzenjer, administracija, poslovoda

Check employees:
```sql
SELECT first_name, last_name, role FROM employees;
```

- [ ] Shows 7 employees (Marko, Jelena, Miroslav, Dragan, Ivan, Petar, Ana)
- [ ] Roles include direktor, inzenjer, administracija, poslovoda, radnik (3x)

Check projects:
```sql
SELECT name, status FROM projects;
```

- [ ] Shows 3 projects:
  - Nove poslovne prostorije
  - Obnova starog objekta
  - Infrastrukturni radovi

Check project assignments:
```sql
SELECT COUNT(*) FROM project_assignments;
```

- [ ] Shows 6 assignments (3 poslovoda + 3 workers)

Check materials:
```sql
SELECT material_name, unit FROM project_materials;
```

- [ ] Shows 3 materials: cement, armatura, oplata

Check employee assets:
```sql
SELECT asset_type, name FROM employee_assets;
```

- [ ] Shows 3 assets: car, tool, equipment

Check audit logs:
```sql
SELECT action, entity_type FROM audit_logs;
```

- [ ] Shows seed insertion audit log

Exit psql:
```sql
\q
```

## 5. Verify Relationships

### Project team
```sql
SELECT e.first_name, e.last_name, p.name, pa.role_on_project
FROM project_assignments pa
JOIN employees e ON pa.employee_id = e.id
JOIN projects p ON pa.project_id = p.id
WHERE pa.active = true;
```

- [ ] Shows assignments of poslovoda and workers to projects

### Poslovoda's projects
```sql
SELECT p.name
FROM projects p
JOIN project_assignments pa ON p.id = pa.project_id
JOIN employees e ON pa.employee_id = e.id
WHERE e.first_name = 'Dragan' AND pa.role_on_project = 'poslovoda';
```

- [ ] Shows 3 projects (Dragan is poslovoda on all)

### Project materials
```sql
SELECT material_name, planned_quantity, unit
FROM project_materials
WHERE project_id = (SELECT id FROM projects WHERE name = 'Nove poslovne prostorije');
```

- [ ] Shows 3 materials with quantities

## 6. Verify Constraints

### Test enum validation
```sql
INSERT INTO employees (company_id, first_name, last_name, role)
VALUES ('10000000-0000-0000-0000-000000000001'::uuid, 'Test', 'User', 'invalid_role');
```

- [ ] Error: `violates check constraint "employees_role_check"`

### Test required fields
```sql
INSERT INTO users (company_id, email, password_hash, role)
VALUES ('10000000-0000-0000-0000-000000000001'::uuid, null, 'hash', 'direktor');
```

- [ ] Error: `null value in column "email" violates not-null constraint`

### Test hours validation
```sql
INSERT INTO daily_report_worker_hours (company_id, daily_report_id, worker_id, hours_worked)
VALUES ('10000000-0000-0000-0000-000000000001'::uuid, 'fake-uuid'::uuid, 'fake-uuid'::uuid, 25);
```

- [ ] Error: `violates check constraint "daily_report_worker_hours_hours_worked_check"`

## 7. Test Soft Deletes

```sql
UPDATE employees SET active = false WHERE first_name = 'Test';
SELECT COUNT(*) FROM employees WHERE active = true;
SELECT COUNT(*) FROM employees WHERE active = false;
```

- [ ] Inactive employees excluded from queries with WHERE active = true

## 8. Test Triggers

```sql
SELECT updated_at FROM employees LIMIT 1;
UPDATE employees SET first_name = 'Updated' WHERE id = (SELECT id FROM employees LIMIT 1);
SELECT updated_at FROM employees LIMIT 1;
```

- [ ] updated_at timestamp was automatically updated

## 9. Verify Indexes

```sql
\di
```

- [ ] Shows indexes on:
  - Foreign keys (FK columns)
  - Filter columns (role, status, active)
  - Search columns (email)
  - Date columns (created_at, report_date)

## 10. Test API Connection from Frontend

In frontend terminal:
```bash
cd apps/web
npm run dev
```

Visit http://localhost:3000

- [ ] Home page loads
- [ ] "Backend API Status" shows "connected"

## Summary

- [ ] All 18 tables exist
- [ ] Seed data loaded (1 company, 4 users, 7 employees, 3 projects, etc.)
- [ ] Foreign keys work
- [ ] Constraints enforced
- [ ] Triggers auto-update timestamps
- [ ] Soft deletes work
- [ ] Indexes created
- [ ] API endpoints respond
- [ ] Frontend can connect

## Troubleshooting

### Migrations didn't run
```bash
docker-compose exec postgres psql -U gradiliste -d gradiliste -f /docker-entrypoint-initdb.d/001_extensions.sql
docker-compose exec postgres psql -U gradiliste -d gradiliste -f /docker-entrypoint-initdb.d/002_base_tables.sql
# ... etc
```

### Seed data missing
```bash
docker-compose exec postgres psql -U gradiliste -d gradiliste -f /docker-entrypoint-initdb.d/seed_phase2.sql
```

### API can't connect to DB
- Check docker-compose logs: `docker-compose logs postgres`
- Verify DB_HOST is "postgres" in container (or "localhost" for local)
- Wait 10 seconds after postgres starts before running API

### Tables don't exist
- Check if migrations ran: `\dt` in psql
- Check if any SQL errors: `docker-compose logs postgres | grep ERROR`

## Next Steps

✅ **Phase 2 complete** — Database is ready for business logic  
→ **Phase 3** — Implement repository layer and CRUD endpoints  
→ **Phase 4** — Build frontend screens
