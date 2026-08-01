-- Migration 020: Add work_description to worker_daily_hours
-- Poslovođa can optionally describe the work performed when submitting hours.
-- Kept separate from notes (which is a generic free-text memo field).

ALTER TABLE worker_daily_hours
    ADD COLUMN IF NOT EXISTS work_description TEXT;
