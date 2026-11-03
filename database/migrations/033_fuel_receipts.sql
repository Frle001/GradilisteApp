CREATE TABLE fuel_receipts (
    id                   UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           UUID           NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    submitted_by         UUID           NOT NULL REFERENCES users(id),
    receipt_date         DATE           NOT NULL,
    amount_eur           NUMERIC(10, 2) NOT NULL CHECK (amount_eur > 0),
    storage_key          TEXT           NOT NULL,
    original_filename    TEXT           NOT NULL,
    mime_type            TEXT           NOT NULL,
    file_size            BIGINT         NOT NULL CHECK (file_size > 0),
    notes                TEXT,
    vehicle_registration TEXT,
    fuel_liters          NUMERIC(6, 2)  CHECK (fuel_liters IS NULL OR fuel_liters > 0),
    fuel_type            TEXT,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fuel_receipts_company      ON fuel_receipts(company_id);
CREATE INDEX idx_fuel_receipts_submitted_by ON fuel_receipts(submitted_by);
CREATE INDEX idx_fuel_receipts_receipt_date ON fuel_receipts(receipt_date DESC);
