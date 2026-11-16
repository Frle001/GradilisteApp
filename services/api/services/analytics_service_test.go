package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Mock repository ───────────────────────────────────────────────────────────

type mockAnalyticsRepo struct {
	plans        []repositories.CompensationPlanRow
	dailyHours   []repositories.DailyHoursRecord
	projectHours []repositories.ProjectHours
	employees    []repositories.AnalyticsEmployeeRow
	openPlan     *repositories.CompensationPlanRow
	hasOverlap   bool

	createdPlan      *repositories.CompensationPlanRow
	closedPlanID     string
	closedPlanDate   time.Time
	getPlanByIDError error
	updatedPlan      *repositories.CompensationPlanRow

	// Contribution stubs
	contribution     *repositories.ContributionRow
	contribsForMonth []repositories.ContributionRow
	contribsForYear  []repositories.ContributionRow
	upsertedContrib  *repositories.ContributionRow

	// Material cost stubs
	matCostsByProject map[string]float64
	matBreakdown      []repositories.ProjectMaterialCostRow
	matCostsByMonth   map[int]float64
}

func (m *mockAnalyticsRepo) CreateCompensationPlan(_ context.Context, p repositories.CreateCompensationParams) (*repositories.CompensationPlanRow, error) {
	if m.createdPlan != nil {
		return m.createdPlan, nil
	}
	return &repositories.CompensationPlanRow{
		ID:                "new-plan-id",
		CompanyID:         p.CompanyID,
		EmployeeID:        p.EmployeeID,
		EmployeeName:      "Test Radnik",
		PayType:           p.PayType,
		PayAmount:         p.PayAmount,
		CompanyCostAmount: p.CompanyCostAmount,
		EffectiveFrom:     p.EffectiveFrom,
		EffectiveTo:       p.EffectiveTo,
		CreatedBy:         p.CreatedBy,
		CreatedAt:         time.Now(),
	}, nil
}

func (m *mockAnalyticsRepo) UpdatePlanEffectiveTo(_ context.Context, _, planID string, to time.Time) error {
	m.closedPlanID = planID
	m.closedPlanDate = to
	return nil
}

func (m *mockAnalyticsRepo) ListCompensationPlans(_ context.Context, _, _ string) ([]repositories.CompensationPlanRow, error) {
	return m.plans, nil
}

func (m *mockAnalyticsRepo) GetPlansInDateRange(_ context.Context, _, _ string, _, _ time.Time) ([]repositories.CompensationPlanRow, error) {
	return m.plans, nil
}

func (m *mockAnalyticsRepo) HasOverlappingPlan(_ context.Context, _, _ string, _ time.Time, _ *time.Time, _ *string) (bool, error) {
	return m.hasOverlap, nil
}

func (m *mockAnalyticsRepo) FindOpenPlan(_ context.Context, _, _ string) (*repositories.CompensationPlanRow, error) {
	if m.openPlan != nil {
		return m.openPlan, nil
	}
	return nil, pgx.ErrNoRows
}

func (m *mockAnalyticsRepo) GetDailyHoursForEmployee(_ context.Context, _, _ string, _, _ time.Time) ([]repositories.DailyHoursRecord, error) {
	return m.dailyHours, nil
}

func (m *mockAnalyticsRepo) GetHoursByProject(_ context.Context, _, _ string, _, _ time.Time) ([]repositories.ProjectHours, error) {
	return m.projectHours, nil
}

func (m *mockAnalyticsRepo) ListEligibleEmployees(_ context.Context, _ string, _, _ time.Time) ([]repositories.AnalyticsEmployeeRow, error) {
	return m.employees, nil
}

func (m *mockAnalyticsRepo) GetEmployee(_ context.Context, _, _ string) (*repositories.AnalyticsEmployeeRow, error) {
	if len(m.employees) == 0 {
		return nil, pgx.ErrNoRows
	}
	return &m.employees[0], nil
}

func (m *mockAnalyticsRepo) GetCompensationPlanByID(_ context.Context, _, planID string) (*repositories.CompensationPlanRow, error) {
	if m.getPlanByIDError != nil {
		return nil, m.getPlanByIDError
	}
	for _, p := range m.plans {
		if p.ID == planID {
			return &p, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (m *mockAnalyticsRepo) UpdateCompensationPlanFields(_ context.Context, p repositories.UpdateCompensationPlanParams) (*repositories.CompensationPlanRow, error) {
	if m.updatedPlan != nil {
		return m.updatedPlan, nil
	}
	return &repositories.CompensationPlanRow{
		ID:                p.PlanID,
		CompanyID:         p.CompanyID,
		EmployeeID:        "e1",
		EmployeeName:      "Test Radnik",
		PayType:           p.PayType,
		PayAmount:         p.PayAmount,
		CompanyCostAmount: p.CompanyCostAmount,
		EffectiveFrom:     p.EffectiveFrom,
		EffectiveTo:       p.EffectiveTo,
		CreatedAt:         time.Now(),
	}, nil
}

func (m *mockAnalyticsRepo) GetContribution(_ context.Context, _, _ string, _ time.Time) (*repositories.ContributionRow, error) {
	return m.contribution, nil
}

func (m *mockAnalyticsRepo) GetContributionsForMonth(_ context.Context, _ string, _ time.Time) ([]repositories.ContributionRow, error) {
	return m.contribsForMonth, nil
}

func (m *mockAnalyticsRepo) GetContributionsForEmployeeYear(_ context.Context, _, _ string, _ int) ([]repositories.ContributionRow, error) {
	return m.contribsForYear, nil
}

func (m *mockAnalyticsRepo) GetContributionsForCompanyYear(_ context.Context, _ string, _ int) ([]repositories.ContributionRow, error) {
	return m.contribsForYear, nil
}

func (m *mockAnalyticsRepo) GetCompanyPlansForYear(_ context.Context, _ string, _, _ time.Time) ([]repositories.CompensationPlanRow, error) {
	return m.plans, nil
}

func (m *mockAnalyticsRepo) GetCompanyDailyHoursForYear(_ context.Context, _ string, _, _ time.Time) ([]repositories.CompanyDailyHoursRecord, error) {
	var out []repositories.CompanyDailyHoursRecord
	for _, h := range m.dailyHours {
		out = append(out, repositories.CompanyDailyHoursRecord{
			EmployeeID:  "e1",
			WorkDate:    h.WorkDate,
			ProjectID:   h.ProjectID,
			ProjectName: h.ProjectName,
			Hours:       h.Hours,
		})
	}
	return out, nil
}

func (m *mockAnalyticsRepo) GetProjectMaterialCostsForMonth(_ context.Context, _ string, _, _ int) (map[string]float64, error) {
	if m.matCostsByProject != nil {
		return m.matCostsByProject, nil
	}
	return map[string]float64{}, nil
}

func (m *mockAnalyticsRepo) GetProjectMaterialBreakdown(_ context.Context, _, _ string, _, _ int) ([]repositories.ProjectMaterialCostRow, error) {
	return m.matBreakdown, nil
}

func (m *mockAnalyticsRepo) GetProjectMaterialCostsForYear(_ context.Context, _, _ string, _ int) (map[int]float64, error) {
	if m.matCostsByMonth != nil {
		return m.matCostsByMonth, nil
	}
	return map[int]float64{}, nil
}

func (m *mockAnalyticsRepo) UpsertContribution(_ context.Context, p repositories.UpsertContributionParams) (*repositories.ContributionRow, error) {
	if m.upsertedContrib != nil {
		return m.upsertedContrib, nil
	}
	return &repositories.ContributionRow{
		ID:                  "contrib-id",
		CompanyID:           p.CompanyID,
		EmployeeID:          p.EmployeeID,
		PayrollMonth:        p.PayrollMonth,
		ContributionsAmount: p.ContributionsAmount,
		UpdatedAt:           time.Now(),
	}, nil
}

// ── Helpers (an-prefix to avoid conflicts with other test files in this package) ──

func anSvc(repo *mockAnalyticsRepo) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func anF64(v float64) *float64 { return &v }

func anD(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func anEmp(id, name, role string) repositories.AnalyticsEmployeeRow {
	return repositories.AnalyticsEmployeeRow{ID: id, Name: name, Role: role}
}

func anFixed(amount float64, from, to string) repositories.CompensationPlanRow {
	p := repositories.CompensationPlanRow{
		ID:            "plan-fixed",
		PayType:       dto.PayTypeFixed,
		PayAmount:     amount,
		EffectiveFrom: anD(from),
	}
	if to != "" {
		t := anD(to)
		p.EffectiveTo = &t
	}
	return p
}

func anHourly(rate float64, from, to string) repositories.CompensationPlanRow {
	p := repositories.CompensationPlanRow{
		ID:            "plan-hourly",
		PayType:       dto.PayTypeHourly,
		PayAmount:     rate,
		EffectiveFrom: anD(from),
	}
	if to != "" {
		t := anD(to)
		p.EffectiveTo = &t
	}
	return p
}

func anDH(dateStr, projectID, projectName string, hours float64) repositories.DailyHoursRecord {
	return repositories.DailyHoursRecord{WorkDate: anD(dateStr), ProjectID: projectID, ProjectName: projectName, Hours: hours}
}

// ── Authorization tests ───────────────────────────────────────────────────────

func TestAnalytics_DirektorAllowed(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan H.", "radnik")}})
	_, err := svc.GetEmployeeLaborCost(context.Background(), "co1", "direktor", "e1", 2026, 8)
	if err != nil {
		t.Errorf("direktor should be allowed, got: %v", err)
	}
}

func TestAnalytics_InzenjerAllowed(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana P.", "radnik")}})
	_, err := svc.GetEmployeeLaborCost(context.Background(), "co1", "inzenjer", "e1", 2026, 8)
	if err != nil {
		t.Errorf("inzenjer should be allowed, got: %v", err)
	}
}

func TestAnalytics_AdministracijaForbidden(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{})
	_, err := svc.GetEmployeeLaborCost(context.Background(), "co1", "administracija", "e1", 2026, 8)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestAnalytics_PoslovodaForbidden(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{})
	_, err := svc.GetMonthlyLaborSummary(context.Background(), "co1", "poslovoda", 2026, 8)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestAnalytics_RadnikForbidden(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{})
	_, err := svc.GetMonthlyLaborSummary(context.Background(), "co1", "radnik", 2026, 8)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestAnalytics_CrossCompanyEmployeeRejected(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{}})
	_, err := svc.GetEmployeeLaborCost(context.Background(), "other-company", "direktor", "e-from-other-company", 2026, 8)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-company should return ErrNotFound, got: %v", err)
	}
}

// ── Compensation management tests ─────────────────────────────────────────────

func TestCreateCompensation_FixedPlan(t *testing.T) {
	repo := &mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")}}
	svc := anSvc(repo)
	plan, err := svc.CreateCompensationPlan(context.Background(), "co1", "direktor", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1800.0, EffectiveFrom: "2026-08-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.PayType != dto.PayTypeFixed {
		t.Errorf("expected fixed_monthly, got %s", plan.PayType)
	}
	if plan.PayAmount != 1800.0 {
		t.Errorf("expected 1800.0, got %f", plan.PayAmount)
	}
}

func TestCreateCompensation_HourlyPlan(t *testing.T) {
	repo := &mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")}}
	svc := anSvc(repo)
	plan, err := svc.CreateCompensationPlan(context.Background(), "co1", "direktor", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: dto.PayTypeHourly, PayAmount: 9.5, EffectiveFrom: "2026-08-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.PayType != dto.PayTypeHourly {
		t.Errorf("expected hourly, got %s", plan.PayType)
	}
}

func TestCreateCompensation_InvalidAmount(t *testing.T) {
	repo := &mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")}}
	svc := anSvc(repo)
	_, err := svc.CreateCompensationPlan(context.Background(), "co1", "direktor", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 0, EffectiveFrom: "2026-08-01",
	})
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestCreateCompensation_NegativeAmount(t *testing.T) {
	repo := &mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")}}
	svc := anSvc(repo)
	_, err := svc.CreateCompensationPlan(context.Background(), "co1", "direktor", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: -100, EffectiveFrom: "2026-08-01",
	})
	if err == nil {
		t.Error("expected error for negative amount")
	}
}

func TestCreateCompensation_InvalidPayType(t *testing.T) {
	repo := &mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")}}
	svc := anSvc(repo)
	_, err := svc.CreateCompensationPlan(context.Background(), "co1", "direktor", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: "unknown", PayAmount: 1000, EffectiveFrom: "2026-08-01",
	})
	if err == nil {
		t.Error("expected error for invalid pay_type")
	}
}

func TestCreateCompensation_OverlappingPeriodRejected(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees:  []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")},
		hasOverlap: true,
	}
	svc := anSvc(repo)
	_, err := svc.CreateCompensationPlan(context.Background(), "co1", "direktor", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1800, EffectiveFrom: "2026-08-01",
	})
	if err == nil {
		t.Error("expected error for overlapping period")
	}
}

func TestCreateCompensation_HistoryPreserved_OpenPlanClosed(t *testing.T) {
	openPlan := anFixed(1500, "2026-01-01", "")
	openPlan.ID = "old-plan"
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")},
		openPlan:  &openPlan,
	}
	svc := anSvc(repo)
	_, err := svc.CreateCompensationPlan(context.Background(), "co1", "direktor", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1800, EffectiveFrom: "2026-08-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.closedPlanID != "old-plan" {
		t.Errorf("expected old-plan to be closed, got closedPlanID=%q", repo.closedPlanID)
	}
	expectedClose := anD("2026-07-31")
	if !repo.closedPlanDate.Equal(expectedClose) {
		t.Errorf("expected close date 2026-07-31, got %s", repo.closedPlanDate.Format("2006-01-02"))
	}
}

func TestCreateCompensation_ForbiddenForAdministracija(t *testing.T) {
	repo := &mockAnalyticsRepo{employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")}}
	svc := anSvc(repo)
	_, err := svc.CreateCompensationPlan(context.Background(), "co1", "administracija", "u1", "e1", dto.CreateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1800, EffectiveFrom: "2026-08-01",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

// ── Pure computation tests ────────────────────────────────────────────────────

func TestCompute_FixedMonthly_Normal(t *testing.T) {
	emp := anEmp("e1", "Ivan H.", "radnik")
	plans := []repositories.CompensationPlanRow{anFixed(1800, "2026-08-01", "")}
	projectHours := []repositories.ProjectHours{
		{ProjectID: "p1", ProjectName: "Projekt A", Hours: 100},
		{ProjectID: "p2", ProjectName: "Projekt B", Hours: 76},
	}
	dailyHours := make([]repositories.DailyHoursRecord, 0, 176)
	for i := 0; i < 100; i++ {
		dailyHours = append(dailyHours, anDH("2026-08-01", "p1", "Projekt A", 1))
	}
	for i := 0; i < 76; i++ {
		dailyHours = append(dailyHours, anDH("2026-08-02", "p2", "Projekt B", 1))
	}

	result := computeEmployeeLaborCost(emp, plans, dailyHours, projectHours)

	if result.TotalHours != 176 {
		t.Errorf("expected 176 hours, got %.1f", result.TotalHours)
	}
	if result.CompanyAnalyticalCost != 1800 {
		t.Errorf("expected analytical cost 1800, got %.2f", result.CompanyAnalyticalCost)
	}
	if result.EffectiveCostPerHour == nil {
		t.Fatal("expected effective cost per hour, got nil")
	}
	expected := 1800.0 / 176.0
	if *result.EffectiveCostPerHour != expected {
		t.Errorf("expected %.4f €/h, got %.4f", expected, *result.EffectiveCostPerHour)
	}
	if len(result.ProjectAllocations) != 2 {
		t.Errorf("expected 2 project allocations, got %d", len(result.ProjectAllocations))
	}
	if result.Warning != nil {
		t.Errorf("unexpected warning: %s", *result.Warning)
	}
}

func TestCompute_FixedMonthly_CompanyCostOverride(t *testing.T) {
	emp := anEmp("e1", "Ivan H.", "radnik")
	plan := anFixed(1800, "2026-08-01", "")
	plan.CompanyCostAmount = anF64(2200.0)
	plans := []repositories.CompensationPlanRow{plan}
	projectHours := []repositories.ProjectHours{{ProjectID: "p1", ProjectName: "A", Hours: 176}}
	dailyHours := []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "A", 176)}

	result := computeEmployeeLaborCost(emp, plans, dailyHours, projectHours)

	if result.CompanyAnalyticalCost != 2200 {
		t.Errorf("expected company cost 2200, got %.2f", result.CompanyAnalyticalCost)
	}
}

func TestCompute_FixedMonthly_ZeroHours_ShowsCostWithWarning(t *testing.T) {
	emp := anEmp("e1", "Ivan H.", "radnik")
	plans := []repositories.CompensationPlanRow{anFixed(1800, "2026-08-01", "")}

	result := computeEmployeeLaborCost(emp, plans, nil, nil)

	if result.CompanyAnalyticalCost != 1800 {
		t.Errorf("expected monthly cost 1800 even with zero hours, got %.2f", result.CompanyAnalyticalCost)
	}
	if result.EffectiveCostPerHour != nil {
		t.Error("expected no effective cost per hour for zero hours")
	}
	if result.Warning == nil {
		t.Error("expected warning for zero hours")
	}
}

func TestCompute_FixedMonthly_MidMonthTransition_Warning(t *testing.T) {
	emp := anEmp("e1", "Ivan H.", "radnik")
	plan1 := anFixed(1500, "2026-08-01", "2026-08-15")
	plan2 := anFixed(1800, "2026-08-16", "")
	plans := []repositories.CompensationPlanRow{plan1, plan2}
	dailyHours := []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "A", 8)}

	result := computeEmployeeLaborCost(emp, plans, dailyHours, nil)

	if !result.HasMidMonthTransition {
		t.Error("expected HasMidMonthTransition = true")
	}
	if result.Warning == nil {
		t.Error("expected warning for mid-month transition")
	}
}

func TestCompute_Hourly_SingleRate(t *testing.T) {
	emp := anEmp("e1", "Marko P.", "radnik")
	plans := []repositories.CompensationPlanRow{anHourly(9.5, "2026-08-01", "")}
	dailyHours := []repositories.DailyHoursRecord{
		anDH("2026-08-01", "p1", "A", 8),
		anDH("2026-08-02", "p1", "A", 8),
	}

	result := computeEmployeeLaborCost(emp, plans, dailyHours, nil)

	if result.TotalHours != 16 {
		t.Errorf("expected 16 hours, got %.1f", result.TotalHours)
	}
	expectedCost := 16 * 9.5
	if result.CompanyAnalyticalCost != expectedCost {
		t.Errorf("expected cost %.2f, got %.2f", expectedCost, result.CompanyAnalyticalCost)
	}
}

func TestCompute_Hourly_CompanyCostOverride(t *testing.T) {
	emp := anEmp("e1", "Marko P.", "radnik")
	plan := anHourly(9.5, "2026-08-01", "")
	plan.CompanyCostAmount = anF64(12.0)
	plans := []repositories.CompensationPlanRow{plan}
	dailyHours := []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "A", 8)}

	result := computeEmployeeLaborCost(emp, plans, dailyHours, nil)

	expectedCost := 8 * 12.0
	if result.CompanyAnalyticalCost != expectedCost {
		t.Errorf("expected company cost %.2f (using company_cost_amount), got %.2f", expectedCost, result.CompanyAnalyticalCost)
	}
}

func TestCompute_Hourly_MidMonthRateChange(t *testing.T) {
	emp := anEmp("e1", "Marko P.", "radnik")
	plan1 := anHourly(8.0, "2026-08-01", "2026-08-15")
	plan2 := anHourly(10.0, "2026-08-16", "")
	plans := []repositories.CompensationPlanRow{plan1, plan2}
	dailyHours := []repositories.DailyHoursRecord{
		anDH("2026-08-10", "p1", "A", 10),
		anDH("2026-08-20", "p1", "A", 5),
	}

	result := computeEmployeeLaborCost(emp, plans, dailyHours, nil)

	if result.TotalHours != 15 {
		t.Errorf("expected 15 hours, got %.1f", result.TotalHours)
	}
	expectedCost := 10*8.0 + 5*10.0
	if result.CompanyAnalyticalCost != expectedCost {
		t.Errorf("expected cost %.2f (correct per-day rate), got %.2f", expectedCost, result.CompanyAnalyticalCost)
	}
}

func TestCompute_Hourly_MultipleProjects(t *testing.T) {
	emp := anEmp("e1", "Marko P.", "radnik")
	plans := []repositories.CompensationPlanRow{anHourly(10.0, "2026-08-01", "")}
	dailyHours := []repositories.DailyHoursRecord{
		anDH("2026-08-01", "p1", "A", 6),
		anDH("2026-08-01", "p2", "B", 2),
	}

	result := computeEmployeeLaborCost(emp, plans, dailyHours, nil)

	if len(result.ProjectAllocations) != 2 {
		t.Errorf("expected 2 project allocations, got %d", len(result.ProjectAllocations))
	}
	totalCost := 0.0
	for _, a := range result.ProjectAllocations {
		totalCost += a.LaborCost
	}
	if totalCost != 80.0 {
		t.Errorf("expected total allocation cost 80.0, got %.2f", totalCost)
	}
}

func TestCompute_NoCompensation_WithHours_Flagged(t *testing.T) {
	emp := anEmp("e1", "Ana S.", "radnik")
	dailyHours := []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "A", 8)}

	result := computeEmployeeLaborCost(emp, nil, dailyHours, nil)

	if result.HasCompensation {
		t.Error("expected HasCompensation = false")
	}
	if result.Warning == nil {
		t.Error("expected warning for missing compensation with hours")
	}
	if result.CompanyAnalyticalCost != 0 {
		t.Errorf("expected 0 cost, got %.2f", result.CompanyAnalyticalCost)
	}
}

func TestCompute_NoCompensation_NoHours_NoWarning(t *testing.T) {
	emp := anEmp("e1", "Ana S.", "radnik")
	result := computeEmployeeLaborCost(emp, nil, nil, nil)

	if result.Warning != nil {
		t.Errorf("no warning expected when employee has no hours and no plan, got: %s", *result.Warning)
	}
}

func TestCompute_FindPlanForDate_CorrectLookup(t *testing.T) {
	eto := anD("2026-07-31")
	plans := []repositories.CompensationPlanRow{
		{PayType: dto.PayTypeHourly, PayAmount: 8.0, EffectiveFrom: anD("2026-01-01"), EffectiveTo: &eto},
		{PayType: dto.PayTypeHourly, PayAmount: 10.0, EffectiveFrom: anD("2026-08-01")},
	}

	if p := findPlanForDate(plans, anD("2026-06-15")); p == nil || p.PayAmount != 8.0 {
		t.Errorf("expected plan with 8.0 rate for June, got %v", p)
	}
	if p := findPlanForDate(plans, anD("2026-08-19")); p == nil || p.PayAmount != 10.0 {
		t.Errorf("expected plan with 10.0 rate for August, got %v", p)
	}
	if p := findPlanForDate(plans, anD("2025-12-31")); p != nil {
		t.Errorf("expected nil for date before any plan, got %v", p)
	}
}

// ── Edit compensation plan tests ──────────────────────────────────────────────

func TestUpdateCompensation_UnauthorizedRoleRejected(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{})
	_, err := svc.UpdateCompensationPlan(context.Background(), "co1", "administracija", "plan-1", dto.UpdateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1800, EffectiveFrom: "2026-08-01",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestUpdateCompensation_CrossCompanyRejected(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{getPlanByIDError: pgx.ErrNoRows})
	_, err := svc.UpdateCompensationPlan(context.Background(), "other-company", "direktor", "plan-1", dto.UpdateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1800, EffectiveFrom: "2026-08-01",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-company plan lookup, got: %v", err)
	}
}

func TestUpdateCompensation_InvalidPayAmount(t *testing.T) {
	plan := anFixed(1500, "2026-08-01", "")
	plan.ID = "plan-1"
	repo := &mockAnalyticsRepo{
		plans:     []repositories.CompensationPlanRow{plan},
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")},
	}
	_, err := anSvc(repo).UpdateCompensationPlan(context.Background(), "co1", "direktor", "plan-1", dto.UpdateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 0, EffectiveFrom: "2026-08-01",
	})
	if err == nil {
		t.Error("expected validation error for zero pay_amount")
	}
}

func TestUpdateCompensation_OverlapAfterEditRejected(t *testing.T) {
	plan := anFixed(1500, "2026-08-01", "2026-08-31")
	plan.ID = "plan-1"
	repo := &mockAnalyticsRepo{
		plans:      []repositories.CompensationPlanRow{plan},
		employees:  []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")},
		hasOverlap: true,
	}
	_, err := anSvc(repo).UpdateCompensationPlan(context.Background(), "co1", "direktor", "plan-1", dto.UpdateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1800, EffectiveFrom: "2026-07-01",
	})
	if err == nil {
		t.Error("expected error for overlapping period after edit")
	}
}

func TestUpdateCompensation_SelfNoConflict(t *testing.T) {
	plan := anFixed(1500, "2026-08-01", "2026-08-31")
	plan.ID = "plan-self"
	repo := &mockAnalyticsRepo{
		plans:      []repositories.CompensationPlanRow{plan},
		employees:  []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")},
		hasOverlap: false,
	}
	updated, err := anSvc(repo).UpdateCompensationPlan(context.Background(), "co1", "direktor", "plan-self", dto.UpdateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 2000, EffectiveFrom: "2026-08-01",
	})
	if err != nil {
		t.Fatalf("self-edit should not conflict, got: %v", err)
	}
	if updated.PayAmount != 2000 {
		t.Errorf("expected updated pay_amount 2000, got %f", updated.PayAmount)
	}
}

func TestUpdateCompensation_UpdateCurrentPlan(t *testing.T) {
	plan := anFixed(1500, "2026-08-01", "")
	plan.ID = "plan-current"
	repo := &mockAnalyticsRepo{
		plans:     []repositories.CompensationPlanRow{plan},
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")},
	}
	updated, err := anSvc(repo).UpdateCompensationPlan(context.Background(), "co1", "direktor", "plan-current", dto.UpdateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 2000, EffectiveFrom: "2026-08-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.PayAmount != 2000 {
		t.Errorf("expected 2000, got %f", updated.PayAmount)
	}
	if updated.PayType != dto.PayTypeFixed {
		t.Errorf("expected fixed_monthly, got %s", updated.PayType)
	}
	if updated.EffectiveTo != nil {
		t.Errorf("expected open-ended plan, got effective_to=%s", *updated.EffectiveTo)
	}
}

func TestUpdateCompensation_UpdateHistoricalPlan(t *testing.T) {
	past := anFixed(1200, "2025-01-01", "2025-12-31")
	past.ID = "plan-hist"
	repo := &mockAnalyticsRepo{
		plans:     []repositories.CompensationPlanRow{past},
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Test", "radnik")},
	}
	eto := "2025-12-31"
	updated, err := anSvc(repo).UpdateCompensationPlan(context.Background(), "co1", "direktor", "plan-hist", dto.UpdateCompensationPlanReq{
		PayType: dto.PayTypeFixed, PayAmount: 1300, EffectiveFrom: "2025-01-01", EffectiveTo: &eto,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.PayAmount != 1300 {
		t.Errorf("expected 1300, got %f", updated.PayAmount)
	}
	if updated.EffectiveTo == nil || *updated.EffectiveTo != "2025-12-31" {
		t.Errorf("expected effective_to 2025-12-31, got %v", updated.EffectiveTo)
	}
}

func TestUpdateCompensation_ChangedAmountAffectsComputation(t *testing.T) {
	emp := anEmp("e1", "Ivan H.", "radnik")
	dailyHours := []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "A", 176)}

	before := computeEmployeeLaborCost(emp, []repositories.CompensationPlanRow{anFixed(1800, "2026-08-01", "")}, dailyHours, nil)
	after := computeEmployeeLaborCost(emp, []repositories.CompensationPlanRow{anFixed(2200, "2026-08-01", "")}, dailyHours, nil)

	if before.CompanyAnalyticalCost == after.CompanyAnalyticalCost {
		t.Error("expected different analytical cost after amount change")
	}
	if after.CompanyAnalyticalCost != 2200 {
		t.Errorf("expected 2200 after update, got %.2f", after.CompanyAnalyticalCost)
	}
}

func TestUpdateCompensation_ChangedCompanyCostAffectsComputation(t *testing.T) {
	emp := anEmp("e1", "Ivan H.", "radnik")
	dailyHours := []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "A", 176)}

	planWithCost := anFixed(1800, "2026-08-01", "")
	planWithCost.CompanyCostAmount = anF64(2400.0)

	result := computeEmployeeLaborCost(emp, []repositories.CompensationPlanRow{planWithCost}, dailyHours, nil)

	if result.CompanyAnalyticalCost != 2400 {
		t.Errorf("expected company_cost_amount 2400 used as analytical cost, got %.2f", result.CompanyAnalyticalCost)
	}
}

func TestUpdateCompensation_ChangedHourlyRateRecalculates(t *testing.T) {
	emp := anEmp("e1", "Marko P.", "radnik")
	dailyHours := []repositories.DailyHoursRecord{anDH("2026-08-10", "p1", "A", 8)}

	before := computeEmployeeLaborCost(emp, []repositories.CompensationPlanRow{anHourly(9.5, "2026-08-01", "")}, dailyHours, nil)
	after := computeEmployeeLaborCost(emp, []repositories.CompensationPlanRow{anHourly(12.0, "2026-08-01", "")}, dailyHours, nil)

	if before.CompanyAnalyticalCost == after.CompanyAnalyticalCost {
		t.Error("expected different cost after hourly rate change")
	}
	if after.CompanyAnalyticalCost != 8*12.0 {
		t.Errorf("expected %.2f, got %.2f", 8*12.0, after.CompanyAnalyticalCost)
	}
}

func TestGetMonthlyLaborSummary_EmployeesWithoutCompensationCounted(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{
			anEmp("e1", "Ivan H.", "radnik"),
			anEmp("e2", "Ana S.", "radnik"),
		},
		plans: nil,
		dailyHours: []repositories.DailyHoursRecord{
			anDH("2026-08-01", "p1", "A", 8),
		},
	}
	svc := anSvc(repo)
	summary, err := svc.GetMonthlyLaborSummary(context.Background(), "co1", "direktor", 2026, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.EmployeesWithoutCompensation != 2 {
		t.Errorf("expected 2 employees without compensation, got %d", summary.EmployeesWithoutCompensation)
	}
}

// ── Contribution tests ────────────────────────────────────────────────────────

func TestContributions_MissingShowsNil(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees:  []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
		plans:      []repositories.CompensationPlanRow{anFixed(1800, "2026-01-01", "")},
		dailyHours: []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "Proj", 8)},
		// contribution is nil — not entered
	}
	svc := anSvc(repo)
	cost, err := svc.GetEmployeeLaborCost(context.Background(), "co1", "direktor", "e1", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !cost.ContributionsMissing {
		t.Error("expected ContributionsMissing=true when no contribution record")
	}
	if cost.Contributions != nil {
		t.Error("expected Contributions=nil")
	}
	if cost.Bruto != nil {
		t.Error("expected Bruto=nil when contributions missing")
	}
	if cost.Neto != 1800 {
		t.Errorf("expected Neto=1800, got %f", cost.Neto)
	}
}

func TestContributions_ZeroIsNotMissing(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees:    []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
		plans:        []repositories.CompensationPlanRow{anFixed(1800, "2026-01-01", "")},
		dailyHours:   []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "Proj", 8)},
		contribution: &repositories.ContributionRow{ContributionsAmount: 0},
	}
	svc := anSvc(repo)
	cost, err := svc.GetEmployeeLaborCost(context.Background(), "co1", "direktor", "e1", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if cost.ContributionsMissing {
		t.Error("expected ContributionsMissing=false for explicit zero")
	}
	if cost.Contributions == nil || *cost.Contributions != 0 {
		t.Errorf("expected Contributions=0, got %v", cost.Contributions)
	}
	if cost.Bruto == nil || *cost.Bruto != 1800 {
		t.Errorf("expected Bruto=1800 (neto+0), got %v", cost.Bruto)
	}
}

func TestContributions_BrutoFormula(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees:    []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
		plans:        []repositories.CompensationPlanRow{anFixed(1800, "2026-01-01", "")},
		dailyHours:   []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "Proj", 8)},
		contribution: &repositories.ContributionRow{ContributionsAmount: 500},
	}
	svc := anSvc(repo)
	cost, err := svc.GetEmployeeLaborCost(context.Background(), "co1", "direktor", "e1", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if cost.Neto != 1800 {
		t.Errorf("expected Neto=1800, got %f", cost.Neto)
	}
	if cost.Contributions == nil || *cost.Contributions != 500 {
		t.Errorf("expected Contributions=500, got %v", cost.Contributions)
	}
	if cost.Bruto == nil || *cost.Bruto != 2300 {
		t.Errorf("expected Bruto=2300 (1800+500), got %v", cost.Bruto)
	}
}

func TestContributions_MidMonthTransitionNoBruto(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
		plans: []repositories.CompensationPlanRow{
			anFixed(1800, "2026-08-01", "2026-08-14"),
			anFixed(2000, "2026-08-15", ""),
		},
		dailyHours:   []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "Proj", 8)},
		contribution: &repositories.ContributionRow{ContributionsAmount: 500},
	}
	svc := anSvc(repo)
	cost, err := svc.GetEmployeeLaborCost(context.Background(), "co1", "direktor", "e1", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if cost.Bruto != nil {
		t.Error("expected Bruto=nil for mid-month fixed transition")
	}
	// Contributions still set even without bruto
	if cost.Contributions == nil {
		t.Error("expected Contributions set even when bruto unavailable")
	}
}

func TestContributions_UpsertForbiddenRole(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
	}
	svc := anSvc(repo)
	_, err := svc.UpsertContribution(context.Background(), "co1", "radnik", "u1", "e1", 2026, 8, dto.UpsertContributionReq{ContributionsAmount: 500})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestContributions_UpsertNegativeAmount(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
	}
	svc := anSvc(repo)
	_, err := svc.UpsertContribution(context.Background(), "co1", "direktor", "u1", "e1", 2026, 8, dto.UpsertContributionReq{ContributionsAmount: -1})
	if err == nil {
		t.Error("expected validation error for negative amount")
	}
}

func TestContributions_UpsertSuccess(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
	}
	svc := anSvc(repo)
	result, err := svc.UpsertContribution(context.Background(), "co1", "direktor", "u1", "e1", 2026, 8, dto.UpsertContributionReq{ContributionsAmount: 500})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContributionsAmount != 500 {
		t.Errorf("expected ContributionsAmount=500, got %f", result.ContributionsAmount)
	}
}

func TestContributions_SummaryTotals(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees:  []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik"), anEmp("e2", "Ivan", "radnik")},
		plans:      []repositories.CompensationPlanRow{anFixed(1800, "2026-01-01", "")},
		dailyHours: []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "Proj", 8)},
		contribsForMonth: []repositories.ContributionRow{
			{EmployeeID: "e1", ContributionsAmount: 500},
		},
		// e2 has no contribution record
	}
	svc := anSvc(repo)
	summary, err := svc.GetMonthlyLaborSummary(context.Background(), "co1", "direktor", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalNeto != 3600 {
		t.Errorf("expected TotalNeto=3600 (2×1800), got %f", summary.TotalNeto)
	}
	if summary.TotalContributions != 500 {
		t.Errorf("expected TotalContributions=500, got %f", summary.TotalContributions)
	}
	if summary.EmployeesMissingContributions != 1 {
		t.Errorf("expected 1 missing contribution, got %d", summary.EmployeesMissingContributions)
	}
	if summary.BrutoComplete {
		t.Error("expected BrutoComplete=false when any employee is missing contributions")
	}
}

func TestContributions_SummaryBrutoComplete(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees:  []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
		plans:      []repositories.CompensationPlanRow{anFixed(1800, "2026-01-01", "")},
		dailyHours: []repositories.DailyHoursRecord{anDH("2026-08-01", "p1", "Proj", 8)},
		contribsForMonth: []repositories.ContributionRow{
			{EmployeeID: "e1", ContributionsAmount: 400},
		},
	}
	svc := anSvc(repo)
	summary, err := svc.GetMonthlyLaborSummary(context.Background(), "co1", "direktor", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.BrutoComplete {
		t.Error("expected BrutoComplete=true when all employees have contributions")
	}
	if summary.TotalBruto != 2200 {
		t.Errorf("expected TotalBruto=2200 (1800+400), got %f", summary.TotalBruto)
	}
}

// ── Annual summary tests ──────────────────────────────────────────────────────

func TestAnnualPayroll_ForbiddenRole(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{})
	_, err := svc.GetAnnualPayrollSummary(context.Background(), "co1", "radnik", 2026, "")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestAnnualPayroll_NoData(t *testing.T) {
	svc := anSvc(&mockAnalyticsRepo{employees: nil})
	summary, err := svc.GetAnnualPayrollSummary(context.Background(), "co1", "direktor", 2026, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Months) != 12 {
		t.Errorf("expected 12 months, got %d", len(summary.Months))
	}
	if summary.TotalNeto != 0 || summary.TotalContributions != 0 {
		t.Error("expected all-zero totals with no employees")
	}
}

func TestAnnualPayroll_AllEmployees_AggregatesCorrectly(t *testing.T) {
	// Employee e1 has fixed 1800 plan all year with contributions 500 in every month.
	// Daily hours in January and August only (8h each).
	// Annual bruto = 1800 + 500 = 2300 per month × 12 = 27600 (fixed monthly neto counts even with no hours).
	jan1 := anD("2026-01-15")
	aug1 := anD("2026-08-10")
	contribs := make([]repositories.ContributionRow, 12)
	for mo := 1; mo <= 12; mo++ {
		contribs[mo-1] = repositories.ContributionRow{
			EmployeeID:          "e1",
			PayrollMonth:        time.Date(2026, time.Month(mo), 1, 0, 0, 0, 0, time.UTC),
			ContributionsAmount: 500,
		}
	}
	plan := anFixed(1800, "2026-01-01", "")
	plan.EmployeeID = "e1"
	repo := &mockAnalyticsRepo{
		employees:       []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan", "radnik")},
		plans:           []repositories.CompensationPlanRow{plan},
		dailyHours:      []repositories.DailyHoursRecord{{WorkDate: jan1, ProjectID: "p1", ProjectName: "P", Hours: 8}, {WorkDate: aug1, ProjectID: "p1", ProjectName: "P", Hours: 8}},
		contribsForYear: contribs,
	}
	svc := anSvc(repo)
	summary, err := svc.GetAnnualPayrollSummary(context.Background(), "co1", "direktor", 2026, "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalEmployees != 1 {
		t.Errorf("expected 1 employee, got %d", summary.TotalEmployees)
	}
	// Months with hours have neto=1800, contributions=500, bruto=2300.
	// Months without hours: fixed monthly pay still = 1800 neto, but Warning="Nema evidentiranih sati", so bruto is set.
	// Actually with fixed monthly and no hours: computeEmployeeLaborCost returns cost with zero hours
	// and warning "Nema evidentiranih sati", HasCompensation=true.
	// enrichWithContribution: brutoReady = HasCompensation && !HasMidMonthTransition &&
	//   !(Warning != nil && CompanyAnalyticalCost == 0)
	// With no hours: CompanyAnalyticalCost = 1800, warning set => brutoReady = true (warning exists but cost != 0).
	// Wait, re-reading the logic: !(cost.Warning != nil && cost.CompanyAnalyticalCost == 0)
	// So warning with non-zero cost: !(true && false) = !(false) = true => brutoReady = true.
	// So bruto IS computed for months with no hours but with compensation.
	// Annual bruto = 1800+500 = 2300 * 12 months = 27600.
	if summary.TotalBruto != 27600 {
		t.Errorf("expected TotalBruto=27600, got %f", summary.TotalBruto)
	}
	if summary.MonthsWithIncompleteData != 0 {
		t.Errorf("expected 0 incomplete months, got %d", summary.MonthsWithIncompleteData)
	}
	if !summary.BrutoComplete {
		t.Error("expected BrutoComplete=true")
	}
}

func TestAnnualPayroll_MonthsWithIncompleteData(t *testing.T) {
	// Employee has a plan all year but no contributions entered for any month.
	plan2 := anFixed(2000, "2026-01-01", "")
	plan2.EmployeeID = "e1"
	repo := &mockAnalyticsRepo{
		employees:       []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ana", "radnik")},
		plans:           []repositories.CompensationPlanRow{plan2},
		dailyHours:      []repositories.DailyHoursRecord{anDH("2026-03-01", "p1", "P", 8)},
		contribsForYear: nil, // no contributions for any month
	}
	svc := anSvc(repo)
	summary, err := svc.GetAnnualPayrollSummary(context.Background(), "co1", "direktor", 2026, "")
	if err != nil {
		t.Fatal(err)
	}
	// All 12 months have compensation → all 12 have incomplete contributions.
	if summary.MonthsWithIncompleteData != 12 {
		t.Errorf("expected 12 incomplete months, got %d", summary.MonthsWithIncompleteData)
	}
	if summary.BrutoComplete {
		t.Error("expected BrutoComplete=false when contributions are missing")
	}
	if summary.TotalBruto != 0 {
		t.Errorf("expected TotalBruto=0 when contributions missing, got %f", summary.TotalBruto)
	}
}

// ── Project allocation tests ──────────────────────────────────────────────────

func TestProjectAllocation_ForbiddenRole(t *testing.T) {
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan", "radnik")},
	}
	_, err := anSvc(repo).GetMonthlyProjectAllocation(context.Background(), "co1", "radnik", 2026, 8)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectAllocation_AggregatesAcrossEmployees(t *testing.T) {
	// Two employees, both with 8h on project p1 and fixed plan at 2000.
	// Expected: p1.TotalHours=16, p1.EmployeeCount=2, TotalLaborCost=4000.
	contribsForMonth := []repositories.ContributionRow{
		{EmployeeID: "e1", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
		{EmployeeID: "e2", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
	}
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{
			anEmp("e1", "Ivan", "radnik"),
			anEmp("e2", "Ana", "radnik"),
		},
		plans: []repositories.CompensationPlanRow{anFixed(2000, "2026-01-01", "")},
		dailyHours: []repositories.DailyHoursRecord{
			anDH("2026-08-05", "p1", "Projekt Alpha", 8),
		},
		projectHours: []repositories.ProjectHours{
			{ProjectID: "p1", ProjectName: "Projekt Alpha", Hours: 8},
		},
		contribsForMonth: contribsForMonth,
	}
	svc := anSvc(repo)
	result, err := svc.GetMonthlyProjectAllocation(context.Background(), "co1", "direktor", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Allocations) != 1 {
		t.Fatalf("expected 1 project allocation, got %d", len(result.Allocations))
	}
	p := result.Allocations[0]
	if p.ProjectID != "p1" {
		t.Errorf("expected project p1, got %s", p.ProjectID)
	}
	if p.TotalHours != 16 {
		t.Errorf("expected 16 total hours, got %f", p.TotalHours)
	}
	if p.EmployeeCount != 2 {
		t.Errorf("expected 2 employees on project, got %d", p.EmployeeCount)
	}
	// LaborCost: each employee earns 2000/8h * 8h = 2000; total = 4000
	if p.LaborCost != 4000 {
		t.Errorf("expected labor cost 4000, got %f", p.LaborCost)
	}
	if result.TotalHours != 16 {
		t.Errorf("expected total hours = 16, got %f", result.TotalHours)
	}
	if result.UnassignedHours != 0 {
		t.Errorf("expected no unassigned hours, got %f", result.UnassignedHours)
	}
}

func TestProjectAllocation_IncludesContributions(t *testing.T) {
	// Employee with fixed plan 2000, 8h on p1 (all hours), contributions 500.
	// Expected: p1.NetoCost=2000, p1.ContributionsCost=500 (100% of hours on p1), p1.BrutoCost=2500.
	plan := anFixed(2000, "2026-01-01", "")
	plan.EmployeeID = "e1"
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan", "radnik")},
		plans:     []repositories.CompensationPlanRow{plan},
		dailyHours: []repositories.DailyHoursRecord{
			anDH("2026-08-05", "p1", "Alpha", 8),
		},
		projectHours: []repositories.ProjectHours{
			{ProjectID: "p1", ProjectName: "Alpha", Hours: 8},
		},
		contribsForMonth: []repositories.ContributionRow{
			{EmployeeID: "e1", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
		},
	}
	result, err := anSvc(repo).GetMonthlyProjectAllocation(context.Background(), "co1", "direktor", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(result.Allocations))
	}
	p := result.Allocations[0]
	if p.NetoCost != 2000 {
		t.Errorf("expected NetoCost=2000, got %f", p.NetoCost)
	}
	if p.ContributionsCost != 500 {
		t.Errorf("expected ContributionsCost=500, got %f", p.ContributionsCost)
	}
	if p.BrutoCost != 2500 {
		t.Errorf("expected BrutoCost=2500, got %f", p.BrutoCost)
	}
	if p.ContributionsIncomplete {
		t.Error("expected ContributionsIncomplete=false")
	}
	if result.TotalNeto != 2000 {
		t.Errorf("expected TotalNeto=2000, got %f", result.TotalNeto)
	}
	if result.TotalContributions != 500 {
		t.Errorf("expected TotalContributions=500, got %f", result.TotalContributions)
	}
	if result.TotalBruto != 2500 {
		t.Errorf("expected TotalBruto=2500, got %f", result.TotalBruto)
	}
}

func TestProjectAllocation_MissingContributions_MarkedIncomplete(t *testing.T) {
	// Employee with fixed plan 2000, 8h on p1, NO contribution record.
	// Expected: p1.ContributionsIncomplete=true, p1.ContributionsCost=0.
	plan := anFixed(2000, "2026-01-01", "")
	plan.EmployeeID = "e1"
	repo := &mockAnalyticsRepo{
		employees:        []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan", "radnik")},
		plans:            []repositories.CompensationPlanRow{plan},
		dailyHours:       []repositories.DailyHoursRecord{anDH("2026-08-05", "p1", "Alpha", 8)},
		projectHours:     []repositories.ProjectHours{{ProjectID: "p1", ProjectName: "Alpha", Hours: 8}},
		contribsForMonth: nil, // no contributions entered
	}
	result, err := anSvc(repo).GetMonthlyProjectAllocation(context.Background(), "co1", "direktor", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(result.Allocations))
	}
	p := result.Allocations[0]
	if !p.ContributionsIncomplete {
		t.Error("expected ContributionsIncomplete=true when no contribution record")
	}
	if p.ContributionsCost != 0 {
		t.Errorf("expected ContributionsCost=0 when missing, got %f", p.ContributionsCost)
	}
	if p.NetoCost != 2000 {
		t.Errorf("expected NetoCost=2000, got %f", p.NetoCost)
	}
}

func TestProjectDetail_ForbiddenRole(t *testing.T) {
	_, err := anSvc(&mockAnalyticsRepo{}).GetProjectDetail(context.Background(), "co1", "radnik", "p1", 2026, 8)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectDetail_EmployeeBreakdown(t *testing.T) {
	// Single employee, 8h on p1, fixed plan 2000, contributions 500.
	// All hours on p1 → 100% allocation → neto=2000, contrib=500, bruto=2500.
	plan := anFixed(2000, "2026-01-01", "")
	plan.EmployeeID = "e1"
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan", "radnik")},
		plans:     []repositories.CompensationPlanRow{plan},
		dailyHours: []repositories.DailyHoursRecord{
			anDH("2026-08-05", "p1", "Alpha", 8),
		},
		projectHours: []repositories.ProjectHours{
			{ProjectID: "p1", ProjectName: "Alpha", Hours: 8},
		},
		contribsForMonth: []repositories.ContributionRow{
			{EmployeeID: "e1", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
		},
		contribsForYear: []repositories.ContributionRow{
			{EmployeeID: "e1", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
		},
	}
	result, err := anSvc(repo).GetProjectDetail(context.Background(), "co1", "direktor", "p1", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProjectID != "p1" {
		t.Errorf("expected project_id p1, got %s", result.ProjectID)
	}
	if result.TotalHours != 8 {
		t.Errorf("expected TotalHours=8, got %f", result.TotalHours)
	}
	if result.TotalNeto != 2000 {
		t.Errorf("expected TotalNeto=2000, got %f", result.TotalNeto)
	}
	if result.TotalContributions != 500 {
		t.Errorf("expected TotalContributions=500, got %f", result.TotalContributions)
	}
	if result.TotalBruto != 2500 {
		t.Errorf("expected TotalBruto=2500, got %f", result.TotalBruto)
	}
	if len(result.Employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(result.Employees))
	}
	emp := result.Employees[0]
	if emp.EmployeeID != "e1" {
		t.Errorf("expected employee e1, got %s", emp.EmployeeID)
	}
	if emp.Hours != 8 {
		t.Errorf("expected 8h, got %f", emp.Hours)
	}
	if emp.NetoCost != 2000 {
		t.Errorf("expected NetoCost=2000, got %f", emp.NetoCost)
	}
	if emp.ContributionsCost == nil || *emp.ContributionsCost != 500 {
		t.Errorf("expected ContributionsCost=500, got %v", emp.ContributionsCost)
	}
	if emp.BrutoCost == nil || *emp.BrutoCost != 2500 {
		t.Errorf("expected BrutoCost=2500, got %v", emp.BrutoCost)
	}
	if len(result.MonthlyTrend) != 12 {
		t.Errorf("expected 12 monthly trend entries, got %d", len(result.MonthlyTrend))
	}
	// August (month 8, index 7) should have data
	aug := result.MonthlyTrend[7]
	if !aug.HasData {
		t.Error("expected August to have data")
	}
	if aug.Neto != 2000 {
		t.Errorf("expected August Neto=2000, got %f", aug.Neto)
	}
}

// ── Material cost tests ───────────────────────────────────────────────────────

func TestProjectAllocation_IncludesMaterialCost(t *testing.T) {
	// Employee with 8h on p1, neto 2000, contrib 500 → bruto 2500.
	// Material cost for p1 = 1200 (from mock).
	// Expected: p1.MaterialCost=1200, p1.TotalDirectCost=3700, summary.TotalMaterialCost=1200.
	plan := anFixed(2000, "2026-01-01", "")
	plan.EmployeeID = "e1"
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan", "radnik")},
		plans:     []repositories.CompensationPlanRow{plan},
		dailyHours: []repositories.DailyHoursRecord{
			anDH("2026-08-05", "p1", "Alpha", 8),
		},
		projectHours: []repositories.ProjectHours{
			{ProjectID: "p1", ProjectName: "Alpha", Hours: 8},
		},
		contribsForMonth: []repositories.ContributionRow{
			{EmployeeID: "e1", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
		},
		matCostsByProject: map[string]float64{"p1": 1200},
	}
	result, err := anSvc(repo).GetMonthlyProjectAllocation(context.Background(), "co1", "direktor", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(result.Allocations))
	}
	p := result.Allocations[0]
	if p.MaterialCost != 1200 {
		t.Errorf("expected MaterialCost=1200, got %f", p.MaterialCost)
	}
	if p.TotalDirectCost != 3700 {
		t.Errorf("expected TotalDirectCost=3700 (2500+1200), got %f", p.TotalDirectCost)
	}
	if result.TotalMaterialCost != 1200 {
		t.Errorf("expected TotalMaterialCost=1200, got %f", result.TotalMaterialCost)
	}
	if result.TotalDirectCost != 3700 {
		t.Errorf("expected summary TotalDirectCost=3700, got %f", result.TotalDirectCost)
	}
}

func TestProjectDetail_IncludesMaterialBreakdown(t *testing.T) {
	// Employee 8h on p1, neto 2000, contrib 500, bruto 2500.
	// Material breakdown: 1 item with total_cost=800.
	// Material in August (month 8): cost=800 from matCostsByMonth.
	// Expected: TotalMaterialCost=800, TotalDirectCost=3300, Materials has 1 item.
	// Annual trend: August entry has MaterialCost=800, TotalCost=bruto+800.
	plan := anFixed(2000, "2026-01-01", "")
	plan.EmployeeID = "e1"
	matCode := "MAT-01"
	repo := &mockAnalyticsRepo{
		employees: []repositories.AnalyticsEmployeeRow{anEmp("e1", "Ivan", "radnik")},
		plans:     []repositories.CompensationPlanRow{plan},
		dailyHours: []repositories.DailyHoursRecord{
			anDH("2026-08-05", "p1", "Alpha", 8),
		},
		projectHours: []repositories.ProjectHours{
			{ProjectID: "p1", ProjectName: "Alpha", Hours: 8},
		},
		contribsForMonth: []repositories.ContributionRow{
			{EmployeeID: "e1", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
		},
		contribsForYear: []repositories.ContributionRow{
			{EmployeeID: "e1", PayrollMonth: anD("2026-08-01"), ContributionsAmount: 500},
		},
		matBreakdown: []repositories.ProjectMaterialCostRow{
			{
				ProjectMaterialID: "mat-1",
				MaterialName:      "Cement",
				MaterialCode:      &matCode,
				InstalledQty:      4,
				Unit:              "vreća",
				UnitPrice:         200,
				TotalCost:         800,
				Source:            "manual",
			},
		},
		matCostsByMonth: map[int]float64{8: 800},
	}
	result, err := anSvc(repo).GetProjectDetail(context.Background(), "co1", "direktor", "p1", 2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalMaterialCost != 800 {
		t.Errorf("expected TotalMaterialCost=800, got %f", result.TotalMaterialCost)
	}
	if result.TotalDirectCost != 3300 {
		t.Errorf("expected TotalDirectCost=3300 (2500+800), got %f", result.TotalDirectCost)
	}
	if len(result.Materials) != 1 {
		t.Fatalf("expected 1 material item, got %d", len(result.Materials))
	}
	mat := result.Materials[0]
	if mat.MaterialName != "Cement" {
		t.Errorf("expected MaterialName=Cement, got %s", mat.MaterialName)
	}
	if mat.TotalCost != 800 {
		t.Errorf("expected material TotalCost=800, got %f", mat.TotalCost)
	}
	if len(result.MonthlyTrend) != 12 {
		t.Fatalf("expected 12 trend months, got %d", len(result.MonthlyTrend))
	}
	aug := result.MonthlyTrend[7] // month 8, index 7
	if aug.MaterialCost != 800 {
		t.Errorf("expected August MaterialCost=800, got %f", aug.MaterialCost)
	}
	if aug.TotalCost != aug.Bruto+800 {
		t.Errorf("expected August TotalCost=bruto+800 (%f), got %f", aug.Bruto+800, aug.TotalCost)
	}
}
