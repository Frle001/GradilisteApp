-- Monthly employee contributions for bruto payroll calculations.
-- Bruto = Neto + Contributions (application-defined formula; no statutory tax logic).
-- A NULL record means contributions were not entered (not the same as zero).

BEGIN;

CREATE TABLE employee_monthly_contributions (
    id                   UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id           UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id          UUID          NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    payroll_month        DATE          NOT NULL,  -- always first day of month
    contributions_amount NUMERIC(12,2) NOT NULL CHECK (contributions_amount >= 0),
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by           UUID          REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (company_id, employee_id, payroll_month)
);

CREATE INDEX idx_emp_contrib_company_month  ON employee_monthly_contributions (company_id, payroll_month);
CREATE INDEX idx_emp_contrib_employee_month ON employee_monthly_contributions (company_id, employee_id, payroll_month);

COMMIT;
