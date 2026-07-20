-- Enforces at most one active poslovoda per project at the database level.
--
-- Pre-deployment guard: run this query before applying the migration.
-- It must return 0 rows. If it returns any rows, resolve the duplicate active
-- poslovoda assignments for those projects before continuing.
--
-- SELECT company_id, project_id, COUNT(*) AS active_poslovoda_count
-- FROM project_assignments
-- WHERE role_on_project = 'poslovoda' AND active = true
-- GROUP BY company_id, project_id
-- HAVING COUNT(*) > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_project_active_poslovoda
    ON project_assignments (company_id, project_id)
    WHERE role_on_project = 'poslovoda' AND active = true;
