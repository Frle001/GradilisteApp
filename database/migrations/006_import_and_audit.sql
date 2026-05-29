-- Phase 2 Migration 6: Import tracking and audit logs
-- Excel import workflow and comprehensive audit trail

-- Import jobs
-- Tracks Excel file uploads and processing status
CREATE TABLE IF NOT EXISTS import_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
  import_type TEXT NOT NULL,
  original_filename TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'uploaded',
  total_rows INTEGER NOT NULL DEFAULT 0,
  valid_rows INTEGER NOT NULL DEFAULT 0,
  invalid_rows INTEGER NOT NULL DEFAULT 0,
  error_message TEXT,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (import_type IN ('project_materials', 'employees', 'other')),
  CHECK (status IN ('uploaded', 'parsed', 'confirmed', 'failed'))
);

COMMENT ON TABLE import_jobs IS 'Tracks Excel imports: status, validation, error tracking';
COMMENT ON COLUMN import_jobs.import_type IS 'Type of import: project_materials, employees, other';
COMMENT ON COLUMN import_jobs.status IS 'Lifecycle: uploaded (received), parsed (validated), confirmed (applied), failed';

CREATE INDEX idx_import_jobs_company_id ON import_jobs(company_id);
CREATE INDEX idx_import_jobs_project_id ON import_jobs(project_id);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);
CREATE INDEX idx_import_jobs_created_at ON import_jobs(created_at);

CREATE TRIGGER trg_import_jobs_updated_at
  BEFORE UPDATE ON import_jobs
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Import job rows
-- Individual rows from an import, with validation status and errors
CREATE TABLE IF NOT EXISTS import_job_rows (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  import_job_id UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
  row_number INTEGER NOT NULL,
  raw_data JSONB NOT NULL,
  normalized_data JSONB,
  status TEXT NOT NULL,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (status IN ('valid', 'invalid', 'imported'))
);

COMMENT ON TABLE import_job_rows IS 'Individual rows from an import job, with validation details';
COMMENT ON COLUMN import_job_rows.raw_data IS 'Original data as received (JSON)';
COMMENT ON COLUMN import_job_rows.normalized_data IS 'Cleaned/transformed data (JSON)';

CREATE INDEX idx_import_job_rows_company_id ON import_job_rows(company_id);
CREATE INDEX idx_import_job_rows_import_job_id ON import_job_rows(import_job_id);
CREATE INDEX idx_import_job_rows_status ON import_job_rows(status);

-- Audit logs
-- Comprehensive audit trail for important actions
CREATE TABLE IF NOT EXISTS audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  employee_id UUID REFERENCES employees(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id UUID,
  old_data JSONB,
  new_data JSONB,
  ip_address TEXT,
  user_agent TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE audit_logs IS 'Complete audit trail of important system actions for compliance and debugging';
COMMENT ON COLUMN audit_logs.action IS 'Action performed: create, update, delete, approve, reject, etc.';
COMMENT ON COLUMN audit_logs.entity_type IS 'What was affected: project, employee, report, material, etc.';
COMMENT ON COLUMN audit_logs.entity_id IS 'ID of affected entity';
COMMENT ON COLUMN audit_logs.old_data IS 'Previous state (for updates)';
COMMENT ON COLUMN audit_logs.new_data IS 'New state (for updates)';

CREATE INDEX idx_audit_logs_company_id ON audit_logs(company_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_employee_id ON audit_logs(employee_id);
CREATE INDEX idx_audit_logs_entity_type_id ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
