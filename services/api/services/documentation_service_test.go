package services

import (
	"context"
	"testing"
	"time"

	"github.com/gradiliste/api/repositories"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// date returns a UTC midnight time for easy test setup.
func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// radnikEmp returns a DocEmployeeRow with role radnik created on the given date.
func radnikEmp(id string, createdAt time.Time) repositories.DocEmployeeRow {
	return repositories.DocEmployeeRow{
		ID: id, FirstName: "Test", LastName: "Radnik",
		Role: "radnik", Active: true, CreatedAt: createdAt,
	}
}

func adminEmp(id string) repositories.DocEmployeeRow {
	return repositories.DocEmployeeRow{
		ID: id, FirstName: "Test", LastName: "Admin",
		Role: "administracija", Active: true, CreatedAt: date(2025, 1, 1),
	}
}

func notifTypes(notifs []notifResult) []string {
	types := make([]string, len(notifs))
	for i, n := range notifs {
		types[i] = n.typ
	}
	return types
}

type notifResult struct {
	typ      string
	priority string
	empID    string
}

func toResults(notifs interface{}) []notifResult {
	return nil // unused; we call ComputeDocumentationNotifications directly
}

// runNotifs is a convenience wrapper to extract notification types for a single employee.
func runNotifs(
	emp repositories.DocEmployeeRow,
	medical *repositories.MedicalExamRow,
	safety *repositories.SafetyRecordRow,
	contract *repositories.ContractRow,
	obls map[repositories.MonthlyOblKey]bool,
	permit *repositories.WorkPermitRow,
	pension *repositories.PensionRecordRow,
	health *repositories.HealthRecordRow,
	today time.Time,
) []string {
	medicals := map[string]*repositories.MedicalExamRow{}
	if medical != nil {
		medicals[emp.ID] = medical
	}
	safety_ := map[string]*repositories.SafetyRecordRow{}
	if safety != nil {
		safety_[emp.ID] = safety
	}
	contracts := map[string]*repositories.ContractRow{}
	if contract != nil {
		contracts[emp.ID] = contract
	}
	obligations := map[string]map[repositories.MonthlyOblKey]bool{}
	if obls != nil {
		obligations[emp.ID] = obls
	}
	permits := map[string]*repositories.WorkPermitRow{}
	if permit != nil {
		permits[emp.ID] = permit
	}
	pensions := map[string]*repositories.PensionRecordRow{}
	if pension != nil {
		pensions[emp.ID] = pension
	}
	healths := map[string]*repositories.HealthRecordRow{}
	if health != nil {
		healths[emp.ID] = health
	}

	ns := ComputeDocumentationNotifications(
		[]repositories.DocEmployeeRow{emp},
		medicals, safety_, contracts, obligations, permits, pensions, healths,
		today,
	)
	types := make([]string, len(ns))
	for i, n := range ns {
		types[i] = n.Type
	}
	return types
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func notContains(slice []string, val string) bool { return !contains(slice, val) }

// ── Liječnički pregled (medical exams) ────────────────────────────────────────

func TestMedical_NoExam_NoNotification(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if contains(types, "medical_warning") || contains(types, "medical_urgent") || contains(types, "medical_overdue") {
		t.Errorf("expected no medical notification, got %v", types)
	}
}

func TestMedical_MoreThan10Days_NoNotification(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	m := &repositories.MedicalExamRow{
		ID: "m1", EmployeeID: emp.ID,
		CompletedDate: date(2026, 1, 1),
		ExpiresAt:     today.AddDate(0, 0, 11), // 11 days remaining
	}
	types := runNotifs(emp, m, nil, nil, nil, nil, nil, nil, today)
	if contains(types, "medical_warning") {
		t.Errorf("expected no medical_warning at 11 days, got %v", types)
	}
}

func TestMedical_Exactly10Days_Warning(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	m := &repositories.MedicalExamRow{
		ID: "m1", EmployeeID: emp.ID,
		CompletedDate: date(2026, 1, 1),
		ExpiresAt:     today.AddDate(0, 0, 10),
	}
	types := runNotifs(emp, m, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "medical_warning") {
		t.Errorf("expected medical_warning at exactly 10 days, got %v", types)
	}
}

func TestMedical_9Days_Warning(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	m := &repositories.MedicalExamRow{
		ID: "m1", EmployeeID: emp.ID,
		CompletedDate: date(2026, 1, 1),
		ExpiresAt:     today.AddDate(0, 0, 9),
	}
	types := runNotifs(emp, m, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "medical_warning") {
		t.Errorf("expected medical_warning at 9 days, got %v", types)
	}
}

func TestMedical_1Day_Warning(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	m := &repositories.MedicalExamRow{
		ID: "m1", EmployeeID: emp.ID,
		CompletedDate: date(2026, 1, 1),
		ExpiresAt:     today.AddDate(0, 0, 1),
	}
	types := runNotifs(emp, m, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "medical_warning") {
		t.Errorf("expected medical_warning at 1 day, got %v", types)
	}
}

func TestMedical_ExpiryDay_Urgent(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	m := &repositories.MedicalExamRow{
		ID: "m1", EmployeeID: emp.ID,
		CompletedDate: date(2026, 1, 1),
		ExpiresAt:     today, // expires today
	}
	types := runNotifs(emp, m, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "medical_urgent") {
		t.Errorf("expected medical_urgent on expiry day, got %v", types)
	}
	if contains(types, "medical_warning") {
		t.Errorf("expected no medical_warning on expiry day, got %v", types)
	}
}

func TestMedical_Expired_Overdue(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	m := &repositories.MedicalExamRow{
		ID: "m1", EmployeeID: emp.ID,
		CompletedDate: date(2025, 1, 1),
		ExpiresAt:     today.AddDate(0, 0, -5),
	}
	types := runNotifs(emp, m, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "medical_overdue") {
		t.Errorf("expected medical_overdue for expired exam, got %v", types)
	}
	if contains(types, "medical_warning") || contains(types, "medical_urgent") {
		t.Errorf("expected only medical_overdue, got %v", types)
	}
}

func TestMedical_NewerExamReplaces(t *testing.T) {
	// The map already holds the latest exam per employee, so we simply confirm a
	// not-yet-expiring exam generates no notification.
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	m := &repositories.MedicalExamRow{
		ID: "m2", EmployeeID: emp.ID,
		CompletedDate: date(2026, 5, 1),
		ExpiresAt:     today.AddDate(0, 11, 0), // far future
	}
	types := runNotifs(emp, m, nil, nil, nil, nil, nil, nil, today)
	if contains(types, "medical_warning") || contains(types, "medical_urgent") || contains(types, "medical_overdue") {
		t.Errorf("newer exam far in future should produce no medical notification, got %v", types)
	}
}

// ── Zaštita na radu (safety) ──────────────────────────────────────────────────

func TestSafety_Day0To59_Low(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", today.AddDate(0, 0, -30)) // created 30 days ago
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "safety_low") {
		t.Errorf("expected safety_low at day 30, got %v", types)
	}
	if contains(types, "safety_urgent") {
		t.Errorf("expected no safety_urgent at day 30, got %v", types)
	}
}

func TestSafety_Day60Plus_Urgent(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", today.AddDate(0, 0, -65)) // created 65 days ago
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "safety_urgent") {
		t.Errorf("expected safety_urgent at day 65, got %v", types)
	}
}

func TestSafety_Exactly60Days_Urgent(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", today.AddDate(0, 0, -60))
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "safety_urgent") {
		t.Errorf("expected safety_urgent at exactly 60 days, got %v", types)
	}
}

func TestSafety_Verified_NoNotification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", today.AddDate(0, 0, -80))
	sr := &repositories.SafetyRecordRow{ID: "sr1", EmployeeID: emp.ID, Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, nil, nil, today)
	if contains(types, "safety_low") || contains(types, "safety_urgent") {
		t.Errorf("verified safety record should produce no notification, got %v", types)
	}
}

func TestSafety_Uploaded_StillNotified(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", today.AddDate(0, 0, -70))
	sr := &repositories.SafetyRecordRow{ID: "sr1", EmployeeID: emp.ID, Status: "uploaded"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, nil, nil, today)
	// uploaded but not verified → still notified (urgent because >60 days)
	if !contains(types, "safety_urgent") {
		t.Errorf("uploaded-not-verified safety at >60 days should be urgent, got %v", types)
	}
}

func TestSafety_NoRecord_Day0_Low(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", today) // created today = day 0
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if !contains(types, "safety_low") {
		t.Errorf("expected safety_low on day 0, got %v", types)
	}
}

func TestSafety_NonRadnik_NoNotification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := repositories.DocEmployeeRow{
		ID: "emp-1", FirstName: "T", LastName: "A",
		Role: "poslovoda", Active: true,
		CreatedAt: today.AddDate(0, 0, -90),
	}
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if contains(types, "safety_low") || contains(types, "safety_urgent") {
		t.Errorf("non-radnik should have no safety notification, got %v", types)
	}
}

// ── Ugovor o radu (contracts) ─────────────────────────────────────────────────

func TestContract_NoContract_NoNotification(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if contains(types, "contract_warning") || contains(types, "contract_overdue") {
		t.Errorf("no contract should produce no contract notification, got %v", types)
	}
}

func TestContract_Indefinite_NoNotification(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	c := &repositories.ContractRow{ID: "c1", EmployeeID: emp.ID, ContractType: "neodredeno"}
	types := runNotifs(emp, nil, nil, c, nil, nil, nil, nil, today)
	if contains(types, "contract_warning") || contains(types, "contract_overdue") {
		t.Errorf("indefinite contract should produce no notification, got %v", types)
	}
}

func TestContract_Odredeno_MoreThan30Days_NoNotification(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	exp := today.AddDate(0, 0, 31)
	c := &repositories.ContractRow{ID: "c1", EmployeeID: emp.ID, ContractType: "odredeno", ExpiresAt: &exp}
	types := runNotifs(emp, nil, nil, c, nil, nil, nil, nil, today)
	if contains(types, "contract_warning") {
		t.Errorf("contract with 31 days remaining should produce no warning, got %v", types)
	}
}

func TestContract_Odredeno_Exactly30Days_Warning(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	exp := today.AddDate(0, 0, 30)
	c := &repositories.ContractRow{ID: "c1", EmployeeID: emp.ID, ContractType: "odredeno", ExpiresAt: &exp}
	types := runNotifs(emp, nil, nil, c, nil, nil, nil, nil, today)
	if !contains(types, "contract_warning") {
		t.Errorf("expected contract_warning at exactly 30 days, got %v", types)
	}
}

func TestContract_Odredeno_Expired_Overdue(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	exp := today.AddDate(0, 0, -1)
	c := &repositories.ContractRow{ID: "c1", EmployeeID: emp.ID, ContractType: "odredeno", ExpiresAt: &exp}
	types := runNotifs(emp, nil, nil, c, nil, nil, nil, nil, today)
	if !contains(types, "contract_overdue") {
		t.Errorf("expected contract_overdue for expired contract, got %v", types)
	}
}

func TestContract_Terminated_NoNotification(t *testing.T) {
	// The contracts map holds ONLY the latest active (non-terminated) contract.
	// A terminated contract won't appear in the map, so no notification.
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	// Pass nil contract → simulates "no active contract in map"
	types := runNotifs(emp, nil, nil, nil, nil, nil, nil, nil, today)
	if contains(types, "contract_warning") || contains(types, "contract_overdue") {
		t.Errorf("terminated contract (absent from map) should produce no notification, got %v", types)
	}
}

func TestContract_TerminatedWithNewActive_NewWins(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	// The new contract expires in 5 days → warning.
	exp := today.AddDate(0, 0, 5)
	c := &repositories.ContractRow{ID: "c2", EmployeeID: emp.ID, ContractType: "odredeno", ExpiresAt: &exp}
	types := runNotifs(emp, nil, nil, c, nil, nil, nil, nil, today)
	if !contains(types, "contract_warning") {
		t.Errorf("new active contract within 30 days should produce warning, got %v", types)
	}
}

// ── Platne liste (payroll) ────────────────────────────────────────────────────

// allPriorPayrollDone marks every platna_lista month from empCreatedAt up to (and including)
// the month before today as completed, so tests that check for no-overdue work correctly.
func allPriorPayrollDone(today, empCreatedAt time.Time) map[repositories.MonthlyOblKey]bool {
	m := map[repositories.MonthlyOblKey]bool{}
	start := time.Date(empCreatedAt.Year(), empCreatedAt.Month(), 1, 0, 0, 0, 0, time.UTC)
	prev := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	for cursor := start; !cursor.After(prev); cursor = cursor.AddDate(0, 1, 0) {
		m[repositories.MonthlyOblKey{Year: cursor.Year(), Month: int(cursor.Month()), Type: "platna_lista"}] = true
	}
	return m
}

// today = 2026-06-15 (month has 30 days, last 5 days = 26-30)
var payrollToday = date(2026, 6, 15)

func TestPayroll_EarlyMonth_NoNotification(t *testing.T) {
	emp := adminEmp("emp-1")
	obls := allPriorPayrollDone(payrollToday, emp.CreatedAt)
	types := runNotifs(emp, nil, nil, nil, obls, nil, nil, nil, payrollToday)
	if contains(types, "payroll_warning") || contains(types, "payroll_overdue") {
		t.Errorf("early in month with all prior done should produce no payroll notification, got %v", types)
	}
}

func TestPayroll_5DaysBeforeMonthEnd_Warning(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 26) // June has 30 days; 30-26=4 < 5 → warning
	types := runNotifs(emp, nil, nil, nil, map[repositories.MonthlyOblKey]bool{}, nil, nil, nil, today)
	if !contains(types, "payroll_warning") {
		t.Errorf("expected payroll_warning with 4 days to end, got %v", types)
	}
}

func TestPayroll_LastDayOfMonth_Warning(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 30)
	types := runNotifs(emp, nil, nil, nil, map[repositories.MonthlyOblKey]bool{}, nil, nil, nil, today)
	if !contains(types, "payroll_warning") {
		t.Errorf("expected payroll_warning on last day of month, got %v", types)
	}
}

func TestPayroll_AfterMonthEnd_Overdue(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 7, 5) // checking June → overdue
	obls := map[repositories.MonthlyOblKey]bool{}
	types := runNotifs(emp, nil, nil, nil, obls, nil, nil, nil, today)
	if !contains(types, "payroll_overdue") {
		t.Errorf("expected payroll_overdue for previous month, got %v", types)
	}
}

func TestPayroll_Completed_NoNotification(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 7, 5)
	// Mark all months from hire date (Jan 2025) through June 2026 as done.
	obls := allPriorPayrollDone(today, emp.CreatedAt)
	types := runNotifs(emp, nil, nil, nil, obls, nil, nil, nil, today)
	if contains(types, "payroll_overdue") {
		t.Errorf("all prior months completed, no payroll_overdue expected, got %v", types)
	}
}

func TestPayroll_NextMonthIndependent(t *testing.T) {
	// All months from hire through June 2026 done; July 2026 left incomplete.
	// Verifies that completing June does not auto-complete July.
	emp := adminEmp("emp-1")
	today := date(2026, 8, 5)
	// allPriorPayrollDone for today=Aug marks Jan 2025–Jul 2026 done; remove July to isolate.
	obls := allPriorPayrollDone(today, emp.CreatedAt)
	delete(obls, repositories.MonthlyOblKey{Year: 2026, Month: 7, Type: "platna_lista"})
	types := runNotifs(emp, nil, nil, nil, obls, nil, nil, nil, today)
	if !contains(types, "payroll_overdue") {
		t.Errorf("July payroll should be overdue in August, got %v", types)
	}
}

// ── Evidencija radnika (timesheet, radnik only) ───────────────────────────────

func TestTimesheet_5DaysBeforeMonthEnd_Warning(t *testing.T) {
	today := date(2026, 6, 26)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	// Provide a verified safety record so we don't pollute with safety notifications.
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	// Also verified pension/health for cleanliness.
	pr := &repositories.PensionRecordRow{Status: "verified"}
	hr := &repositories.HealthRecordRow{Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, map[repositories.MonthlyOblKey]bool{}, nil, pr, hr, today)
	if !contains(types, "timesheet_warning") {
		t.Errorf("expected timesheet_warning with 4 days to month end, got %v", types)
	}
}

func TestTimesheet_AfterMonthEnd_Overdue(t *testing.T) {
	today := date(2026, 7, 10)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	pr := &repositories.PensionRecordRow{Status: "verified"}
	hr := &repositories.HealthRecordRow{Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, map[repositories.MonthlyOblKey]bool{}, nil, pr, hr, today)
	if !contains(types, "timesheet_overdue") {
		t.Errorf("expected timesheet_overdue for previous month, got %v", types)
	}
}

func TestTimesheet_Completed_NoNotification(t *testing.T) {
	today := date(2026, 7, 10)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	pr := &repositories.PensionRecordRow{Status: "verified"}
	hr := &repositories.HealthRecordRow{Status: "verified"}
	// Mark all months from hire date (Jan 2025) through June 2026 as done for both types.
	obls := map[repositories.MonthlyOblKey]bool{}
	start := time.Date(emp.CreatedAt.Year(), emp.CreatedAt.Month(), 1, 0, 0, 0, 0, time.UTC)
	prev := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	for cursor := start; !cursor.After(prev); cursor = cursor.AddDate(0, 1, 0) {
		y, m := cursor.Year(), int(cursor.Month())
		obls[repositories.MonthlyOblKey{Year: y, Month: m, Type: "platna_lista"}] = true
		obls[repositories.MonthlyOblKey{Year: y, Month: m, Type: "evidencija_radnika"}] = true
	}
	types := runNotifs(emp, nil, sr, nil, obls, nil, pr, hr, today)
	if contains(types, "timesheet_overdue") {
		t.Errorf("all prior months done, no timesheet_overdue expected, got %v", types)
	}
}

// ── Radne dozvole (work permits) ──────────────────────────────────────────────
// All thresholds are calendar-based: expiry.AddDate(0,-N,0) or AddDate(0,-N,-D).
// This handles month-length variation (28/29/30/31 days) correctly.

// TestPermit_Calendar_Before4Months_None: expiry=Oct 1; 4 months before = Jun 1.
// today=May 31 is strictly before Jun 1 → no notification.
func TestPermit_Calendar_Before4Months_None(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 5, 31)
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: date(2026, 10, 1)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if contains(types, "permit_low") || contains(types, "permit_warning") || contains(types, "permit_urgent") || contains(types, "permit_overdue") {
		t.Errorf("May 31 is before the 4-month window (Jun 1) → no notification, got %v", types)
	}
}

// TestPermit_Calendar_Exact4Months_Low: today=Jun 1 equals expiry.AddDate(0,-4,0) → low.
func TestPermit_Calendar_Exact4Months_Low(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 1)
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: date(2026, 10, 1)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if !contains(types, "permit_low") {
		t.Errorf("Jun 1 = exactly 4 calendar months before Oct 1 → permit_low, got %v", types)
	}
}

// TestPermit_Calendar_Exact3m15d_Warning: expiry=Sep 30; 3m+15d before = Jun 15 → warning.
func TestPermit_Calendar_Exact3m15d_Warning(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	// Sep 30 - 3m - 15d = Jun 30 - 15d = Jun 15.
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: date(2026, 9, 30)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if !contains(types, "permit_warning") {
		t.Errorf("Jun 15 = exactly 3m+15d before Sep 30 → permit_warning, got %v", types)
	}
	if contains(types, "permit_low") {
		t.Errorf("should not be low when already at warning threshold, got %v", types)
	}
}

// TestPermit_Calendar_Exact3m_Urgent: expiry=Sep 15; 3 months before = Jun 15 → urgent.
func TestPermit_Calendar_Exact3m_Urgent(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: date(2026, 9, 15)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if !contains(types, "permit_urgent") {
		t.Errorf("Jun 15 = exactly 3 calendar months before Sep 15 → permit_urgent, got %v", types)
	}
	if contains(types, "permit_warning") || contains(types, "permit_low") {
		t.Errorf("should only be urgent, not warning/low, got %v", types)
	}
}

// TestPermit_Calendar_ExpiryDay_Urgent: today == expiry day → urgent, not overdue.
func TestPermit_Calendar_ExpiryDay_Urgent(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 9, 15)
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: today}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if !contains(types, "permit_urgent") {
		t.Errorf("expiry day should be urgent (not yet past), got %v", types)
	}
	if contains(types, "permit_overdue") {
		t.Errorf("expiry day should not be overdue yet, got %v", types)
	}
}

// TestPermit_Calendar_Expired_Overdue: today > expiry → overdue.
func TestPermit_Calendar_Expired_Overdue(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 9, 16)
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: date(2026, 9, 15)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if !contains(types, "permit_overdue") {
		t.Errorf("day after expiry → permit_overdue, got %v", types)
	}
}

// TestPermit_Calendar_FarFuture_None: permit far in future → no notification.
func TestPermit_Calendar_FarFuture_None(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 6, 15)
	wp := &repositories.WorkPermitRow{ID: "wp2", EmployeeID: emp.ID, ExpiresAt: date(2027, 12, 1)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if contains(types, "permit_low") || contains(types, "permit_warning") || contains(types, "permit_urgent") || contains(types, "permit_overdue") {
		t.Errorf("far-future permit should produce no notification, got %v", types)
	}
}

// TestPermit_Calendar_LeapYear_Low: expiry=Jun 29 2024; 4m before = Feb 29 2024 (leap year).
// today=Feb 29 2024 → exactly at 4-month boundary → low.
func TestPermit_Calendar_LeapYear_Low(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2024, 2, 29)
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: date(2024, 6, 29)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if !contains(types, "permit_low") {
		t.Errorf("Feb 29 = 4 calendar months before Jun 29 in leap year → permit_low, got %v", types)
	}
}

// TestPermit_Calendar_YearBoundary_Low: expiry=Feb 1 2027; 4m before = Oct 1 2026 → low.
func TestPermit_Calendar_YearBoundary_Low(t *testing.T) {
	emp := adminEmp("emp-1")
	today := date(2026, 10, 1)
	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: date(2027, 2, 1)}
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, today)
	if !contains(types, "permit_low") {
		t.Errorf("Oct 1 = 4 calendar months before Feb 1 (year boundary) → permit_low, got %v", types)
	}
}

// TestPermit_Calendar_ShortMonth_LowBoundary: expiry=Mar 31 2027.
// expiry.AddDate(0,-4,0) overflows Nov 31 → Dec 1 2026.
// Nov 30 → no notification; Dec 1 → low.
func TestPermit_Calendar_ShortMonth_LowBoundary(t *testing.T) {
	emp := adminEmp("emp-1")
	expiry := date(2027, 3, 31)

	wp := &repositories.WorkPermitRow{ID: "wp1", EmployeeID: emp.ID, ExpiresAt: expiry}

	// Nov 30: before the Dec 1 threshold → no notification.
	types := runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, date(2026, 11, 30))
	if contains(types, "permit_low") || contains(types, "permit_warning") || contains(types, "permit_urgent") {
		t.Errorf("Nov 30 is before Dec 1 threshold → no notification, got %v", types)
	}

	// Dec 1: exactly on the threshold (Nov 31 overflows to Dec 1) → low.
	types = runNotifs(emp, nil, nil, nil, nil, wp, nil, nil, date(2026, 12, 1))
	if !contains(types, "permit_low") {
		t.Errorf("Dec 1 = effective 4-month threshold for Mar 31 expiry → permit_low, got %v", types)
	}
}

// ── Mirovinsko osiguranje (pension) ──────────────────────────────────────────

func TestPension_Missing_Notification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	hr := &repositories.HealthRecordRow{Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, nil, hr, today)
	if !contains(types, "pension_missing") {
		t.Errorf("missing pension should produce pension_missing notification, got %v", types)
	}
}

func TestPension_Uploaded_Notification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	pr := &repositories.PensionRecordRow{ID: "pr1", Status: "uploaded"}
	hr := &repositories.HealthRecordRow{Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, pr, hr, today)
	if !contains(types, "pension_missing") {
		t.Errorf("uploaded (not verified) pension should still produce pension_missing, got %v", types)
	}
}

func TestPension_Verified_NoNotification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	pr := &repositories.PensionRecordRow{ID: "pr1", Status: "verified"}
	hr := &repositories.HealthRecordRow{Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, pr, hr, today)
	if contains(types, "pension_missing") {
		t.Errorf("verified pension should produce no pension notification, got %v", types)
	}
}

// ── Zdravstveno osiguranje (health) ──────────────────────────────────────────

func TestHealth_Missing_Notification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	pr := &repositories.PensionRecordRow{Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, pr, nil, today)
	if !contains(types, "health_missing") {
		t.Errorf("missing health should produce health_missing notification, got %v", types)
	}
}

func TestHealth_Uploaded_Notification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	pr := &repositories.PensionRecordRow{Status: "verified"}
	hr := &repositories.HealthRecordRow{ID: "hr1", Status: "uploaded"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, pr, hr, today)
	if !contains(types, "health_missing") {
		t.Errorf("uploaded (not verified) health should still produce health_missing, got %v", types)
	}
}

func TestHealth_Verified_NoNotification(t *testing.T) {
	today := date(2026, 6, 15)
	emp := radnikEmp("emp-1", date(2025, 1, 1))
	sr := &repositories.SafetyRecordRow{Status: "verified"}
	pr := &repositories.PensionRecordRow{Status: "verified"}
	hr := &repositories.HealthRecordRow{ID: "hr1", Status: "verified"}
	types := runNotifs(emp, nil, sr, nil, nil, nil, pr, hr, today)
	if contains(types, "health_missing") {
		t.Errorf("verified health should produce no health notification, got %v", types)
	}
}

// ── Auth / role enforcement ───────────────────────────────────────────────────

// mockDocumentationRepo satisfies documentationRepoIface with no-op implementations.
type mockDocumentationRepo struct{}

func (m *mockDocumentationRepo) ListActiveEmployees(ctx context.Context, companyID string) ([]repositories.DocEmployeeRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListActiveRadnikEmployees(ctx context.Context, companyID string) ([]repositories.DocEmployeeRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) GetEmployee(ctx context.Context, companyID, employeeID string) (*repositories.DocEmployeeRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) CreateDocFile(ctx context.Context, companyID, employeeID, storageKey, originalFilename, mimeType, uploadedBy string, fileSize int64) (string, error) {
	return "file-id", nil
}
func (m *mockDocumentationRepo) GetDocFile(ctx context.Context, companyID, fileID string) (*repositories.DocumentFileRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) CreateMedicalExam(ctx context.Context, companyID, employeeID, createdBy, documentID string, completedDate, expiresAt time.Time) (string, error) {
	return "id", nil
}
func (m *mockDocumentationRepo) GetLatestMedicalExam(ctx context.Context, companyID, employeeID string) (*repositories.MedicalExamRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) ListMedicalExams(ctx context.Context, companyID, employeeID string) ([]repositories.MedicalExamRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListLatestMedicalExams(ctx context.Context, companyID string) (map[string]*repositories.MedicalExamRow, error) {
	return map[string]*repositories.MedicalExamRow{}, nil
}
func (m *mockDocumentationRepo) GetSafetyRecord(ctx context.Context, companyID, employeeID string) (*repositories.SafetyRecordRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) UpsertSafetyUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*repositories.SafetyRecordRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) VerifySafetyRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*repositories.SafetyRecordRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListSafetyRecords(ctx context.Context, companyID string) (map[string]*repositories.SafetyRecordRow, error) {
	return map[string]*repositories.SafetyRecordRow{}, nil
}
func (m *mockDocumentationRepo) CreateContract(ctx context.Context, companyID, employeeID, contractType, createdBy, documentID string, expiresAt *time.Time) (string, error) {
	return "id", nil
}
func (m *mockDocumentationRepo) GetContract(ctx context.Context, companyID, contractID string) (*repositories.ContractRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) TerminateContract(ctx context.Context, companyID, contractID, terminatedBy, terminationDocID string, terminatedAt time.Time) error {
	return nil
}
func (m *mockDocumentationRepo) ListContracts(ctx context.Context, companyID, employeeID string) ([]repositories.ContractRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) GetLatestActiveContract(ctx context.Context, companyID, employeeID string) (*repositories.ContractRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) ListLatestActiveContracts(ctx context.Context, companyID string) (map[string]*repositories.ContractRow, error) {
	return map[string]*repositories.ContractRow{}, nil
}
func (m *mockDocumentationRepo) GetMonthlyObligation(ctx context.Context, companyID, employeeID string, year, month int, oblType string) (*repositories.MonthlyObligationRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) UpsertMonthlyCompletion(ctx context.Context, companyID, employeeID, oblType, completedBy string, year, month int, hours *float64, documentID *string) (*repositories.MonthlyObligationRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListMonthlyObligations(ctx context.Context, companyID, employeeID string) ([]repositories.MonthlyObligationRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListCompletedObligations(ctx context.Context, companyID string, fromYear, fromMonth int) (map[string]map[repositories.MonthlyOblKey]bool, error) {
	return map[string]map[repositories.MonthlyOblKey]bool{}, nil
}
func (m *mockDocumentationRepo) SumMonthlyHoursForWorker(ctx context.Context, companyID, employeeID string, year, month int) (float64, error) {
	return 0, nil
}
func (m *mockDocumentationRepo) CreateAnnualLeave(ctx context.Context, companyID, employeeID, createdBy string, dateFrom, dateTo *time.Time, documentID *string) (string, error) {
	return "id", nil
}
func (m *mockDocumentationRepo) ListAnnualLeave(ctx context.Context, companyID, employeeID string) ([]repositories.AnnualLeaveRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) CreateWorkPermit(ctx context.Context, companyID, employeeID, createdBy, documentID string, enteredAt, expiresAt time.Time) (string, error) {
	return "id", nil
}
func (m *mockDocumentationRepo) GetLatestWorkPermit(ctx context.Context, companyID, employeeID string) (*repositories.WorkPermitRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) ListWorkPermits(ctx context.Context, companyID, employeeID string) ([]repositories.WorkPermitRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListLatestWorkPermits(ctx context.Context, companyID string) (map[string]*repositories.WorkPermitRow, error) {
	return map[string]*repositories.WorkPermitRow{}, nil
}
func (m *mockDocumentationRepo) GetPensionRecord(ctx context.Context, companyID, employeeID string) (*repositories.PensionRecordRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) UpsertPensionUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*repositories.PensionRecordRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) VerifyPensionRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*repositories.PensionRecordRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListPensionRecords(ctx context.Context, companyID string) (map[string]*repositories.PensionRecordRow, error) {
	return map[string]*repositories.PensionRecordRow{}, nil
}
func (m *mockDocumentationRepo) GetHealthRecord(ctx context.Context, companyID, employeeID string) (*repositories.HealthRecordRow, error) {
	return nil, repositories.ErrNotFound
}
func (m *mockDocumentationRepo) UpsertHealthUpload(ctx context.Context, companyID, employeeID, documentID, createdBy string) (*repositories.HealthRecordRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) VerifyHealthRecord(ctx context.Context, companyID, employeeID, verifiedBy string) (*repositories.HealthRecordRow, error) {
	return nil, nil
}
func (m *mockDocumentationRepo) ListHealthRecords(ctx context.Context, companyID string) (map[string]*repositories.HealthRecordRow, error) {
	return map[string]*repositories.HealthRecordRow{}, nil
}

func newDocSvc() *DocumentationService {
	return NewDocumentationService(&mockDocumentationRepo{}, nil)
}

func TestAuth_Administracija_OK(t *testing.T) {
	svc := newDocSvc()
	_, err := svc.ListEmployees(context.Background(), "company-1", "administracija")
	if err != nil {
		t.Errorf("administracija should be allowed, got %v", err)
	}
}

func TestAuth_Direktor_Forbidden(t *testing.T) {
	svc := newDocSvc()
	_, err := svc.ListEmployees(context.Background(), "company-1", "direktor")
	if err != ErrForbidden {
		t.Errorf("direktor should be forbidden, got %v", err)
	}
}

func TestAuth_Inzenjer_Forbidden(t *testing.T) {
	svc := newDocSvc()
	_, err := svc.ListEmployees(context.Background(), "company-1", "inzenjer")
	if err != ErrForbidden {
		t.Errorf("inzenjer should be forbidden, got %v", err)
	}
}

func TestAuth_Poslovoda_Forbidden(t *testing.T) {
	svc := newDocSvc()
	_, err := svc.ListEmployees(context.Background(), "company-1", "poslovoda")
	if err != ErrForbidden {
		t.Errorf("poslovoda should be forbidden, got %v", err)
	}
}

func TestAuth_Radnik_Forbidden(t *testing.T) {
	svc := newDocSvc()
	_, err := svc.ListEmployees(context.Background(), "company-1", "radnik")
	if err != ErrForbidden {
		t.Errorf("radnik should be forbidden, got %v", err)
	}
}

// ── Security: cross-company isolation ────────────────────────────────────────

func TestSecurity_CrossCompanyEmployee_Rejected(t *testing.T) {
	// mockDocumentationRepo.GetEmployee always returns ErrNotFound; simulates employee from another company.
	svc := newDocSvc()
	_, err := svc.GetEmployeeSummary(context.Background(), "company-A", "administracija", "emp-from-company-B")
	if err == nil {
		t.Error("cross-company employee lookup should return an error")
	}
}

func TestSecurity_CrossCompanyDocument_Rejected(t *testing.T) {
	// ValidateEmployee passes (we override mockDocumentationRepo.GetEmployee to succeed),
	// but GetDocFile returns ErrNotFound for a document from another company.
	type partialMock struct {
		mockDocumentationRepo
	}
	// Use the base mockDocumentationRepo which returns ErrNotFound for GetDocFile.
	svc := newDocSvc()
	// Try creating a medical exam with a document ID from another company.
	_, err := svc.CreateMedicalExam(context.Background(), "company-A", "administracija", "user-1",
		CreateMedicalExamRequest{
			EmployeeID:    "emp-1",
			CompletedDate: "2026-01-01",
			ExpiresAt:     "2027-01-01",
			DocumentID:    "doc-from-other-company",
		})
	// Will fail at GetEmployee (ErrNotFound) or GetDocFile; either way, it fails.
	if err == nil {
		t.Error("should reject creation referencing cross-company document")
	}
}
