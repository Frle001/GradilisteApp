package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Row types ─────────────────────────────────────────────────────────────────

type ColdOpsCategoryRow struct {
	ID          string
	CompanyID   string
	Name        string
	Description *string
	Color       *string
	IsActive    bool
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ColdOpsExpenseRow struct {
	ID            string
	CompanyID     string
	CategoryID    *string
	CategoryName  *string
	Amount        float64
	ExpenseDate   time.Time
	Title         string
	Notes         *string
	Supplier      *string
	PaymentMethod *string
	ExpenseType   string
	CreatedBy     string
	CreatedAt     time.Time
}

type RecurringDefRow struct {
	ID            string
	CompanyID     string
	CategoryID    *string
	CategoryName  *string
	Title         string
	Frequency     string
	DayOfMonth    int
	MonthOfYear   *int
	StartDate     time.Time
	EndDate       *time.Time
	Supplier      *string
	PaymentMethod *string
	Notes         *string
	IsActive      bool
	CreatedBy     string
	CreatedAt     time.Time
}

type RecurringAmountRow struct {
	ID            string
	DefinitionID  string
	Amount        float64
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

type ColdOpsOccurrenceRow struct {
	ID              string
	DefinitionID    string
	DefinitionTitle string
	CompanyID       string
	OccurrenceKey   string
	DueDate         time.Time
	Status          string
	ExpenseID       *string
	Amount          *float64
	Notes           *string
	HandledBy       *string
	HandledAt       *time.Time
	CreatedAt       time.Time
}

type ColdOpsBudgetRow struct {
	ID           string
	CompanyID    string
	CategoryID   *string
	CategoryName *string
	Year         int
	Month        int
	BudgetAmount float64
	CreatedBy    string
}

type TrendPointRow struct {
	Year  int
	Month int
	Total float64
	Count int
}

type CategoryStatsRow struct {
	CategoryID   *string
	CategoryName *string
	Color        *string
	Total        float64
	Count        int
}

type FixedVariableRow struct {
	FixedTotal    float64
	VariableTotal float64
	FixedCount    int
	VariableCount int
}

// ── Params types ──────────────────────────────────────────────────────────────

type CreateCategoryParams struct {
	CompanyID   string
	Name        string
	Description *string
	Color       *string
	CreatedBy   string
}

type UpdateCategoryParams struct {
	ID          string
	CompanyID   string
	Name        string
	Description *string
	Color       *string
	IsActive    bool
}

type CreateExpenseParams struct {
	CompanyID     string
	CategoryID    *string
	Amount        float64
	ExpenseDate   time.Time
	Title         string
	Notes         *string
	Supplier      *string
	PaymentMethod *string
	ExpenseType   string
	CreatedBy     string
}

type UpdateExpenseParams struct {
	ID            string
	CompanyID     string
	CategoryID    *string
	Amount        float64
	ExpenseDate   time.Time
	Title         string
	Notes         *string
	Supplier      *string
	PaymentMethod *string
	ExpenseType   string
}

type CreateRecurringDefParams struct {
	CompanyID     string
	CategoryID    *string
	Title         string
	Frequency     string
	DayOfMonth    int
	MonthOfYear   *int
	StartDate     time.Time
	EndDate       *time.Time
	Supplier      *string
	PaymentMethod *string
	Notes         *string
	CreatedBy     string
}

type UpdateRecurringDefParams struct {
	ID            string
	CompanyID     string
	CategoryID    *string
	Title         string
	DayOfMonth    int
	MonthOfYear   *int
	EndDate       *time.Time
	Supplier      *string
	PaymentMethod *string
	Notes         *string
	IsActive      bool
}

type CreateRecurringAmountParams struct {
	DefinitionID  string
	CompanyID     string
	Amount        float64
	EffectiveFrom time.Time
	CreatedBy     string
}

type UpsertOccurrenceParams struct {
	DefinitionID  string
	CompanyID     string
	OccurrenceKey string
	DueDate       time.Time
}

type UpdateOccurrenceParams struct {
	ID        string
	CompanyID string
	Status    string
	Amount    *float64
	Notes     *string
	ExpenseID *string
	HandledBy *string
	HandledAt *time.Time
}

type UpsertBudgetParams struct {
	CompanyID    string
	CategoryID   *string
	Year         int
	Month        int
	BudgetAmount float64
	CreatedBy    string
}

// ── Repository ────────────────────────────────────────────────────────────────

type ColdOpsRepository struct {
	db *pgxpool.Pool
}

func NewColdOpsRepository(db *pgxpool.Pool) *ColdOpsRepository {
	return &ColdOpsRepository{db: db}
}

// ── Categories ────────────────────────────────────────────────────────────────

func (r *ColdOpsRepository) ListCategories(ctx context.Context, companyID string) ([]ColdOpsCategoryRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, company_id, name, description, color, is_active, created_by, created_at, updated_at
		FROM cold_ops_categories
		WHERE company_id = $1::uuid
		ORDER BY name ASC
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ColdOpsCategoryRow
	for rows.Next() {
		var c ColdOpsCategoryRow
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.Name, &c.Description, &c.Color,
			&c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ColdOpsRepository) CreateCategory(ctx context.Context, p CreateCategoryParams) (*ColdOpsCategoryRow, error) {
	var c ColdOpsCategoryRow
	err := r.db.QueryRow(ctx, `
		INSERT INTO cold_ops_categories (company_id, name, description, color, created_by)
		VALUES ($1::uuid, $2, $3, $4, $5::uuid)
		RETURNING id, company_id, name, description, color, is_active, created_by, created_at, updated_at
	`, p.CompanyID, p.Name, p.Description, p.Color, p.CreatedBy).
		Scan(&c.ID, &c.CompanyID, &c.Name, &c.Description, &c.Color,
			&c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ColdOpsRepository) GetCategory(ctx context.Context, companyID, id string) (*ColdOpsCategoryRow, error) {
	var c ColdOpsCategoryRow
	err := r.db.QueryRow(ctx, `
		SELECT id, company_id, name, description, color, is_active, created_by, created_at, updated_at
		FROM cold_ops_categories WHERE id=$1::uuid AND company_id=$2::uuid
	`, id, companyID).Scan(&c.ID, &c.CompanyID, &c.Name, &c.Description, &c.Color,
		&c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *ColdOpsRepository) UpdateCategory(ctx context.Context, p UpdateCategoryParams) (*ColdOpsCategoryRow, error) {
	var c ColdOpsCategoryRow
	err := r.db.QueryRow(ctx, `
		UPDATE cold_ops_categories
		SET name=$3, description=$4, color=$5, is_active=$6, updated_at=NOW()
		WHERE id=$1::uuid AND company_id=$2::uuid
		RETURNING id, company_id, name, description, color, is_active, created_by, created_at, updated_at
	`, p.ID, p.CompanyID, p.Name, p.Description, p.Color, p.IsActive).
		Scan(&c.ID, &c.CompanyID, &c.Name, &c.Description, &c.Color,
			&c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *ColdOpsRepository) DeleteCategory(ctx context.Context, companyID, id string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM cold_ops_categories WHERE id=$1::uuid AND company_id=$2::uuid
	`, id, companyID)
	return err
}

func (r *ColdOpsRepository) CountExpensesForCategory(ctx context.Context, companyID, categoryID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM cold_ops_expenses WHERE company_id=$1::uuid AND category_id=$2::uuid`,
		companyID, categoryID).Scan(&n)
	return n, err
}

// ── Expenses ──────────────────────────────────────────────────────────────────

func (r *ColdOpsRepository) ListExpenses(ctx context.Context, companyID string,
	categoryID, expenseType *string, from, to *time.Time, offset, limit int) ([]ColdOpsExpenseRow, error) {

	where := "e.company_id = $1::uuid"
	args := []any{companyID}
	n := 2
	if categoryID != nil {
		where += fmt.Sprintf(" AND e.category_id = $%d::uuid", n)
		args = append(args, *categoryID)
		n++
	}
	if expenseType != nil {
		where += fmt.Sprintf(" AND e.expense_type = $%d", n)
		args = append(args, *expenseType)
		n++
	}
	if from != nil {
		where += fmt.Sprintf(" AND e.expense_date >= $%d", n)
		args = append(args, *from)
		n++
	}
	if to != nil {
		where += fmt.Sprintf(" AND e.expense_date <= $%d", n)
		args = append(args, *to)
		n++
	}
	args = append(args, limit, offset)

	q := fmt.Sprintf(`
		SELECT e.id, e.company_id, e.category_id, c.name, e.amount, e.expense_date,
			e.title, e.notes, e.supplier, e.payment_method, e.expense_type, e.created_by, e.created_at
		FROM cold_ops_expenses e
		LEFT JOIN cold_ops_categories c ON c.id = e.category_id
		WHERE %s
		ORDER BY e.expense_date DESC, e.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExpenseRows(rows)
}

func (r *ColdOpsRepository) CreateExpense(ctx context.Context, p CreateExpenseParams) (*ColdOpsExpenseRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH ins AS (
			INSERT INTO cold_ops_expenses
				(company_id, category_id, amount, expense_date, title, notes, supplier, payment_method, expense_type, created_by)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10::uuid)
			RETURNING *
		)
		SELECT e.id, e.company_id, e.category_id, c.name, e.amount, e.expense_date,
			e.title, e.notes, e.supplier, e.payment_method, e.expense_type, e.created_by, e.created_at
		FROM ins e LEFT JOIN cold_ops_categories c ON c.id = e.category_id
	`, p.CompanyID, p.CategoryID, p.Amount, p.ExpenseDate, p.Title,
		p.Notes, p.Supplier, p.PaymentMethod, p.ExpenseType, p.CreatedBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanExpenseRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (r *ColdOpsRepository) GetExpense(ctx context.Context, companyID, id string) (*ColdOpsExpenseRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id, e.company_id, e.category_id, c.name, e.amount, e.expense_date,
			e.title, e.notes, e.supplier, e.payment_method, e.expense_type, e.created_by, e.created_at
		FROM cold_ops_expenses e
		LEFT JOIN cold_ops_categories c ON c.id = e.category_id
		WHERE e.id=$1::uuid AND e.company_id=$2::uuid
	`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanExpenseRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (r *ColdOpsRepository) UpdateExpense(ctx context.Context, p UpdateExpenseParams) (*ColdOpsExpenseRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH upd AS (
			UPDATE cold_ops_expenses
			SET category_id=$3::uuid, amount=$4, expense_date=$5, title=$6,
				notes=$7, supplier=$8, payment_method=$9, expense_type=$10, updated_at=NOW()
			WHERE id=$1::uuid AND company_id=$2::uuid
			RETURNING *
		)
		SELECT e.id, e.company_id, e.category_id, c.name, e.amount, e.expense_date,
			e.title, e.notes, e.supplier, e.payment_method, e.expense_type, e.created_by, e.created_at
		FROM upd e LEFT JOIN cold_ops_categories c ON c.id = e.category_id
	`, p.ID, p.CompanyID, p.CategoryID, p.Amount, p.ExpenseDate, p.Title,
		p.Notes, p.Supplier, p.PaymentMethod, p.ExpenseType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanExpenseRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (r *ColdOpsRepository) DeleteExpense(ctx context.Context, companyID, id string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM cold_ops_expenses WHERE id=$1::uuid AND company_id=$2::uuid`, id, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanExpenseRows(rows pgx.Rows) ([]ColdOpsExpenseRow, error) {
	var out []ColdOpsExpenseRow
	for rows.Next() {
		var e ColdOpsExpenseRow
		if err := rows.Scan(&e.ID, &e.CompanyID, &e.CategoryID, &e.CategoryName,
			&e.Amount, &e.ExpenseDate, &e.Title, &e.Notes, &e.Supplier,
			&e.PaymentMethod, &e.ExpenseType, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ── Recurring definitions ─────────────────────────────────────────────────────

const recurringDefSelect = `
	SELECT d.id, d.company_id, d.category_id, c.name,
		d.title, d.frequency, d.day_of_month, d.month_of_year,
		d.start_date, d.end_date, d.supplier, d.payment_method, d.notes,
		d.is_active, d.created_by, d.created_at
	FROM cold_ops_recurring_defs d
	LEFT JOIN cold_ops_categories c ON c.id = d.category_id
`

func (r *ColdOpsRepository) ListRecurringDefs(ctx context.Context, companyID string, activeOnly bool) ([]RecurringDefRow, error) {
	where := "d.company_id = $1::uuid"
	if activeOnly {
		where += " AND d.is_active = true"
	}
	rows, err := r.db.Query(ctx, recurringDefSelect+` WHERE `+where+` ORDER BY d.title ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecurringDefRows(rows)
}

func (r *ColdOpsRepository) CreateRecurringDef(ctx context.Context, p CreateRecurringDefParams) (*RecurringDefRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH ins AS (
			INSERT INTO cold_ops_recurring_defs
				(company_id, category_id, title, frequency, day_of_month, month_of_year,
				 start_date, end_date, supplier, payment_method, notes, created_by)
			VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::uuid)
			RETURNING *
		)
		`+recurringDefSelect+` FROM ins d LEFT JOIN cold_ops_categories c ON c.id = d.category_id
	`, p.CompanyID, p.CategoryID, p.Title, p.Frequency, p.DayOfMonth, p.MonthOfYear,
		p.StartDate, p.EndDate, p.Supplier, p.PaymentMethod, p.Notes, p.CreatedBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanRecurringDefRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (r *ColdOpsRepository) GetRecurringDef(ctx context.Context, companyID, id string) (*RecurringDefRow, error) {
	rows, err := r.db.Query(ctx, recurringDefSelect+` WHERE d.id=$1::uuid AND d.company_id=$2::uuid`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanRecurringDefRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (r *ColdOpsRepository) UpdateRecurringDef(ctx context.Context, p UpdateRecurringDefParams) (*RecurringDefRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH upd AS (
			UPDATE cold_ops_recurring_defs
			SET category_id=$3::uuid, title=$4, day_of_month=$5, month_of_year=$6,
				end_date=$7, supplier=$8, payment_method=$9, notes=$10, is_active=$11,
				updated_at=NOW()
			WHERE id=$1::uuid AND company_id=$2::uuid
			RETURNING *
		)
		`+recurringDefSelect+` FROM upd d LEFT JOIN cold_ops_categories c ON c.id = d.category_id
	`, p.ID, p.CompanyID, p.CategoryID, p.Title, p.DayOfMonth, p.MonthOfYear,
		p.EndDate, p.Supplier, p.PaymentMethod, p.Notes, p.IsActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanRecurringDefRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func scanRecurringDefRows(rows pgx.Rows) ([]RecurringDefRow, error) {
	var out []RecurringDefRow
	for rows.Next() {
		var d RecurringDefRow
		if err := rows.Scan(&d.ID, &d.CompanyID, &d.CategoryID, &d.CategoryName,
			&d.Title, &d.Frequency, &d.DayOfMonth, &d.MonthOfYear,
			&d.StartDate, &d.EndDate, &d.Supplier, &d.PaymentMethod, &d.Notes,
			&d.IsActive, &d.CreatedBy, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ── Recurring amounts ─────────────────────────────────────────────────────────

func (r *ColdOpsRepository) CreateRecurringAmount(ctx context.Context, p CreateRecurringAmountParams) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	_, err = tx.Exec(ctx, `
		UPDATE cold_ops_recurring_amounts
		SET effective_to = $3::date - INTERVAL '1 day'
		WHERE definition_id=$1::uuid AND company_id=$2::uuid AND effective_to IS NULL
	`, p.DefinitionID, p.CompanyID, p.EffectiveFrom)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO cold_ops_recurring_amounts
			(definition_id, company_id, amount, effective_from, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid)
	`, p.DefinitionID, p.CompanyID, p.Amount, p.EffectiveFrom, p.CreatedBy)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ColdOpsRepository) GetAmountForDate(ctx context.Context, companyID, defID string, d time.Time) (*float64, error) {
	var amt float64
	err := r.db.QueryRow(ctx, `
		SELECT amount FROM cold_ops_recurring_amounts
		WHERE definition_id=$1::uuid AND company_id=$2::uuid
		  AND effective_from <= $3
		  AND (effective_to IS NULL OR effective_to >= $3)
		ORDER BY effective_from DESC LIMIT 1
	`, defID, companyID, d).Scan(&amt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &amt, err
}

// ── Occurrences ───────────────────────────────────────────────────────────────

func (r *ColdOpsRepository) BulkUpsertOccurrences(ctx context.Context, params []UpsertOccurrenceParams) error {
	for _, p := range params {
		_, err := r.db.Exec(ctx, `
			INSERT INTO cold_ops_occurrences (definition_id, company_id, occurrence_key, due_date)
			VALUES ($1::uuid, $2::uuid, $3, $4)
			ON CONFLICT (definition_id, occurrence_key) DO NOTHING
		`, p.DefinitionID, p.CompanyID, p.OccurrenceKey, p.DueDate)
		if err != nil {
			return err
		}
	}
	return nil
}

const occurrenceSelect = `
	SELECT o.id, o.definition_id, d.title, o.company_id, o.occurrence_key,
		o.due_date, o.status, o.expense_id, o.amount, o.notes,
		o.handled_by, o.handled_at, o.created_at
	FROM cold_ops_occurrences o
	JOIN cold_ops_recurring_defs d ON d.id = o.definition_id
`

func (r *ColdOpsRepository) ListOccurrences(ctx context.Context, companyID, defID string,
	from, to *time.Time, status *string) ([]ColdOpsOccurrenceRow, error) {

	where := "o.company_id=$1::uuid AND o.definition_id=$2::uuid"
	args := []any{companyID, defID}
	n := 3
	if from != nil {
		where += fmt.Sprintf(" AND o.due_date >= $%d", n)
		args = append(args, *from)
		n++
	}
	if to != nil {
		where += fmt.Sprintf(" AND o.due_date <= $%d", n)
		args = append(args, *to)
		n++
	}
	if status != nil {
		where += fmt.Sprintf(" AND o.status = $%d", n)
		args = append(args, *status)
	}
	rows, err := r.db.Query(ctx, occurrenceSelect+` WHERE `+where+` ORDER BY o.due_date ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrenceRows(rows)
}

func (r *ColdOpsRepository) ListUpcomingOccurrences(ctx context.Context, companyID string, horizonDays int) ([]ColdOpsOccurrenceRow, error) {
	rows, err := r.db.Query(ctx, occurrenceSelect+`
		WHERE o.company_id=$1::uuid AND o.status='planned'
		  AND o.due_date BETWEEN CURRENT_DATE AND CURRENT_DATE + ($2 * INTERVAL '1 day')
		ORDER BY o.due_date ASC LIMIT 50
	`, companyID, horizonDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOccurrenceRows(rows)
}

func (r *ColdOpsRepository) GetOccurrence(ctx context.Context, companyID, id string) (*ColdOpsOccurrenceRow, error) {
	rows, err := r.db.Query(ctx, occurrenceSelect+` WHERE o.id=$1::uuid AND o.company_id=$2::uuid`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanOccurrenceRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func (r *ColdOpsRepository) UpdateOccurrence(ctx context.Context, p UpdateOccurrenceParams) (*ColdOpsOccurrenceRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH upd AS (
			UPDATE cold_ops_occurrences
			SET status=$3, amount=$4, notes=$5, expense_id=$6::uuid,
				handled_by=$7::uuid, handled_at=$8, updated_at=NOW()
			WHERE id=$1::uuid AND company_id=$2::uuid
			RETURNING *
		)
		`+occurrenceSelect+` FROM upd o JOIN cold_ops_recurring_defs d ON d.id = o.definition_id
	`, p.ID, p.CompanyID, p.Status, p.Amount, p.Notes, p.ExpenseID, p.HandledBy, p.HandledAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanOccurrenceRows(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func scanOccurrenceRows(rows pgx.Rows) ([]ColdOpsOccurrenceRow, error) {
	var out []ColdOpsOccurrenceRow
	for rows.Next() {
		var o ColdOpsOccurrenceRow
		if err := rows.Scan(&o.ID, &o.DefinitionID, &o.DefinitionTitle, &o.CompanyID,
			&o.OccurrenceKey, &o.DueDate, &o.Status, &o.ExpenseID, &o.Amount,
			&o.Notes, &o.HandledBy, &o.HandledAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ── Budgets ───────────────────────────────────────────────────────────────────

func (r *ColdOpsRepository) ListBudgets(ctx context.Context, companyID string, year, month int) ([]ColdOpsBudgetRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.company_id, b.category_id, c.name, b.year, b.month, b.budget_amount, b.created_by
		FROM cold_ops_budgets b
		LEFT JOIN cold_ops_categories c ON c.id = b.category_id
		WHERE b.company_id=$1::uuid AND b.year=$2 AND b.month=$3
		ORDER BY c.name ASC NULLS FIRST
	`, companyID, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ColdOpsBudgetRow
	for rows.Next() {
		var b ColdOpsBudgetRow
		if err := rows.Scan(&b.ID, &b.CompanyID, &b.CategoryID, &b.CategoryName,
			&b.Year, &b.Month, &b.BudgetAmount, &b.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *ColdOpsRepository) UpsertBudget(ctx context.Context, p UpsertBudgetParams) (*ColdOpsBudgetRow, error) {
	// Delete existing entry with same key, then insert fresh.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var delErr error
	if p.CategoryID != nil {
		_, delErr = tx.Exec(ctx, `
			DELETE FROM cold_ops_budgets
			WHERE company_id=$1::uuid AND category_id=$2::uuid AND year=$3 AND month=$4
		`, p.CompanyID, *p.CategoryID, p.Year, p.Month)
	} else {
		_, delErr = tx.Exec(ctx, `
			DELETE FROM cold_ops_budgets
			WHERE company_id=$1::uuid AND category_id IS NULL AND year=$2 AND month=$3
		`, p.CompanyID, p.Year, p.Month)
	}
	if delErr != nil {
		return nil, delErr
	}

	var b ColdOpsBudgetRow
	err = tx.QueryRow(ctx, `
		INSERT INTO cold_ops_budgets (company_id, category_id, year, month, budget_amount, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid)
		RETURNING id, company_id, category_id, NULL::text, year, month, budget_amount, created_by
	`, p.CompanyID, p.CategoryID, p.Year, p.Month, p.BudgetAmount, p.CreatedBy).
		Scan(&b.ID, &b.CompanyID, &b.CategoryID, &b.CategoryName, &b.Year, &b.Month, &b.BudgetAmount, &b.CreatedBy)
	if err != nil {
		return nil, err
	}
	return &b, tx.Commit(ctx)
}

func (r *ColdOpsRepository) DeleteBudget(ctx context.Context, companyID, id string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM cold_ops_budgets WHERE id=$1::uuid AND company_id=$2::uuid`, id, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ── Analytics aggregations ────────────────────────────────────────────────────

func (r *ColdOpsRepository) GetPeriodTotal(ctx context.Context, companyID string, from, to time.Time) (float64, int, error) {
	var total float64
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount),0), COUNT(*)
		FROM cold_ops_expenses
		WHERE company_id=$1::uuid AND expense_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&total, &count)
	return total, count, err
}

func (r *ColdOpsRepository) GetMonthlyTrend(ctx context.Context, companyID string, months int) ([]TrendPointRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT EXTRACT(YEAR FROM expense_date)::int,
			   EXTRACT(MONTH FROM expense_date)::int,
			   COALESCE(SUM(amount),0),
			   COUNT(*)
		FROM cold_ops_expenses
		WHERE company_id=$1::uuid
		  AND expense_date >= date_trunc('month', NOW()) - (($2 - 1) * INTERVAL '1 month')
		GROUP BY 1, 2
		ORDER BY 1, 2
	`, companyID, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendPointRow
	for rows.Next() {
		var t TrendPointRow
		if err := rows.Scan(&t.Year, &t.Month, &t.Total, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *ColdOpsRepository) GetCategoryDistribution(ctx context.Context, companyID string, from, to time.Time) ([]CategoryStatsRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.category_id, c.name, c.color, COALESCE(SUM(e.amount),0), COUNT(*)
		FROM cold_ops_expenses e
		LEFT JOIN cold_ops_categories c ON c.id = e.category_id
		WHERE e.company_id=$1::uuid AND e.expense_date BETWEEN $2 AND $3
		GROUP BY e.category_id, c.name, c.color
		ORDER BY SUM(e.amount) DESC
	`, companyID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CategoryStatsRow
	for rows.Next() {
		var s CategoryStatsRow
		if err := rows.Scan(&s.CategoryID, &s.CategoryName, &s.Color, &s.Total, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ColdOpsRepository) GetFixedVsVariable(ctx context.Context, companyID string, from, to time.Time) (FixedVariableRow, error) {
	var fv FixedVariableRow
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN expense_type='fixed' THEN amount ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN expense_type='variable' THEN amount ELSE 0 END),0),
			COUNT(CASE WHEN expense_type='fixed' THEN 1 END),
			COUNT(CASE WHEN expense_type='variable' THEN 1 END)
		FROM cold_ops_expenses
		WHERE company_id=$1::uuid AND expense_date BETWEEN $2 AND $3
	`, companyID, from, to).Scan(&fv.FixedTotal, &fv.VariableTotal, &fv.FixedCount, &fv.VariableCount)
	return fv, err
}

func (r *ColdOpsRepository) GetPlannedOccurrenceTotals(ctx context.Context, companyID string, from, to time.Time) ([]TrendPointRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT EXTRACT(YEAR FROM due_date)::int, EXTRACT(MONTH FROM due_date)::int,
			COALESCE(SUM(amount),0), COUNT(*)
		FROM cold_ops_occurrences
		WHERE company_id=$1::uuid AND status='planned' AND due_date BETWEEN $2 AND $3
		GROUP BY 1, 2
		ORDER BY 1, 2
	`, companyID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrendPointRow
	for rows.Next() {
		var t TrendPointRow
		if err := rows.Scan(&t.Year, &t.Month, &t.Total, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
