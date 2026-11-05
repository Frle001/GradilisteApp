package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// â”€â”€ Repo interface â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type coldOpsRepoIface interface {
	ListCategories(ctx context.Context, companyID string) ([]repositories.ColdOpsCategoryRow, error)
	CreateCategory(ctx context.Context, p repositories.CreateCategoryParams) (*repositories.ColdOpsCategoryRow, error)
	GetCategory(ctx context.Context, companyID, id string) (*repositories.ColdOpsCategoryRow, error)
	UpdateCategory(ctx context.Context, p repositories.UpdateCategoryParams) (*repositories.ColdOpsCategoryRow, error)
	DeleteCategory(ctx context.Context, companyID, id string) error
	CountExpensesForCategory(ctx context.Context, companyID, categoryID string) (int, error)

	ListExpenses(ctx context.Context, companyID string, categoryID, expenseType *string, from, to *time.Time, offset, limit int) ([]repositories.ColdOpsExpenseRow, error)
	CreateExpense(ctx context.Context, p repositories.CreateExpenseParams) (*repositories.ColdOpsExpenseRow, error)
	GetExpense(ctx context.Context, companyID, id string) (*repositories.ColdOpsExpenseRow, error)
	UpdateExpense(ctx context.Context, p repositories.UpdateExpenseParams) (*repositories.ColdOpsExpenseRow, error)
	DeleteExpense(ctx context.Context, companyID, id string) error

	ListRecurringDefs(ctx context.Context, companyID string, activeOnly bool) ([]repositories.RecurringDefRow, error)
	CreateRecurringDef(ctx context.Context, p repositories.CreateRecurringDefParams) (*repositories.RecurringDefRow, error)
	GetRecurringDef(ctx context.Context, companyID, id string) (*repositories.RecurringDefRow, error)
	UpdateRecurringDef(ctx context.Context, p repositories.UpdateRecurringDefParams) (*repositories.RecurringDefRow, error)
	CreateRecurringAmount(ctx context.Context, p repositories.CreateRecurringAmountParams) error
	GetAmountForDate(ctx context.Context, companyID, defID string, d time.Time) (*float64, error)

	BulkUpsertOccurrences(ctx context.Context, params []repositories.UpsertOccurrenceParams) error
	ListOccurrences(ctx context.Context, companyID, defID string, from, to *time.Time, status *string) ([]repositories.ColdOpsOccurrenceRow, error)
	ListUpcomingOccurrences(ctx context.Context, companyID string, horizonDays int) ([]repositories.ColdOpsOccurrenceRow, error)
	GetOccurrence(ctx context.Context, companyID, id string) (*repositories.ColdOpsOccurrenceRow, error)
	UpdateOccurrence(ctx context.Context, p repositories.UpdateOccurrenceParams) (*repositories.ColdOpsOccurrenceRow, error)

	ListBudgets(ctx context.Context, companyID string, year, month int) ([]repositories.ColdOpsBudgetRow, error)
	UpsertBudget(ctx context.Context, p repositories.UpsertBudgetParams) (*repositories.ColdOpsBudgetRow, error)
	DeleteBudget(ctx context.Context, companyID, id string) error

	GetPeriodTotal(ctx context.Context, companyID string, from, to time.Time) (float64, int, error)
	GetMonthlyTrend(ctx context.Context, companyID string, months int) ([]repositories.TrendPointRow, error)
	GetCategoryDistribution(ctx context.Context, companyID string, from, to time.Time) ([]repositories.CategoryStatsRow, error)
	GetFixedVsVariable(ctx context.Context, companyID string, from, to time.Time) (repositories.FixedVariableRow, error)
	GetPlannedOccurrenceTotals(ctx context.Context, companyID string, from, to time.Time) ([]repositories.TrendPointRow, error)
}

// â”€â”€ Service â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type ColdOpsService struct {
	repo coldOpsRepoIface
}

func NewColdOpsService(repo *repositories.ColdOpsRepository) *ColdOpsService {
	return &ColdOpsService{repo: repo}
}

// â”€â”€ Date helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// clampDayOfMonth returns day clamped to the last valid day of the given month.
func clampDayOfMonth(year, month, day int) int {
	last := daysInMonth(year, month)
	if day > last {
		return last
	}
	return day
}

func daysInMonth(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func occurrenceKey(frequency string, year, month int) string {
	if frequency == "yearly" {
		return fmt.Sprintf("%04d", year)
	}
	return fmt.Sprintf("%04d-%02d", year, month)
}

// â”€â”€ Occurrence generation â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// generateOccurrences lazily inserts planned occurrences for all active
// definitions up to horizonMonths in the future. Idempotent: ON CONFLICT DO NOTHING.
func (s *ColdOpsService) generateOccurrences(ctx context.Context, companyID string, horizonMonths int) error {
	defs, err := s.repo.ListRecurringDefs(ctx, companyID, true)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	horizon := now.AddDate(0, horizonMonths, 0)

	var batch []repositories.UpsertOccurrenceParams
	for _, def := range defs {
		if def.Frequency == "monthly" {
			cur := def.StartDate
			for !cur.After(horizon) {
				if def.EndDate != nil && cur.After(*def.EndDate) {
					break
				}
				day := clampDayOfMonth(cur.Year(), int(cur.Month()), def.DayOfMonth)
				due := time.Date(cur.Year(), cur.Month(), day, 0, 0, 0, 0, time.UTC)
				key := occurrenceKey("monthly", cur.Year(), int(cur.Month()))
				batch = append(batch, repositories.UpsertOccurrenceParams{
					DefinitionID:  def.ID,
					CompanyID:     companyID,
					OccurrenceKey: key,
					DueDate:       due,
				})
				cur = cur.AddDate(0, 1, 0)
			}
		} else if def.Frequency == "yearly" && def.MonthOfYear != nil {
			// Iterate year by year; for each year compute the target date.
			maxYear := horizon.Year() + 1
			for y := def.StartDate.Year(); y <= maxYear; y++ {
				day := clampDayOfMonth(y, *def.MonthOfYear, def.DayOfMonth)
				due := time.Date(y, time.Month(*def.MonthOfYear), day, 0, 0, 0, 0, time.UTC)
				if due.Before(def.StartDate) {
					continue
				}
				if due.After(horizon) {
					break
				}
				if def.EndDate != nil && due.After(*def.EndDate) {
					break
				}
				key := occurrenceKey("yearly", y, 0)
				batch = append(batch, repositories.UpsertOccurrenceParams{
					DefinitionID:  def.ID,
					CompanyID:     companyID,
					OccurrenceKey: key,
					DueDate:       due,
				})
			}
		}
	}
	if len(batch) == 0 {
		return nil
	}
	return s.repo.BulkUpsertOccurrences(ctx, batch)
}

// â”€â”€ Categories â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *ColdOpsService) ListCategories(ctx context.Context, companyID, callerRole string) ([]dto.ColdOpsCategory, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListCategories(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("cold_ops.ListCategories: %w", err)
	}
	out := make([]dto.ColdOpsCategory, 0, len(rows))
	for _, r := range rows {
		out = append(out, categoryRowToDTO(r))
	}
	return out, nil
}

func (s *ColdOpsService) CreateCategory(ctx context.Context, companyID, callerID, callerRole string, req dto.CreateCategoryReq) (*dto.ColdOpsCategory, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, &ValidationError{Message: "name is required"}
	}
	row, err := s.repo.CreateCategory(ctx, repositories.CreateCategoryParams{
		CompanyID:   companyID,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		CreatedBy:   callerID,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.CreateCategory: %w", err)
	}
	v := categoryRowToDTO(*row)
	return &v, nil
}

func (s *ColdOpsService) UpdateCategory(ctx context.Context, companyID, callerRole, id string, req dto.UpdateCategoryReq) (*dto.ColdOpsCategory, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, &ValidationError{Message: "name is required"}
	}
	row, err := s.repo.UpdateCategory(ctx, repositories.UpdateCategoryParams{
		ID:          id,
		CompanyID:   companyID,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		IsActive:    req.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.UpdateCategory: %w", err)
	}
	if row == nil {
		return nil, ErrNotFound
	}
	v := categoryRowToDTO(*row)
	return &v, nil
}

func (s *ColdOpsService) DeleteCategory(ctx context.Context, companyID, callerRole, id string) error {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return err
	}
	cat, err := s.repo.GetCategory(ctx, companyID, id)
	if err != nil {
		return fmt.Errorf("cold_ops.DeleteCategory: %w", err)
	}
	if cat == nil {
		return ErrNotFound
	}
	count, err := s.repo.CountExpensesForCategory(ctx, companyID, id)
	if err != nil {
		return fmt.Errorf("cold_ops.DeleteCategory count: %w", err)
	}
	if count > 0 {
		// Soft-delete: deactivate so history is preserved
		_, err = s.repo.UpdateCategory(ctx, repositories.UpdateCategoryParams{
			ID:          id,
			CompanyID:   companyID,
			Name:        cat.Name,
			Description: cat.Description,
			Color:       cat.Color,
			IsActive:    false,
		})
		return err
	}
	return s.repo.DeleteCategory(ctx, companyID, id)
}

// â”€â”€ Expenses â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *ColdOpsService) ListExpenses(ctx context.Context, companyID, callerRole string, f dto.ListExpensesFilter) ([]dto.ColdOpsExpense, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	offset := 0
	limit := 50
	if f.PageSize > 0 {
		limit = f.PageSize
	}
	if f.Page > 1 {
		offset = (f.Page - 1) * limit
	}
	rows, err := s.repo.ListExpenses(ctx, companyID, f.CategoryID, f.ExpenseType, f.From, f.To, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("cold_ops.ListExpenses: %w", err)
	}
	out := make([]dto.ColdOpsExpense, 0, len(rows))
	for _, r := range rows {
		out = append(out, expenseRowToDTO(r))
	}
	return out, nil
}

func (s *ColdOpsService) CreateExpense(ctx context.Context, companyID, callerID, callerRole string, req dto.CreateExpenseReq) (*dto.ColdOpsExpense, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if err := validateExpenseReq(req.Title, req.Amount, req.ExpenseDate, req.ExpenseType, req.PaymentMethod); err != nil {
		return nil, err
	}
	expDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		return nil, &ValidationError{Message: "must be YYYY-MM-DD"}
	}
	row, err := s.repo.CreateExpense(ctx, repositories.CreateExpenseParams{
		CompanyID:     companyID,
		CategoryID:    req.CategoryID,
		Amount:        req.Amount,
		ExpenseDate:   expDate,
		Title:         req.Title,
		Notes:         req.Notes,
		Supplier:      req.Supplier,
		PaymentMethod: req.PaymentMethod,
		ExpenseType:   req.ExpenseType,
		CreatedBy:     callerID,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.CreateExpense: %w", err)
	}
	v := expenseRowToDTO(*row)
	return &v, nil
}

func (s *ColdOpsService) GetExpense(ctx context.Context, companyID, callerRole, id string) (*dto.ColdOpsExpense, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	row, err := s.repo.GetExpense(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("cold_ops.GetExpense: %w", err)
	}
	if row == nil {
		return nil, ErrNotFound
	}
	v := expenseRowToDTO(*row)
	return &v, nil
}

func (s *ColdOpsService) UpdateExpense(ctx context.Context, companyID, callerRole, id string, req dto.UpdateExpenseReq) (*dto.ColdOpsExpense, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if err := validateExpenseReq(req.Title, req.Amount, req.ExpenseDate, req.ExpenseType, req.PaymentMethod); err != nil {
		return nil, err
	}
	expDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		return nil, &ValidationError{Message: "must be YYYY-MM-DD"}
	}
	row, err := s.repo.UpdateExpense(ctx, repositories.UpdateExpenseParams{
		ID:            id,
		CompanyID:     companyID,
		CategoryID:    req.CategoryID,
		Amount:        req.Amount,
		ExpenseDate:   expDate,
		Title:         req.Title,
		Notes:         req.Notes,
		Supplier:      req.Supplier,
		PaymentMethod: req.PaymentMethod,
		ExpenseType:   req.ExpenseType,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.UpdateExpense: %w", err)
	}
	if row == nil {
		return nil, ErrNotFound
	}
	v := expenseRowToDTO(*row)
	return &v, nil
}

func (s *ColdOpsService) DeleteExpense(ctx context.Context, companyID, callerRole, id string) error {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return err
	}
	if err := s.repo.DeleteExpense(ctx, companyID, id); err != nil {
		return ErrNotFound
	}
	return nil
}

// â”€â”€ Recurring definitions â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *ColdOpsService) ListRecurringDefs(ctx context.Context, companyID, callerRole string, activeOnly bool) ([]dto.RecurringDef, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListRecurringDefs(ctx, companyID, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("cold_ops.ListRecurringDefs: %w", err)
	}
	out := make([]dto.RecurringDef, 0, len(rows))
	for _, r := range rows {
		out = append(out, recurringDefRowToDTO(r))
	}
	return out, nil
}

func (s *ColdOpsService) CreateRecurringDef(ctx context.Context, companyID, callerID, callerRole string, req dto.CreateRecurringReq) (*dto.RecurringDef, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if err := validateRecurringReq(req.Title, req.Frequency, req.DayOfMonth, req.MonthOfYear, req.Amount, req.StartDate); err != nil {
		return nil, err
	}
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, &ValidationError{Message: "must be YYYY-MM-DD"}
	}
	var endDate *time.Time
	if req.EndDate != nil {
		t, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, &ValidationError{Message: "must be YYYY-MM-DD"}
		}
		endDate = &t
	}
	row, err := s.repo.CreateRecurringDef(ctx, repositories.CreateRecurringDefParams{
		CompanyID:     companyID,
		CategoryID:    req.CategoryID,
		Title:         req.Title,
		Frequency:     req.Frequency,
		DayOfMonth:    req.DayOfMonth,
		MonthOfYear:   req.MonthOfYear,
		StartDate:     start,
		EndDate:       endDate,
		Supplier:      req.Supplier,
		PaymentMethod: req.PaymentMethod,
		Notes:         req.Notes,
		CreatedBy:     callerID,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.CreateRecurringDef: %w", err)
	}
	// Record initial amount
	if err := s.repo.CreateRecurringAmount(ctx, repositories.CreateRecurringAmountParams{
		DefinitionID:  row.ID,
		CompanyID:     companyID,
		Amount:        req.Amount,
		EffectiveFrom: start,
		CreatedBy:     callerID,
	}); err != nil {
		return nil, fmt.Errorf("cold_ops.CreateRecurringAmount: %w", err)
	}
	v := recurringDefRowToDTO(*row)
	v.CurrentAmount = &req.Amount
	return &v, nil
}

func (s *ColdOpsService) UpdateRecurringDef(ctx context.Context, companyID, callerID, callerRole, id string, req dto.UpdateRecurringReq) (*dto.RecurringDef, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if req.Title == "" {
		return nil, &ValidationError{Message: "title is required"}
	}
	var endDate *time.Time
	if req.EndDate != nil {
		t, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, &ValidationError{Message: "must be YYYY-MM-DD"}
		}
		endDate = &t
	}
	row, err := s.repo.UpdateRecurringDef(ctx, repositories.UpdateRecurringDefParams{
		ID:            id,
		CompanyID:     companyID,
		CategoryID:    req.CategoryID,
		Title:         req.Title,
		DayOfMonth:    req.DayOfMonth,
		MonthOfYear:   req.MonthOfYear,
		EndDate:       endDate,
		Supplier:      req.Supplier,
		PaymentMethod: req.PaymentMethod,
		Notes:         req.Notes,
		IsActive:      req.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.UpdateRecurringDef: %w", err)
	}
	if row == nil {
		return nil, ErrNotFound
	}
	// If a new amount is provided, record it effective today
	if req.NewAmount != nil {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if err := s.repo.CreateRecurringAmount(ctx, repositories.CreateRecurringAmountParams{
			DefinitionID:  id,
			CompanyID:     companyID,
			Amount:        *req.NewAmount,
			EffectiveFrom: today,
			CreatedBy:     callerID,
		}); err != nil {
			return nil, fmt.Errorf("cold_ops.UpdateRecurringDef amount: %w", err)
		}
	}
	v := recurringDefRowToDTO(*row)
	if req.NewAmount != nil {
		v.CurrentAmount = req.NewAmount
	}
	return &v, nil
}

// â”€â”€ Occurrences â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *ColdOpsService) ListOccurrences(ctx context.Context, companyID, callerRole, defID string, from, to *time.Time, status *string) ([]dto.ColdOpsOccurrence, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	// Generate up to 3 months ahead before listing
	_ = s.generateOccurrences(ctx, companyID, 3)

	rows, err := s.repo.ListOccurrences(ctx, companyID, defID, from, to, status)
	if err != nil {
		return nil, fmt.Errorf("cold_ops.ListOccurrences: %w", err)
	}
	out := make([]dto.ColdOpsOccurrence, 0, len(rows))
	for _, r := range rows {
		out = append(out, occurrenceRowToDTO(r))
	}
	return out, nil
}

func (s *ColdOpsService) ConfirmOccurrence(ctx context.Context, companyID, callerID, callerRole, id string, req dto.ConfirmOccurrenceReq) (*dto.ColdOpsOccurrence, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, &ValidationError{Message: "amount must be > 0"}
	}
	now := time.Now().UTC()
	row, err := s.repo.UpdateOccurrence(ctx, repositories.UpdateOccurrenceParams{
		ID:        id,
		CompanyID: companyID,
		Status:    "confirmed",
		Amount:    &req.Amount,
		Notes:     req.Notes,
		HandledBy: &callerID,
		HandledAt: &now,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.ConfirmOccurrence: %w", err)
	}
	if row == nil {
		return nil, ErrNotFound
	}
	v := occurrenceRowToDTO(*row)
	return &v, nil
}

func (s *ColdOpsService) SkipOccurrence(ctx context.Context, companyID, callerID, callerRole, id string, req dto.SkipOccurrenceReq) (*dto.ColdOpsOccurrence, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row, err := s.repo.UpdateOccurrence(ctx, repositories.UpdateOccurrenceParams{
		ID:        id,
		CompanyID: companyID,
		Status:    "skipped",
		Notes:     req.Notes,
		HandledBy: &callerID,
		HandledAt: &now,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.SkipOccurrence: %w", err)
	}
	if row == nil {
		return nil, ErrNotFound
	}
	v := occurrenceRowToDTO(*row)
	return &v, nil
}

// â”€â”€ Budgets â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *ColdOpsService) ListBudgets(ctx context.Context, companyID, callerRole string, year, month int) ([]dto.ColdOpsBudget, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListBudgets(ctx, companyID, year, month)
	if err != nil {
		return nil, fmt.Errorf("cold_ops.ListBudgets: %w", err)
	}
	// Get actuals for the month to populate utilization
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1)
	catDist, err := s.repo.GetCategoryDistribution(ctx, companyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("cold_ops.ListBudgets dist: %w", err)
	}
	actuals := make(map[string]float64)
	for _, c := range catDist {
		if c.CategoryID != nil {
			actuals[*c.CategoryID] = c.Total
		}
	}
	out := make([]dto.ColdOpsBudget, 0, len(rows))
	for _, r := range rows {
		actual := 0.0
		if r.CategoryID != nil {
			actual = actuals[*r.CategoryID]
		}
		util := 0.0
		if r.BudgetAmount > 0 {
			util = actual / r.BudgetAmount * 100
		}
		out = append(out, dto.ColdOpsBudget{
			ID:           r.ID,
			CategoryID:   r.CategoryID,
			CategoryName: r.CategoryName,
			Year:         r.Year,
			Month:        r.Month,
			BudgetAmount: r.BudgetAmount,
			ActualAmount: actual,
			Utilization:  util,
		})
	}
	return out, nil
}

func (s *ColdOpsService) UpsertBudget(ctx context.Context, companyID, callerID, callerRole string, req dto.UpsertBudgetReq) (*dto.ColdOpsBudget, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	if req.BudgetAmount <= 0 {
		return nil, &ValidationError{Message: "must be > 0"}
	}
	if req.Month < 1 || req.Month > 12 {
		return nil, &ValidationError{Message: "must be 1-12"}
	}
	row, err := s.repo.UpsertBudget(ctx, repositories.UpsertBudgetParams{
		CompanyID:    companyID,
		CategoryID:   req.CategoryID,
		Year:         req.Year,
		Month:        req.Month,
		BudgetAmount: req.BudgetAmount,
		CreatedBy:    callerID,
	})
	if err != nil {
		return nil, fmt.Errorf("cold_ops.UpsertBudget: %w", err)
	}
	return &dto.ColdOpsBudget{
		ID:           row.ID,
		CategoryID:   row.CategoryID,
		CategoryName: row.CategoryName,
		Year:         row.Year,
		Month:        row.Month,
		BudgetAmount: row.BudgetAmount,
	}, nil
}

func (s *ColdOpsService) DeleteBudget(ctx context.Context, companyID, callerRole, id string) error {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return err
	}
	if err := s.repo.DeleteBudget(ctx, companyID, id); err != nil {
		return ErrNotFound
	}
	return nil
}

// â”€â”€ Analytics summary â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *ColdOpsService) GetSummary(ctx context.Context, companyID, callerRole string) (*dto.ColdOpsSummary, error) {
	if err := requireAnalyticsAccess(callerRole); err != nil {
		return nil, err
	}
	// Generate occurrences up to 3 months ahead
	_ = s.generateOccurrences(ctx, companyID, 3)

	now := time.Now().UTC()
	curMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	curMonthEnd := curMonthStart.AddDate(0, 1, -1)
	prevMonthStart := curMonthStart.AddDate(0, -1, 0)
	prevMonthEnd := curMonthStart.AddDate(0, 0, -1)
	curYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	curYearEnd := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
	prevYearStart := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, time.UTC)
	prevYearEnd := time.Date(now.Year()-1, 12, 31, 0, 0, 0, 0, time.UTC)

	curMonTotal, curMonCount, err := s.repo.GetPeriodTotal(ctx, companyID, curMonthStart, curMonthEnd)
	if err != nil {
		return nil, err
	}
	prevMonTotal, prevMonCount, err := s.repo.GetPeriodTotal(ctx, companyID, prevMonthStart, prevMonthEnd)
	if err != nil {
		return nil, err
	}
	curYearTotal, curYearCount, err := s.repo.GetPeriodTotal(ctx, companyID, curYearStart, curYearEnd)
	if err != nil {
		return nil, err
	}
	prevYearTotal, prevYearCount, err := s.repo.GetPeriodTotal(ctx, companyID, prevYearStart, prevYearEnd)
	if err != nil {
		return nil, err
	}
	trend, err := s.repo.GetMonthlyTrend(ctx, companyID, 12)
	if err != nil {
		return nil, err
	}
	catDist, err := s.repo.GetCategoryDistribution(ctx, companyID, curYearStart, curYearEnd)
	if err != nil {
		return nil, err
	}
	fv, err := s.repo.GetFixedVsVariable(ctx, companyID, curYearStart, curYearEnd)
	if err != nil {
		return nil, err
	}
	upcoming, err := s.repo.ListUpcomingOccurrences(ctx, companyID, 30)
	if err != nil {
		return nil, err
	}
	forecastFrom := curMonthStart
	forecastTo := curMonthStart.AddDate(0, 3, -1)
	planned, err := s.repo.GetPlannedOccurrenceTotals(ctx, companyID, forecastFrom, forecastTo)
	if err != nil {
		return nil, err
	}
	budgets, err := s.repo.ListBudgets(ctx, companyID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, err
	}

	curMonPct := pctChange(prevMonTotal, curMonTotal)
	curYearPct := pctChange(prevYearTotal, curYearTotal)

	trendPoints := make([]dto.ColdOpsTrendPoint, 0, len(trend))
	for _, t := range trend {
		trendPoints = append(trendPoints, dto.ColdOpsTrendPoint{
			Year: t.Year, Month: t.Month, Total: t.Total, Count: t.Count,
		})
	}

	distTotal := 0.0
	for _, c := range catDist {
		distTotal += c.Total
	}
	catStats := make([]dto.ColdOpsCategoryStats, 0, len(catDist))
	for _, c := range catDist {
		name := "Bez kategorije"
		if c.CategoryName != nil {
			name = *c.CategoryName
		}
		pct := 0.0
		if distTotal > 0 {
			pct = c.Total / distTotal * 100
		}
		catStats = append(catStats, dto.ColdOpsCategoryStats{
			CategoryID:   c.CategoryID,
			CategoryName: name,
			Color:        c.Color,
			Total:        c.Total,
			Count:        c.Count,
			Pct:          pct,
		})
	}

	// Build forecast: past months = confirmed, future = confirmed + planned
	forecastMap := make(map[string]float64)
	for _, p := range planned {
		forecastMap[fmt.Sprintf("%04d-%02d", p.Year, p.Month)] = p.Total
	}
	confirmedMap := make(map[string]float64)
	for _, t := range trend {
		confirmedMap[fmt.Sprintf("%04d-%02d", t.Year, t.Month)] = t.Total
	}
	var forecast []dto.ColdOpsForecastPoint
	cur := curMonthStart
	for !cur.After(forecastTo) {
		k := fmt.Sprintf("%04d-%02d", cur.Year(), int(cur.Month()))
		confirmed := confirmedMap[k]
		planned := forecastMap[k]
		isProj := cur.After(now)
		forecast = append(forecast, dto.ColdOpsForecastPoint{
			Year:            cur.Year(),
			Month:           int(cur.Month()),
			ConfirmedAmount: confirmed,
			ForecastAmount:  confirmed + planned,
			IsProjection:    isProj,
		})
		cur = cur.AddDate(0, 1, 0)
	}

	// Budget utilization
	budgetUtil := make([]dto.ColdOpsBudgetUtil, 0, len(budgets))
	actualByCat := make(map[string]float64)
	for _, c := range catDist {
		if c.CategoryID != nil {
			actualByCat[*c.CategoryID] = c.Total
		}
	}
	for _, b := range budgets {
		actual := 0.0
		if b.CategoryID != nil {
			actual = actualByCat[*b.CategoryID]
		}
		util := 0.0
		if b.BudgetAmount > 0 {
			util = actual / b.BudgetAmount * 100
		}
		name := "Ukupno"
		if b.CategoryName != nil {
			name = *b.CategoryName
		}
		budgetUtil = append(budgetUtil, dto.ColdOpsBudgetUtil{
			CategoryID:   b.CategoryID,
			CategoryName: name,
			BudgetAmount: b.BudgetAmount,
			ActualAmount: actual,
			Utilization:  util,
			IsOverBudget: actual > b.BudgetAmount,
		})
	}

	// Annualized commitment = sum of active monthly recurring amounts * 12
	// + yearly recurring amounts
	defs, err := s.repo.ListRecurringDefs(ctx, companyID, true)
	if err != nil {
		return nil, err
	}
	annualized := 0.0
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for _, d := range defs {
		amt, err := s.repo.GetAmountForDate(ctx, companyID, d.ID, today)
		if err != nil || amt == nil {
			continue
		}
		if d.Frequency == "monthly" {
			annualized += *amt * 12
		} else {
			annualized += *amt
		}
	}

	upcomingDTOs := make([]dto.ColdOpsOccurrence, 0, len(upcoming))
	for _, o := range upcoming {
		upcomingDTOs = append(upcomingDTOs, occurrenceRowToDTO(o))
	}

	return &dto.ColdOpsSummary{
		CurrentMonth:         dto.ColdOpsPeriodStats{Total: curMonTotal, Count: curMonCount, PctChange: curMonPct},
		PrevMonth:            dto.ColdOpsPeriodStats{Total: prevMonTotal, Count: prevMonCount},
		CurrentYear:          dto.ColdOpsPeriodStats{Total: curYearTotal, Count: curYearCount, PctChange: curYearPct},
		PrevYear:             dto.ColdOpsPeriodStats{Total: prevYearTotal, Count: prevYearCount},
		MonthlyTrend:         trendPoints,
		CategoryDistribution: catStats,
		FixedVsVariable:      dto.ColdOpsFixedVariable{FixedTotal: fv.FixedTotal, VariableTotal: fv.VariableTotal, FixedCount: fv.FixedCount, VariableCount: fv.VariableCount},
		UpcomingOccurrences:  upcomingDTOs,
		Forecast:             forecast,
		BudgetUtilization:    budgetUtil,
		AnnualizedCommitment: annualized,
	}, nil
}

// â”€â”€ Conversion helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func categoryRowToDTO(r repositories.ColdOpsCategoryRow) dto.ColdOpsCategory {
	return dto.ColdOpsCategory{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Color:       r.Color,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
	}
}

func expenseRowToDTO(r repositories.ColdOpsExpenseRow) dto.ColdOpsExpense {
	return dto.ColdOpsExpense{
		ID:            r.ID,
		CategoryID:    r.CategoryID,
		CategoryName:  r.CategoryName,
		Amount:        r.Amount,
		ExpenseDate:   r.ExpenseDate.Format("2006-01-02"),
		Title:         r.Title,
		Notes:         r.Notes,
		Supplier:      r.Supplier,
		PaymentMethod: r.PaymentMethod,
		ExpenseType:   r.ExpenseType,
		CreatedBy:     r.CreatedBy,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
	}
}

func recurringDefRowToDTO(r repositories.RecurringDefRow) dto.RecurringDef {
	d := dto.RecurringDef{
		ID:            r.ID,
		CategoryID:    r.CategoryID,
		CategoryName:  r.CategoryName,
		Title:         r.Title,
		Frequency:     r.Frequency,
		DayOfMonth:    r.DayOfMonth,
		MonthOfYear:   r.MonthOfYear,
		StartDate:     r.StartDate.Format("2006-01-02"),
		Supplier:      r.Supplier,
		PaymentMethod: r.PaymentMethod,
		Notes:         r.Notes,
		IsActive:      r.IsActive,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
	}
	if r.EndDate != nil {
		s := r.EndDate.Format("2006-01-02")
		d.EndDate = &s
	}
	return d
}

func occurrenceRowToDTO(r repositories.ColdOpsOccurrenceRow) dto.ColdOpsOccurrence {
	o := dto.ColdOpsOccurrence{
		ID:              r.ID,
		DefinitionID:    r.DefinitionID,
		DefinitionTitle: r.DefinitionTitle,
		OccurrenceKey:   r.OccurrenceKey,
		DueDate:         r.DueDate.Format("2006-01-02"),
		Status:          r.Status,
		Amount:          r.Amount,
		ExpenseID:       r.ExpenseID,
		Notes:           r.Notes,
		HandledBy:       r.HandledBy,
	}
	if r.HandledAt != nil {
		s := r.HandledAt.Format(time.RFC3339)
		o.HandledAt = &s
	}
	return o
}

// â”€â”€ Validation helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

var validPaymentMethods = map[string]bool{"cash": true, "card": true, "transfer": true, "other": true}
var validExpenseTypes = map[string]bool{"fixed": true, "variable": true}

func validateExpenseReq(title string, amount float64, expenseDate, expenseType string, paymentMethod *string) error {
	if title == "" {
		return &ValidationError{Message: "title is required"}
	}
	if amount <= 0 {
		return &ValidationError{Message: "must be > 0"}
	}
	if expenseDate == "" {
		return &ValidationError{Message: "required"}
	}
	if !validExpenseTypes[expenseType] {
		return &ValidationError{Message: "must be fixed or variable"}
	}
	if paymentMethod != nil && !validPaymentMethods[*paymentMethod] {
		return &ValidationError{Message: "must be cash, card, transfer or other"}
	}
	return nil
}

func validateRecurringReq(title, frequency string, dayOfMonth int, monthOfYear *int, amount float64, startDate string) error {
	if title == "" {
		return &ValidationError{Message: "required"}
	}
	if frequency != "monthly" && frequency != "yearly" {
		return &ValidationError{Message: "must be monthly or yearly"}
	}
	if dayOfMonth < 1 || dayOfMonth > 31 {
		return &ValidationError{Message: "must be 1-31"}
	}
	if frequency == "yearly" && (monthOfYear == nil || *monthOfYear < 1 || *monthOfYear > 12) {
		return &ValidationError{Message: "required for yearly frequency"}
	}
	if amount <= 0 {
		return &ValidationError{Message: "must be > 0"}
	}
	if startDate == "" {
		return &ValidationError{Message: "required"}
	}
	return nil
}

func pctChange(prev, cur float64) *float64 {
	if prev == 0 {
		return nil
	}
	v := (cur - prev) / prev * 100
	return &v
}
