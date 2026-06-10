# Database Backups and Restore

## Automatic backups (Render PostgreSQL)

Render managed PostgreSQL automatically creates daily backups retained for 7 days (Starter plan) or 30 days (Standard plan).

**To access backups:** Render Dashboard → PostgreSQL → Backups tab

## Manual backup

Create a snapshot before any destructive operation (migration, bulk update, rollback):

```bash
# Export via pg_dump (requires psql client and the External Database URL)
pg_dump "$DATABASE_URL" --format=custom --file="backup_$(date +%Y%m%d_%H%M%S).dump"
```

Store the dump file in a safe location outside the Render container (e.g. local machine, cloud storage, encrypted archive).

## Restore from backup

```bash
# Restore to the same or a new database
pg_restore --dbname="$DATABASE_URL" --clean --no-owner backup_20240101_120000.dump
```

> `--clean` drops existing objects before restoring. Use with caution on a live database.
> For production restores, take the service offline first, restore, verify, then bring back up.

## Pre-migration checklist

Before applying any migration to production:

1. Trigger a manual backup (pg_dump) and verify it is not empty
2. Apply the migration to staging first and verify the app works
3. Review the migration for irreversible operations (DROP TABLE, DROP COLUMN, NOT NULL without default)
4. Apply to production during low-traffic window
5. Immediately smoke-test critical paths after the migration

## Receipt file backup (R2)

R2 provides eleven nines of durability by default. For an additional backup layer, use `rclone` to mirror objects to another bucket:

```bash
rclone copy r2:gradiliste-receipts r2:gradiliste-receipts-backup --progress
```

Configure rclone with R2 credentials using the S3-compatible provider.

## Staging data policy

- Never copy production data to staging without anonymizing employee names, emails, and company details.
- Use the `create-admin` tool to create synthetic test accounts in staging.
- Staging database may be wiped and re-seeded at any time.

## Retention policy

| Data | Retention |
|---|---|
| Database backups (Render) | 7 days (Starter) / 30 days (Standard) |
| Manual pg_dump snapshots | Keep indefinitely for pre-migration checkpoints |
| R2 receipt files | Indefinitely while company is active |
