-- Migration 025: Add tracking_type to project_materials
-- 'stock': physical material with an available-quantity pool (default, existing behavior)
-- 'work':  measurable work activity (e.g. Štemanje); accumulates completed quantity,
--           has no stock pool — approval never fails for insufficient quantity

ALTER TABLE project_materials
    ADD COLUMN IF NOT EXISTS tracking_type VARCHAR NOT NULL DEFAULT 'stock'
    CHECK (tracking_type IN ('stock', 'work'));

COMMENT ON COLUMN project_materials.tracking_type IS
    'stock = physical stock with available_quantity management; work = cumulative work progress only';
