-- Migration 040: Make warehouse_receipt_lines.unit_price nullable.
-- Separates "unknown price" (NULL) from "explicitly free" (0.00).
-- Existing rows keep their current value; new receipts may omit a price.
ALTER TABLE warehouse_receipt_lines
    ALTER COLUMN unit_price DROP NOT NULL;
