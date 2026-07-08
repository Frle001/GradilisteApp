-- Migration 013: Track material quantity changes on daily report approval
-- Each row records the effect one activity had on project_materials.available_quantity.
-- UNIQUE (report_id, activity_id) makes approval idempotent at the DB level.

CREATE TABLE IF NOT EXISTS report_material_effects (
    id                  UUID           NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id          UUID           NOT NULL REFERENCES companies(id)               ON DELETE CASCADE,
    report_id           UUID           NOT NULL REFERENCES daily_reports(id)           ON DELETE CASCADE,
    activity_id         UUID           NOT NULL REFERENCES daily_report_activities(id) ON DELETE CASCADE,
    project_id          UUID           NOT NULL REFERENCES projects(id)                ON DELETE CASCADE,
    project_material_id UUID                    REFERENCES project_materials(id)       ON DELETE SET NULL,
    effect_type         TEXT           NOT NULL CHECK (effect_type IN ('montaza', 'demontaza')),
    quantity_delta      NUMERIC(12, 2) NOT NULL,
    unit                TEXT           NOT NULL,
    applied_by          UUID                    REFERENCES users(id)                   ON DELETE SET NULL,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_report_material_effect UNIQUE (report_id, activity_id)
);

CREATE INDEX idx_rme_company_id  ON report_material_effects (company_id);
CREATE INDEX idx_rme_report_id   ON report_material_effects (report_id);
CREATE INDEX idx_rme_project_id  ON report_material_effects (project_id);
CREATE INDEX idx_rme_material_id ON report_material_effects (project_material_id);
