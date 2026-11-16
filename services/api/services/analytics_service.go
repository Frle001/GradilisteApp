package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Interface ─────────────────────────────────────────────────────────────────

type analyticsRepoIface interface {
	// Compensation plans
	CreateCompensationPlan(ctx context.Context, p repositories.CreateCompensationParams) (*repositories.CompensationPlanRow, error)
	UpdatePlanEffectiveTo(ctx context.Context, companyID, planID string, to time.Time) error
	ListCompensationPlans(ctx context.Context, companyID, employeeID string) ([]repositories.CompensationPlanRow, error)
	GetPlansInDateRange(ctx context.Context, companyID, employeeID string, from, to time.Time) ([]repositories.CompensationPlanRow, error)
	HasOverlappingPlan(ctx context.Context, companyID, employeeID string, from time.Time, to *time.Time, excludeID *string) (bool, error)
	FindOpenPlan(ctx context.Context, companyID, employeeID string) (*repositories.CompensationPlanRow, error)
	GetCompensationPlanByID(ctx context.Context, companyID, planID string) (*repositories.CompensationPlanRow, error)
	UpdateCompensationPlanFields(ctx context.Context, p repositories.UpdateCompensationPlanParams) (*repositories.CompensationPlanRow, error)
	// Worker hours
	GetDailyHoursForEmployee(ctx context.Context, companyID, employeeID string, from, to time.Time) ([]repositories.DailyHoursRecord, error)
	GetHoursByProject(ctx context.Context, companyID, employeeID string, from, to time.Time) ([]repositories.ProjectHours, error)
	// Employees
	ListEligibleEmployees(ctx context.Context, companyID string, from, to time.Time) ([]repositories.AnalyticsEmployeeRow, error)
	GetEmployee(ctx context.Context, companyID, employeeID string) (*repositories.AnalyticsEmployeeRow, error)
	// Contributions
	GetContribution(ctx context.Context, companyID, employeeID string, month time.Time) (*repositories.ContributionRow, error)
	GetContributionsForMonth(ctx context.Context, companyID string, month time.Time) ([]repositories.ContributionRow, error)
	GetContributionsForEmployeeYear(ctx context.Context, companyID, employeeID string, year int) ([]repositories.ContributionRow, error)
	GetContributionsForCompanyYear(ctx context.Context, companyID string, year int) ([]repositories.ContributionRow, error)
	UpsertContribution(ctx context.Context, p repositories.UpsertContributionParams) (*repositories.ContributionRow, error)
	// Annual / company-wide
	GetCompanyPlansForYear(ctx context.Context, companyID string, from, to time.Time) ([]repositories.CompensationPlanRow, error)
	GetCompanyDailyHoursForYear(ctx context.Context, companyID string, from, to time.Time) ([]repositories.CompanyDailyHoursRecord, error)
	// Material costs
	GetProjectMaterialCostsForMonth(ctx context.Context, companyID string, year, month int) (map[string]float64, error)
	GetProjectMaterialBreakdown(ctx context.Context, companyID, projectID string, year, month int) ([]repositories.ProjectMaterialCostRow, error)
	GetProjectMaterialCostsForYear(ctx context.Context, companyID, projectID string, year int) (map[int]float64, error)
}

// ── Service ───────────────────────────────────────────────────────────────────

// contribKey is used in annual/multi-month computations to look up a contribution
// by employee ID and month in a flat slice loaded from the DB.
type contribKey struct {
	empID string
	month time.Month
}

type AnalyticsService struct {
	repo analyticsRepoIface
}

func NewAnalyticsService(repo *repositories.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// ── Access control ────────────────────────────────────────────────────────────

func requireAnalyticsAccess(role string) error {
	if role != "direktor" && role != "inzenjer" {
		return ErrForbidden
	}
	return nil
}

// ── Public methods ────────────────────────────────────────────────────────────

func (s *AnalyticsService) ListCompensationPlans(ctx context.Context, companyID, callerRole, employeeID string) ([]dto.CompensationPlan, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetEmployee(ctx, companyID, employeeID); err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.repo.ListCompensationPlans(ctx, companyID, employeeID)
	if err != nil {
		return nil, fmt.Errorf("analytics.ListCompensationPlans: %w", err)
	}
	out := make([]dto.CompensationPlan, 0, len(rows))
	for _, r := range rows {
		out = append(out, planRowToDTO(r))
	}
	return out, nil
}

func (s *AnalyticsService) CreateCompensationPlan(ctx context.Context, companyID, callerRole, callerUserID, employeeID string, req dto.CreateCompensationPlanReq) (*dto.CompensationPlan, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}

	// Validate employee belongs to company
	if _, err := s.repo.GetEmployee(ctx, companyID, employeeID); err != nil {
		return nil, ErrNotFound
	}

	// Validate pay_type
	if req.PayType != dto.PayTypeFixed && req.PayType != dto.PayTypeHourly {
		return nil, validationErr("Neispravni model plaće")
	}

	// Validate amounts
	if req.PayAmount <= 0 {
		return nil, validationErr("Iznos mora biti veći od nule")
	}
	if req.CompanyCostAmount != nil && *req.CompanyCostAmount <= 0 {
		return nil, validationErr("Trošak firme mora biti veći od nule")
	}

	// Parse effective_from
	effectiveFrom, err := time.Parse("2006-01-02", req.EffectiveFrom)
	if err != nil {
		return nil, validationErr("Neispravan datum početka (format: YYYY-MM-DD)")
	}

	// Auto-close any open-ended plan if it starts before the new one
	openPlan, err := s.repo.FindOpenPlan(ctx, companyID, employeeID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("analytics.CreateCompensationPlan: find open plan: %w", err)
	}
	if openPlan != nil {
		if openPlan.EffectiveFrom.Before(effectiveFrom) {
			// Close the open plan one day before the new one starts
			closeDate := effectiveFrom.AddDate(0, 0, -1)
			if err := s.repo.UpdatePlanEffectiveTo(ctx, companyID, openPlan.ID, closeDate); err != nil {
				return nil, fmt.Errorf("analytics.CreateCompensationPlan: close open plan: %w", err)
			}
		} else {
			return nil, validationErr("Postoji aktivni plan koji počinje istog ili kasnijeg datuma")
		}
	}

	// Check for remaining overlaps with closed plans
	overlap, err := s.repo.HasOverlappingPlan(ctx, companyID, employeeID, effectiveFrom, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("analytics.CreateCompensationPlan: overlap check: %w", err)
	}
	if overlap {
		return nil, validationErr("Odabrani period se preklapa s postojećim planom")
	}

	row, err := s.repo.CreateCompensationPlan(ctx, repositories.CreateCompensationParams{
		CompanyID:         companyID,
		EmployeeID:        employeeID,
		PayType:           req.PayType,
		PayAmount:         req.PayAmount,
		CompanyCostAmount: req.CompanyCostAmount,
		EffectiveFrom:     effectiveFrom,
		EffectiveTo:       nil,
		CreatedBy:         callerUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("analytics.CreateCompensationPlan: insert: %w", err)
	}

	out := planRowToDTO(*row)
	return &out, nil
}

func (s *AnalyticsService) UpdateCompensationPlan(ctx context.Context, companyID, callerRole, planID string, req dto.UpdateCompensationPlanReq) (*dto.CompensationPlan, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetCompensationPlanByID(ctx, companyID, planID)
	if err != nil {
		return nil, ErrNotFound
	}

	if _, err := s.repo.GetEmployee(ctx, companyID, existing.EmployeeID); err != nil {
		return nil, ErrNotFound
	}

	if req.PayType != dto.PayTypeFixed && req.PayType != dto.PayTypeHourly {
		return nil, validationErr("Neispravni model plaće")
	}
	if req.PayAmount <= 0 {
		return nil, validationErr("Iznos mora biti veći od nule")
	}
	if req.CompanyCostAmount != nil && *req.CompanyCostAmount <= 0 {
		return nil, validationErr("Trošak firme mora biti veći od nule")
	}

	effectiveFrom, err := time.Parse("2006-01-02", req.EffectiveFrom)
	if err != nil {
		return nil, validationErr("Neispravan datum početka (format: YYYY-MM-DD)")
	}

	var effectiveTo *time.Time
	if req.EffectiveTo != nil && *req.EffectiveTo != "" {
		t, err2 := time.Parse("2006-01-02", *req.EffectiveTo)
		if err2 != nil {
			return nil, validationErr("Neispravan datum završetka (format: YYYY-MM-DD)")
		}
		if t.Before(effectiveFrom) {
			return nil, validationErr("Datum završetka mora biti nakon datuma početka")
		}
		effectiveTo = &t
	}

	overlap, err := s.repo.HasOverlappingPlan(ctx, companyID, existing.EmployeeID, effectiveFrom, effectiveTo, &planID)
	if err != nil {
		return nil, fmt.Errorf("analytics.UpdateCompensationPlan: overlap check: %w", err)
	}
	if overlap {
		return nil, validationErr("Odabrani period se preklapa s postojećim planom")
	}

	row, err := s.repo.UpdateCompensationPlanFields(ctx, repositories.UpdateCompensationPlanParams{
		PlanID:            planID,
		CompanyID:         companyID,
		PayType:           req.PayType,
		PayAmount:         req.PayAmount,
		CompanyCostAmount: req.CompanyCostAmount,
		EffectiveFrom:     effectiveFrom,
		EffectiveTo:       effectiveTo,
	})
	if err != nil {
		return nil, fmt.Errorf("analytics.UpdateCompensationPlan: update: %w", err)
	}

	out := planRowToDTO(*row)
	return &out, nil
}

func (s *AnalyticsService) GetEmployeeLaborCost(ctx context.Context, companyID, callerRole, employeeID string, year, month int) (*dto.EmployeeLaborCost, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	emp, err := s.repo.GetEmployee(ctx, companyID, employeeID)
	if err != nil {
		return nil, ErrNotFound
	}
	from, to := monthBounds(year, month)
	contrib, err := s.repo.GetContribution(ctx, companyID, employeeID, from)
	if err != nil {
		return nil, fmt.Errorf("analytics: get contribution: %w", err)
	}
	return s.computeForEmployee(ctx, companyID, *emp, from, to, contrib)
}

func (s *AnalyticsService) GetMonthlyLaborSummary(ctx context.Context, companyID, callerRole string, year, month int) (*dto.MonthlyLaborSummary, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	from, to := monthBounds(year, month)

	employees, err := s.repo.ListEligibleEmployees(ctx, companyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetMonthlyLaborSummary: list employees: %w", err)
	}

	// Batch-fetch all contributions for this month to avoid N+1 queries.
	contribs, err := s.repo.GetContributionsForMonth(ctx, companyID, from)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetMonthlyLaborSummary: get contributions: %w", err)
	}
	contribMap := make(map[string]*repositories.ContributionRow, len(contribs))
	for i := range contribs {
		contribMap[contribs[i].EmployeeID] = &contribs[i]
	}

	summary := &dto.MonthlyLaborSummary{
		Year:      year,
		Month:     month,
		Employees: make([]dto.EmployeeLaborCost, 0, len(employees)),
	}

	for _, emp := range employees {
		cost, err := s.computeForEmployee(ctx, companyID, emp, from, to, contribMap[emp.ID])
		if err != nil {
			return nil, err
		}
		summary.Employees = append(summary.Employees, *cost)
		summary.TotalHours += cost.TotalHours
		summary.TotalKnownCost += cost.CompanyAnalyticalCost
		summary.TotalNeto += cost.Neto
		if cost.Contributions != nil {
			summary.TotalContributions += *cost.Contributions
		}
		if cost.Bruto != nil {
			summary.TotalBruto += *cost.Bruto
		}
		if !cost.HasCompensation && cost.TotalHours > 0 {
			summary.EmployeesWithoutCompensation++
		}
		if cost.HasCompensation && cost.ContributionsMissing {
			summary.EmployeesMissingContributions++
		}
	}

	if summary.TotalHours > 0 && summary.TotalKnownCost > 0 {
		avg := summary.TotalKnownCost / summary.TotalHours
		summary.AvgCostPerHour = &avg
	}

	employeesWithCompensation := len(employees) - summary.EmployeesWithoutCompensation
	summary.BrutoComplete = employeesWithCompensation > 0 && summary.EmployeesMissingContributions == 0

	return summary, nil
}

// ── New public methods ────────────────────────────────────────────────────────

func (s *AnalyticsService) UpsertContribution(ctx context.Context, companyID, callerRole, callerUserID, employeeID string, year, month int, req dto.UpsertContributionReq) (*dto.EmployeeContribution, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetEmployee(ctx, companyID, employeeID); err != nil {
		return nil, ErrNotFound
	}
	if req.ContributionsAmount < 0 {
		return nil, validationErr("Iznos doprinosa ne može biti negativan")
	}
	from, _ := monthBounds(year, month)
	row, err := s.repo.UpsertContribution(ctx, repositories.UpsertContributionParams{
		CompanyID:           companyID,
		EmployeeID:          employeeID,
		PayrollMonth:        from,
		ContributionsAmount: req.ContributionsAmount,
		UpdatedBy:           callerUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("analytics.UpsertContribution: %w", err)
	}
	out := dto.EmployeeContribution{
		ID:                  row.ID,
		EmployeeID:          row.EmployeeID,
		PayrollMonth:        row.PayrollMonth.Format("2006-01-02"),
		ContributionsAmount: row.ContributionsAmount,
		UpdatedAt:           row.UpdatedAt,
	}
	return &out, nil
}

func (s *AnalyticsService) GetEmployeePayrollHistory(ctx context.Context, companyID, callerRole, employeeID string, year int) (*dto.EmployeePayrollHistory, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	emp, err := s.repo.GetEmployee(ctx, companyID, employeeID)
	if err != nil {
		return nil, ErrNotFound
	}

	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)

	plans, err := s.repo.GetPlansInDateRange(ctx, companyID, employeeID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetEmployeePayrollHistory: get plans: %w", err)
	}
	allHours, err := s.repo.GetDailyHoursForEmployee(ctx, companyID, employeeID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetEmployeePayrollHistory: get hours: %w", err)
	}
	contribs, err := s.repo.GetContributionsForEmployeeYear(ctx, companyID, employeeID, year)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetEmployeePayrollHistory: get contributions: %w", err)
	}
	contribMap := make(map[time.Month]*repositories.ContributionRow, len(contribs))
	for i := range contribs {
		contribMap[contribs[i].PayrollMonth.Month()] = &contribs[i]
	}

	history := &dto.EmployeePayrollHistory{
		EmployeeID:   emp.ID,
		EmployeeName: emp.Name,
		Year:         year,
		Months:       make([]dto.MonthlyPayrollEntry, 0, 12),
	}

	inf := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

	for mo := 1; mo <= 12; mo++ {
		from, to := monthBounds(year, mo)

		var monthPlans []repositories.CompensationPlanRow
		for _, p := range plans {
			pFrom := p.EffectiveFrom.Truncate(24 * time.Hour)
			pTo := inf
			if p.EffectiveTo != nil {
				pTo = p.EffectiveTo.Truncate(24 * time.Hour)
			}
			if !pFrom.After(to) && !pTo.Before(from) {
				monthPlans = append(monthPlans, p)
			}
		}

		var monthHours []repositories.DailyHoursRecord
		for _, h := range allHours {
			d := h.WorkDate.Truncate(24 * time.Hour)
			if !d.Before(from) && !d.After(to) {
				monthHours = append(monthHours, h)
			}
		}

		cost := computeEmployeeLaborCost(*emp, monthPlans, monthHours, nil)
		enrichWithContribution(&cost, contribMap[time.Month(mo)])

		entry := dto.MonthlyPayrollEntry{
			Year:                 year,
			Month:                mo,
			Neto:                 cost.Neto,
			Contributions:        cost.Contributions,
			Bruto:                cost.Bruto,
			ContributionsMissing: cost.ContributionsMissing,
			HasCompensation:      cost.HasCompensation,
			TotalHours:           cost.TotalHours,
			Warning:              cost.Warning,
		}
		history.Months = append(history.Months, entry)

		history.TotalNeto += cost.Neto
		if cost.Contributions != nil {
			history.TotalContrib += *cost.Contributions
		}
		if cost.Bruto != nil {
			history.TotalBruto += *cost.Bruto
		}
	}

	return history, nil
}

// GetAnnualPayrollSummary computes 12-month aggregated payroll data for the given year.
// If employeeID is non-empty, it filters to that one employee (uses GetEmployeePayrollHistory).
// Otherwise it aggregates across all eligible employees using 4 company-wide DB queries.
func (s *AnalyticsService) GetAnnualPayrollSummary(ctx context.Context, companyID, callerRole string, year int, employeeID string) (*dto.AnnualPayrollSummary, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}

	// Single-employee path: reuse existing history logic.
	if employeeID != "" {
		hist, err := s.GetEmployeePayrollHistory(ctx, companyID, callerRole, employeeID, year)
		if err != nil {
			return nil, err
		}
		summary := &dto.AnnualPayrollSummary{
			Year:                 year,
			TotalNeto:            hist.TotalNeto,
			TotalContributions:   hist.TotalContrib,
			TotalBruto:           hist.TotalBruto,
			TotalEmployees:       1,
			EmployeeFilter:       &employeeID,
			FilteredEmployeeName: &hist.EmployeeName,
			Months:               make([]dto.MonthlyPayrollAggregate, 0, 12),
		}
		for _, m := range hist.Months {
			incomplete := 0
			if m.HasCompensation && m.ContributionsMissing {
				incomplete = 1
			}
			agg := dto.MonthlyPayrollAggregate{
				Year:               year,
				Month:              m.Month,
				TotalNeto:          m.Neto,
				TotalContributions: 0,
				TotalBruto:         0,
				TotalHours:         m.TotalHours,
				EmployeeCount:      1,
				IncompleteCount:    incomplete,
				HasData:            m.HasCompensation || m.TotalHours > 0,
			}
			if m.Contributions != nil {
				agg.TotalContributions = *m.Contributions
			}
			if m.Bruto != nil {
				agg.TotalBruto = *m.Bruto
			}
			if incomplete > 0 {
				summary.MonthsWithIncompleteData++
			}
			summary.Months = append(summary.Months, agg)
		}
		summary.BrutoComplete = summary.TotalEmployees > 0 && summary.MonthsWithIncompleteData == 0
		return summary, nil
	}

	// All-employees path: 4 company-wide queries, compute in-memory.
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)

	employees, err := s.repo.ListEligibleEmployees(ctx, companyID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetAnnualPayrollSummary: list employees: %w", err)
	}

	allPlans, err := s.repo.GetCompanyPlansForYear(ctx, companyID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetAnnualPayrollSummary: get plans: %w", err)
	}

	allHours, err := s.repo.GetCompanyDailyHoursForYear(ctx, companyID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetAnnualPayrollSummary: get hours: %w", err)
	}

	allContribs, err := s.repo.GetContributionsForCompanyYear(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetAnnualPayrollSummary: get contributions: %w", err)
	}

	// Build per-employee maps.
	plansMap := make(map[string][]repositories.CompensationPlanRow)
	for _, p := range allPlans {
		plansMap[p.EmployeeID] = append(plansMap[p.EmployeeID], p)
	}
	hoursMap := make(map[string][]repositories.DailyHoursRecord)
	for _, h := range allHours {
		rec := repositories.DailyHoursRecord{WorkDate: h.WorkDate, ProjectID: h.ProjectID, ProjectName: h.ProjectName, Hours: h.Hours}
		hoursMap[h.EmployeeID] = append(hoursMap[h.EmployeeID], rec)
	}
	// contribMap[employeeID][month] = *ContributionRow
	contribMap := make(map[contribKey]*repositories.ContributionRow, len(allContribs))
	for i := range allContribs {
		c := &allContribs[i]
		contribMap[contribKey{c.EmployeeID, c.PayrollMonth.Month()}] = c
	}

	inf := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

	summary := &dto.AnnualPayrollSummary{
		Year:           year,
		TotalEmployees: len(employees),
		Months:         make([]dto.MonthlyPayrollAggregate, 0, 12),
	}

	for mo := 1; mo <= 12; mo++ {
		from, to := monthBounds(year, mo)
		agg := dto.MonthlyPayrollAggregate{Year: year, Month: mo}

		for _, emp := range employees {
			// Filter plans to this month.
			var monthPlans []repositories.CompensationPlanRow
			for _, p := range plansMap[emp.ID] {
				pFrom := p.EffectiveFrom.Truncate(24 * time.Hour)
				pTo := inf
				if p.EffectiveTo != nil {
					pTo = p.EffectiveTo.Truncate(24 * time.Hour)
				}
				if !pFrom.After(to) && !pTo.Before(from) {
					monthPlans = append(monthPlans, p)
				}
			}
			// Filter hours to this month.
			var monthHours []repositories.DailyHoursRecord
			for _, h := range hoursMap[emp.ID] {
				d := h.WorkDate.Truncate(24 * time.Hour)
				if !d.Before(from) && !d.After(to) {
					monthHours = append(monthHours, h)
				}
			}

			cost := computeEmployeeLaborCost(emp, monthPlans, monthHours, nil)
			enrichWithContribution(&cost, contribMap[contribKey{emp.ID, time.Month(mo)}])

			agg.TotalHours += cost.TotalHours
			agg.TotalNeto += cost.Neto
			if cost.Contributions != nil {
				agg.TotalContributions += *cost.Contributions
			}
			if cost.Bruto != nil {
				agg.TotalBruto += *cost.Bruto
			}
			if cost.HasCompensation && cost.ContributionsMissing {
				agg.IncompleteCount++
			}
			if cost.HasCompensation || cost.TotalHours > 0 {
				agg.EmployeeCount++
				agg.HasData = true
			}
		}

		if agg.IncompleteCount > 0 {
			summary.MonthsWithIncompleteData++
		}
		summary.TotalNeto += agg.TotalNeto
		summary.TotalContributions += agg.TotalContributions
		summary.TotalBruto += agg.TotalBruto
		summary.TotalHours += agg.TotalHours
		summary.Months = append(summary.Months, agg)
	}

	summary.BrutoComplete = summary.TotalEmployees > 0 && summary.MonthsWithIncompleteData == 0
	return summary, nil
}

// GetMonthlyProjectAllocation aggregates per-employee project labor allocations
// into company-wide project totals for the given month.
func (s *AnalyticsService) GetMonthlyProjectAllocation(ctx context.Context, companyID, callerRole string, year, month int) (*dto.MonthlyProjectAllocationSummary, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}

	monthly, err := s.GetMonthlyLaborSummary(ctx, companyID, callerRole, year, month)
	if err != nil {
		return nil, err
	}

	type projAgg struct {
		name              string
		hours             float64
		netoCost          float64
		contribCost       float64
		contribIncomplete bool
		emps              map[string]struct{}
	}
	byProject := make(map[string]*projAgg)
	var totalProjectHours float64

	for _, emp := range monthly.Employees {
		for _, pa := range emp.ProjectAllocations {
			if _, ok := byProject[pa.ProjectID]; !ok {
				byProject[pa.ProjectID] = &projAgg{name: pa.ProjectName, emps: make(map[string]struct{})}
			}
			agg := byProject[pa.ProjectID]
			agg.hours += pa.Hours
			agg.netoCost += pa.LaborCost
			agg.emps[emp.EmployeeID] = struct{}{}
			totalProjectHours += pa.Hours

			// Distribute this employee's contributions proportionally to their project hours.
			if emp.TotalHours > 0 && emp.Contributions != nil {
				agg.contribCost += *emp.Contributions * (pa.Hours / emp.TotalHours)
			} else if emp.HasCompensation && emp.ContributionsMissing {
				agg.contribIncomplete = true
			}
		}
	}

	unassigned := monthly.TotalHours - totalProjectHours
	if unassigned < 0 {
		unassigned = 0
	}

	// Fetch material costs for this month (single DB query, map[projectID]→cost).
	matCosts, err := s.repo.GetProjectMaterialCostsForMonth(ctx, companyID, year, month)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetMonthlyProjectAllocation: material costs: %w", err)
	}

	var totalNeto, totalContrib, totalBruto, totalMaterial float64
	allocations := make([]dto.MonthlyProjectAllocation, 0, len(byProject))
	for projID, agg := range byProject {
		pct := 0.0
		if monthly.TotalHours > 0 {
			pct = agg.hours / monthly.TotalHours * 100
		}
		bruto := agg.netoCost + agg.contribCost
		matCost := matCosts[projID]
		totalNeto += agg.netoCost
		totalContrib += agg.contribCost
		totalBruto += bruto
		totalMaterial += matCost
		allocations = append(allocations, dto.MonthlyProjectAllocation{
			ProjectID:               projID,
			ProjectName:             agg.name,
			TotalHours:              agg.hours,
			LaborCost:               agg.netoCost, // backward compat
			NetoCost:                agg.netoCost,
			ContributionsCost:       agg.contribCost,
			BrutoCost:               bruto,
			MaterialCost:            matCost,
			TotalDirectCost:         bruto + matCost,
			EmployeeCount:           len(agg.emps),
			PctOfTotal:              pct,
			ContributionsIncomplete: agg.contribIncomplete,
		})
	}
	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].TotalHours > allocations[j].TotalHours
	})

	return &dto.MonthlyProjectAllocationSummary{
		Year:               year,
		Month:              month,
		TotalHours:         monthly.TotalHours,
		TotalLaborCost:     totalNeto, // backward compat
		TotalNeto:          totalNeto,
		TotalContributions: totalContrib,
		TotalBruto:         totalBruto,
		TotalMaterialCost:  totalMaterial,
		TotalDirectCost:    totalBruto + totalMaterial,
		UnassignedHours:    unassigned,
		BrutoComplete:      monthly.BrutoComplete,
		Allocations:        allocations,
	}, nil
}

// GetProjectDetail returns per-employee cost breakdown for a specific project in the
// given month, plus a 12-month cost trend for the year. Uses company-wide DB queries
// so only 6 total round-trips are needed regardless of employee count.
func (s *AnalyticsService) GetProjectDetail(ctx context.Context, companyID, callerRole, projectID string, year, month int) (*dto.ProjectDetailSummary, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}

	// Monthly employee breakdown: reuse existing monthly summary.
	monthly, err := s.GetMonthlyLaborSummary(ctx, companyID, callerRole, year, month)
	if err != nil {
		return nil, err
	}

	result := &dto.ProjectDetailSummary{
		ProjectID:    projectID,
		Year:         year,
		Month:        month,
		Employees:    []dto.ProjectEmployeeAllocation{},
		MonthlyTrend: make([]dto.ProjectMonthlyTrend, 0, 12),
	}

	for _, emp := range monthly.Employees {
		for _, pa := range emp.ProjectAllocations {
			if pa.ProjectID != projectID {
				continue
			}
			if result.ProjectName == "" {
				result.ProjectName = pa.ProjectName
			}
			result.TotalHours += pa.Hours
			result.TotalNeto += pa.LaborCost

			empAlloc := dto.ProjectEmployeeAllocation{
				EmployeeID:           emp.EmployeeID,
				EmployeeName:         emp.EmployeeName,
				Hours:                pa.Hours,
				NetoCost:             pa.LaborCost,
				ContributionsMissing: emp.ContributionsMissing,
			}
			if emp.TotalHours > 0 && emp.Contributions != nil {
				share := *emp.Contributions * (pa.Hours / emp.TotalHours)
				empAlloc.ContributionsCost = &share
				b := pa.LaborCost + share
				empAlloc.BrutoCost = &b
				result.TotalContributions += share
			}
			result.Employees = append(result.Employees, empAlloc)
		}
	}
	result.TotalBruto = result.TotalNeto + result.TotalContributions

	// PctOfProject and sort
	for i := range result.Employees {
		if result.TotalHours > 0 {
			result.Employees[i].PctOfProject = result.Employees[i].Hours / result.TotalHours * 100
		}
	}
	sort.Slice(result.Employees, func(i, j int) bool {
		return result.Employees[i].Hours > result.Employees[j].Hours
	})

	// Determine bruto completeness for the month: missing if any emp has compensation + missing contrib
	result.BrutoComplete = monthly.BrutoComplete

	// Material breakdown for this project+month.
	matRows, err := s.repo.GetProjectMaterialBreakdown(ctx, companyID, projectID, year, month)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetProjectDetail: material breakdown: %w", err)
	}
	result.Materials = make([]dto.ProjectMaterialCostItem, 0, len(matRows))
	for _, m := range matRows {
		result.TotalMaterialCost += m.TotalCost
		result.Materials = append(result.Materials, dto.ProjectMaterialCostItem{
			ProjectMaterialID: m.ProjectMaterialID,
			MaterialName:      m.MaterialName,
			MaterialCode:      m.MaterialCode,
			InstalledQty:      m.InstalledQty,
			Unit:              m.Unit,
			UnitPrice:         m.UnitPrice,
			TotalCost:         m.TotalCost,
			Source:            m.Source,
		})
	}
	result.TotalDirectCost = result.TotalBruto + result.TotalMaterialCost

	// Annual trend: 4 company-wide queries for the year + 1 material cost query, compute in memory.
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)

	employees, err := s.repo.ListEligibleEmployees(ctx, companyID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetProjectDetail: list employees: %w", err)
	}
	allPlans, err := s.repo.GetCompanyPlansForYear(ctx, companyID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetProjectDetail: get plans: %w", err)
	}
	allHours, err := s.repo.GetCompanyDailyHoursForYear(ctx, companyID, yearStart, yearEnd)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetProjectDetail: get hours: %w", err)
	}
	allContribs, err := s.repo.GetContributionsForCompanyYear(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetProjectDetail: get contributions: %w", err)
	}

	// Material costs per month for the full year (single query).
	matByMonth, err := s.repo.GetProjectMaterialCostsForYear(ctx, companyID, projectID, year)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetProjectDetail: material costs for year: %w", err)
	}

	plansMap := make(map[string][]repositories.CompensationPlanRow)
	for _, p := range allPlans {
		plansMap[p.EmployeeID] = append(plansMap[p.EmployeeID], p)
	}
	hoursMap := make(map[string][]repositories.DailyHoursRecord)
	for _, h := range allHours {
		rec := repositories.DailyHoursRecord{WorkDate: h.WorkDate, ProjectID: h.ProjectID, ProjectName: h.ProjectName, Hours: h.Hours}
		hoursMap[h.EmployeeID] = append(hoursMap[h.EmployeeID], rec)
	}
	ckMap := make(map[contribKey]*repositories.ContributionRow, len(allContribs))
	for i := range allContribs {
		c := &allContribs[i]
		ckMap[contribKey{c.EmployeeID, c.PayrollMonth.Month()}] = c
	}

	inf := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

	for mo := 1; mo <= 12; mo++ {
		from, to := monthBounds(year, mo)
		trend := dto.ProjectMonthlyTrend{Month: mo}

		for _, emp := range employees {
			var monthPlans []repositories.CompensationPlanRow
			for _, p := range plansMap[emp.ID] {
				pFrom := p.EffectiveFrom.Truncate(24 * time.Hour)
				pTo := inf
				if p.EffectiveTo != nil {
					pTo = p.EffectiveTo.Truncate(24 * time.Hour)
				}
				if !pFrom.After(to) && !pTo.Before(from) {
					monthPlans = append(monthPlans, p)
				}
			}
			var monthHours []repositories.DailyHoursRecord
			for _, h := range hoursMap[emp.ID] {
				d := h.WorkDate.Truncate(24 * time.Hour)
				if !d.Before(from) && !d.After(to) {
					monthHours = append(monthHours, h)
				}
			}

			// Build aggregated project hours from daily hours for this month.
			projHoursMap := make(map[string]*repositories.ProjectHours)
			for _, h := range monthHours {
				if _, ok := projHoursMap[h.ProjectID]; !ok {
					projHoursMap[h.ProjectID] = &repositories.ProjectHours{ProjectID: h.ProjectID, ProjectName: h.ProjectName}
				}
				projHoursMap[h.ProjectID].Hours += h.Hours
			}
			projHours := make([]repositories.ProjectHours, 0, len(projHoursMap))
			for _, ph := range projHoursMap {
				projHours = append(projHours, *ph)
			}

			cost := computeEmployeeLaborCost(emp, monthPlans, monthHours, projHours)
			enrichWithContribution(&cost, ckMap[contribKey{emp.ID, time.Month(mo)}])

			for _, pa := range cost.ProjectAllocations {
				if pa.ProjectID != projectID {
					continue
				}
				trend.Hours += pa.Hours
				trend.Neto += pa.LaborCost
				if cost.Contributions != nil && cost.TotalHours > 0 {
					trend.Contributions += *cost.Contributions * (pa.Hours / cost.TotalHours)
				}
				if cost.HasCompensation && cost.ContributionsMissing {
					trend.Incomplete = true
				}
				trend.HasData = true
			}
		}
		trend.Bruto = trend.Neto + trend.Contributions
		trend.MaterialCost = matByMonth[mo]
		trend.TotalCost = trend.Bruto + trend.MaterialCost
		if trend.MaterialCost > 0 {
			trend.HasData = true
		}
		result.MonthlyTrend = append(result.MonthlyTrend, trend)
	}

	return result, nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (s *AnalyticsService) computeForEmployee(ctx context.Context, companyID string, emp repositories.AnalyticsEmployeeRow, from, to time.Time, contrib *repositories.ContributionRow) (*dto.EmployeeLaborCost, error) {
	plans, err := s.repo.GetPlansInDateRange(ctx, companyID, emp.ID, from, to)
	if err != nil {
		return nil, fmt.Errorf("analytics: get plans for %s: %w", emp.ID, err)
	}
	dailyHours, err := s.repo.GetDailyHoursForEmployee(ctx, companyID, emp.ID, from, to)
	if err != nil {
		return nil, fmt.Errorf("analytics: get daily hours for %s: %w", emp.ID, err)
	}
	projectHours, err := s.repo.GetHoursByProject(ctx, companyID, emp.ID, from, to)
	if err != nil {
		return nil, fmt.Errorf("analytics: get project hours for %s: %w", emp.ID, err)
	}
	cost := computeEmployeeLaborCost(emp, plans, dailyHours, projectHours)
	enrichWithContribution(&cost, contrib)
	return &cost, nil
}

// enrichWithContribution sets neto/contributions/bruto on a computed EmployeeLaborCost.
// contrib == nil means no record was entered (ContributionsMissing = true).
// Bruto is only computed when neto is reliably known (no mid-month transition or mixed-type error).
func enrichWithContribution(cost *dto.EmployeeLaborCost, contrib *repositories.ContributionRow) {
	cost.Neto = cost.CompanyAnalyticalCost
	// Bruto is available when compensation is set, no mid-month transition occurred, and
	// any warning did not cause the neto to be reset to zero.
	brutoReady := cost.HasCompensation && !cost.HasMidMonthTransition &&
		!(cost.Warning != nil && cost.CompanyAnalyticalCost == 0)
	if contrib == nil {
		cost.ContributionsMissing = true
		return
	}
	d := contrib.ContributionsAmount
	cost.Contributions = &d
	if brutoReady {
		b := cost.Neto + d
		cost.Bruto = &b
	}
}

// monthBounds returns the first and last day of the given year/month as time.Time.
func monthBounds(year, month int) (time.Time, time.Time) {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1)
	return from, to
}

// planRowToDTO converts a CompensationPlanRow to a DTO, formatting dates.
func planRowToDTO(r repositories.CompensationPlanRow) dto.CompensationPlan {
	plan := dto.CompensationPlan{
		ID:                r.ID,
		EmployeeID:        r.EmployeeID,
		EmployeeName:      r.EmployeeName,
		PayType:           r.PayType,
		PayAmount:         r.PayAmount,
		CompanyCostAmount: r.CompanyCostAmount,
		EffectiveFrom:     r.EffectiveFrom.Format("2006-01-02"),
		CreatedBy:         r.CreatedBy,
		CreatedAt:         r.CreatedAt,
	}
	if r.EffectiveTo != nil {
		s := r.EffectiveTo.Format("2006-01-02")
		plan.EffectiveTo = &s
	}
	return plan
}

// computeEmployeeLaborCost is pure (no IO) — used by the service and tests.
func computeEmployeeLaborCost(
	emp repositories.AnalyticsEmployeeRow,
	plans []repositories.CompensationPlanRow,
	dailyHours []repositories.DailyHoursRecord,
	projectHours []repositories.ProjectHours,
) dto.EmployeeLaborCost {
	result := dto.EmployeeLaborCost{
		EmployeeID:         emp.ID,
		EmployeeName:       emp.Name,
		EmployeeRole:       emp.Role,
		ProjectAllocations: []dto.ProjectLaborAllocation{},
	}

	for _, d := range dailyHours {
		result.TotalHours += d.Hours
	}

	if len(plans) == 0 {
		if result.TotalHours > 0 {
			w := "Plaća nije definirana"
			result.Warning = &w
		}
		return result
	}

	result.HasCompensation = true

	// Detect mixed pay types
	fixedCount, hourlyCount := 0, 0
	for _, p := range plans {
		if p.PayType == dto.PayTypeFixed {
			fixedCount++
		} else {
			hourlyCount++
		}
	}
	if fixedCount > 0 && hourlyCount > 0 {
		w := "Prijelaz između modela plaće unutar odabranog perioda"
		result.Warning = &w
		return result
	}

	payType := plans[0].PayType
	result.PayType = &payType
	amt := plans[0].PayAmount
	result.PayAmount = &amt

	if payType == dto.PayTypeFixed {
		if len(plans) > 1 {
			result.HasMidMonthTransition = true
			w := "Prijelaz fiksne plaće unutar odabranog perioda – automatski izračun nije dostupan"
			result.Warning = &w
			return result
		}
		plan := plans[0]
		analyticalCost := plan.PayAmount
		if plan.CompanyCostAmount != nil {
			analyticalCost = *plan.CompanyCostAmount
		}
		result.CompanyAnalyticalCost = analyticalCost

		if result.TotalHours == 0 {
			w := "Nema evidentiranih sati"
			result.Warning = &w
			return result
		}

		effectiveCPH := analyticalCost / result.TotalHours
		result.EffectiveCostPerHour = &effectiveCPH

		for _, ph := range projectHours {
			pct := ph.Hours / result.TotalHours * 100
			result.ProjectAllocations = append(result.ProjectAllocations, dto.ProjectLaborAllocation{
				ProjectID:   ph.ProjectID,
				ProjectName: ph.ProjectName,
				Hours:       ph.Hours,
				Percentage:  pct,
				LaborCost:   ph.Hours * effectiveCPH,
			})
		}

	} else { // hourly
		type bucket struct {
			name  string
			hours float64
			cost  float64
		}
		buckets := map[string]*bucket{}
		totalCost := 0.0
		uncoveredHours := 0.0

		for _, rec := range dailyHours {
			if _, ok := buckets[rec.ProjectID]; !ok {
				buckets[rec.ProjectID] = &bucket{name: rec.ProjectName}
			}
			b := buckets[rec.ProjectID]
			b.hours += rec.Hours

			plan := findPlanForDate(plans, rec.WorkDate)
			if plan == nil {
				uncoveredHours += rec.Hours
			} else {
				rate := plan.PayAmount
				if plan.CompanyCostAmount != nil {
					rate = *plan.CompanyCostAmount
				}
				cost := rec.Hours * rate
				b.cost += cost
				totalCost += cost
			}
		}

		result.CompanyAnalyticalCost = totalCost

		if uncoveredHours > 0 {
			w := fmt.Sprintf("%.1f h bez definirane plaće u odabranom periodu", uncoveredHours)
			result.Warning = &w
		}

		if result.TotalHours > 0 && totalCost > 0 {
			avg := totalCost / result.TotalHours
			result.EffectiveCostPerHour = &avg
		}

		// Build project allocations from buckets
		for projID, b := range buckets {
			pct := 0.0
			if result.TotalHours > 0 {
				pct = b.hours / result.TotalHours * 100
			}
			result.ProjectAllocations = append(result.ProjectAllocations, dto.ProjectLaborAllocation{
				ProjectID:   projID,
				ProjectName: b.name,
				Hours:       b.hours,
				Percentage:  pct,
				LaborCost:   b.cost,
			})
		}
		sort.Slice(result.ProjectAllocations, func(i, j int) bool {
			return result.ProjectAllocations[i].Hours > result.ProjectAllocations[j].Hours
		})
	}

	return result
}

// findPlanForDate returns the plan effective on a given date, or nil.
func findPlanForDate(plans []repositories.CompensationPlanRow, date time.Time) *repositories.CompensationPlanRow {
	d := date.Truncate(24 * time.Hour)
	for i := range plans {
		p := &plans[i]
		from := p.EffectiveFrom.Truncate(24 * time.Hour)
		if d.Before(from) {
			continue
		}
		if p.EffectiveTo == nil {
			return p
		}
		eto := p.EffectiveTo.Truncate(24 * time.Hour)
		if !d.After(eto) {
			return p
		}
	}
	return nil
}
