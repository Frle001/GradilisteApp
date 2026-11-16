-- Phase 5: Project revenue tracking

CREATE TABLE project_contracts (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    project_id    UUID          NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    contract_type TEXT          NOT NULL DEFAULT 'initial',
    reference     TEXT,
    contract_date DATE,
    net_value     NUMERIC(15,2) NOT NULL,
    notes         TEXT,
    created_by    UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_pc_type CHECK (contract_type IN ('initial', 'amendment'))
);

COMMENT ON TABLE project_contracts IS 'Contracted values per project: initial contract and any amendments.';

CREATE INDEX idx_project_contracts_company ON project_contracts(company_id);
CREATE INDEX idx_project_contracts_project ON project_contracts(project_id);

CREATE TRIGGER trg_project_contracts_updated_at
    BEFORE UPDATE ON project_contracts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE project_invoices (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id         UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    project_id         UUID          NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    invoice_number     TEXT          NOT NULL,
    invoice_date       DATE          NOT NULL,
    due_date           DATE,
    net_amount         NUMERIC(15,2) NOT NULL,
    vat_amount         NUMERIC(15,2) NOT NULL DEFAULT 0,
    gross_amount       NUMERIC(15,2) NOT NULL GENERATED ALWAYS AS (net_amount + vat_amount) STORED,
    status             TEXT          NOT NULL DEFAULT 'issued',
    cancels_invoice_id UUID          REFERENCES project_invoices(id) ON DELETE SET NULL,
    notes              TEXT,
    created_by         UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_pi_status       CHECK (status IN ('issued', 'partial', 'paid', 'cancelled')),
    CONSTRAINT ck_pi_net_positive CHECK (net_amount > 0),
    CONSTRAINT ck_pi_vat_nonneg   CHECK (vat_amount >= 0),
    UNIQUE (company_id, invoice_number)
);

COMMENT ON TABLE project_invoices IS 'Outgoing customer invoices (izlazni racuni) linked to projects. Separate from supplier invoices in company_invoices.';
COMMENT ON COLUMN project_invoices.gross_amount IS 'Computed: net_amount + vat_amount. Stored for query efficiency.';
COMMENT ON COLUMN project_invoices.cancels_invoice_id IS 'If set, this invoice is a corrective/cancellation invoice for the referenced one.';

CREATE INDEX idx_project_invoices_company ON project_invoices(company_id);
CREATE INDEX idx_project_invoices_project ON project_invoices(project_id);
CREATE INDEX idx_project_invoices_date    ON project_invoices(invoice_date DESC);
CREATE INDEX idx_project_invoices_status  ON project_invoices(status);

CREATE TRIGGER trg_project_invoices_updated_at
    BEFORE UPDATE ON project_invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE project_invoice_payments (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    invoice_id   UUID          NOT NULL REFERENCES project_invoices(id) ON DELETE CASCADE,
    payment_date DATE          NOT NULL,
    amount       NUMERIC(15,2) NOT NULL,
    notes        TEXT,
    created_by   UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_pip_amount_positive CHECK (amount > 0)
);

COMMENT ON TABLE project_invoice_payments IS 'Partial or full payments received against outgoing project invoices.';

CREATE INDEX idx_project_invoice_payments_invoice ON project_invoice_payments(invoice_id);
CREATE INDEX idx_project_invoice_payments_company ON project_invoice_payments(company_id);
CREATE INDEX idx_project_invoice_payments_date    ON project_invoice_payments(payment_date DESC);
