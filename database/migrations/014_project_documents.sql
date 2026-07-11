-- Migration 014: Project document/file storage
-- Files uploaded per-project. Soft-deleted (deleted_at) so storage cleanup can be async.

CREATE TABLE IF NOT EXISTS project_documents (
    id            UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id    UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    project_id    UUID        NOT NULL REFERENCES projects(id)  ON DELETE CASCADE,
    uploaded_by   UUID                 REFERENCES users(id)     ON DELETE SET NULL,
    file_key      TEXT        NOT NULL,
    original_name TEXT        NOT NULL,
    content_type  TEXT        NOT NULL,
    file_size     BIGINT      NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_project_documents_project_id ON project_documents (project_id);
CREATE INDEX idx_project_documents_company_id ON project_documents (company_id);
CREATE INDEX idx_project_documents_active     ON project_documents (project_id, company_id) WHERE deleted_at IS NULL;
