-- Phase 4 improvement: track whether a user must change their temporary password on next login.
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT false;
