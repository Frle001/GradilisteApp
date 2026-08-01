-- Migration 022: Private director/engineer comments on worker hours entries
-- Comments are invisible to radnik and poslovoda.

CREATE TABLE IF NOT EXISTS worker_daily_hours_comments (
    id                    UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id            UUID        NOT NULL REFERENCES companies(id)          ON DELETE CASCADE,
    worker_daily_hours_id UUID        NOT NULL REFERENCES worker_daily_hours(id) ON DELETE CASCADE,
    author_user_id        UUID        REFERENCES users(id) ON DELETE SET NULL,
    comment_text          TEXT        NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wdh_comments_entry
    ON worker_daily_hours_comments (worker_daily_hours_id);
CREATE INDEX IF NOT EXISTS idx_wdh_comments_company
    ON worker_daily_hours_comments (company_id);
