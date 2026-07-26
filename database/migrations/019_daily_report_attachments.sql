-- Migration 019: Daily report image attachments
-- Photos uploaded per-report. Soft-deleted (deleted_at) so storage cleanup can be async.

CREATE TABLE IF NOT EXISTS daily_report_attachments (
    id               UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id       UUID        NOT NULL REFERENCES companies(id)     ON DELETE CASCADE,
    daily_report_id  UUID        NOT NULL REFERENCES daily_reports(id) ON DELETE CASCADE,
    uploaded_by      UUID                 REFERENCES users(id)         ON DELETE SET NULL,
    file_key         TEXT        NOT NULL,
    original_name    TEXT        NOT NULL,
    content_type     TEXT        NOT NULL,
    file_size        BIGINT      NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_daily_report_attachments_report_id ON daily_report_attachments (daily_report_id);
CREATE INDEX idx_daily_report_attachments_company_id ON daily_report_attachments (company_id);
CREATE INDEX idx_daily_report_attachments_active     ON daily_report_attachments (daily_report_id, company_id) WHERE deleted_at IS NULL;
