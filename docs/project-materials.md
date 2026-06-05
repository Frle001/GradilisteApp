# Project Materials — Phase 6

## Overview

Project materials (`project_materials`) represent the planned material inventory for a construction project. Each entry has a planned quantity, used quantity (updated by purchase tracking in a future phase), and available quantity (computed as `planned - used`).

Materials are never physically deleted — they are deactivated (`active = false`).

## Permissions

| Action              | direktor | inženjer | administracija | poslovoda |
|---------------------|----------|----------|----------------|-----------|
| View materials      | ✓        | ✓        | ✓              | own only  |
| Add manually        | ✓        | ✓        | ✗              | ✗         |
| Edit                | ✓        | ✓        | ✗              | ✗         |
| Deactivate          | ✓        | ✓        | ✗              | ✗         |
| Import from Excel   | ✓        | ✓        | ✗              | ✗         |
| Confirm import      | ✓        | ✓        | ✗              | ✗         |

## Uniqueness Constraint

A functional unique index prevents duplicate material entries per project:

```sql
UNIQUE INDEX idx_project_materials_unique_name_unit
ON project_materials (project_id, company_id, LOWER(material_name), unit)
```

Importing the same material name + unit combination **upserts** (updates `planned_quantity`, reactivates if inactive).

## API Endpoints

All routes require a valid JWT. Routes under `/:id/materials` also require `authRequired` (already on the parent `/projects` group).

### `GET /api/projects/:id/materials`

Query params:
- `search` — partial match on `material_name` or `material_code` (ILIKE)
- `active=false` — include inactive materials

Returns `{ materials: [...], count: N }`.

### `POST /api/projects/:id/materials`

Roles: `direktor`, `inženjer`

```json
{
  "material_name": "Beton C25/30",
  "material_code": "B-25-30",
  "planned_quantity": 150,
  "unit": "m³"
}
```

Returns `{ material: {...} }`.

### `PUT /api/projects/:id/materials/:materialId`

Roles: `direktor`, `inženjer`

Same body as POST. `available_quantity` is recomputed as `MAX(0, planned - used)`.

### `PATCH /api/projects/:id/materials/:materialId/deactivate`

Roles: `direktor`, `inženjer`

Soft-deletes the material.

### `POST /api/projects/:id/materials/import/preview`

Roles: `direktor`, `inženjer`

Multipart form upload, field name: `file` (xlsx/xls).

Returns `ImportPreviewResponse` with `import_job_id` and per-row validation results.

Returns 422 if the project is archived.

### `POST /api/projects/:id/materials/import/confirm`

Roles: `direktor`, `inženjer`

```json
{ "import_job_id": "uuid" }
```

Upserts all valid rows from the import job into `project_materials`. Returns `{ inserted, updated }`.

Returns 409 if the import job is not in `parsed` state.

## Frontend Routes

| Route | Description |
|-------|-------------|
| `/dashboard/projects/:id/materials` | Materials list + inline add form + Excel import |

## Manual Testing Checklist

### As direktor / inženjer

- [ ] View materials list for a project
- [ ] Search by name / code
- [ ] Toggle "Prikaži neaktivne" to see deactivated entries
- [ ] Add material manually — verify it appears in list
- [ ] Edit material — verify `available_quantity` updates
- [ ] Deactivate material — verify it disappears unless inactive toggle is on
- [ ] Upload valid Excel file → preview appears with row count
- [ ] Confirm import → inserted/updated counts shown; materials appear in list
- [ ] Upload file with invalid rows → invalid rows shown in red in preview
- [ ] Re-import same file → existing materials updated, not duplicated
- [ ] Try importing into an archived project → 422 error

### As administracija / poslovoda

- [ ] Can view materials list
- [ ] Cannot add/edit/deactivate (403)
- [ ] Cannot upload Excel import (403)

### Security checks

- [ ] All material queries scoped by `company_id` from JWT
- [ ] Poslovoda can only view materials for own assigned projects
