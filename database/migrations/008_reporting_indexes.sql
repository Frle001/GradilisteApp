-- Phase 8 Migration: Reporting indexes
-- Compound indexes optimised for Građevinski dnevnik and Građevinska knjiga queries.
-- Single-column indexes on these tables already exist from earlier migrations;
-- these add multi-column indexes that the query planner will prefer for the
-- date-range + company_id + status scans used by the reporting endpoints.

-- daily_reports: company-scoped date range (most common reporting scan)
CREATE INDEX IF NOT EXISTS idx_daily_reports_company_date
  ON daily_reports (company_id, report_date DESC);

-- daily_reports: company + poslovoda (poslovoda-role isolation)
CREATE INDEX IF NOT EXISTS idx_daily_reports_company_poslovoda_date
  ON daily_reports (company_id, poslovoda_id, report_date DESC);

-- daily_reports: company + status (status filter)
CREATE INDEX IF NOT EXISTS idx_daily_reports_company_status
  ON daily_reports (company_id, status);

-- daily_reports: company + project (project filter)
CREATE INDEX IF NOT EXISTS idx_daily_reports_company_project
  ON daily_reports (company_id, project_id);

-- daily_report_worker_hours: join path from report to worker
CREATE INDEX IF NOT EXISTS idx_drwh_report_worker
  ON daily_report_worker_hours (daily_report_id, worker_id);

-- daily_report_activities: join path from report
CREATE INDEX IF NOT EXISTS idx_dra_report_id_company
  ON daily_report_activities (daily_report_id, company_id);

-- daily_report_activities: VTK filter
CREATE INDEX IF NOT EXISTS idx_dra_is_vtk
  ON daily_report_activities (company_id, is_vtk);

-- daily_report_activities: activity_type filter
CREATE INDEX IF NOT EXISTS idx_dra_activity_type
  ON daily_report_activities (company_id, activity_type);

-- employees: role lookup for filter-options
CREATE INDEX IF NOT EXISTS idx_employees_company_role
  ON employees (company_id, role);

-- employees: supervisor lookup (workers under a poslovoda)
CREATE INDEX IF NOT EXISTS idx_employees_company_supervisor
  ON employees (company_id, supervisor_id);

-- project_materials: active materials per project
CREATE INDEX IF NOT EXISTS idx_project_materials_project_active
  ON project_materials (project_id, active)
  WHERE active = true;

-- projects: compound lookup for filter-options
CREATE INDEX IF NOT EXISTS idx_projects_company_status_name
  ON projects (company_id, status, name);
