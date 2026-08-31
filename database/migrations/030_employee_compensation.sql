CREATE TABLE IF NOT EXISTS employee_compensation_plans (
    id                  UUID          NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id          UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id         UUID          NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    pay_type            TEXT          NOT NULL CHECK (pay_type IN ('fixed_monthly', 'hourly')),
    pay_amount          NUMERIC(12,2) NOT NULL CHECK (pay_amount > 0),
    company_cost_amount NUMERIC(12,2)           CHECK (company_cost_amount IS NULL OR company_cost_amount > 0),
    effective_from      DATE          NOT NULL,
    effective_to        DATE,
    created_by          UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT effective_to_after_from CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX IF NOT EXISTS idx_emp_comp_company   ON employee_compensation_plans (company_id);
CREATE INDEX IF NOT EXISTS idx_emp_comp_employee  ON employee_compensation_plans (company_id, employee_id);
CREATE INDEX IF NOT EXISTS idx_emp_comp_effective ON employee_compensation_plans (company_id, employee_id, effective_from);
