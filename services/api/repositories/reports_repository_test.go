package repositories

import (
	"strings"
	"testing"

	"github.com/gradiliste/api/dto"
)

// pstr returns a pointer to s.
func pstr(s string) *string { return &s }

// assertNoUUIDCast fails if the WHERE SQL contains any ::uuid cast.
// The combined CTE exposes project_id, worker_id, poslovoda_id as text;
// comparing them with ::uuid causes SQLSTATE 42883 at runtime.
func assertNoUUIDCast(t *testing.T, label, sql string) {
	t.Helper()
	if strings.Contains(sql, "::uuid") {
		t.Errorf("%s: WHERE must not contain ::uuid; got:\n%s", label, sql)
	}
}

// assertContains fails if sql does not contain want.
func assertContains(t *testing.T, label, sql, want string) {
	t.Helper()
	if !strings.Contains(sql, want) {
		t.Errorf("%s: expected %q in WHERE; got:\n%s", label, want, sql)
	}
}

// assertNotContains fails if sql contains the forbidden fragment.
func assertNotContains(t *testing.T, label, sql, bad string) {
	t.Helper()
	if strings.Contains(sql, bad) {
		t.Errorf("%s: forbidden fragment %q found in WHERE; got:\n%s", label, bad, sql)
	}
}

// ── No filter ─────────────────────────────────────────────────────────────────

func TestBuildDnevnikWhere_NoFilter(t *testing.T) {
	w := buildDnevnikWhere("cid", dto.ReportFilter{})
	if w.sql() != "" {
		t.Errorf("expected empty WHERE for no filter, got: %s", w.sql())
	}
	if len(w.args) != 1 {
		t.Errorf("expected 1 arg (companyID), got %d", len(w.args))
	}
}

// ── Project filter ────────────────────────────────────────────────────────────

func TestBuildDnevnikWhere_ProjectFilter(t *testing.T) {
	f := dto.ReportFilter{ProjectID: pstr("proj-uuid")}
	w := buildDnevnikWhere("cid", f)
	sql := w.sql()

	assertNoUUIDCast(t, "project filter", sql)
	// Must use alias-qualified text comparison.
	assertContains(t, "project filter", sql, "c.project_id = $2::text")

	if len(w.args) != 2 {
		t.Errorf("expected 2 args (companyID, projectID), got %d", len(w.args))
	}
	if w.args[1] != "proj-uuid" {
		t.Errorf("expected args[1]=%q, got %v", "proj-uuid", w.args[1])
	}
}

// ── Worker filter ─────────────────────────────────────────────────────────────

func TestBuildDnevnikWhere_WorkerFilter(t *testing.T) {
	f := dto.ReportFilter{WorkerID: pstr("worker-uuid")}
	w := buildDnevnikWhere("cid", f)
	sql := w.sql()

	assertNoUUIDCast(t, "worker filter", sql)
	assertContains(t, "worker filter", sql, "c.worker_id = $2::text")

	if len(w.args) != 2 {
		t.Errorf("expected 2 args (companyID, workerID), got %d", len(w.args))
	}
}

// ── Poslovoda filter ──────────────────────────────────────────────────────────

func TestBuildDnevnikWhere_PoslovodaFilter(t *testing.T) {
	f := dto.ReportFilter{PoslovodaID: pstr("pos-uuid")}
	w := buildDnevnikWhere("cid", f)
	sql := w.sql()

	// No ::uuid anywhere — both $N uses must be ::text.
	assertNoUUIDCast(t, "poslovoda filter", sql)

	// Direct daily-report match uses alias-qualified column and ::text.
	assertContains(t, "poslovoda filter", sql, "c.poslovoda_id = $2::text")
	// EXISTS join must cast pa.employee_id to text, same placeholder ::text.
	assertContains(t, "poslovoda filter", sql, "pa.employee_id::text = $2::text")

	// Correlated column must be explicitly qualified to avoid pa.project_id shadowing.
	assertContains(t, "poslovoda filter", sql, "pa.project_id::text = c.project_id")
	// Reject the unqualified (ambiguous) form.
	assertNotContains(t, "poslovoda filter", sql, "pa.project_id::text = project_id")

	// Radnik branch present.
	assertContains(t, "poslovoda filter", sql, "c.poslovoda_id = ''")
	assertContains(t, "poslovoda filter", sql, "project_assignments")
	assertContains(t, "poslovoda filter", sql, "role_on_project = 'poslovoda'")

	// Only one argument for this condition.
	if len(w.args) != 2 {
		t.Errorf("expected 2 args (companyID, poslovodaID), got %d", len(w.args))
	}
}

// ── Poslovoda + project + worker (max-stress combination) ────────────────────

func TestBuildDnevnikWhere_PoslovodaProjectWorker(t *testing.T) {
	f := dto.ReportFilter{
		PoslovodaID: pstr("pos-uuid"),
		ProjectID:   pstr("proj-uuid"),
		WorkerID:    pstr("worker-uuid"),
	}
	w := buildDnevnikWhere("cid", f)
	sql := w.sql()

	assertNoUUIDCast(t, "poslovoda+project+worker", sql)

	// Arg order: companyID($1), poslovodaID($2), projectID($3), workerID($4).
	assertContains(t, "poslovoda+project+worker", sql, "c.poslovoda_id = $2::text")
	assertContains(t, "poslovoda+project+worker", sql, "pa.employee_id::text = $2::text")
	assertContains(t, "poslovoda+project+worker", sql, "pa.project_id::text = c.project_id")
	assertContains(t, "poslovoda+project+worker", sql, "c.project_id = $3::text")
	assertContains(t, "poslovoda+project+worker", sql, "c.worker_id = $4::text")

	// Unqualified form must not appear.
	assertNotContains(t, "poslovoda+project+worker", sql, "pa.project_id::text = project_id")

	if len(w.args) != 4 {
		t.Errorf("expected 4 args, got %d", len(w.args))
	}
}

// ── Radnik empty poslovoda_id branch ─────────────────────────────────────────

func TestBuildDnevnikWhere_RadnikEmptyPoslovoda(t *testing.T) {
	f := dto.ReportFilter{PoslovodaID: pstr("pos-uuid")}
	w := buildDnevnikWhere("cid", f)
	sql := w.sql()

	assertContains(t, "radnik branch", sql, "c.poslovoda_id = ''")
	assertContains(t, "radnik branch", sql, "EXISTS")
	// Correlated reference must be qualified.
	assertContains(t, "radnik branch", sql, "c.project_id")
	assertNotContains(t, "radnik branch", sql, "pa.project_id::text = project_id")
}

// ── Date filters do not introduce uuid casts ──────────────────────────────────

func TestBuildDnevnikWhere_DateFilters(t *testing.T) {
	f := dto.ReportFilter{
		DateFrom: pstr("2024-01-01"),
		DateTo:   pstr("2024-12-31"),
	}
	w := buildDnevnikWhere("cid", f)
	sql := w.sql()

	assertNoUUIDCast(t, "date filters", sql)
	assertContains(t, "date filters", sql, "c.report_date >=")
	assertContains(t, "date filters", sql, "c.report_date <=")
}

// ── Project-only arg count and next-idx ──────────────────────────────────────

func TestBuildDnevnikWhere_ProjectOnly_ArgCount(t *testing.T) {
	f := dto.ReportFilter{ProjectID: pstr("proj-uuid")}
	w := buildDnevnikWhere("cid", f)
	if len(w.args) != 2 {
		t.Errorf("project-only: want 2 args, got %d", len(w.args))
	}
	if w.idx != 3 {
		t.Errorf("project-only: want next idx=3, got %d", w.idx)
	}
}
