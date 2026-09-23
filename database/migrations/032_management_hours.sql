-- Migration 032: Management-role self-hours
-- Allows direktor, inženjer, and administracija to record their own working hours
-- without a mandatory project assignment.

-- 1. Make project_id nullable so management entries can omit a project.
ALTER TABLE worker_daily_hours ALTER COLUMN project_id DROP NOT NULL;

-- 2. Drop the old single unique constraint that required project_id NOT NULL.
ALTER TABLE worker_daily_hours DROP CONSTRAINT worker_daily_hours_unique_per_day;

-- 3. Worker/poslovoda entries: one row per (company, worker, project, date).
--    The partial index only covers rows where project_id is present,
--    preserving the original uniqueness guarantee for existing data.
CREATE UNIQUE INDEX worker_daily_hours_unique_worker
    ON worker_daily_hours (company_id, worker_id, project_id, work_date)
    WHERE project_id IS NOT NULL;

-- 4. Management entries: one row per (company, worker, date) with no project.
--    This prevents a director or engineer from double-logging the same day.
CREATE UNIQUE INDEX worker_daily_hours_unique_management
    ON worker_daily_hours (company_id, worker_id, work_date)
    WHERE project_id IS NULL;
