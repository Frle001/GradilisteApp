# Final QA Checklist

## Auth

- [ ] Login with valid credentials returns `access_token` and sets `refresh_token` HTTP-only cookie
- [ ] Login with wrong password returns 401
- [ ] Login with inactive account returns 401
- [ ] `POST /api/auth/refresh` with valid cookie returns new access token and rotates cookie
- [ ] `POST /api/auth/refresh` with revoked cookie returns 401
- [ ] `POST /api/auth/refresh` with expired cookie returns 401
- [ ] `POST /api/auth/logout` revokes DB record and clears cookie
- [ ] Accessing protected route without token returns 401
- [ ] Accessing protected route with expired token returns 401
- [ ] Login works for: direktor, inženjer, administracija, poslovoda
- [ ] `PATCH /api/auth/change-password` requires current password

## Rate limiting

- [ ] Rapid `POST /auth/login` requests (>5) trigger 429
- [ ] Rapid `POST /auth/refresh` requests (>5) trigger 429

## CORS

- [ ] In development: requests from any origin succeed
- [ ] In production: requests from unlisted origin return 403 on preflight
- [ ] `Access-Control-Allow-Credentials: true` present on allowed origins
- [ ] `*` wildcard is NOT used in production CORS

## Company isolation

- [ ] All major queries filter by `company_id` from JWT
- [ ] `company_id` is never accepted from request body
- [ ] User from company A cannot read or mutate company B data

## Employees

- [ ] Direktor/inženjer can create/edit/deactivate employees
- [ ] Poslovoda can only see employees assigned to them
- [ ] Poslovoda cannot deactivate employees

## Projects

- [ ] Create/edit/assign poslovoda
- [ ] Close project → appears in archive, blocks new work
- [ ] Archive project → appears in archive, blocks new work
- [ ] Reactivate → returns to active
- [ ] Poslovoda only sees assigned active projects
- [ ] Administracija cannot create/close/archive projects

## Project materials

- [ ] Excel preview shows valid/invalid rows before confirming
- [ ] Confirmed import adds materials
- [ ] Invalid rows are skipped with error detail
- [ ] Closed/archived project rejects new materials (422)
- [ ] Deactivated material hidden by default, visible via filter

## Daily reports

- [ ] Poslovoda creates report for assigned project
- [ ] Duplicate date on same project returns error
- [ ] Only workers assigned to poslovoda can be reported
- [ ] Materials referenced must belong to the project

## Reports / Export

- [ ] Građevinski dnevnik filters by project, date range, worker
- [ ] Građevinska knjiga filters by project, date range
- [ ] Excel export respects all active filters
- [ ] Exported data is scoped to caller's company

## Material purchases

- [ ] Upload receipt image succeeds (≤10 MB)
- [ ] Files >10 MB return 413
- [ ] Receipt is only accessible to authorized users (no public URL)
- [ ] Purchase quantities update available_quantity
- [ ] Closed/archived project rejects new purchases
- [ ] Responsibility records update on purchase confirmation

## Inventory / Transfers

- [ ] Personal inventory view shows correct quantities
- [ ] Transfer of partial quantity succeeds
- [ ] Transfer of full quantity leaves 0 (or removes record)
- [ ] Transfer below 0 is rejected
- [ ] Employee can only transfer their own inventory
- [ ] Asset transfer records in history

## Archive

- [ ] Closed/archived projects appear in archive list
- [ ] Archive list filters work (search, status, date range)
- [ ] Archive summary shows correct counts
- [ ] Reactivate restores project to active
- [ ] No new daily reports/materials/purchases on closed project

## Security

- [ ] Passwords never appear in any API response
- [ ] Tokens never appear in logs
- [ ] `JWT_SECRET` not in any response or log
- [ ] Rate limiting working on auth endpoints
- [ ] Request body >10 MB returns 413
- [ ] `GET /health` returns 200 without authentication
- [ ] `GET /api/ready` returns 503 if DB is unreachable
- [ ] Audit logs created for login, employee create/deactivate, project close/archive

## Frontend

- [ ] 401 triggers refresh attempt; on success request is retried transparently
- [ ] After refresh failure, user is redirected to /login
- [ ] Role-based dashboard cards match user role
- [ ] Buttons hidden for unauthorized actions (backed by API enforcement)
- [ ] Clear Croatian error messages shown on API errors
- [ ] Loading states shown on data fetches
- [ ] Empty states shown when lists are empty
- [ ] Confirmation dialogs for: deactivate employee, close project, archive project, transfer
