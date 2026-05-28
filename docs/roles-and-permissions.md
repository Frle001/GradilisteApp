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

### Authentication & Authorization

Currently **NOT IMPLEMENTED** — Phase 2 task.

When implemented:
1. Users login with email + password
2. System issues JWT token with role embedded
3. Each request includes token in `Authorization` header
4. Middleware validates token and extracts role
5. Handlers check role before processing request

### Database Schema

Roles are stored as strings (not enum ID references):

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    role VARCHAR(50) NOT NULL CHECK (
        role IN ('direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik')
    )
);
```

Constants defined in `models/models.go`:
```go
const (
    RoleDirektor       = "direktor"
    RoleInzenjer       = "inzenjer"
    RoleAdministracija = "administracija"
    RolePoslovoda      = "poslovoda"
    RoleRadnik         = "radnik"
)
```

### Future: Permission Middleware

```go
// Example (not yet implemented)
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole := c.GetString("user_role")
        for _, role := range allowedRoles {
            if userRole == role {
                c.Next()
                return
            }
        }
        c.JSON(403, gin.H{"error": "Forbidden"})
    }
}

// Usage
router.POST("/projects", RequireRole("direktor", "inzenjer"), handlers.CreateProject)
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

**Note:** Not yet implemented in Phase 1.

## Security Considerations

**Phase 1:** No security checks (foundation only)

**Phase 2 priorities:**
1. Password hashing (bcrypt)
2. JWT token validation
3. Role-based access control
4. Audit logging
5. Rate limiting
6. SQL injection prevention (already using pgx prepared statements)
7. XSS prevention (frontend validation)

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
