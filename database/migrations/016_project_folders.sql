-- Migration 016: Project folder hierarchy for the document browser.
-- Adds project_folders and a folder_id column on project_documents.
--
-- Uniqueness notes:
--   PostgreSQL treats NULL != NULL in regular unique indexes, so two root-level
--   folders with the same name would not collide on (company_id, project_id,
--   parent_id, lower(name)). Two partial unique indexes (one WHERE parent_id IS NULL,
--   one WHERE parent_id IS NOT NULL) solve this correctly.
--
-- Pre-deployment guard: this query must return 0 rows before running the
-- document name uniqueness indexes. Duplicate active document names in the
-- same folder would violate uq_document_root / uq_document_folder.
--
-- SELECT company_id, project_id, COALESCE(folder_id::text, 'ROOT') AS folder,
--        lower(original_name), COUNT(*)
-- FROM project_documents
-- WHERE deleted_at IS NULL
-- GROUP BY company_id, project_id, COALESCE(folder_id::text,'ROOT'), lower(original_name)
-- HAVING COUNT(*) > 1;

CREATE TABLE IF NOT EXISTS project_folders (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    company_id  UUID        NOT NULL REFERENCES companies(id)       ON DELETE CASCADE,
    project_id  UUID        NOT NULL REFERENCES projects(id)        ON DELETE CASCADE,
    parent_id   UUID                 REFERENCES project_folders(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    created_by  UUID                 REFERENCES users(id)           ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_project_folders_project
    ON project_folders (company_id, project_id);

-- Root-level folders: unique name per project (parent_id IS NULL).
CREATE UNIQUE INDEX IF NOT EXISTS uq_folder_root
    ON project_folders (company_id, project_id, lower(name))
    WHERE parent_id IS NULL;

-- Child folders: unique name under each parent.
CREATE UNIQUE INDEX IF NOT EXISTS uq_folder_child
    ON project_folders (company_id, project_id, parent_id, lower(name))
    WHERE parent_id IS NOT NULL;

-- Add folder reference to project_documents (NULL = project root, not a subfolder).
ALTER TABLE project_documents
    ADD COLUMN IF NOT EXISTS folder_id UUID REFERENCES project_folders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_project_documents_folder
    ON project_documents (folder_id)
    WHERE folder_id IS NOT NULL;

-- Documents at root level: unique active name per project.
CREATE UNIQUE INDEX IF NOT EXISTS uq_document_root
    ON project_documents (company_id, project_id, lower(original_name))
    WHERE folder_id IS NULL AND deleted_at IS NULL;

-- Documents inside a folder: unique active name per folder.
CREATE UNIQUE INDEX IF NOT EXISTS uq_document_folder
    ON project_documents (company_id, project_id, folder_id, lower(original_name))
    WHERE folder_id IS NOT NULL AND deleted_at IS NULL;
