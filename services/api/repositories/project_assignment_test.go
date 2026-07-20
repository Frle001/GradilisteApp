package repositories

import (
	"os"
	"strings"
	"testing"
)

// ── SQL structure tests ───────────────────────────────────────────────────────
//
// These tests verify the critical clauses in the poslovoda assignment SQL
// constants without requiring a live database. They guard against accidental
// regression of the three root-cause bugs fixed in this change:
//
//  1. poslovodaUpsertSQL must set role_on_project in DO UPDATE so that an
//     employee who already has a non-poslovoda row is correctly promoted.
//  2. poslovodaLockSQL must use FOR UPDATE to prevent concurrent reassignments
//     from both committing and leaving two active poslovoda rows.
//  3. poslovodaIDSubquery and poslovodaNameSubquery must use LIMIT 1 so that
//     a project with (temporarily) multiple active poslovoda rows never
//     produces duplicate rows in List/GetByID results.
//
// Full integration tests (DB required) are documented below each SQL test
// as the scenarios they would cover.

func TestPoslovodaUpsertSQL_SetsRoleOnConflict(t *testing.T) {
	doIdx := strings.Index(poslovodaUpsertSQL, "DO UPDATE")
	if doIdx < 0 {
		t.Fatal("poslovodaUpsertSQL must contain a DO UPDATE clause")
	}
	doUpdate := poslovodaUpsertSQL[doIdx:]

	if !strings.Contains(doUpdate, "role_on_project = 'poslovoda'") {
		t.Error("DO UPDATE must set role_on_project = 'poslovoda' so that an employee " +
			"with a prior non-poslovoda assignment on the project is correctly promoted")
	}
	if !strings.Contains(doUpdate, "active = true") {
		t.Error("DO UPDATE must set active = true to reactivate a previously inactive assignment")
	}
	if !strings.Contains(doUpdate, "assigned_by = EXCLUDED.assigned_by") {
		t.Error("DO UPDATE must update assigned_by from EXCLUDED to record who made the change")
	}
}

// Integration scenarios covered by the above SQL correction:
//   Scenario A  — Project has poslovoda A; reassign to B (B has no prior row)
//                 → INSERT succeeds with role_on_project='poslovoda', active=true.
//   Scenario B  — B already has an inactive poslovoda assignment
//                 → conflict → DO UPDATE reactivates and keeps role_on_project='poslovoda'.
//   Scenario C  — B has an old assignment with a different role
//                 → conflict → DO UPDATE now also sets role_on_project='poslovoda'.
//   Scenario D  — Reassign A to A (idempotent)
//                 → lock → deactivate (A.active=false) → upsert conflict → A reactivated.

func TestPoslovodaLockSQL_ContainsForUpdate(t *testing.T) {
	if !strings.Contains(poslovodaLockSQL, "FOR UPDATE") {
		t.Error("poslovodaLockSQL must contain FOR UPDATE to serialise concurrent reassignments")
	}
	if !strings.Contains(poslovodaLockSQL, "role_on_project = 'poslovoda'") {
		t.Error("lock query must filter by role_on_project = 'poslovoda'")
	}
	if !strings.Contains(poslovodaLockSQL, "project_id = $1") {
		t.Error("lock query must be scoped to the target project")
	}
}

// Integration scenarios covered by the locking fix:
//   Scenario E  — Two simultaneous PATCH /assign-poslovoda for the same project
//                 → second transaction blocks on FOR UPDATE until first commits
//                 → net result: exactly one active poslovoda, no duplicates.

func TestPoslovodaDeactivateSQL_CorrectCondition(t *testing.T) {
	if !strings.Contains(poslovodaDeactivateSQL, "role_on_project = 'poslovoda'") {
		t.Error("deactivate query must only deactivate poslovoda rows, not worker rows")
	}
	if !strings.Contains(poslovodaDeactivateSQL, "active = true") {
		t.Error("deactivate query must filter to active rows to avoid unnecessary writes")
	}
}

// Integration scenarios covered by the deactivate SQL:
//   Scenario F  — Worker assignments on the project must remain active after reassignment.
//                 The WHERE clause scopes deactivation to role_on_project='poslovoda' only.

func TestPoslovodaSubqueries_ContainLimit1(t *testing.T) {
	if !strings.Contains(poslovodaIDSubquery, "LIMIT 1") {
		t.Error("poslovodaIDSubquery must use LIMIT 1 to prevent duplicate project rows " +
			"when multiple active poslovoda rows exist (defensive guard)")
	}
	if !strings.Contains(poslovodaNameSubquery, "LIMIT 1") {
		t.Error("poslovodaNameSubquery must use LIMIT 1 for the same reason")
	}
	if !strings.Contains(poslovodaIDSubquery, "role_on_project = 'poslovoda'") {
		t.Error("poslovodaIDSubquery must filter by role")
	}
	if !strings.Contains(poslovodaNameSubquery, "role_on_project = 'poslovoda'") {
		t.Error("poslovodaNameSubquery must filter by role")
	}
}

// Integration scenarios covered by LIMIT 1 subqueries:
//   Scenario G  — Project detail shows B (not A) immediately after reassignment.
//   Scenario H  — Project list shows each project exactly once even during a
//                 concurrent reassignment window.
//   Scenario I  — Exactly one active poslovoda assignment exists after success.

func TestPoslovodaSubqueries_ContainOrderBy(t *testing.T) {
	if !strings.Contains(poslovodaIDSubquery, "ORDER BY") {
		t.Error("poslovodaIDSubquery must ORDER BY to make LIMIT 1 deterministic")
	}
	if !strings.Contains(poslovodaIDSubquery, "updated_at") {
		t.Error("poslovodaIDSubquery ORDER BY must include updated_at for recency ordering")
	}
	if !strings.Contains(poslovodaNameSubquery, "ORDER BY") {
		t.Error("poslovodaNameSubquery must ORDER BY to make LIMIT 1 deterministic")
	}
	if !strings.Contains(poslovodaNameSubquery, "updated_at") {
		t.Error("poslovodaNameSubquery ORDER BY must include updated_at for recency ordering")
	}
}

func TestProjectLockSQL_LocksProjectRow(t *testing.T) {
	if !strings.Contains(projectLockSQL, "FROM projects") {
		t.Error("projectLockSQL must lock a row in the projects table, not project_assignments")
	}
	if !strings.Contains(projectLockSQL, "FOR UPDATE") {
		t.Error("projectLockSQL must use FOR UPDATE to serialise concurrent reassignments")
	}
	if !strings.Contains(projectLockSQL, "$1") {
		t.Error("projectLockSQL must be scoped to the target project ID ($1)")
	}
	if !strings.Contains(projectLockSQL, "$2") {
		t.Error("projectLockSQL must be scoped to the company ID ($2)")
	}
}

// Integration scenarios covered by the project row lock:
//   Scenario J  — Two simultaneous PATCH /assign-poslovoda for the same project:
//                 second transaction blocks on FOR UPDATE until first commits.
//                 Combined with the partial unique index (015 migration), even if
//                 the block is somehow bypassed the INSERT will fail at the DB level.

func TestMigration015_ActivePoslovodaUniqueIndex(t *testing.T) {
	// Resolve path relative to this test file's package directory.
	data, err := os.ReadFile("../../../database/migrations/015_active_poslovoda_unique_index.sql")
	if err != nil {
		t.Fatalf("migration 015 not found — create database/migrations/015_active_poslovoda_unique_index.sql: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "uq_project_active_poslovoda") {
		t.Error("migration must create the uq_project_active_poslovoda index")
	}
	if !strings.Contains(content, "role_on_project = 'poslovoda'") {
		t.Error("migration WHERE clause must filter by role_on_project = 'poslovoda'")
	}
	if !strings.Contains(content, "active = true") {
		t.Error("migration WHERE clause must include active = true")
	}
}
