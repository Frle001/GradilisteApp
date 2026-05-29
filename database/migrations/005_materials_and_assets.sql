-- Phase 2 Migration 5: Materials, assets, and transfers
-- Material purchases, employee assets (cars, tools), and transfer tracking

-- Employee assets (cars, tools, equipment assigned to individuals)
CREATE TABLE IF NOT EXISTS employee_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  asset_type TEXT NOT NULL,
  name TEXT NOT NULL,
  quantity NUMERIC(12, 2) NOT NULL DEFAULT 1,
  unit TEXT,
  serial_number TEXT,
  notes TEXT,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (asset_type IN ('car', 'tool', 'equipment', 'other')),
  CHECK (quantity > 0)
);

COMMENT ON TABLE employee_assets IS 'Physical assets assigned to individual employees: cars, tools, equipment';
COMMENT ON COLUMN employee_assets.asset_type IS 'Type: car, tool, equipment, other';
COMMENT ON COLUMN employee_assets.active IS 'Soft delete: false when asset is no longer with employee';

CREATE INDEX idx_employee_assets_company_id ON employee_assets(company_id);
CREATE INDEX idx_employee_assets_employee_id ON employee_assets(employee_id);
CREATE INDEX idx_employee_assets_asset_type ON employee_assets(asset_type);
CREATE INDEX idx_employee_assets_active ON employee_assets(active);

CREATE TRIGGER trg_employee_assets_updated_at
  BEFORE UPDATE ON employee_assets
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Material purchase sessions
-- Groups material purchases for a specific project (usually with receipt)
CREATE TABLE IF NOT EXISTS material_purchase_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  buyer_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  receipt_file_url TEXT,
  receipt_file_key TEXT,
  receipt_original_filename TEXT,
  purchased_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  notes TEXT,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE material_purchase_sessions IS 'Groups purchased materials for a project. Usually includes receipt image/file.';
COMMENT ON COLUMN material_purchase_sessions.buyer_id IS 'Employee who bought the materials (responsible party)';
COMMENT ON COLUMN material_purchase_sessions.receipt_file_key IS 'S3/storage key (for future cloud storage)';

CREATE INDEX idx_material_purchase_sessions_company_id ON material_purchase_sessions(company_id);
CREATE INDEX idx_material_purchase_sessions_project_id ON material_purchase_sessions(project_id);
CREATE INDEX idx_material_purchase_sessions_buyer_id ON material_purchase_sessions(buyer_id);
CREATE INDEX idx_material_purchase_sessions_purchased_at ON material_purchase_sessions(purchased_at);

CREATE TRIGGER trg_material_purchase_sessions_updated_at
  BEFORE UPDATE ON material_purchase_sessions
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Material purchase items
-- Individual items within a purchase session
CREATE TABLE IF NOT EXISTS material_purchase_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  purchase_session_id UUID NOT NULL REFERENCES material_purchase_sessions(id) ON DELETE CASCADE,
  project_material_id UUID NOT NULL REFERENCES project_materials(id) ON DELETE CASCADE,
  quantity NUMERIC(12, 2) NOT NULL,
  unit TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (quantity > 0)
);

COMMENT ON TABLE material_purchase_items IS 'Individual items in a purchase session';

CREATE INDEX idx_material_purchase_items_company_id ON material_purchase_items(company_id);
CREATE INDEX idx_material_purchase_items_purchase_session_id ON material_purchase_items(purchase_session_id);
CREATE INDEX idx_material_purchase_items_project_material_id ON material_purchase_items(project_material_id);

CREATE TRIGGER trg_material_purchase_items_updated_at
  BEFORE UPDATE ON material_purchase_items
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Employee material responsibility
-- Tracks materials personally assigned/responsible to an employee
CREATE TABLE IF NOT EXISTS employee_material_responsibility (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  project_material_id UUID NOT NULL REFERENCES project_materials(id) ON DELETE CASCADE,
  quantity NUMERIC(12, 2) NOT NULL,
  unit TEXT NOT NULL,
  source_purchase_session_id UUID REFERENCES material_purchase_sessions(id) ON DELETE SET NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (quantity > 0)
);

COMMENT ON TABLE employee_material_responsibility IS 'Materials assigned to specific employees for tracking and accountability';
COMMENT ON COLUMN employee_material_responsibility.source_purchase_session_id IS 'Where this material came from (optional)';
COMMENT ON COLUMN employee_material_responsibility.active IS 'Soft delete: false when material transferred away';

CREATE INDEX idx_employee_material_responsibility_company_id ON employee_material_responsibility(company_id);
CREATE INDEX idx_employee_material_responsibility_employee_id ON employee_material_responsibility(employee_id);
CREATE INDEX idx_employee_material_responsibility_project_id ON employee_material_responsibility(project_id);
CREATE INDEX idx_employee_material_responsibility_project_material_id ON employee_material_responsibility(project_material_id);
CREATE INDEX idx_employee_material_responsibility_active ON employee_material_responsibility(active);

CREATE TRIGGER trg_employee_material_responsibility_updated_at
  BEFORE UPDATE ON employee_material_responsibility
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Asset transfers
-- Audit trail of transfers between employees
CREATE TABLE IF NOT EXISTS asset_transfers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  from_employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  to_employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  asset_type TEXT NOT NULL,
  employee_asset_id UUID REFERENCES employee_assets(id) ON DELETE SET NULL,
  employee_material_responsibility_id UUID REFERENCES employee_material_responsibility(id) ON DELETE SET NULL,
  quantity NUMERIC(12, 2) NOT NULL DEFAULT 1,
  project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
  transferred_by UUID REFERENCES users(id) ON DELETE SET NULL,
  transferred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (asset_type IN ('car', 'tool', 'equipment', 'material', 'other')),
  CHECK (
    (asset_type = 'material' AND employee_material_responsibility_id IS NOT NULL) OR
    (asset_type IN ('car', 'tool', 'equipment', 'other') AND employee_asset_id IS NOT NULL)
  ),
  CHECK (quantity > 0)
);

COMMENT ON TABLE asset_transfers IS 'Complete audit trail of asset/material transfers between employees';
COMMENT ON COLUMN asset_transfers.from_employee_id IS 'Employee giving the asset';
COMMENT ON COLUMN asset_transfers.to_employee_id IS 'Employee receiving the asset';
COMMENT ON COLUMN asset_transfers.transferred_by IS 'User who recorded the transfer';

CREATE INDEX idx_asset_transfers_company_id ON asset_transfers(company_id);
CREATE INDEX idx_asset_transfers_from_employee_id ON asset_transfers(from_employee_id);
CREATE INDEX idx_asset_transfers_to_employee_id ON asset_transfers(to_employee_id);
CREATE INDEX idx_asset_transfers_asset_type ON asset_transfers(asset_type);
CREATE INDEX idx_asset_transfers_transferred_at ON asset_transfers(transferred_at);
