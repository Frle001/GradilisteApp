-- Phase 2 Migration 3: Projects and assignments
-- Core project management and team assignments

CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  address TEXT,
  description TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  start_date DATE,
  end_date DATE,
  closed_at TIMESTAMPTZ,
  closed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (status IN ('active', 'closed', 'archived'))
);

COMMENT ON TABLE projects IS 'Construction projects (gradilišta). Status: active (ongoing), closed (finished), archived (historical)';
COMMENT ON COLUMN projects.status IS 'Project lifecycle: active → closed → archived. Archived projects stay visible but read-only.';
COMMENT ON COLUMN projects.closed_by IS 'User who closed/archived the project for audit trail';

CREATE INDEX idx_projects_company_id ON projects(company_id);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_active ON projects(status) WHERE status = 'active';
CREATE INDEX idx_projects_created_at ON projects(created_at);

CREATE TRIGGER trg_projects_updated_at
  BEFORE UPDATE ON projects
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Project assignments
-- Tracks who is assigned to each project and in what role
CREATE TABLE IF NOT EXISTS project_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  role_on_project TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (role_on_project IN ('poslovoda', 'worker', 'engineer')),
  UNIQUE (project_id, employee_id, company_id)
);

COMMENT ON TABLE project_assignments IS 'Team assignments: who works on which project and in what role';
COMMENT ON COLUMN project_assignments.role_on_project IS 'Role on this specific project: poslovoda (manager), worker, engineer';
COMMENT ON COLUMN project_assignments.active IS 'Soft delete: false means employee was removed from project but history is preserved';

CREATE INDEX idx_project_assignments_company_id ON project_assignments(company_id);
CREATE INDEX idx_project_assignments_project_id ON project_assignments(project_id);
CREATE INDEX idx_project_assignments_employee_id ON project_assignments(employee_id);
CREATE INDEX idx_project_assignments_active ON project_assignments(active);

CREATE TRIGGER trg_project_assignments_updated_at
  BEFORE UPDATE ON project_assignments
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Project materials
-- Unique material list for each project, usually imported from Excel
CREATE TABLE IF NOT EXISTS project_materials (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  material_name TEXT NOT NULL,
  material_code TEXT,
  planned_quantity NUMERIC(12, 2) NOT NULL DEFAULT 0,
  used_quantity NUMERIC(12, 2) NOT NULL DEFAULT 0,
  available_quantity NUMERIC(12, 2) NOT NULL DEFAULT 0,
  unit TEXT NOT NULL,
  source TEXT,
  source_excel_row INTEGER,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (planned_quantity >= 0),
  CHECK (used_quantity >= 0),
  CHECK (available_quantity >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_project_materials_unique_name_unit
ON project_materials (project_id, company_id, LOWER(material_name), unit);

COMMENT ON TABLE project_materials IS 'Project-specific material inventory list, typically imported from Excel';
COMMENT ON COLUMN project_materials.planned_quantity IS 'Quantity planned for this project';
COMMENT ON COLUMN project_materials.used_quantity IS 'Quantity used on site';
COMMENT ON COLUMN project_materials.available_quantity IS 'Remaining quantity';
COMMENT ON COLUMN project_materials.source_excel_row IS 'Original row number in Excel import for traceability';

CREATE INDEX idx_project_materials_company_id ON project_materials(company_id);
CREATE INDEX idx_project_materials_project_id ON project_materials(project_id);
CREATE INDEX idx_project_materials_active ON project_materials(active);
CREATE INDEX idx_project_materials_name_lower ON project_materials(LOWER(material_name));

CREATE TRIGGER trg_project_materials_updated_at
  BEFORE UPDATE ON project_materials
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
