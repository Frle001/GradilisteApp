-- Migration 021: Director/engineer correction history for worker hours

CREATE TABLE IF NOT EXISTS worker_daily_hours_revisions (
    id                    UUID         NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id            UUID         NOT NULL REFERENCES companies(id)           ON DELETE CASCADE,
    worker_daily_hours_id UUID         NOT NULL REFERENCES worker_daily_hours(id)  ON DELETE CASCADE,
    previous_hours        NUMERIC(5,2) NOT NULL,
    new_hours             NUMERIC(5,2) NOT NULL,
    reason                TEXT         NOT NULL,
    changed_by            UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wdh_revisions_entry
    ON worker_daily_hours_revisions (worker_daily_hours_id);
CREATE INDEX IF NOT EXISTS idx_wdh_revisions_company
    ON worker_daily_hours_revisions (company_id);
