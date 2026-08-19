-- ── Company Invoices (Računi) ─────────────────────────────────────────────────

CREATE TABLE company_invoices (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id        UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    invoice_type      TEXT        NOT NULL,
    supplier          TEXT,
    leasing_company   TEXT,
    storage_key       TEXT        NOT NULL,
    original_filename TEXT        NOT NULL,
    mime_type         TEXT        NOT NULL,
    file_size         BIGINT      NOT NULL CHECK (file_size > 0),
    created_by        UUID        NOT NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_invoice_type CHECK (
        invoice_type IN ('materijal', 'gorivo', 'leasing', 'alati', 'oprema')
    ),
    CONSTRAINT ck_invoice_supplier CHECK (
        supplier IS NULL OR supplier IN ('Energy Centar', 'Pondeljak', 'Lipa Promet')
    ),
    CONSTRAINT ck_invoice_leasing_company CHECK (
        leasing_company IS NULL OR leasing_company IN ('Impuls', 'Unicredit Leasing')
    ),
    CONSTRAINT ck_materijal_requires_supplier CHECK (
        invoice_type != 'materijal' OR supplier IS NOT NULL
    ),
    CONSTRAINT ck_leasing_requires_company CHECK (
        invoice_type != 'leasing' OR leasing_company IS NOT NULL
    ),
    CONSTRAINT ck_supplier_only_for_materijal CHECK (
        invoice_type = 'materijal' OR supplier IS NULL
    ),
    CONSTRAINT ck_leasing_company_only_for_leasing CHECK (
        invoice_type = 'leasing' OR leasing_company IS NULL
    )
);

CREATE INDEX idx_company_invoices_company    ON company_invoices(company_id);
CREATE INDEX idx_company_invoices_type       ON company_invoices(invoice_type);
CREATE INDEX idx_company_invoices_created_by ON company_invoices(created_by);
CREATE INDEX idx_company_invoices_created_at ON company_invoices(created_at DESC);

-- ── R1 Receipts ───────────────────────────────────────────────────────────────

CREATE TABLE r1_receipts (
    id                UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id        UUID           NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    submitted_by      UUID           NOT NULL REFERENCES users(id),
    price             NUMERIC(12, 2) NOT NULL CHECK (price > 0),
    storage_key       TEXT           NOT NULL,
    original_filename TEXT           NOT NULL,
    mime_type         TEXT           NOT NULL,
    file_size         BIGINT         NOT NULL CHECK (file_size > 0),
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_r1_receipts_company      ON r1_receipts(company_id);
CREATE INDEX idx_r1_receipts_submitted_by ON r1_receipts(submitted_by);
CREATE INDEX idx_r1_receipts_created_at   ON r1_receipts(created_at DESC);
