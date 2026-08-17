-- Migration 027: Leasing extension for company assets (Zaduženja module)
-- Adds leasing fields to vehicles and a monthly payment completion history table.

-- ── Extend company_assets with leasing fields (vozilo only) ──────────────────

ALTER TABLE company_assets
    ADD COLUMN IF NOT EXISTS is_leasing       BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS leasing_company  TEXT,
    ADD COLUMN IF NOT EXISTS leasing_end_date DATE;

COMMENT ON COLUMN company_assets.is_leasing       IS 'True when this vehicle is under a leasing contract.';
COMMENT ON COLUMN company_assets.leasing_company  IS 'Name of the leasing company; only meaningful when is_leasing = TRUE.';
COMMENT ON COLUMN company_assets.leasing_end_date IS 'Date when the leasing contract ends; no new reminders generated for months after this date.';

-- ── Monthly leasing payment history ──────────────────────────────────────────

CREATE TABLE IF NOT EXISTS company_asset_leasing_payments (
    id               UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id       UUID        NOT NULL REFERENCES companies(id)      ON DELETE CASCADE,
    company_asset_id UUID        NOT NULL REFERENCES company_assets(id) ON DELETE CASCADE,
    period_month     DATE        NOT NULL, -- always normalised to first day of month (YYYY-MM-01)
    completed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_by     UUID        NOT NULL REFERENCES users(id)          ON DELETE RESTRICT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_leasing_payment_asset_month UNIQUE (company_asset_id, period_month)
);

COMMENT ON TABLE  company_asset_leasing_payments             IS 'Monthly leasing obligation completion records for the Zaduženja module.';
COMMENT ON COLUMN company_asset_leasing_payments.period_month   IS 'Calendar month being completed, stored as first day of that month (YYYY-MM-01).';
COMMENT ON COLUMN company_asset_leasing_payments.completed_by   IS 'User ID (administracija) who marked this payment as completed.';

CREATE INDEX IF NOT EXISTS idx_leasing_payments_company
    ON company_asset_leasing_payments (company_id);

CREATE INDEX IF NOT EXISTS idx_leasing_payments_asset
    ON company_asset_leasing_payments (company_asset_id);

CREATE INDEX IF NOT EXISTS idx_leasing_payments_asset_period
    ON company_asset_leasing_payments (company_asset_id, period_month DESC);
