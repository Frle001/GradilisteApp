# Authentication & Authorization

## Overview

Gradilište App uses **email/password login with JWT access tokens**.

- Passwords hashed with **bcrypt** (cost 12 in production, 10 in dev)
- Tokens are **HS256 JWT** signed with `JWT_SECRET`
- **No refresh tokens** in Phase 3 — logout is client-side token removal
- **Public registration is disabled** — accounts are created by administrators

---

## JWT Token Structure

```json
{
  "user_id":     "uuid",
  "company_id":  "uuid",
  "employee_id": "uuid or empty string",
  "role":        "direktor | inzenjer | administracija | poslovoda",
  "email":       "user@example.com",
  "iat":         1234567890,
  "exp":         1234567890
}
```

**Rule: never trust role, company_id, or user_id from the request body.**
Always derive them from the validated JWT via `GetAuthUser(c)` or `CompanyID(c)`.

---

## Login Flow

```
POST /api/auth/login
{ "email": "...", "password": "..." }

1. Look up user by email
2. Check user.active = true
3. bcrypt.CompareHashAndPassword
4. UPDATE users SET last_login_at = NOW()
5. jwt.Sign(claims, JWT_SECRET)
6. Write async audit log (action: "auth.login")
7. Return { access_token, user }
```

Response:
```json
{
  "access_token": "eyJ...",
  "user": {
    "id": "...",
    "company_id": "...",
    "employee_id": "...",
    "email": "...",
    "role": "direktor"
  }
}
```

`password_hash` is **never** returned.

---

## Auth Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/login` | — | Email/password login |
| POST | `/api/auth/register` | — | Always 403 (disabled) |
| GET | `/api/auth/me` | Required | Fresh user + employee data |
| POST | `/api/auth/logout` | Required | Stateless — clears client token |

---

## Middleware

### `AuthRequired()`

Reads `Authorization: Bearer <token>`, validates JWT, injects `AuthContext` into Gin context.

Returns `401` for:
- Missing Authorization header
- Malformed token
- Expired token
- Invalid signature

### `RequireRoles(roles ...string)`

Must follow `AuthRequired()`. Returns `403` if the user's role is not in the allowed list.

```go
// Examples
RequireRoles("direktor", "inzenjer")
RequireRoles("direktor", "inzenjer", "administracija")
```

### Reading auth context in handlers

```go
u := GetAuthUser(c)
// u.UserID, u.CompanyID, u.EmployeeID, u.Role, u.Email

// Convenience helper for multi-tenant scoping:
cid := CompanyID(c) // always use this when building DB queries
```

---

## Company Isolation

**Every business query must include `WHERE company_id = $N::uuid`.**

Use `CompanyID(c)` from the auth middleware, never accept `company_id` from request params:

```go
// ✅ Correct
rows, err := db.Query(ctx,
    "SELECT * FROM projects WHERE company_id = $1::uuid AND id = $2::uuid",
    CompanyID(c), projectID,
)

// ❌ Wrong — never trust user-supplied company_id
rows, err := db.Query(ctx,
    "SELECT * FROM projects WHERE company_id = $1::uuid",
    c.Param("company_id"),
)
```

This applies to all future modules: employees, projects, reports, materials, assets.

---

## Protected Test Routes

These verify that middleware is wired up correctly. Remove when real modules exist.

| Route | Required Role |
|-------|--------------|
| GET `/api/protected/me` | Any authenticated user |
| GET `/api/protected/director-engineer` | direktor, inzenjer |
| GET `/api/protected/admin` | direktor, inzenjer, administracija |
| GET `/api/protected/poslovoda` | direktor, inzenjer, poslovoda |

---

## Development Seed Users

Default seed users after running `seed_phase2.sql`:

| Email | Role | Password (after genhash) |
|-------|------|--------------------------|
| direktor@example.com | direktor | password123 |
| inzenjer@example.com | inzenjer | password123 |
| admin@example.com | administracija | password123 |
| poslovoda@example.com | poslovoda | password123 |

**The seed SQL contains placeholder hashes. Run the setup below before testing login.**

### Setting up dev passwords

```bash
# From services/api/:
go run ./cmd/genhash/ | psql -h localhost -U gradiliste -d gradiliste
```

This generates bcrypt hashes of `password123` and updates all seed users.

You can override the password or cost:
```bash
SEED_PASSWORD=mydevpass BCRYPT_COST=10 go run ./cmd/genhash/ | psql ...
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes (prod) | dev fallback | HS256 signing key |
| `JWT_EXPIRES_IN` | No | `24h` | Token lifetime (Go duration string) |
| `BCRYPT_COST` | No | `12` | bcrypt work factor |
| `ENV` | No | `production` | `development` enables debug routes |

Generate a secure JWT_SECRET:
```bash
openssl rand -base64 48
```

---

## Why Registration is Disabled

This is a company-internal application. Users are employees of a specific company.
Allowing self-registration would let anyone create accounts.
Accounts must be created by a `direktor` or `administracija` user via the admin interface (Phase 4+).

---

## Frontend Auth Flow

```
1. Visit /login
2. Submit email + password
3. Store access_token + user in localStorage
4. Redirect to /dashboard

5. Dashboard calls GET /api/auth/me on mount
6. If 401 → clearAuth() + redirect /login
7. Render role-based sections

8. Logout button:
   POST /api/auth/logout → clearAuth() → redirect /login
```

Token is attached automatically by the Axios interceptor in `lib/api-client.ts`:
```typescript
config.headers.Authorization = `Bearer ${token}`
```

On 401 response, the interceptor calls `clearAuth()` so the next navigation
lands on `/login`.

---

## Audit Logging

Login events are written to `audit_logs`:

```json
{
  "action": "auth.login",
  "entity_type": "user",
  "entity_id": "<user_uuid>",
  "company_id": "...",
  "user_id": "...",
  "ip_address": "...",
  "user_agent": "..."
}
```

Use `CreateAuditLog(ctx, db, AuditLogParams{...})` in any handler that needs auditing.
It is safe to call in a goroutine — errors are logged, never returned.
