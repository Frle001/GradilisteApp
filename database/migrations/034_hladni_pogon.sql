-- 034_hladni_pogon.sql
-- Hladni pogon (Cold Operations): operating expense tracking with categories,
-- recurring definitions, occurrence management and optional budgets.

BEGIN;

-- ── Expense categories ────────────────────────────────────────────────────────

CREATE TABLE cold_ops_categories (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID    NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name        TEXT    NOT NULL,
    description TEXT,
    color       TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_by  UUID    NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, name)
);

CREATE INDEX idx_cold_ops_categories_company ON cold_ops_categories(company_id);

-- ── Expense records ───────────────────────────────────────────────────────────

CREATE TABLE cold_ops_expenses (
    id               UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID    NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    category_id      UUID    REFERENCES cold_ops_categories(id) ON DELETE SET NULL,
    amount           NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    expense_date     DATE    NOT NULL,
    title            TEXT    NOT NULL,
    notes            TEXT,
    supplier         TEXT,
    payment_method   TEXT    CHECK (payment_method IS NULL OR payment_method IN ('cash','card','transfer','other')),
    expense_type     TEXT    NOT NULL DEFAULT 'variable' CHECK (expense_type IN ('fixed','variable')),
    -- Optional supporting document (upload endpoint added separately)
    storage_key      TEXT,
    original_filename TEXT,
    mime_type        TEXT,
    file_size        BIGINT  CHECK (file_size IS NULL OR file_size > 0),
    created_by       UUID    NOT NULL REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cold_ops_expenses_company      ON cold_ops_expenses(company_id);
CREATE INDEX idx_cold_ops_expenses_category     ON cold_ops_expenses(category_id);
CREATE INDEX idx_cold_ops_expenses_date         ON cold_ops_expenses(expense_date DESC);
CREATE INDEX idx_cold_ops_expenses_company_date ON cold_ops_expenses(company_id, expense_date DESC);

-- ── Recurring expense definitions ─────────────────────────────────────────────

CREATE TABLE cold_ops_recurring_defs (
    id             UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id     UUID    NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    category_id    UUID    REFERENCES cold_ops_categories(id) ON DELETE SET NULL,
    title          TEXT    NOT NULL,
    frequency      TEXT    NOT NULL CHECK (frequency IN ('monthly','yearly')),
    day_of_month   INTEGER NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
    month_of_year  INTEGER CHECK (month_of_year IS NULL OR month_of_year BETWEEN 1 AND 12),
    start_date     DATE    NOT NULL,
    end_date       DATE,
    supplier       TEXT,
    payment_method TEXT    CHECK (payment_method IS NULL OR payment_method IN ('cash','card','transfer','other')),
    notes          TEXT,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_by     UUID    NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cold_ops_recurring_defs_company ON cold_ops_recurring_defs(company_id);

-- ── Effective-dated amount history for recurring definitions ──────────────────

CREATE TABLE cold_ops_recurring_amounts (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    definition_id UUID    NOT NULL REFERENCES cold_ops_recurring_defs(id) ON DELETE CASCADE,
    company_id    UUID    NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    amount        NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    effective_from DATE   NOT NULL,
    effective_to  DATE,
    created_by    UUID    NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cold_ops_recurring_amounts_def ON cold_ops_recurring_amounts(definition_id);

-- ── Recurring occurrence tracking ─────────────────────────────────────────────

CREATE TABLE cold_ops_occurrences (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    definition_id UUID    NOT NULL REFERENCES cold_ops_recurring_defs(id) ON DELETE CASCADE,
    company_id    UUID    NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    occurrence_key TEXT   NOT NULL,
    due_date      DATE    NOT NULL,
    status        TEXT    NOT NULL DEFAULT 'planned' CHECK (status IN ('planned','confirmed','skipped')),
    expense_id    UUID    REFERENCES cold_ops_expenses(id) ON DELETE SET NULL,
    amount        NUMERIC(12, 2) CHECK (amount IS NULL OR amount > 0),
    notes         TEXT,
    handled_by    UUID    REFERENCES users(id),
    handled_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (definition_id, occurrence_key)
);

CREATE INDEX idx_cold_ops_occurrences_company    ON cold_ops_occurrences(company_id);
CREATE INDEX idx_cold_ops_occurrences_definition ON cold_ops_occurrences(definition_id);
CREATE INDEX idx_cold_ops_occurrences_due_date   ON cold_ops_occurrences(due_date);
CREATE INDEX idx_cold_ops_occurrences_status     ON cold_ops_occurrences(company_id, status, due_date);

-- ── Optional monthly category budgets ────────────────────────────────────────

CREATE TABLE cold_ops_budgets (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID    NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    category_id   UUID    REFERENCES cold_ops_categories(id) ON DELETE CASCADE,
    year          INTEGER NOT NULL,
    month         INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
    budget_amount NUMERIC(12, 2) NOT NULL CHECK (budget_amount > 0),
    created_by    UUID    NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Separate partial indexes handle nullable category_id uniqueness correctly.
CREATE UNIQUE INDEX idx_cold_ops_budgets_with_cat
    ON cold_ops_budgets (company_id, category_id, year, month)
    WHERE category_id IS NOT NULL;

CREATE UNIQUE INDEX idx_cold_ops_budgets_no_cat
    ON cold_ops_budgets (company_id, year, month)
    WHERE category_id IS NULL;

COMMIT;
