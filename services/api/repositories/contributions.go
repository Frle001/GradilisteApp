package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ── Row / param types ─────────────────────────────────────────────────────────

type ContributionRow struct {
	ID                  string
	CompanyID           string
	EmployeeID          string
	PayrollMonth        time.Time
	ContributionsAmount float64
	UpdatedBy           *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type UpsertContributionParams struct {
	CompanyID           string
	EmployeeID          string
	PayrollMonth        time.Time
	ContributionsAmount float64
	UpdatedBy           string
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

const contribScanCols = `id::text, company_id::text, employee_id::text, payroll_month,
	contributions_amount, updated_by::text, created_at, updated_at`

func scanContribRow(row interface {
	Scan(dest ...any) error
}, r *ContributionRow) error {
	return row.Scan(
		&r.ID, &r.CompanyID, &r.EmployeeID, &r.PayrollMonth,
		&r.ContributionsAmount, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
}

// ── Methods ───────────────────────────────────────────────────────────────────

// GetContribution returns the contribution record for a given employee/month,
// or nil if no record has been entered yet (not the same as zero contributions).
func (r *AnalyticsRepository) GetContribution(ctx context.Context, companyID, employeeID string, month time.Time) (*ContributionRow, error) {
	monthFirst := firstOfMonth(month)
	var row ContributionRow
	err := r.db.QueryRow(ctx, `
		SELECT `+contribScanCols+`
		FROM employee_monthly_contributions
		WHERE company_id = $1::uuid AND employee_id = $2::uuid AND payroll_month = $3::date
	`, companyID, employeeID, monthFirst).Scan(
		&row.ID, &row.CompanyID, &row.EmployeeID, &row.PayrollMonth,
		&row.ContributionsAmount, &row.UpdatedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetContributionsForMonth returns all contribution records for a company in the given month.
func (r *AnalyticsRepository) GetContributionsForMonth(ctx context.Context, companyID string, month time.Time) ([]ContributionRow, error) {
	monthFirst := firstOfMonth(month)
	rows, err := r.db.Query(ctx, `
		SELECT `+contribScanCols+`
		FROM employee_monthly_contributions
		WHERE company_id = $1::uuid AND payroll_month = $2::date
	`, companyID, monthFirst)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ContributionRow
	for rows.Next() {
		var row ContributionRow
		if err := rows.Scan(
			&row.ID, &row.CompanyID, &row.EmployeeID, &row.PayrollMonth,
			&row.ContributionsAmount, &row.UpdatedBy, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetContributionsForEmployeeYear returns all contributions for an employee in the given calendar year.
func (r *AnalyticsRepository) GetContributionsForEmployeeYear(ctx context.Context, companyID, employeeID string, year int) ([]ContributionRow, error) {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 1, 0, 0, 0, 0, time.UTC)
	rows, err := r.db.Query(ctx, `
		SELECT `+contribScanCols+`
		FROM employee_monthly_contributions
		WHERE company_id = $1::uuid AND employee_id = $2::uuid
		  AND payroll_month >= $3::date AND payroll_month <= $4::date
		ORDER BY payroll_month
	`, companyID, employeeID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ContributionRow
	for rows.Next() {
		var row ContributionRow
		if err := rows.Scan(
			&row.ID, &row.CompanyID, &row.EmployeeID, &row.PayrollMonth,
			&row.ContributionsAmount, &row.UpdatedBy, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetContributionsForCompanyYear returns all contributions for all employees in the company
// for the given calendar year.
func (r *AnalyticsRepository) GetContributionsForCompanyYear(ctx context.Context, companyID string, year int) ([]ContributionRow, error) {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 1, 0, 0, 0, 0, time.UTC)
	rows, err := r.db.Query(ctx, `
		SELECT `+contribScanCols+`
		FROM employee_monthly_contributions
		WHERE company_id = $1::uuid
		  AND payroll_month >= $2::date AND payroll_month <= $3::date
		ORDER BY payroll_month, employee_id
	`, companyID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ContributionRow
	for rows.Next() {
		var row ContributionRow
		if err := rows.Scan(
			&row.ID, &row.CompanyID, &row.EmployeeID, &row.PayrollMonth,
			&row.ContributionsAmount, &row.UpdatedBy, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// UpsertContribution inserts or updates the contribution record for an employee/month.
// contrib_amount = 0 is a valid entry (zero contributions explicitly recorded).
func (r *AnalyticsRepository) UpsertContribution(ctx context.Context, p UpsertContributionParams) (*ContributionRow, error) {
	monthFirst := firstOfMonth(p.PayrollMonth)
	var updatedBy *string
	if p.UpdatedBy != "" {
		updatedBy = &p.UpdatedBy
	}
	var row ContributionRow
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_monthly_contributions
			(company_id, employee_id, payroll_month, contributions_amount, updated_by)
		VALUES ($1::uuid, $2::uuid, $3::date, $4, $5::uuid)
		ON CONFLICT (company_id, employee_id, payroll_month)
		DO UPDATE SET
			contributions_amount = EXCLUDED.contributions_amount,
			updated_by           = EXCLUDED.updated_by,
			updated_at           = NOW()
		RETURNING `+contribScanCols,
		p.CompanyID, p.EmployeeID, monthFirst, p.ContributionsAmount, updatedBy,
	).Scan(
		&row.ID, &row.CompanyID, &row.EmployeeID, &row.PayrollMonth,
		&row.ContributionsAmount, &row.UpdatedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}
