# Database Seeds

Seed files to populate the database with initial/test data.

## File Naming Convention

`seed_*.sql` or similar descriptive names.

## Applying Seeds

After migrations are applied, seed the database:

```bash
psql -h localhost -U gradiliste -d gradiliste -f seeds/seed_initial.sql
```

Or with docker-compose (if mapped):

```bash
docker-compose exec postgres psql -U gradiliste -d gradiliste -f /seeds/seed_initial.sql
```

## Notes

- Seeds are separate from migrations
- Safe to re-run (use `INSERT ... ON CONFLICT` or `IF NOT EXISTS`)
- Use for initial test data and fixtures
- Never use seeds for schema changes
- Keep seed data realistic for testing purposes

## Development vs Production

- **Dev/Local:** Use full seed files with test data
- **Production:** Minimal or no seeds (use one-time setup scripts)

This is controlled by environment or manually.
