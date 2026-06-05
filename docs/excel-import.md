# Excel Import — Phase 6

## Overview

The Excel import system allows direktor and inženjer to bulk-import project materials from `.xlsx` / `.xls` files. The flow is two-step: **preview** (parse + validate + store) → **confirm** (upsert into `project_materials`).

## Supported Column Names

Column detection is case-insensitive and order-independent. Aliases for each field:

| Field | Accepted Column Headers |
|-------|------------------------|
| Material name | `naziv materijala`, `naziv`, `materijal`, `opis`, `material_name`, `name` |
| Quantity | `količina`, `kolicina`, `qty`, `quantity`, `planned_quantity`, `planirano` |
| Unit | `jedinica mjere`, `jed. mjere`, `jm`, `unit`, `mjera`, `j.m.` |
| Code (optional) | `šifra`, `sifra`, `code`, `material_code`, `kod`, `broj` |

Material name, quantity, and unit are **required**. Code is optional.

## Number Format

Quantity accepts both `.` and `,` as decimal separators:
- `1500` → 1500
- `1500,50` → 1500.5
- `1.500,50` → 1500.5 (thousands separator dot removed)

## Validation Rules

A row is `valid` if:
- `material_name` is non-empty
- `unit` is non-empty
- `quantity` is a parseable number >= 0

Invalid rows are stored in `import_job_rows` with `status = 'invalid'` and shown in the preview table highlighted in red. They are skipped during confirm.

## Import Flow

```
POST /api/projects/:id/materials/import/preview
  ← multipart/form-data, field: file

Response: {
  import_job_id: "uuid",
  filename: "materials.xlsx",
  total_rows: 50,
  valid_rows: 48,
  invalid_rows: 2,
  rows: [...]
}

↓ user reviews preview

POST /api/projects/:id/materials/import/confirm
  ← { import_job_id: "uuid" }

Response: { inserted: 45, updated: 3 }
```

## Upsert Behavior

Confirm uses `ON CONFLICT (project_id, company_id, LOWER(material_name), unit) DO UPDATE`:
- Matching row exists and is **active** → `planned_quantity` is updated
- Matching row exists but is **inactive** → it is **reactivated** and quantity updated
- No match → new row inserted

`available_quantity` is recalculated as `MAX(0, planned_quantity - used_quantity)` on conflict.

## Data Storage

| Table | Purpose |
|-------|---------|
| `import_jobs` | One record per upload: filename, status (`uploaded → parsed → confirmed`), row counts |
| `import_job_rows` | One record per data row: raw JSON, normalized JSON, status (`valid / invalid / imported`) |

Once confirmed, `import_job_rows.status` changes to `imported` and `import_jobs.status` to `confirmed`. A confirmed job cannot be re-confirmed (returns 409).

## Guards

- Archived projects cannot receive new imports (422 response)
- Import confirm is restricted to `direktor` and `inženjer` (403 for others)
- Import preview is also restricted to `direktor` and `inženjer`

## Excel File Requirements

- File must have at least one sheet
- First row must be the header row
- At least one data row must be present
- Required columns: name, quantity, unit (in any order)
- Blank rows (all three required fields empty) are silently skipped
