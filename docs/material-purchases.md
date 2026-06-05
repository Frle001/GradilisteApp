# Material Purchases (Upis materijala)

## Business purpose

Poslovođa buys materials for a project and needs to log: what was bought, how much, for which project, and upload the receipt photo. This creates an audit trail and increases `project_materials.available_quantity` for tracking.

## Purchase flow

1. Poslovođa opens **Upis materijala** → **Novi upis**
2. Selects active assigned project (gradilište)
3. Selects material from the project's material list
4. Enters quantity; unit is auto-filled from the material record
5. Clicks **Dodaj na listu** (repeat for multiple materials)
6. Uploads receipt photo (optional but recommended)
7. Clicks **Spremi upis materijala**
8. Backend creates the purchase session in a DB transaction, redirects to detail page

## Permissions

| Role | List | View detail | Create | View receipt |
|------|------|-------------|--------|--------------|
| direktor | All company purchases | Any | Yes | Yes |
| inzenjer | All company purchases | Any | Yes | Yes |
| administracija | All company purchases | Any | No | Yes |
| poslovoda | Own purchases only | Own only | Own projects only | Own only |

**Poslovoda restrictions enforced in backend:**
- Can only create for projects where they have an active assignment
- `buyer_id` is always set to `auth.employee_id` (cannot specify another buyer)
- List is filtered to `buyer_id = auth.employee_id`

## API endpoints

```
GET    /api/material-purchases                  - List purchases (paginated)
GET    /api/material-purchases/form-data        - Projects + materials for form
POST   /api/material-purchases                  - Create purchase (multipart/form-data)
GET    /api/material-purchases/:id              - Purchase detail with items
GET    /api/material-purchases/:id/receipt      - Serve receipt file (auth-protected)
DELETE /api/material-purchases/:id              - Returns 405 (not deleted in this phase)
```

All endpoints require JWT authentication (`Authorization: Bearer <token>`).

## Create request format

`POST /api/material-purchases` — `Content-Type: multipart/form-data`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | string (UUID) | Yes | Active project |
| `items` | string (JSON array) | Yes | List of items (see below) |
| `purchased_at` | string | No | Date (YYYY-MM-DD). Defaults to today |
| `notes` | string | No | Optional notes |
| `receipt` | file | No | Receipt image/PDF (JPEG/PNG/WEBP/PDF, max 10 MB) |

`items` JSON example:
```json
[
  {"project_material_id": "uuid", "quantity": 20, "unit": "kom"},
  {"project_material_id": "uuid", "quantity": 5, "unit": "m"}
]
```

## Database transaction (on Create)

All writes happen in a single transaction:

1. `INSERT INTO material_purchase_sessions` → returns `session_id`
2. For each item:
   - `INSERT INTO material_purchase_items`
   - `UPDATE project_materials SET available_quantity = available_quantity + qty`
   - `SELECT FOR UPDATE` on `employee_material_responsibility` (employee + project + material)
     - If exists and active: `UPDATE quantity += qty`
     - Else: `INSERT` new responsibility row linked to this session
3. Audit log: `material_purchase.create`

If the transaction fails and a receipt file was already saved, the service attempts to delete the file to avoid orphaned uploads.

## Quantity logic

```
project_materials.available_quantity += purchased_quantity
```

This represents material that has been physically acquired. It does NOT reduce `available_quantity` (that would represent usage, which happens through daily report activities).

`employee_material_responsibility` tracks how much of each material the buyer is personally responsible for.

## Receipt file storage

See [storage.md](./storage.md) for storage architecture.

Key points:
- Stored at `$UPLOADS_DIR/receipts/{company_id}/{project_id}/{uuid}.{ext}`
- Never served publicly — always through `/api/material-purchases/:id/receipt`
- Original filename stored but not used for file path
- Allowed: JPEG, PNG, WEBP, PDF. Max 10 MB.

## Audit log entries

| Action | When |
|--------|------|
| `material_purchase.create` | Purchase session created |

New data includes: `session_id`, `project_id`, `project_name`, `buyer_id`, `items_count`, `purchased_at`, `has_receipt`.

## Manual testing checklist

### As poslovoda

- [ ] Open Upis materijala → Novi upis
- [ ] See only assigned active projects in dropdown
- [ ] Select project → materials load correctly
- [ ] Add one material to list
- [ ] Add multiple materials (quantities merge if same material added twice)
- [ ] Remove a material from the list
- [ ] Upload receipt image (JPEG/PNG)
- [ ] Submit → redirected to detail page
- [ ] Detail shows correct project, buyer, date, items
- [ ] Receipt is visible in detail page
- [ ] Try to load a project not assigned to you via direct API call → 403
- [ ] Try to submit a material from another project via direct API → 422
- [ ] Try to submit with empty items → 422
- [ ] Try to upload a 15 MB file → 400 (validation error)
- [ ] Confirm `project_materials.available_quantity` increased in database
- [ ] Confirm `employee_material_responsibility` created for buyer/project/material
- [ ] Submit second purchase for same material → responsibility quantity increases (not duplicate row)

### As direktor / inzenjer

- [ ] View list → see all company purchases
- [ ] Filter by project, date range, search
- [ ] View purchase detail
- [ ] View receipt image
- [ ] Create purchase (optional — primary creator is poslovoda)

### As administracija

- [ ] View list → see all company purchases
- [ ] View detail and receipt
- [ ] Attempt POST /api/material-purchases → 403 Forbidden

### Security checks

- [ ] All purchase queries include `company_id` filter from JWT
- [ ] Receipt files are rejected without valid auth token
- [ ] `buyer_id` always set from JWT, never from request body
- [ ] File type validation blocks non-image/PDF uploads
- [ ] File size validation blocks oversized files
- [ ] DB transaction is atomic: no partial states on failure
