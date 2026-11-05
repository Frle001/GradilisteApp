package dto

import "time"

// ── Categories ────────────────────────────────────────────────────────────────

type ColdOpsCategory struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	IsActive    bool    `json:"is_active"`
	CreatedAt   string  `json:"created_at"`
}

type CreateCategoryReq struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
}

type UpdateCategoryReq struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
	IsActive    bool    `json:"is_active"`
}

// ── Expenses ──────────────────────────────────────────────────────────────────

type ColdOpsExpense struct {
	ID            string  `json:"id"`
	CategoryID    *string `json:"category_id,omitempty"`
	CategoryName  *string `json:"category_name,omitempty"`
	Amount        float64 `json:"amount"`
	ExpenseDate   string  `json:"expense_date"`
	Title         string  `json:"title"`
	Notes         *string `json:"notes,omitempty"`
	Supplier      *string `json:"supplier,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	ExpenseType   string  `json:"expense_type"`
	CreatedBy     string  `json:"created_by"`
	CreatedAt     string  `json:"created_at"`
}

type CreateExpenseReq struct {
	CategoryID    *string `json:"category_id"`
	Amount        float64 `json:"amount"`
	ExpenseDate   string  `json:"expense_date"`
	Title         string  `json:"title"`
	Notes         *string `json:"notes"`
	Supplier      *string `json:"supplier"`
	PaymentMethod *string `json:"payment_method"`
	ExpenseType   string  `json:"expense_type"`
}

type UpdateExpenseReq struct {
	CategoryID    *string `json:"category_id"`
	Amount        float64 `json:"amount"`
	ExpenseDate   string  `json:"expense_date"`
	Title         string  `json:"title"`
	Notes         *string `json:"notes"`
	Supplier      *string `json:"supplier"`
	PaymentMethod *string `json:"payment_method"`
	ExpenseType   string  `json:"expense_type"`
}

type ListExpensesFilter struct {
	CategoryID  *string
	ExpenseType *string
	From        *time.Time
	To          *time.Time
	Page        int
	PageSize    int
}

// ── Recurring definitions ─────────────────────────────────────────────────────

type RecurringDef struct {
	ID            string   `json:"id"`
	CategoryID    *string  `json:"category_id,omitempty"`
	CategoryName  *string  `json:"category_name,omitempty"`
	Title         string   `json:"title"`
	Frequency     string   `json:"frequency"`
	DayOfMonth    int      `json:"day_of_month"`
	MonthOfYear   *int     `json:"month_of_year,omitempty"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
	Supplier      *string  `json:"supplier,omitempty"`
	PaymentMethod *string  `json:"payment_method,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	IsActive      bool     `json:"is_active"`
	CurrentAmount *float64 `json:"current_amount,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

type CreateRecurringReq struct {
	CategoryID    *string `json:"category_id"`
	Title         string  `json:"title"`
	Frequency     string  `json:"frequency"`
	DayOfMonth    int     `json:"day_of_month"`
	MonthOfYear   *int    `json:"month_of_year"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date"`
	Amount        float64 `json:"amount"`
	Supplier      *string `json:"supplier"`
	PaymentMethod *string `json:"payment_method"`
	Notes         *string `json:"notes"`
}

type UpdateRecurringReq struct {
	CategoryID    *string  `json:"category_id"`
	Title         string   `json:"title"`
	DayOfMonth    int      `json:"day_of_month"`
	MonthOfYear   *int     `json:"month_of_year"`
	EndDate       *string  `json:"end_date"`
	NewAmount     *float64 `json:"new_amount"`
	Supplier      *string  `json:"supplier"`
	PaymentMethod *string  `json:"payment_method"`
	Notes         *string  `json:"notes"`
	IsActive      bool     `json:"is_active"`
}

// ── Occurrences ───────────────────────────────────────────────────────────────

type ColdOpsOccurrence struct {
	ID              string   `json:"id"`
	DefinitionID    string   `json:"definition_id"`
	DefinitionTitle string   `json:"definition_title,omitempty"`
	OccurrenceKey   string   `json:"occurrence_key"`
	DueDate         string   `json:"due_date"`
	Status          string   `json:"status"`
	Amount          *float64 `json:"amount,omitempty"`
	ExpenseID       *string  `json:"expense_id,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
	HandledBy       *string  `json:"handled_by,omitempty"`
	HandledAt       *string  `json:"handled_at,omitempty"`
}

type ConfirmOccurrenceReq struct {
	Amount float64 `json:"amount"`
	Notes  *string `json:"notes"`
}

type SkipOccurrenceReq struct {
	Notes *string `json:"notes"`
}

// ── Budgets ───────────────────────────────────────────────────────────────────

type ColdOpsBudget struct {
	ID           string  `json:"id"`
	CategoryID   *string `json:"category_id,omitempty"`
	CategoryName *string `json:"category_name,omitempty"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	BudgetAmount float64 `json:"budget_amount"`
	ActualAmount float64 `json:"actual_amount"`
	Utilization  float64 `json:"utilization_pct"`
}

type UpsertBudgetReq struct {
	CategoryID   *string `json:"category_id"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	BudgetAmount float64 `json:"budget_amount"`
}

// ── Analytics summary ─────────────────────────────────────────────────────────

type ColdOpsPeriodStats struct {
	Total     float64  `json:"total"`
	Count     int      `json:"count"`
	PctChange *float64 `json:"pct_change,omitempty"`
}

type ColdOpsTrendPoint struct {
	Year  int     `json:"year"`
	Month int     `json:"month"`
	Total float64 `json:"total"`
	Count int     `json:"count"`
}

type ColdOpsCategoryStats struct {
	CategoryID   *string `json:"category_id,omitempty"`
	CategoryName string  `json:"category_name"`
	Color        *string `json:"color,omitempty"`
	Total        float64 `json:"total"`
	Count        int     `json:"count"`
	Pct          float64 `json:"pct"`
}

type ColdOpsFixedVariable struct {
	FixedTotal    float64 `json:"fixed_total"`
	VariableTotal float64 `json:"variable_total"`
	FixedCount    int     `json:"fixed_count"`
	VariableCount int     `json:"variable_count"`
}

type ColdOpsForecastPoint struct {
	Year            int     `json:"year"`
	Month           int     `json:"month"`
	ConfirmedAmount float64 `json:"confirmed_amount"`
	ForecastAmount  float64 `json:"forecast_amount"`
	IsProjection    bool    `json:"is_projection"`
}

type ColdOpsBudgetUtil struct {
	CategoryID   *string `json:"category_id,omitempty"`
	CategoryName string  `json:"category_name"`
	BudgetAmount float64 `json:"budget_amount"`
	ActualAmount float64 `json:"actual_amount"`
	Utilization  float64 `json:"utilization_pct"`
	IsOverBudget bool    `json:"is_over_budget"`
}

type ColdOpsSummary struct {
	CurrentMonth         ColdOpsPeriodStats     `json:"current_month"`
	CurrentYear          ColdOpsPeriodStats     `json:"current_year"`
	PrevMonth            ColdOpsPeriodStats     `json:"prev_month"`
	PrevYear             ColdOpsPeriodStats     `json:"prev_year"`
	MonthlyTrend         []ColdOpsTrendPoint    `json:"monthly_trend"`
	CategoryDistribution []ColdOpsCategoryStats `json:"category_distribution"`
	FixedVsVariable      ColdOpsFixedVariable   `json:"fixed_vs_variable"`
	UpcomingOccurrences  []ColdOpsOccurrence    `json:"upcoming_occurrences"`
	Forecast             []ColdOpsForecastPoint `json:"forecast"`
	BudgetUtilization    []ColdOpsBudgetUtil    `json:"budget_utilization"`
	AnnualizedCommitment float64                `json:"annualized_commitment"`
}
