# Post-Deployment Smoke Test

Run this checklist after every production or staging deploy. It should take under 10 minutes.

## 1. Health checks

- [ ] `GET /health` returns `{"status":"ok"}`
- [ ] `GET /api/health` returns `{"status":"ok","env":"production"}`
- [ ] `GET /api/ready` returns `{"status":"ready","database":"ok","storage":"ok"}`

## 2. Authentication

- [ ] Log in with a valid pilot account — receives access token
- [ ] Refresh token works — silent re-login after token expiry
- [ ] Attempt login with wrong password — returns 401 (not 500)
- [ ] Log out — refresh cookie cleared

## 3. Employees

- [ ] List employees page loads without error
- [ ] Create a new employee (test account; delete after)
- [ ] Edit the employee's name
- [ ] Deactivate, then reactivate the employee

## 4. Projects

- [ ] List projects page loads
- [ ] Create a new test project
- [ ] Edit the project
- [ ] Archive the project — verify it moves to archive list

## 5. Daily reports

- [ ] Create a daily report for the test project
- [ ] Add worker hours
- [ ] Submit the report
- [ ] Approve the report (as direktor/inzenjer)

## 6. Material purchases and receipts

- [ ] Create a material purchase with a receipt (JPEG or PDF, ≤10 MB)
- [ ] Download / view the receipt — file opens correctly
- [ ] Verify the receipt is served via `/api/...` (not a direct R2 URL)

## 7. PWA / mobile

- [ ] App loads on a mobile browser
- [ ] Bottom navigation is visible and functional
- [ ] Offline page appears when network is disabled

## 8. Security checks

- [ ] Try to access another company's data by guessing UUIDs — must return 403 or 404
- [ ] Try to call `/api/employees` without a JWT — must return 401
- [ ] `POST /api/auth/register` always returns 403

## After the checklist

- If all items pass: deployment is approved
- If any item fails: roll back immediately (see [releases-and-rollback.md](releases-and-rollback.md)) and investigate before re-deploying
