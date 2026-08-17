-- Migration 028: Employee documentation and HR compliance tracking

-- Uploaded file metadata (reused across all document types)
CREATE TABLE IF NOT EXISTS employee_documents (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id      UUID        NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    storage_key      TEXT        NOT NULL,
    original_filename TEXT        NOT NULL,
    mime_type        TEXT        NOT NULL,
    file_size        BIGINT      NOT NULL,
    uploaded_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by      UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_emp_documents_company    ON employee_documents(company_id);
CREATE INDEX IF NOT EXISTS idx_emp_documents_employee   ON employee_documents(employee_id);

-- Liječnički pregled (history preserved)
CREATE TABLE IF NOT EXISTS employee_medical_exams (
    id             UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id     UUID  NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id    UUID  NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    completed_date DATE  NOT NULL,
    expires_at     DATE  NOT NULL,
    document_id    UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     UUID  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT ck_medical_exam_dates CHECK (expires_at >= completed_date)
);
CREATE INDEX IF NOT EXISTS idx_emp_medical_company  ON employee_medical_exams(company_id);
CREATE INDEX IF NOT EXISTS idx_emp_medical_employee ON employee_medical_exams(employee_id, completed_date DESC);

-- Zaštita na radu (one record per employee; upserted on upload/verify)
CREATE TABLE IF NOT EXISTS employee_safety_records (
    id          UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID  NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id UUID  NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    status      TEXT  NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','uploaded','verified')),
    document_id UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    verified_at TIMESTAMPTZ,
    verified_by UUID  REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_safety_employee UNIQUE (company_id, employee_id)
);
CREATE INDEX IF NOT EXISTS idx_emp_safety_company ON employee_safety_records(company_id);

-- Ugovor o radu (history preserved; termination stored in-row)
CREATE TABLE IF NOT EXISTS employee_contracts (
    id                      UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id              UUID  NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id             UUID  NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    contract_type           TEXT  NOT NULL CHECK (contract_type IN ('odredeno','neodredeno')),
    expires_at              DATE,
    document_id             UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    terminated_at           DATE,
    termination_document_id UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    terminated_by           UUID  REFERENCES users(id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT ck_contract_odredeno CHECK (
        (contract_type = 'odredeno'  AND expires_at IS NOT NULL) OR
        (contract_type = 'neodredeno')
    )
);
CREATE INDEX IF NOT EXISTS idx_emp_contracts_company  ON employee_contracts(company_id);
CREATE INDEX IF NOT EXISTS idx_emp_contracts_employee ON employee_contracts(employee_id, created_at DESC);

-- Platne liste + Evidencija radnika (shared monthly obligation table)
CREATE TABLE IF NOT EXISTS employee_monthly_obligations (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id     UUID          NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    year            INTEGER       NOT NULL,
    month           INTEGER       NOT NULL CHECK (month BETWEEN 1 AND 12),
    obligation_type TEXT          NOT NULL CHECK (obligation_type IN ('platna_lista','evidencija_radnika')),
    status          TEXT          NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','completed')),
    hours           NUMERIC(7,2),
    document_id     UUID          REFERENCES employee_documents(id) ON DELETE SET NULL,
    completed_at    TIMESTAMPTZ,
    completed_by    UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by      UUID          NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_monthly_obligation UNIQUE (company_id, employee_id, year, month, obligation_type)
);
CREATE INDEX IF NOT EXISTS idx_emp_monthly_company  ON employee_monthly_obligations(company_id);
CREATE INDEX IF NOT EXISTS idx_emp_monthly_employee ON employee_monthly_obligations(employee_id, year DESC, month DESC);

-- Godišnji odmor (history preserved)
CREATE TABLE IF NOT EXISTS employee_annual_leave (
    id          UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID  NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id UUID  NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    date_from   DATE,
    date_to     DATE,
    document_id UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID  NOT NULL REFERENCES users(id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_emp_leave_company  ON employee_annual_leave(company_id);
CREATE INDEX IF NOT EXISTS idx_emp_leave_employee ON employee_annual_leave(employee_id, created_at DESC);

-- Radne dozvole (history preserved; latest by expires_at determines notification)
CREATE TABLE IF NOT EXISTS employee_work_permits (
    id          UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID  NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id UUID  NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    entered_at  DATE  NOT NULL,
    expires_at  DATE  NOT NULL,
    document_id UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT ck_work_permit_dates CHECK (expires_at >= entered_at)
);
CREATE INDEX IF NOT EXISTS idx_emp_permits_company  ON employee_work_permits(company_id);
CREATE INDEX IF NOT EXISTS idx_emp_permits_employee ON employee_work_permits(employee_id, expires_at DESC);

-- Mirovinsko osiguranje (one record per employee)
CREATE TABLE IF NOT EXISTS employee_pension_records (
    id          UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID  NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id UUID  NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    status      TEXT  NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','uploaded','verified')),
    document_id UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    verified_at TIMESTAMPTZ,
    verified_by UUID  REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_pension_employee UNIQUE (company_id, employee_id)
);
CREATE INDEX IF NOT EXISTS idx_emp_pension_company ON employee_pension_records(company_id);

-- Zdravstveno osiguranje (one record per employee)
CREATE TABLE IF NOT EXISTS employee_health_records (
    id          UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID  NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_id UUID  NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    status      TEXT  NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','uploaded','verified')),
    document_id UUID  REFERENCES employee_documents(id) ON DELETE SET NULL,
    verified_at TIMESTAMPTZ,
    verified_by UUID  REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  UUID  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_health_employee UNIQUE (company_id, employee_id)
);
CREATE INDEX IF NOT EXISTS idx_emp_health_company ON employee_health_records(company_id);
