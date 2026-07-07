-- Migration 010: Allow multiple daily reports per project/poslovođa/day
-- Drops the unique constraint that limited one report per (project, poslovoda, date, company).
-- Replaces it with a non-unique index so queries on those columns remain fast.

ALTER TABLE daily_reports
  DROP CONSTRAINT daily_reports_project_id_poslovoda_id_report_date_company_i_key;

CREATE INDEX idx_daily_reports_project_poslovoda_date
  ON daily_reports (company_id, project_id, poslovoda_id, report_date);
