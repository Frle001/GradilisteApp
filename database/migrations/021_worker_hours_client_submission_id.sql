-- Migration 021: client_submission_id for idempotent worker-hours submission.
--
-- Without this, a network timeout after the server committed causes the retry
-- to blindly overwrite whatever the server wrote, with no way to detect that
-- the row was already created. The partial unique index prevents two distinct
-- payloads from claiming the same submission ID within a company, while leaving
-- existing rows with NULL unaffected.

ALTER TABLE worker_daily_hours
  ADD COLUMN IF NOT EXISTS client_submission_id uuid;

CREATE UNIQUE INDEX IF NOT EXISTS idx_worker_daily_hours_client_submission_id
  ON worker_daily_hours (company_id, client_submission_id)
  WHERE client_submission_id IS NOT NULL;
