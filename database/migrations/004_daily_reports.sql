-- Phase 2 Migration 4: Daily reports and work tracking
-- Poslovođa creates daily reports with worker hours and activities

CREATE TABLE IF NOT EXISTS daily_reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  poslovoda_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  report_date DATE NOT NULL,
  status TEXT NOT NULL DEFAULT 'submitted',
  notes TEXT,
  submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (status IN ('draft', 'submitted', 'approved', 'rejected')),
  UNIQUE (project_id, poslovoda_id, report_date, company_id)
);

COMMENT ON TABLE daily_reports IS 'Daily work reports submitted by poslovođa (site manager) for each project';
COMMENT ON COLUMN daily_reports.status IS 'Report lifecycle: draft (in progress), submitted (ready for review), approved, rejected';
COMMENT ON COLUMN daily_reports.poslovoda_id IS 'The poslovođa who submitted this report';

CREATE INDEX idx_daily_reports_company_id ON daily_reports(company_id);
CREATE INDEX idx_daily_reports_project_id ON daily_reports(project_id);
CREATE INDEX idx_daily_reports_poslovoda_id ON daily_reports(poslovoda_id);
CREATE INDEX idx_daily_reports_report_date ON daily_reports(report_date);
CREATE INDEX idx_daily_reports_status ON daily_reports(status);

CREATE TRIGGER trg_daily_reports_updated_at
  BEFORE UPDATE ON daily_reports
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Daily report worker hours
-- Records how many hours each worker worked on a specific report
CREATE TABLE IF NOT EXISTS daily_report_worker_hours (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  daily_report_id UUID NOT NULL REFERENCES daily_reports(id) ON DELETE CASCADE,
  worker_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  hours_worked NUMERIC(5, 2) NOT NULL,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (hours_worked >= 0 AND hours_worked <= 24),
  UNIQUE (daily_report_id, worker_id, company_id)
);

COMMENT ON TABLE daily_report_worker_hours IS 'Worker hours recorded in a daily report';
COMMENT ON COLUMN daily_report_worker_hours.hours_worked IS 'Hours worked today (0-24)';

CREATE INDEX idx_daily_report_worker_hours_company_id ON daily_report_worker_hours(company_id);
CREATE INDEX idx_daily_report_worker_hours_daily_report_id ON daily_report_worker_hours(daily_report_id);
CREATE INDEX idx_daily_report_worker_hours_worker_id ON daily_report_worker_hours(worker_id);

CREATE TRIGGER trg_daily_report_worker_hours_updated_at
  BEFORE UPDATE ON daily_report_worker_hours
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Daily report activities
-- Records what work/materials were used on a specific day
-- Can reference project_materials (standard list) or be VTK (custom manual entry)
CREATE TABLE IF NOT EXISTS daily_report_activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  daily_report_id UUID NOT NULL REFERENCES daily_reports(id) ON DELETE CASCADE,
  project_material_id UUID REFERENCES project_materials(id) ON DELETE SET NULL,
  custom_material_name TEXT,
  quantity NUMERIC(12, 2) NOT NULL,
  unit TEXT NOT NULL,
  activity_type TEXT NOT NULL,
  is_vtk BOOLEAN NOT NULL DEFAULT false,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (activity_type IN ('montaza', 'demontaza', 'other')),
  CHECK (
    (is_vtk = false AND project_material_id IS NOT NULL) OR
    (is_vtk = true AND custom_material_name IS NOT NULL)
  )
);

COMMENT ON TABLE daily_report_activities IS 'Activity entries in a daily report: materials used, installed, removed, etc.';
COMMENT ON COLUMN daily_report_activities.project_material_id IS 'References standard project material (NULL if VTK)';
COMMENT ON COLUMN daily_report_activities.custom_material_name IS 'Custom material name if VTK (NULL if from project_materials)';
COMMENT ON COLUMN daily_report_activities.is_vtk IS 'VTK = custom/manual entry (not from project material list)';
COMMENT ON COLUMN daily_report_activities.activity_type IS 'montaza (installation), demontaza (removal), other (documentation, cleanup, etc.)';

CREATE INDEX idx_daily_report_activities_company_id ON daily_report_activities(company_id);
CREATE INDEX idx_daily_report_activities_daily_report_id ON daily_report_activities(daily_report_id);
CREATE INDEX idx_daily_report_activities_project_material_id ON daily_report_activities(project_material_id);

CREATE TRIGGER trg_daily_report_activities_updated_at
  BEFORE UPDATE ON daily_report_activities
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
