-- Migration 038: Central warehouse (Skladište) module
-- Central stock registry with moving weighted-average cost valuation.
-- Atomic warehouse↔project transfers reference the existing project_materials table.

-- ── Stock balances (one row per material per company) ─────────────────────────
CREATE TABLE IF NOT EXISTS warehouse_balances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    material_name   VARCHAR(255) NOT NULL,
    material_code   VARCHAR(100),
    unit            VARCHAR(50) NOT NULL,
    quantity        NUMERIC(14, 4) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    avg_cost        NUMERIC(12, 4) NOT NULL DEFAULT 0 CHECK (avg_cost >= 0),
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, LOWER(material_name), unit)
);

-- ── Receipt documents (Primke – incoming goods) ───────────────────────────────
CREATE TABLE IF NOT EXISTS warehouse_receipts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    receipt_number  VARCHAR(100),
    supplier        VARCHAR(255),
    receipt_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    notes           TEXT,
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Receipt line items ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS warehouse_receipt_lines (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id              UUID NOT NULL REFERENCES warehouse_receipts(id) ON DELETE CASCADE,
    warehouse_balance_id    UUID NOT NULL REFERENCES warehouse_balances(id),
    quantity                NUMERIC(14, 4) NOT NULL CHECK (quantity > 0),
    unit_price              NUMERIC(12, 4) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Movement audit log ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS warehouse_movements (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id              UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    warehouse_balance_id    UUID NOT NULL REFERENCES warehouse_balances(id),
    movement_type           VARCHAR(50) NOT NULL CHECK (movement_type IN ('receipt', 'issue', 'return', 'adjustment')),
    quantity_delta          NUMERIC(14, 4) NOT NULL,
    quantity_before         NUMERIC(14, 4) NOT NULL,
    quantity_after          NUMERIC(14, 4) NOT NULL,
    unit_price              NUMERIC(12, 4) NOT NULL DEFAULT 0,
    avg_cost_before         NUMERIC(12, 4) NOT NULL DEFAULT 0,
    avg_cost_after          NUMERIC(12, 4) NOT NULL DEFAULT 0,
    receipt_line_id         UUID REFERENCES warehouse_receipt_lines(id),
    project_id              UUID REFERENCES projects(id),
    project_material_id     UUID REFERENCES project_materials(id),
    notes                   TEXT,
    created_by              UUID NOT NULL REFERENCES users(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Indexes ───────────────────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_warehouse_balances_company     ON warehouse_balances(company_id) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_warehouse_movements_company    ON warehouse_movements(company_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_warehouse_movements_balance    ON warehouse_movements(warehouse_balance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_warehouse_receipts_company     ON warehouse_receipts(company_id, created_at DESC);
