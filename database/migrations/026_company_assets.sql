-- Migration 026: Company assets registry (Zaduženja module)
-- A company-level registry of tools (alat), equipment (oprema), and vehicles (vozilo).
-- Assets are owned by the company and may optionally be assigned to an employee.

CREATE TABLE IF NOT EXISTS company_assets (
    id                      UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id              UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    assigned_employee_id    UUID        REFERENCES employees(id) ON DELETE SET NULL,

    asset_type              VARCHAR     NOT NULL CHECK (asset_type IN ('alat', 'oprema', 'vozilo')),
    name                    TEXT        NOT NULL,
    status                  VARCHAR     NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    notes                   TEXT,

    -- Shared: purchase date and warranty expiry (alat + oprema)
    purchased_at            DATE,
    warranty_expires_at     DATE,

    -- Oprema only: clothing/equipment size
    size                    VARCHAR     CHECK (size IN ('XS', 'S', 'M', 'L', 'XL', 'XXL', '3XL', 'Univerzalna')),

    -- Vozilo only: registration fields
    registration_plate      TEXT,
    registration_date       DATE,
    registration_expires_at DATE,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_company_assets_reg_dates
        CHECK (
            registration_expires_at IS NULL
            OR registration_date IS NULL
            OR registration_expires_at >= registration_date
        )
);

COMMENT ON TABLE company_assets IS 'Company-level registry of tools, equipment, and vehicles (Zaduženja module).';
COMMENT ON COLUMN company_assets.asset_type IS 'alat = hand tool; oprema = equipment/clothing; vozilo = vehicle';
COMMENT ON COLUMN company_assets.assigned_employee_id IS 'Currently assigned employee; NULL means unassigned (Nezaduženo).';
COMMENT ON COLUMN company_assets.status IS 'active = in use; inactive = decommissioned/retired';
COMMENT ON COLUMN company_assets.size IS 'Clothing/equipment size; only meaningful for oprema type.';
COMMENT ON COLUMN company_assets.purchased_at IS 'Purchase date; used for alat and oprema.';
COMMENT ON COLUMN company_assets.warranty_expires_at IS 'Warranty expiry date; used for alat and oprema.';
COMMENT ON COLUMN company_assets.registration_plate IS 'Vehicle registration plate; only for vozilo type.';
COMMENT ON COLUMN company_assets.registration_date IS 'Date of current registration; only for vozilo.';
COMMENT ON COLUMN company_assets.registration_expires_at IS 'Registration expiry date; only for vozilo.';

CREATE INDEX IF NOT EXISTS idx_company_assets_company_id
    ON company_assets (company_id);

CREATE INDEX IF NOT EXISTS idx_company_assets_assigned_employee
    ON company_assets (assigned_employee_id);

CREATE INDEX IF NOT EXISTS idx_company_assets_company_type
    ON company_assets (company_id, asset_type);

CREATE INDEX IF NOT EXISTS idx_company_assets_company_status
    ON company_assets (company_id, status);

CREATE TRIGGER trg_company_assets_updated_at
    BEFORE UPDATE ON company_assets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
