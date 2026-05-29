# Database Migrations

SQL migration files for PostgreSQL schema.

## Current Migrations

### Phase 2 (Complete Schema)

1. **001_extensions.sql** — PostgreSQL extensions and trigger functions
2. **002_base_tables.sql** — Companies, users, employees
3. **003_projects.sql** — Projects, assignments, materials
4. **004_daily_reports.sql** — Daily reports and work tracking
5. **005_materials_and_assets.sql** — Material tracking, assets, transfers
6. **006_import_and_audit.sql** — Import jobs and audit logs

**Total tables: 17**
**Features:**
- Multi-company/multi-tenant support
- Complete audit trail
- Employee vs User separation
- Material accountability tracking
- Asset transfer history
- Excel import workflow tracking

## Applying Migrations

### With Docker Compose

Migrations in this folder are automatically applied on first startup via the `docker-entrypoint-initdb.d` volume mount.

### Manually

```bash
psql -h localhost -U gradiliste -d gradiliste -f migrations/001_initial_schema.sql
```

Or:

```bash
psql -h localhost -U gradiliste -d gradiliste < migrations/001_initial_schema.sql
```

## Notes

- Each migration should be **idempotent** (can run multiple times safely)
- Use `IF NOT EXISTS` and `IF EXISTS` clauses
- Keep migrations focused on one logical change
- Never modify existing migrations; create new ones for changes
- Use transactions where appropriate

## Future Tools

When ready, consider using:
- **Flyway** — Java-based migration tool
- **Migrate** — Go-based migration tool
- **sqlc** — SQL compiler for Go
