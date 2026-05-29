-- Phase 2 Migration 1: Extensions and setup
-- Creates UUID support and trigger function for updated_at timestamps

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Updated_at trigger function
-- Automatically sets updated_at to current timestamp on UPDATE
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to verify database connectivity
-- Used by backend health checks
CREATE OR REPLACE FUNCTION verify_db_ready()
RETURNS TABLE (
  extension_uuid_ossp BOOLEAN,
  extension_pgcrypto BOOLEAN,
  function_update_updated_at BOOLEAN,
  function_verify_db_ready BOOLEAN
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp'),
    EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto'),
    EXISTS(SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at_column'),
    EXISTS(SELECT 1 FROM pg_proc WHERE proname = 'verify_db_ready');
END;
$$ LANGUAGE plpgsql;
