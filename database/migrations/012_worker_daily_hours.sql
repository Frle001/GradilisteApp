-- Migration 012: Standalone worker daily hours table
-- Radnik submits/updates their own working hours per project per day.
-- This is separate from daily_report_worker_hours which Poslovođa manages.

CREATE TABLE IF NOT EXISTS worker_daily_hours (
    id           UUID                     NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id   UUID                     NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    worker_id    UUID                     NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    project_id   UUID                     NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    work_date    DATE                     NOT NULL,
    hours_worked NUMERIC(5, 2)            NOT NULL CHECK (hours_worked >= 0 AND hours_worked <= 24),
    notes        TEXT,
    submitted_by UUID                     REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ              NOT NULL DEFAULT NOW(),

    CONSTRAINT worker_daily_hours_unique_per_day
        UNIQUE (company_id, worker_id, project_id, work_date)
);

CREATE INDEX IF NOT EXISTS idx_worker_daily_hours_company_worker_date
    ON worker_daily_hours (company_id, worker_id, work_date);

CREATE INDEX IF NOT EXISTS idx_worker_daily_hours_company_project_date
    ON worker_daily_hours (company_id, project_id, work_date);

CREATE OR REPLACE FUNCTION update_worker_daily_hours_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgname = 'trg_worker_daily_hours_updated_at'
          AND tgrelid = 'worker_daily_hours'::regclass
    ) THEN
        CREATE TRIGGER trg_worker_daily_hours_updated_at
        BEFORE UPDATE ON worker_daily_hours
        FOR EACH ROW EXECUTE FUNCTION update_worker_daily_hours_updated_at();
    END IF;
END;
$$;
