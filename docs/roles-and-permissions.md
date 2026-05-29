# Roles & Permissions

## Role Hierarchy & Responsibilities

### 1. Direktor (Director)

**Permissions:**
- View all projects
- View all employees
- View all reports
- Create/edit/delete projects
- Create/edit/delete employees
- Assign employees to projects
- Manage system settings
- Access to all modules

**Visibility:**
- Full system access

**Notes:**
- Strategic decision-maker
- Full authority over all employees and projects
- Should not be confused with Inženjer (Engineer), despite similar access levels

---

### 2. Inženjer (Engineer)

**Permissions:**
- View all projects
- View all employees (names, roles, assignments)
- View daily reports
- Create/edit/delete projects
- Assign employees to projects (as site manager or team member)
- Technical approval of work quality
- Create technical specifications

**Visibility:**
- Full system access (technical focus)

**Notes:**
- Technical decision-maker
- Same access level as Direktor but separate role for audit trails
- Focus on quality, compliance, technical correctness
- Cannot approve financial/management decisions (Direktor only)

---

### 3. Administracija (Administration)

**Permissions:**
- View employee list
- Create/edit/deactivate employees
- Manage employee attributes (name, role, contact info)
- View employee history/assignments
- Generate employee reports
- Manage user accounts (future)
- Cannot create/edit projects or modify assignments

**Visibility:**
- Employees only
- Cannot see project financials or detailed work orders

**Notes:**
- HR/HR-like role
- Cannot assign employees to projects (Direktor/Inženjer does that)
- Cannot view daily reports or material entries
- Focused on employee data integrity

---

### 4. Poslovođa (Site Manager)

**Permissions:**
- View assigned projects only (not all projects)
- Manage daily reports for assigned projects
- Record material activities and purchases
- Manage personal inventory (tools, materials)
- Record worker hours
- Submit material receipts/images
- Transfers between employees
- Cannot edit project details
- Cannot add/remove employees from projects

**Visibility:**
- Only projects assigned to them
- Only their team members on those projects
- Daily reports and materials for their projects

**Notes:**
- Site-level operations manager
- Responsible for day-to-day project execution
- Cannot make strategic or administrative decisions
- All actions are recorded with their name

---

### 5. Radnik (Worker)

**Status:**
- Employee record only (currently)
- May not have system login (optional)
- Can be assigned under a Poslovođa

**Future Permissions (Phase 3+):**
- Record own work hours
- View personal assignments
- Access limited project information

**Notes:**
- Lowest privilege level
- Primarily for record-keeping
- Login may be optional (Poslovođa records hours on their behalf)

---

## Access Matrix

| Feature | Direktor | Inženjer | Administracija | Poslovođa | Radnik |
|---------|----------|----------|----------------|-----------|--------|
| **Projects** |
| View all | ✓ | ✓ | - | Own only | - |
| Create/Edit | ✓ | ✓ | - | - | - |
| Delete | ✓ | ✓ | - | - | - |
| **Employees** |
| View all | ✓ | ✓ | ✓ | Own team | Own |
| Create | ✓ | - | ✓ | - | - |
| Edit | ✓ | - | ✓ | - | - |
| Deactivate | ✓ | - | ✓ | - | - |
| **Assignments** |
| Assign to project | ✓ | ✓ | - | - | - |
| Remove from project | ✓ | ✓ | - | - | - |
| **Daily Reports** |
| View all | ✓ | ✓ | - | Own | - |
| Create | ✓ | - | - | ✓ | ✓ (future) |
| Edit own | ✓ | - | - | ✓ | ✓ (future) |
| Edit others | ✓ | - | - | - | - |
| **Materials** |
| View all | ✓ | ✓ | - | Own | - |
| Record | ✓ | - | - | ✓ | - |
| Track inventory | ✓ | ✓ | - | ✓ | - |
| **System** |
| Settings | ✓ | - | - | - | - |
| Audit log | ✓ | - | - | - | - |
| Reports | ✓ | ✓ | - | - | - |

## Implementation Notes

### Authentication & Authorization (Phase 3 — Implemented)

1. Users log in at `POST /api/auth/login` with email + password
2. Server returns a signed JWT with role, company_id, user_id embedded
3. Every protected request includes `Authorization: Bearer <token>`
4. `AuthRequired()` middleware validates the token and injects `AuthContext`
5. `RequireRoles(...)` middleware checks role before the handler runs

See [docs/auth.md](auth.md) for full details.

### Database Schema

Roles are stored as `TEXT` with `CHECK` constraints (Phase 2 schema):

```sql
-- users table (login-capable roles only)
CHECK (role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda'))

-- employees table (all roles including radnik)
CHECK (role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik'))
```

Constants defined in `models/models.go`:
```go
const (
    RoleDirektor       = "direktor"
    RoleInzenjer       = "inzenjer"
    RoleAdministracija = "administracija"
    RolePoslovoda      = "poslovoda"
    RoleRadnik         = "radnik"  // employee-only, no login
)
```

### Role Middleware Usage

```go
// In main.go route registration:
protected := api.Group("/protected", AuthRequired())
protected.GET("/reports",
    RequireRoles("direktor", "inzenjer", "poslovoda"),
    handlers.ListReports,
)

// In any handler, read auth context:
u := GetAuthUser(c)
// u.Role, u.UserID, u.CompanyID, u.EmployeeID

// For DB queries — always scope by company:
companyID := CompanyID(c)
```

## Business Rules

### Direktor vs Inženjer

Both have **full system access**, but they are separate roles to:
- Allow audit trails to distinguish decisions
- Enable future "approval workflows" (Direktor approves financials, Inženjer approves technical)
- Support future delegation (each can review the other's work)

### Poslovođa Assignment

- A Poslovođa can be assigned to **multiple projects**
- A project can have **multiple Poslovođas** (e.g., day shift + night shift)
- Assignments track date range (future: when did they start/end?)

### Administracija Limitations

- Cannot see project data (maintain employee data privacy from projects)
- Cannot assign employees (prevents accidental conflicts)
- Focused on pure HR functions

### Radnik Access (Future)

- May eventually have read-only access to own assignments
- Can submit work hours
- Cannot modify others' records
- Cannot access financial or material data

## Audit Trail

All role-based actions should be logged:
- Who did it (user ID + role)
- What they did (action)
- When (timestamp)
- What data changed (audit log)

**Implemented in Phase 3:** Login events are written automatically. Use `CreateAuditLog()` in any handler that requires auditing.

## Security Considerations

**Phase 3 (implemented):**
- ✅ Password hashing — bcrypt (cost 12)
- ✅ JWT token validation — HS256, signed with JWT_SECRET
- ✅ Role-based access control — `AuthRequired()` + `RequireRoles()`
- ✅ Audit logging — login events written to audit_logs
- ✅ Company isolation — all DB queries must use `CompanyID(c)`
- ✅ No password hash in responses — `json:"-"` tag on PasswordHash field
- ✅ SQL injection prevention — pgx parameterized queries throughout

**Phase 4+ (future):**
- Rate limiting on login endpoint
- Token blacklist / refresh tokens
- XSS prevention hardening (Content-Security-Policy)
- Account lockout on repeated failures

## Changing Roles

- A user's role can be updated by Direktor only
- Changing role affects immediate permissions
- Old role access revoked
- Audit trail records role change

## Future Enhancements

- **Team Leaders** — Sub-role of Poslovođa with limited approval authority
- **Reviewer** — Can review but not approve
- **Auditor** — Read-only access to all audit logs
- **Finance** — Separate role for budget/cost management
- **Custom Permissions** — Admin-defined role combinations
