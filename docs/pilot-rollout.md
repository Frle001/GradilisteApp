# Controlled Pilot Rollout Plan

## Objectives

- Validate that the app works correctly with real construction site data
- Identify UX issues before wider rollout
- Confirm that receipts, daily reports, and employee records survive production operations
- Collect structured feedback from the initial users

## Phase 1 — Internal testing (week 1)

**Who:** 2–3 internal users (direktor + poslovoda + administracija roles)

**What to test:**
- Create the first project and employees
- Submit a daily report with worker hours
- Upload a material purchase receipt
- Approve a daily report

**Success criteria:**
- No data loss after 48 hours of use
- All smoke test items pass (see [post-deployment-smoke-test.md](post-deployment-smoke-test.md))
- No authentication failures

## Phase 2 — Extended pilot (weeks 2–4)

**Who:** Full pilot group (up to 10 users across all roles)

**What to expand:**
- Add real projects from the current construction pipeline
- Daily reports submitted from mobile devices on-site
- Material purchases with photo receipts taken on mobile
- Inventory transfers if used

**Monitoring:**
- Check `/api/ready` storage status daily
- Review Render logs for errors (`"level":"error"`)
- Confirm R2 bucket object count grows (receipts are being stored)

## Onboarding new pilot users

1. Run `create-admin` only once to create the first company and first direktor
2. All subsequent users are created via the app's employee + user management UI
3. New users receive a temporary password (set must_change_password=true)
4. Users change their password on first login
5. Share the pilot user guide ([pilot-user-guide.md](pilot-user-guide.md)) before first login

## Feedback collection

During the pilot, collect feedback via:
- Weekly brief check-in (15 min) with the poslovoda user
- Issue list: any blocker, confusion, or data problem is written down immediately
- Screenshot any UI issue with the phone (attach to feedback notes)

## Go/no-go criteria for full rollout

- Zero data loss incidents
- No critical auth bugs
- Daily report flow completes end-to-end on mobile
- Receipt upload and download works for all file types (JPEG, PDF)
- All pilot users report they can complete their daily tasks without developer help

## Data rules during pilot

- Use only real, current project data — not test data mixed in with production
- Do not import historical data unless it has been reviewed for correctness
- Keep the pilot company isolated (do not create multiple test companies)
- If a rollback is needed, notify all pilot users before taking the system down
