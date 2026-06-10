# Database Backup Plan

## Manual backup

```bash
# Custom format (preferred — supports selective restore)
pg_dump -Fc -h localhost -U gradiliste gradiliste \
  > backup_$(date +%Y%m%d_%H%M%S).dump

# With Docker
docker exec gradiliste_db pg_dump -Fc -U gradiliste gradiliste \
  > backup_$(date +%Y%m%d_%H%M%S).dump
```

## Restore

```bash
# From custom format
pg_restore -Fc -h localhost -U gradiliste -d gradiliste backup.dump

# Into Docker container
cat backup.dump | docker exec -i gradiliste_db pg_restore -U gradiliste -d gradiliste
```

## Recommended retention

| Frequency | Keep for |
|---|---|
| Daily | 7 days |
| Weekly | 4 weeks |
| Monthly | 3 months |

## Automated backups

### Option A — Managed hosting (simplest)

Fly.io Postgres, Railway, Render, Neon, and Supabase all include automated daily backups with point-in-time recovery. Enable it in the provider dashboard and verify the retention window.

### Option B — Cron + object storage (VPS)

Add to crontab (`crontab -e`) on the server:
```cron
0 3 * * * docker exec gradiliste_db pg_dump -Fc -U gradiliste gradiliste | \
  gzip > /var/backups/gradiliste_$(date +\%Y\%m\%d).dump.gz && \
  find /var/backups -name "*.dump.gz" -mtime +30 -delete
```

Copy backups off-server to S3/Backblaze B2/Cloudflare R2:
```bash
aws s3 cp /var/backups/gradiliste_$(date +%Y%m%d).dump.gz \
  s3://your-bucket/db-backups/ --storage-class STANDARD_IA
```

## Verifying backups

Restore to a local test DB monthly to confirm integrity:
```bash
createdb gradiliste_test
pg_restore -Fc -h localhost -U gradiliste -d gradiliste_test backup.dump
psql -h localhost -U gradiliste gradiliste_test \
  -c "SELECT COUNT(*) FROM projects; SELECT COUNT(*) FROM daily_reports;"
dropdb gradiliste_test
```

## Receipt/file storage backups

If using `UPLOAD_STORAGE_DRIVER=local`, the uploads directory must be backed up separately:
```bash
tar -czf uploads_$(date +%Y%m%d).tar.gz /app/uploads/
```

For production, use `UPLOAD_STORAGE_DRIVER=s3` with an S3-compatible provider that supports versioning and cross-region replication. Local storage is not safe for production if the server is ephemeral.

## Before running migrations in production

Always take a full backup before applying migrations:
```bash
pg_dump -Fc -h $DB_HOST -U $DB_USER $DB_NAME > pre_migration_backup.dump
```
