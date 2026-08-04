-- Migration 020: add client_submission_id for idempotent report creation.
--
-- Without this, a network timeout after the server committed would create a
-- duplicate report on retry.  The column is nullable so existing rows are
-- unaffected.  The partial unique index ensures only one row per
-- (company, submission_id) pair, while NULLs are excluded (UNIQUE does not
-- cover NULL values in Postgres, but the explicit WHERE makes intent clear).

ALTER TABLE daily_reports
  ADD COLUMN IF NOT EXISTS client_submission_id uuid;

CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_reports_client_submission_id
  ON daily_reports (company_id, client_submission_id)
  WHERE client_submission_id IS NOT NULL;
