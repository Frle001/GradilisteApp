-- Migration 017: Company work schedule — project shifts and assignments

CREATE TABLE project_shifts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID        NOT NULL REFERENCES companies(id)  ON DELETE CASCADE,
    project_id   UUID        NOT NULL REFERENCES projects(id)   ON DELETE CASCADE,
    shift_date   DATE        NOT NULL,
    start_time   TIME        NOT NULL,
    end_time     TIME        NOT NULL,
    notes        TEXT,
    status       TEXT        NOT NULL DEFAULT 'active'
                                 CHECK (status IN ('active', 'cancelled')),
    created_by   UUID        NOT NULL REFERENCES users(id)      ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    CHECK (start_time < end_time),
    CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL))
);

CREATE INDEX idx_project_shifts_company_date  ON project_shifts (company_id, shift_date);
CREATE INDEX idx_project_shifts_project_date  ON project_shifts (project_id, shift_date);

CREATE TRIGGER trg_project_shifts_updated_at
    BEFORE UPDATE ON project_shifts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE project_shift_assignments (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           UUID        NOT NULL REFERENCES companies(id)  ON DELETE CASCADE,
    shift_id             UUID        NOT NULL REFERENCES project_shifts(id) ON DELETE CASCADE,
    employee_id          UUID        NOT NULL REFERENCES employees(id)  ON DELETE CASCADE,
    assigned_by          UUID        NOT NULL REFERENCES users(id)      ON DELETE RESTRICT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    overlap_overridden    BOOLEAN     NOT NULL DEFAULT false,
    overlap_overridden_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    overlap_overridden_at TIMESTAMPTZ,
    UNIQUE (shift_id, employee_id),
    CHECK (
        NOT overlap_overridden
        OR (overlap_overridden_by IS NOT NULL AND overlap_overridden_at IS NOT NULL)
    )
);

CREATE INDEX idx_shift_assignments_shift    ON project_shift_assignments (shift_id);
CREATE INDEX idx_shift_assignments_employee ON project_shift_assignments (employee_id, company_id);
