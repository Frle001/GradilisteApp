# Projects (Gradilišta) — Phase 5

## Overview

A project represents a construction site (gradilište). Every project belongs to exactly one company. Projects are never physically deleted — they are closed or archived instead.

## Project Lifecycle

```
active ──► closed ──► active (reactivated)
  │
  └──► archived ──► active (reactivated)
```

| Status     | Description                                              |
|------------|----------------------------------------------------------|
| `active`   | Project is ongoing; poslovoda can see it                 |
| `closed`   | Work completed; hidden from poslovoda's list             |
| `archived` | Permanent historical record; hidden from poslovoda's list |

Reactivation is possible for both `closed` and `archived` projects by direktor/inženjer.

## Project Assignment Model

Assignments are stored in `project_assignments`. The schema supports multiple employees per project with different `role_on_project` values (`poslovoda`, `worker`, `engineer`).

In Phase 5, only `poslovoda` assignments are created. A project has at most one active poslovoda assignment at a time. Assigning a new poslovoda deactivates the previous assignment before inserting the new one (atomic transaction).

The UNIQUE constraint is `(project_id, employee_id, company_id)`. Re-assigning the same poslovoda uses `ON CONFLICT DO UPDATE SET active = true`.

## Why Projects Are Not Deleted

- Audit trail integrity: daily reports, materials, and assignments reference the project
- Legal requirement: construction documentation must be retained
- Use `close` or `archive` instead

## Permissions

| Action               | direktor | inženjer | administracija | poslovoda |
|----------------------|----------|----------|----------------|-----------|
| View active projects | ✓        | ✓        | ✓              | own only  |
| View closed/archived | ✓        | ✓        | ✓              | ✗         |
| Create project       | ✓        | ✓        | ✗              | ✗         |
| Edit project         | ✓        | ✓        | ✗              | ✗         |
| Assign poslovoda     | ✓        | ✓        | ✗              | ✗         |
| Close project        | ✓        | ✓        | ✗              | ✗         |
| Archive project      | ✓        | ✓        | ✗              | ✗         |
| Reactivate project   | ✓        | ✓        | ✗              | ✗         |

### Poslovoda Project Filtering

Poslovoda sees only projects where:
1. `projects.status = 'active'`
2. An active assignment exists: `project_assignments.employee_id = auth.employee_id AND role_on_project = 'poslovoda' AND active = true`
3. `projects.company_id = auth.company_id`

Accessing a project via direct URL that the poslovoda is not assigned to (or that is not active) returns **403 Forbidden**.

## API Endpoints

All routes require a valid JWT (`Authorization: Bearer <token>`).

### `GET /api/projects`

Query params:
- `search` — partial match on name or address (ILIKE)
- `status` — filter by status (ignored for poslovoda)

Returns `{ projects: [...], count: N }`.

### `GET /api/projects/:id`

Returns `{ project: { ...detail, assignments: [...] } }`.

Counts `materials_count` and `daily_reports_count` are included as placeholders for future modules.

### `POST /api/projects`

Roles: `direktor`, `inženjer`

```json
{
  "name": "Gradilište Osijek Centar",
  "address": "Ulica 1, Osijek",
  "description": "Opis projekta",
  "start_date": "2026-06-01",
  "end_date": null,
  "primary_poslovoda_id": "uuid-or-null"
}
```

Creates project (status=active) and optionally assigns poslovoda, all in one transaction. Returns `{ project: { ...detail } }`.

### `PUT /api/projects/:id`

Roles: `direktor`, `inženjer`

Editable fields: `name`, `address`, `description`, `start_date`, `end_date`. Status not changed here.

### `PATCH /api/projects/:id/assign-poslovoda`

Roles: `direktor`, `inženjer`

```json
{ "poslovoda_id": "uuid" }
```

Deactivates previous poslovoda assignment and creates a new one atomically.

### `PATCH /api/projects/:id/close`

Sets `status = 'closed'`, records `closed_at` and `closed_by`.

### `PATCH /api/projects/:id/archive`

Sets `status = 'archived'`. Uses `COALESCE` so `closed_at`/`closed_by` are not overwritten if already set.

### `PATCH /api/projects/:id/reactivate`

Sets `status = 'active'`, clears `closed_at` and `closed_by`.

### `DELETE /api/projects/:id`

Returns 405 with message: `"Projects are not permanently deleted. Use close or archive instead."`

### `GET /api/projects/:id/assignments`

Returns all assignments (active and historical) for a project.

## Frontend Routes

| Route | Visible to |
|-------|-----------|
| `/dashboard/projects` | direktor, inženjer, administracija, poslovoda |
| `/dashboard/projects/new` | direktor, inženjer |
| `/dashboard/projects/:id` | per backend permissions |
| `/dashboard/projects/:id/edit` | direktor, inženjer |

## Manual Testing Checklist

### As direktor / inženjer

- [ ] View all projects (active, closed, archived)
- [ ] Create project without poslovoda
- [ ] Create project with poslovoda — verify assignment created
- [ ] Edit project details
- [ ] Assign / change poslovoda — verify old assignment deactivated
- [ ] Close active project — verify it disappears from poslovoda's list
- [ ] Archive project
- [ ] Reactivate closed/archived project
- [ ] Verify audit logs: `project.create`, `project.update`, `project.assign_poslovoda`, `project.close`, `project.archive`, `project.reactivate`

### As administracija

- [ ] View all projects (all statuses)
- [ ] Cannot reach POST /api/projects (403)
- [ ] Cannot reach PUT /api/projects/:id (403)
- [ ] Cannot reach PATCH .../close, .../archive, .../reactivate, .../assign-poslovoda (403)

### As poslovoda

- [ ] See only active projects assigned to self
- [ ] Cannot see closed/archived projects
- [ ] Cannot see projects assigned to another poslovoda
- [ ] Direct URL to unassigned project → 403
- [ ] Cannot create/edit/close/archive projects (403)

### Security checks

- [ ] All project queries return only records matching `company_id` from JWT
- [ ] `company_id` from request body/params is never trusted for data scoping
- [ ] DELETE returns 405, not 404
