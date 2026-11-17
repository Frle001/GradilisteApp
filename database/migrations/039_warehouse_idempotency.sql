-- Migration 039: Idempotency keys for warehouse write operations
-- Prevents duplicate stock movements from retries, double-clicks, or network timeouts.
-- The key is scoped to (company_id, idempotency_key); each client generates a UUID per
-- logical operation and reuses it on retries. The stock movement and this record are
-- committed in the same transaction, so a rollback also removes the key.

CREATE TABLE IF NOT EXISTS warehouse_idempotency_keys (
    company_id      UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL,
    operation_type  VARCHAR(50) NOT NULL
        CHECK (operation_type IN ('receipt', 'issue', 'return', 'adjustment')),
    payload_hash    TEXT NOT NULL,
    result_id       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_warehouse_idempotency_company
    ON warehouse_idempotency_keys(company_id, created_at DESC);
