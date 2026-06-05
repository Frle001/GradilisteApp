-- Migration 007: Fix available_quantity initial value in project_materials
--
-- Bug: When materials were created/imported, available_quantity was set to
-- planned_quantity instead of 0. available_quantity represents physically
-- purchased/received stock (increased only via material purchase sessions),
-- so it must start at 0 and grow only through purchases.
--
-- Safe strategy:
--   - Materials with no purchase history: reset to 0.
--   - Materials with purchases: recalculate from actual purchase items so
--     the inflated planned_quantity initial value is removed.

-- 1. Materials never purchased → available = 0
UPDATE project_materials pm
SET available_quantity = 0
WHERE NOT EXISTS (
    SELECT 1
    FROM material_purchase_items mpi
    WHERE mpi.project_material_id = pm.id
);

-- 2. Materials with purchases → available = sum of purchased quantities
UPDATE project_materials pm
SET available_quantity = (
    SELECT COALESCE(SUM(mpi.quantity), 0)
    FROM material_purchase_items mpi
    WHERE mpi.project_material_id = pm.id
)
WHERE EXISTS (
    SELECT 1
    FROM material_purchase_items mpi
    WHERE mpi.project_material_id = pm.id
);

-- Update column comment to reflect correct semantics
COMMENT ON COLUMN project_materials.available_quantity IS 'Purchased/received quantity (increased by material purchases, not derived from planned)';
