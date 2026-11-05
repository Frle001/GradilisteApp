package services

import (
	"context"
	"testing"
	"time"

	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
)

// ── Minimal in-memory stub ────────────────────────────────────────────────────

type mockColdOpsRepo struct {
	categories  []repositories.ColdOpsCategoryRow
	expenses    []repositories.ColdOpsExpenseRow
	recurDefs   []repositories.RecurringDefRow
	occurrences []repositories.ColdOpsOccurrenceRow
	budgets     []repositories.ColdOpsBudgetRow
	amounts     []repositories.RecurringAmountRow
}

func (m *mockColdOpsRepo) ListCategories(_ context.Context, _ string) ([]repositories.ColdOpsCategoryRow, error) {
	return m.categories, nil
}
func (m *mockColdOpsRepo) CreateCategory(_ context.Context, p repositories.CreateCategoryParams) (*repositories.ColdOpsCategoryRow, error) {
	r := repositories.ColdOpsCategoryRow{
		ID: "cat-1", CompanyID: p.CompanyID, Name: p.Name,
		Description: p.Description, Color: p.Color, IsActive: true,
		CreatedBy: p.CreatedBy, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.categories = append(m.categories, r)
	return &r, nil
}
func (m *mockColdOpsRepo) GetCategory(_ context.Context, _, id string) (*repositories.ColdOpsCategoryRow, error) {
	for _, c := range m.categories {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) UpdateCategory(_ context.Context, p repositories.UpdateCategoryParams) (*repositories.ColdOpsCategoryRow, error) {
	for i, c := range m.categories {
		if c.ID == p.ID {
			m.categories[i].Name = p.Name
			m.categories[i].IsActive = p.IsActive
			return &m.categories[i], nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) DeleteCategory(_ context.Context, _, id string) error {
	for i, c := range m.categories {
		if c.ID == id {
			m.categories = append(m.categories[:i], m.categories[i+1:]...)
			return nil
		}
	}
	return nil
}
func (m *mockColdOpsRepo) CountExpensesForCategory(_ context.Context, _, categoryID string) (int, error) {
	count := 0
	for _, e := range m.expenses {
		if e.CategoryID != nil && *e.CategoryID == categoryID {
			count++
		}
	}
	return count, nil
}
func (m *mockColdOpsRepo) ListExpenses(_ context.Context, companyID string, categoryID, expenseType *string, from, to *time.Time, offset, limit int) ([]repositories.ColdOpsExpenseRow, error) {
	return m.expenses, nil
}
func (m *mockColdOpsRepo) CreateExpense(_ context.Context, p repositories.CreateExpenseParams) (*repositories.ColdOpsExpenseRow, error) {
	r := repositories.ColdOpsExpenseRow{
		ID: "exp-1", CompanyID: p.CompanyID, CategoryID: p.CategoryID,
		Amount: p.Amount, ExpenseDate: p.ExpenseDate, Title: p.Title,
		ExpenseType: p.ExpenseType, CreatedBy: p.CreatedBy, CreatedAt: time.Now(),
	}
	m.expenses = append(m.expenses, r)
	return &r, nil
}
func (m *mockColdOpsRepo) GetExpense(_ context.Context, _, id string) (*repositories.ColdOpsExpenseRow, error) {
	for _, e := range m.expenses {
		if e.ID == id {
			return &e, nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) UpdateExpense(_ context.Context, p repositories.UpdateExpenseParams) (*repositories.ColdOpsExpenseRow, error) {
	for i, e := range m.expenses {
		if e.ID == p.ID {
			m.expenses[i].Amount = p.Amount
			return &m.expenses[i], nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) DeleteExpense(_ context.Context, _, id string) error {
	for i, e := range m.expenses {
		if e.ID == id {
			m.expenses = append(m.expenses[:i], m.expenses[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
func (m *mockColdOpsRepo) ListRecurringDefs(_ context.Context, _ string, activeOnly bool) ([]repositories.RecurringDefRow, error) {
	if !activeOnly {
		return m.recurDefs, nil
	}
	var out []repositories.RecurringDefRow
	for _, d := range m.recurDefs {
		if d.IsActive {
			out = append(out, d)
		}
	}
	return out, nil
}
func (m *mockColdOpsRepo) CreateRecurringDef(_ context.Context, p repositories.CreateRecurringDefParams) (*repositories.RecurringDefRow, error) {
	r := repositories.RecurringDefRow{
		ID: "def-1", CompanyID: p.CompanyID, Title: p.Title,
		Frequency: p.Frequency, DayOfMonth: p.DayOfMonth, MonthOfYear: p.MonthOfYear,
		StartDate: p.StartDate, EndDate: p.EndDate, IsActive: true,
		CreatedBy: p.CreatedBy, CreatedAt: time.Now(),
	}
	m.recurDefs = append(m.recurDefs, r)
	return &r, nil
}
func (m *mockColdOpsRepo) GetRecurringDef(_ context.Context, _, id string) (*repositories.RecurringDefRow, error) {
	for _, d := range m.recurDefs {
		if d.ID == id {
			return &d, nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) UpdateRecurringDef(_ context.Context, p repositories.UpdateRecurringDefParams) (*repositories.RecurringDefRow, error) {
	for i, d := range m.recurDefs {
		if d.ID == p.ID {
			m.recurDefs[i].Title = p.Title
			m.recurDefs[i].IsActive = p.IsActive
			return &m.recurDefs[i], nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) CreateRecurringAmount(_ context.Context, p repositories.CreateRecurringAmountParams) error {
	m.amounts = append(m.amounts, repositories.RecurringAmountRow{
		ID: "amt-1", DefinitionID: p.DefinitionID, Amount: p.Amount, EffectiveFrom: p.EffectiveFrom,
	})
	return nil
}
func (m *mockColdOpsRepo) GetAmountForDate(_ context.Context, _, defID string, d time.Time) (*float64, error) {
	var best *repositories.RecurringAmountRow
	for i, a := range m.amounts {
		if a.DefinitionID != defID {
			continue
		}
		if !a.EffectiveFrom.After(d) && (a.EffectiveTo == nil || !a.EffectiveTo.Before(d)) {
			if best == nil || a.EffectiveFrom.After(best.EffectiveFrom) {
				best = &m.amounts[i]
			}
		}
	}
	if best == nil {
		return nil, nil
	}
	return &best.Amount, nil
}
func (m *mockColdOpsRepo) BulkUpsertOccurrences(_ context.Context, params []repositories.UpsertOccurrenceParams) error {
	for _, p := range params {
		found := false
		for _, o := range m.occurrences {
			if o.DefinitionID == p.DefinitionID && o.OccurrenceKey == p.OccurrenceKey {
				found = true
				break
			}
		}
		if !found {
			m.occurrences = append(m.occurrences, repositories.ColdOpsOccurrenceRow{
				ID:            "occ-" + p.OccurrenceKey,
				DefinitionID:  p.DefinitionID,
				CompanyID:     p.CompanyID,
				OccurrenceKey: p.OccurrenceKey,
				DueDate:       p.DueDate,
				Status:        "planned",
				CreatedAt:     time.Now(),
			})
		}
	}
	return nil
}
func (m *mockColdOpsRepo) ListOccurrences(_ context.Context, _, defID string, _, _ *time.Time, status *string) ([]repositories.ColdOpsOccurrenceRow, error) {
	var out []repositories.ColdOpsOccurrenceRow
	for _, o := range m.occurrences {
		if o.DefinitionID != defID {
			continue
		}
		if status != nil && o.Status != *status {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}
func (m *mockColdOpsRepo) ListUpcomingOccurrences(_ context.Context, _ string, _ int) ([]repositories.ColdOpsOccurrenceRow, error) {
	return nil, nil
}
func (m *mockColdOpsRepo) GetOccurrence(_ context.Context, _, id string) (*repositories.ColdOpsOccurrenceRow, error) {
	for _, o := range m.occurrences {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) UpdateOccurrence(_ context.Context, p repositories.UpdateOccurrenceParams) (*repositories.ColdOpsOccurrenceRow, error) {
	for i, o := range m.occurrences {
		if o.ID == p.ID {
			m.occurrences[i].Status = p.Status
			m.occurrences[i].Amount = p.Amount
			m.occurrences[i].Notes = p.Notes
			return &m.occurrences[i], nil
		}
	}
	return nil, nil
}
func (m *mockColdOpsRepo) ListBudgets(_ context.Context, _ string, _, _ int) ([]repositories.ColdOpsBudgetRow, error) {
	return m.budgets, nil
}
func (m *mockColdOpsRepo) UpsertBudget(_ context.Context, p repositories.UpsertBudgetParams) (*repositories.ColdOpsBudgetRow, error) {
	r := repositories.ColdOpsBudgetRow{
		ID: "bud-1", CompanyID: p.CompanyID, CategoryID: p.CategoryID,
		Year: p.Year, Month: p.Month, BudgetAmount: p.BudgetAmount, CreatedBy: p.CreatedBy,
	}
	m.budgets = append(m.budgets, r)
	return &r, nil
}
func (m *mockColdOpsRepo) DeleteBudget(_ context.Context, _, id string) error {
	for i, b := range m.budgets {
		if b.ID == id {
			m.budgets = append(m.budgets[:i], m.budgets[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
func (m *mockColdOpsRepo) GetPeriodTotal(_ context.Context, _ string, _, _ time.Time) (float64, int, error) {
	return 0, 0, nil
}
func (m *mockColdOpsRepo) GetMonthlyTrend(_ context.Context, _ string, _ int) ([]repositories.TrendPointRow, error) {
	return nil, nil
}
func (m *mockColdOpsRepo) GetCategoryDistribution(_ context.Context, _ string, _, _ time.Time) ([]repositories.CategoryStatsRow, error) {
	return nil, nil
}
func (m *mockColdOpsRepo) GetFixedVsVariable(_ context.Context, _ string, _, _ time.Time) (repositories.FixedVariableRow, error) {
	return repositories.FixedVariableRow{}, nil
}
func (m *mockColdOpsRepo) GetPlannedOccurrenceTotals(_ context.Context, _ string, _, _ time.Time) ([]repositories.TrendPointRow, error) {
	return nil, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newSvc() *ColdOpsService {
	return &ColdOpsService{repo: &mockColdOpsRepo{}}
}

func newSvcWithRepo(r *mockColdOpsRepo) *ColdOpsService {
	return &ColdOpsService{repo: r}
}

// ── Authorization tests ───────────────────────────────────────────────────────

func TestColdOps_AuthForbidden(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	forbiddenRoles := []string{"radnik", "poslovoda", "administracija"}
	for _, role := range forbiddenRoles {
		_, err := svc.ListCategories(ctx, "co-1", role)
		if err != ErrForbidden {
			t.Errorf("ListCategories with role %q: want ErrForbidden, got %v", role, err)
		}
		_, err = svc.CreateCategory(ctx, "co-1", "u-1", role, dto.CreateCategoryReq{Name: "X"})
		if err != ErrForbidden {
			t.Errorf("CreateCategory with role %q: want ErrForbidden, got %v", role, err)
		}
		_, err = svc.ListExpenses(ctx, "co-1", role, dto.ListExpensesFilter{})
		if err != ErrForbidden {
			t.Errorf("ListExpenses with role %q: want ErrForbidden, got %v", role, err)
		}
	}
}

func TestColdOps_AuthAllowed(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	for _, role := range []string{"direktor", "inzenjer"} {
		_, err := svc.ListCategories(ctx, "co-1", role)
		if err != nil {
			t.Errorf("ListCategories with role %q: unexpected error %v", role, err)
		}
	}
}

// ── Category CRUD tests ───────────────────────────────────────────────────────

func TestColdOps_CreateCategory_Validation(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	_, err := svc.CreateCategory(ctx, "co-1", "u-1", "direktor", dto.CreateCategoryReq{Name: ""})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestColdOps_CreateCategory_OK(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	cat, err := svc.CreateCategory(ctx, "co-1", "u-1", "direktor", dto.CreateCategoryReq{Name: "Najam"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.Name != "Najam" {
		t.Errorf("want name=Najam, got %q", cat.Name)
	}
	if !cat.IsActive {
		t.Error("new category should be active")
	}
}

func TestColdOps_DeleteCategory_SoftDeleteWhenHasExpenses(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	catID := "cat-soft"
	r.categories = append(r.categories, repositories.ColdOpsCategoryRow{
		ID: catID, CompanyID: "co-1", Name: "Old", IsActive: true, CreatedBy: "u-1",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	expCatID := catID
	r.expenses = append(r.expenses, repositories.ColdOpsExpenseRow{
		ID: "exp-1", CompanyID: "co-1", CategoryID: &expCatID, Amount: 100,
		ExpenseDate: time.Now(), Title: "T", ExpenseType: "variable", CreatedBy: "u-1",
	})

	err := svc.DeleteCategory(ctx, "co-1", "direktor", catID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Category should still exist but be deactivated
	var found *repositories.ColdOpsCategoryRow
	for _, c := range r.categories {
		if c.ID == catID {
			found = &c
			break
		}
	}
	if found == nil {
		t.Fatal("category should not be hard-deleted when it has expenses")
	}
	if found.IsActive {
		t.Error("category should be deactivated (soft-delete)")
	}
}

func TestColdOps_DeleteCategory_HardDeleteWhenNoExpenses(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	catID := "cat-hard"
	r.categories = append(r.categories, repositories.ColdOpsCategoryRow{
		ID: catID, CompanyID: "co-1", Name: "New", IsActive: true, CreatedBy: "u-1",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.DeleteCategory(ctx, "co-1", "direktor", catID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range r.categories {
		if c.ID == catID {
			t.Error("category should be hard-deleted when it has no expenses")
		}
	}
}

func TestColdOps_DeleteCategory_NotFound(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	err := svc.DeleteCategory(ctx, "co-1", "direktor", "no-such-id")
	if err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// ── Expense validation tests ──────────────────────────────────────────────────

func TestColdOps_CreateExpense_Validation(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	cases := []struct {
		req dto.CreateExpenseReq
	}{
		{dto.CreateExpenseReq{Title: "", Amount: 100, ExpenseDate: "2024-01-01", ExpenseType: "fixed"}},
		{dto.CreateExpenseReq{Title: "X", Amount: 0, ExpenseDate: "2024-01-01", ExpenseType: "fixed"}},
		{dto.CreateExpenseReq{Title: "X", Amount: 100, ExpenseDate: "2024-01-01", ExpenseType: "bad"}},
		{dto.CreateExpenseReq{Title: "X", Amount: 100, ExpenseDate: "not-a-date", ExpenseType: "fixed"}},
	}
	for _, c := range cases {
		_, err := svc.CreateExpense(ctx, "co-1", "u-1", "direktor", c.req)
		if err == nil {
			t.Errorf("expected error for req=%+v", c.req)
		}
	}
}

// ── Recurring definition + occurrence generation tests ────────────────────────

func TestColdOps_OccurrenceGeneration_Monthly(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	r.recurDefs = append(r.recurDefs, repositories.RecurringDefRow{
		ID: "def-m", CompanyID: "co-1", Title: "Najam", Frequency: "monthly",
		DayOfMonth: 15, StartDate: start, IsActive: true, CreatedBy: "u-1",
	})

	err := svc.generateOccurrences(ctx, "co-1", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.occurrences) == 0 {
		t.Fatal("expected at least one occurrence to be generated")
	}
	for _, o := range r.occurrences {
		if o.Status != "planned" {
			t.Errorf("new occurrence status should be planned, got %q", o.Status)
		}
	}
}

func TestColdOps_OccurrenceGeneration_Idempotent(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	r.recurDefs = append(r.recurDefs, repositories.RecurringDefRow{
		ID: "def-m", CompanyID: "co-1", Title: "Najam", Frequency: "monthly",
		DayOfMonth: 1, StartDate: start, IsActive: true, CreatedBy: "u-1",
	})

	_ = svc.generateOccurrences(ctx, "co-1", 3)
	countAfterFirst := len(r.occurrences)
	_ = svc.generateOccurrences(ctx, "co-1", 3)
	if len(r.occurrences) != countAfterFirst {
		t.Errorf("generation is not idempotent: first=%d, second=%d", countAfterFirst, len(r.occurrences))
	}
}

func TestColdOps_OccurrenceGeneration_MonthEnd(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	// day_of_month=31 in February should clamp to 28 (or 29)
	start := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	r.recurDefs = append(r.recurDefs, repositories.RecurringDefRow{
		ID: "def-feb", CompanyID: "co-1", Title: "Month End", Frequency: "monthly",
		DayOfMonth: 31, StartDate: start, IsActive: true, CreatedBy: "u-1",
	})

	_ = svc.generateOccurrences(ctx, "co-1", 1)
	var febOcc *repositories.ColdOpsOccurrenceRow
	for _, o := range r.occurrences {
		if o.DefinitionID == "def-feb" {
			occ := o
			febOcc = &occ
			break
		}
	}
	if febOcc == nil {
		t.Fatal("expected occurrence for February")
	}
	// 2024 is a leap year, so Feb has 29 days
	if febOcc.DueDate.Day() != 29 {
		t.Errorf("expected day=29 (leap year 2024 Feb), got %d", febOcc.DueDate.Day())
	}
}

func TestColdOps_OccurrenceGeneration_Yearly(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	month := 6
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	r.recurDefs = append(r.recurDefs, repositories.RecurringDefRow{
		ID: "def-y", CompanyID: "co-1", Title: "Annual", Frequency: "yearly",
		DayOfMonth: 15, MonthOfYear: &month, StartDate: start, IsActive: true, CreatedBy: "u-1",
	})

	_ = svc.generateOccurrences(ctx, "co-1", 24)
	if len(r.occurrences) == 0 {
		t.Fatal("expected at least one yearly occurrence")
	}
	for _, o := range r.occurrences {
		if o.DefinitionID != "def-y" {
			continue
		}
		if o.DueDate.Month() != time.June {
			t.Errorf("yearly occurrence should be in June, got %v", o.DueDate.Month())
		}
		if o.DueDate.Day() != 15 {
			t.Errorf("yearly occurrence should be on day 15, got %d", o.DueDate.Day())
		}
	}
}

// ── clampDayOfMonth unit tests ────────────────────────────────────────────────

func TestClampDayOfMonth(t *testing.T) {
	cases := []struct {
		year, month, day int
		want             int
	}{
		{2024, 2, 31, 29}, // leap year
		{2023, 2, 31, 28}, // non-leap
		{2024, 1, 31, 31},
		{2024, 4, 31, 30}, // April has 30 days
		{2024, 1, 15, 15}, // no clamp needed
	}
	for _, c := range cases {
		got := clampDayOfMonth(c.year, c.month, c.day)
		if got != c.want {
			t.Errorf("clampDayOfMonth(%d,%d,%d)=%d, want %d", c.year, c.month, c.day, got, c.want)
		}
	}
}

// ── Recurring amount change test ──────────────────────────────────────────────

func TestColdOps_UpdateRecurringDef_NewAmount(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	r.recurDefs = append(r.recurDefs, repositories.RecurringDefRow{
		ID: "def-1", CompanyID: "co-1", Title: "Old Title", Frequency: "monthly",
		DayOfMonth: 1, StartDate: time.Now(), IsActive: true, CreatedBy: "u-1",
	})

	newAmt := 500.0
	_, err := svc.UpdateRecurringDef(ctx, "co-1", "u-1", "direktor", "def-1", dto.UpdateRecurringReq{
		Title:      "New Title",
		DayOfMonth: 1,
		IsActive:   true,
		NewAmount:  &newAmt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.amounts) != 1 {
		t.Errorf("expected 1 amount record, got %d", len(r.amounts))
	}
	if r.amounts[0].Amount != 500.0 {
		t.Errorf("expected amount=500, got %f", r.amounts[0].Amount)
	}
}

// ── Budget validation tests ───────────────────────────────────────────────────

func TestColdOps_UpsertBudget_Validation(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()

	_, err := svc.UpsertBudget(ctx, "co-1", "u-1", "direktor", dto.UpsertBudgetReq{
		Year: 2024, Month: 13, BudgetAmount: 1000,
	})
	if err == nil {
		t.Error("expected validation error for month=13")
	}

	_, err = svc.UpsertBudget(ctx, "co-1", "u-1", "direktor", dto.UpsertBudgetReq{
		Year: 2024, Month: 1, BudgetAmount: 0,
	})
	if err == nil {
		t.Error("expected validation error for budget_amount=0")
	}
}

// ── Confirm/Skip occurrence tests ─────────────────────────────────────────────

func TestColdOps_ConfirmOccurrence_OK(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	r.occurrences = append(r.occurrences, repositories.ColdOpsOccurrenceRow{
		ID: "occ-1", DefinitionID: "def-1", CompanyID: "co-1",
		OccurrenceKey: "2024-01", DueDate: time.Now(), Status: "planned",
	})

	occ, err := svc.ConfirmOccurrence(ctx, "co-1", "u-1", "direktor", "occ-1", dto.ConfirmOccurrenceReq{Amount: 350.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if occ.Status != "confirmed" {
		t.Errorf("want status=confirmed, got %q", occ.Status)
	}
}

func TestColdOps_ConfirmOccurrence_ZeroAmount(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	_, err := svc.ConfirmOccurrence(ctx, "co-1", "u-1", "direktor", "occ-1", dto.ConfirmOccurrenceReq{Amount: 0})
	if err == nil {
		t.Error("expected validation error for amount=0")
	}
}

func TestColdOps_SkipOccurrence_OK(t *testing.T) {
	r := &mockColdOpsRepo{}
	svc := newSvcWithRepo(r)
	ctx := context.Background()

	r.occurrences = append(r.occurrences, repositories.ColdOpsOccurrenceRow{
		ID: "occ-2", DefinitionID: "def-1", CompanyID: "co-1",
		OccurrenceKey: "2024-02", DueDate: time.Now(), Status: "planned",
	})

	occ, err := svc.SkipOccurrence(ctx, "co-1", "u-1", "direktor", "occ-2", dto.SkipOccurrenceReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if occ.Status != "skipped" {
		t.Errorf("want status=skipped, got %q", occ.Status)
	}
}
