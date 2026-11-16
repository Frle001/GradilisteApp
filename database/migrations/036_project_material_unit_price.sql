-- Migration 036: Add unit_price to project_materials for cost analytics
-- Enables material cost calculation: cost = SUM(installed_qty × unit_price) per project per month.
-- DEFAULT 0 means existing materials are treated as zero-cost until a price is entered.

ALTER TABLE project_materials
    ADD COLUMN IF NOT EXISTS unit_price NUMERIC(12, 4) NOT NULL DEFAULT 0;

COMMENT ON COLUMN project_materials.unit_price IS
    'Price per unit used for cost analytics. 0 = no price entered (material excluded from cost totals).';
