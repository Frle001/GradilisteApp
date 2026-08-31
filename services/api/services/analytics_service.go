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
}

// ── Service ───────────────────────────────────────────────────────────────────

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
	return s.computeForEmployee(ctx, companyID, *emp, from, to)
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

	summary := &dto.MonthlyLaborSummary{
		Year:      year,
		Month:     month,
		Employees: make([]dto.EmployeeLaborCost, 0, len(employees)),
	}

	for _, emp := range employees {
		cost, err := s.computeForEmployee(ctx, companyID, emp, from, to)
		if err != nil {
			return nil, err
		}
		summary.Employees = append(summary.Employees, *cost)
		summary.TotalHours += cost.TotalHours
		summary.TotalKnownCost += cost.CompanyAnalyticalCost
		if !cost.HasCompensation && cost.TotalHours > 0 {
			summary.EmployeesWithoutCompensation++
		}
	}

	if summary.TotalHours > 0 && summary.TotalKnownCost > 0 {
		avg := summary.TotalKnownCost / summary.TotalHours
		summary.AvgCostPerHour = &avg
	}

	return summary, nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (s *AnalyticsService) computeForEmployee(ctx context.Context, companyID string, emp repositories.AnalyticsEmployeeRow, from, to time.Time) (*dto.EmployeeLaborCost, error) {
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
	return &cost, nil
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
