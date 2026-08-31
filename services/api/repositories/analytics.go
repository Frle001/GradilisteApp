package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Row types ─────────────────────────────────────────────────────────────────

type CompensationPlanRow struct {
	ID                string
	CompanyID         string
	EmployeeID        string
	EmployeeName      string
	PayType           string
	PayAmount         float64
	CompanyCostAmount *float64
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time
	CreatedBy         string
	CreatedAt         time.Time
}

type DailyHoursRecord struct {
	WorkDate    time.Time
	ProjectID   string
	ProjectName string
	Hours       float64
}

type ProjectHours struct {
	ProjectID   string
	ProjectName string
	Hours       float64
}

type AnalyticsEmployeeRow struct {
	ID   string
	Name string
	Role string
}

// ── Create/update params ──────────────────────────────────────────────────────

type CreateCompensationParams struct {
	CompanyID         string
	EmployeeID        string
	PayType           string
	PayAmount         float64
	CompanyCostAmount *float64
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time
	CreatedBy         string
}

type UpdateCompensationPlanParams struct {
	PlanID            string
	CompanyID         string
	PayType           string
	PayAmount         float64
	CompanyCostAmount *float64
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time
}

// ── Repository ────────────────────────────────────────────────────────────────

type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// ── Compensation plan methods ─────────────────────────────────────────────────

func (r *AnalyticsRepository) CreateCompensationPlan(ctx context.Context, p CreateCompensationParams) (*CompensationPlanRow, error) {
	var row CompensationPlanRow
	var createdBy *string
	if p.CreatedBy != "" {
		createdBy = &p.CreatedBy
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO employee_compensation_plans
			(company_id, employee_id, pay_type, pay_amount, company_cost_amount, effective_from, effective_to, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::date, $7::date, $8::uuid)
		RETURNING
			id::text, company_id::text, employee_id::text,
			(SELECT first_name || ' ' || last_name FROM employees WHERE id = employee_compensation_plans.employee_id),
			pay_type, pay_amount, company_cost_amount,
			effective_from, effective_to,
			COALESCE(created_by::text, ''), created_at
	`,
		p.CompanyID, p.EmployeeID, p.PayType, p.PayAmount, p.CompanyCostAmount,
		p.EffectiveFrom, p.EffectiveTo, createdBy,
	).Scan(
		&row.ID, &row.CompanyID, &row.EmployeeID, &row.EmployeeName,
		&row.PayType, &row.PayAmount, &row.CompanyCostAmount,
		&row.EffectiveFrom, &row.EffectiveTo,
		&row.CreatedBy, &row.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AnalyticsRepository) UpdatePlanEffectiveTo(ctx context.Context, companyID, planID string, to time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employee_compensation_plans
		 SET effective_to = $3::date, updated_at = NOW()
		 WHERE company_id = $1::uuid AND id = $2::uuid`,
		companyID, planID, to,
	)
	return err
}

func (r *AnalyticsRepository) ListCompensationPlans(ctx context.Context, companyID, employeeID string) ([]CompensationPlanRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ecp.id::text, ecp.company_id::text, ecp.employee_id::text,
			e.first_name || ' ' || e.last_name,
			ecp.pay_type, ecp.pay_amount, ecp.company_cost_amount,
			ecp.effective_from, ecp.effective_to,
			COALESCE(ecp.created_by::text, ''), ecp.created_at
		FROM employee_compensation_plans ecp
		JOIN employees e ON e.id = ecp.employee_id
		WHERE ecp.company_id = $1::uuid AND ecp.employee_id = $2::uuid
		ORDER BY ecp.effective_from DESC
	`, companyID, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CompensationPlanRow
	for rows.Next() {
		var row CompensationPlanRow
		if err := rows.Scan(
			&row.ID, &row.CompanyID, &row.EmployeeID, &row.EmployeeName,
			&row.PayType, &row.PayAmount, &row.CompanyCostAmount,
			&row.EffectiveFrom, &row.EffectiveTo,
			&row.CreatedBy, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetPlansInDateRange returns all plans for an employee that overlap [from, to].
func (r *AnalyticsRepository) GetPlansInDateRange(ctx context.Context, companyID, employeeID string, from, to time.Time) ([]CompensationPlanRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ecp.id::text, ecp.company_id::text, ecp.employee_id::text,
			e.first_name || ' ' || e.last_name,
			ecp.pay_type, ecp.pay_amount, ecp.company_cost_amount,
			ecp.effective_from, ecp.effective_to,
			COALESCE(ecp.created_by::text, ''), ecp.created_at
		FROM employee_compensation_plans ecp
		JOIN employees e ON e.id = ecp.employee_id
		WHERE ecp.company_id = $1::uuid AND ecp.employee_id = $2::uuid
		  AND ecp.effective_from <= $4::date
		  AND COALESCE(ecp.effective_to, 'infinity'::date) >= $3::date
		ORDER BY ecp.effective_from
	`, companyID, employeeID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CompensationPlanRow
	for rows.Next() {
		var row CompensationPlanRow
		if err := rows.Scan(
			&row.ID, &row.CompanyID, &row.EmployeeID, &row.EmployeeName,
			&row.PayType, &row.PayAmount, &row.CompanyCostAmount,
			&row.EffectiveFrom, &row.EffectiveTo,
			&row.CreatedBy, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// HasOverlappingPlan checks whether any existing plan for the employee overlaps
// the period [from, to] (to=nil means open-ended). excludeID can be set to
// ignore the plan being edited.
func (r *AnalyticsRepository) HasOverlappingPlan(ctx context.Context, companyID, employeeID string, from time.Time, to *time.Time, excludeID *string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM employee_compensation_plans
			WHERE company_id = $1::uuid
			  AND employee_id = $2::uuid
			  AND effective_from <= COALESCE($4::date, 'infinity'::date)
			  AND COALESCE(effective_to, 'infinity'::date) >= $3::date
			  AND ($5::uuid IS NULL OR id != $5::uuid)
		)
	`, companyID, employeeID, from, to, excludeID).Scan(&exists)
	return exists, err
}

// FindOpenPlan returns the first open-ended plan (effective_to IS NULL) for an employee, or nil.
func (r *AnalyticsRepository) FindOpenPlan(ctx context.Context, companyID, employeeID string) (*CompensationPlanRow, error) {
	var row CompensationPlanRow
	err := r.db.QueryRow(ctx, `
		SELECT ecp.id::text, ecp.company_id::text, ecp.employee_id::text,
			e.first_name || ' ' || e.last_name,
			ecp.pay_type, ecp.pay_amount, ecp.company_cost_amount,
			ecp.effective_from, ecp.effective_to,
			COALESCE(ecp.created_by::text, ''), ecp.created_at
		FROM employee_compensation_plans ecp
		JOIN employees e ON e.id = ecp.employee_id
		WHERE ecp.company_id = $1::uuid AND ecp.employee_id = $2::uuid
		  AND ecp.effective_to IS NULL
		ORDER BY ecp.effective_from DESC
		LIMIT 1
	`, companyID, employeeID).Scan(
		&row.ID, &row.CompanyID, &row.EmployeeID, &row.EmployeeName,
		&row.PayType, &row.PayAmount, &row.CompanyCostAmount,
		&row.EffectiveFrom, &row.EffectiveTo,
		&row.CreatedBy, &row.CreatedAt,
	)
	if err != nil {
		return nil, err // caller checks pgx.ErrNoRows
	}
	return &row, nil
}

func (r *AnalyticsRepository) GetCompensationPlanByID(ctx context.Context, companyID, planID string) (*CompensationPlanRow, error) {
	var row CompensationPlanRow
	err := r.db.QueryRow(ctx, `
		SELECT ecp.id::text, ecp.company_id::text, ecp.employee_id::text,
			e.first_name || ' ' || e.last_name,
			ecp.pay_type, ecp.pay_amount, ecp.company_cost_amount,
			ecp.effective_from, ecp.effective_to,
			COALESCE(ecp.created_by::text, ''), ecp.created_at
		FROM employee_compensation_plans ecp
		JOIN employees e ON e.id = ecp.employee_id
		WHERE ecp.company_id = $1::uuid AND ecp.id = $2::uuid
	`, companyID, planID).Scan(
		&row.ID, &row.CompanyID, &row.EmployeeID, &row.EmployeeName,
		&row.PayType, &row.PayAmount, &row.CompanyCostAmount,
		&row.EffectiveFrom, &row.EffectiveTo,
		&row.CreatedBy, &row.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AnalyticsRepository) UpdateCompensationPlanFields(ctx context.Context, p UpdateCompensationPlanParams) (*CompensationPlanRow, error) {
	var row CompensationPlanRow
	err := r.db.QueryRow(ctx, `
		UPDATE employee_compensation_plans
		SET pay_type = $3,
		    pay_amount = $4,
		    company_cost_amount = $5,
		    effective_from = $6::date,
		    effective_to = $7::date,
		    updated_at = NOW()
		WHERE company_id = $1::uuid AND id = $2::uuid
		RETURNING
			id::text, company_id::text, employee_id::text,
			(SELECT first_name || ' ' || last_name FROM employees WHERE id = employee_compensation_plans.employee_id),
			pay_type, pay_amount, company_cost_amount,
			effective_from, effective_to,
			COALESCE(created_by::text, ''), created_at
	`,
		p.CompanyID, p.PlanID, p.PayType, p.PayAmount, p.CompanyCostAmount,
		p.EffectiveFrom, p.EffectiveTo,
	).Scan(
		&row.ID, &row.CompanyID, &row.EmployeeID, &row.EmployeeName,
		&row.PayType, &row.PayAmount, &row.CompanyCostAmount,
		&row.EffectiveFrom, &row.EffectiveTo,
		&row.CreatedBy, &row.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ── Worker hours methods ──────────────────────────────────────────────────────

// GetDailyHoursForEmployee returns per-day records with project name for an employee in [from, to].
func (r *AnalyticsRepository) GetDailyHoursForEmployee(ctx context.Context, companyID, employeeID string, from, to time.Time) ([]DailyHoursRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT wdh.work_date, wdh.project_id::text, COALESCE(p.name, ''), wdh.hours_worked
		FROM worker_daily_hours wdh
		LEFT JOIN projects p ON p.id = wdh.project_id
		WHERE wdh.company_id = $1::uuid AND wdh.worker_id = $2::uuid
		  AND wdh.work_date >= $3::date AND wdh.work_date <= $4::date
		ORDER BY wdh.work_date, p.name
	`, companyID, employeeID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DailyHoursRecord
	for rows.Next() {
		var rec DailyHoursRecord
		if err := rows.Scan(&rec.WorkDate, &rec.ProjectID, &rec.ProjectName, &rec.Hours); err != nil {
			return nil, err
		}
		result = append(result, rec)
	}
	return result, rows.Err()
}

// GetHoursByProject returns hours aggregated by project for an employee in [from, to].
func (r *AnalyticsRepository) GetHoursByProject(ctx context.Context, companyID, employeeID string, from, to time.Time) ([]ProjectHours, error) {
	rows, err := r.db.Query(ctx, `
		SELECT wdh.project_id::text, COALESCE(p.name, ''), SUM(wdh.hours_worked)
		FROM worker_daily_hours wdh
		LEFT JOIN projects p ON p.id = wdh.project_id
		WHERE wdh.company_id = $1::uuid AND wdh.worker_id = $2::uuid
		  AND wdh.work_date >= $3::date AND wdh.work_date <= $4::date
		GROUP BY wdh.project_id, p.name
		ORDER BY SUM(wdh.hours_worked) DESC
	`, companyID, employeeID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProjectHours
	for rows.Next() {
		var ph ProjectHours
		if err := rows.Scan(&ph.ProjectID, &ph.ProjectName, &ph.Hours); err != nil {
			return nil, err
		}
		result = append(result, ph)
	}
	return result, rows.Err()
}

// ── Employee methods ──────────────────────────────────────────────────────────

// ListEligibleEmployees returns active employees in the company who either have
// hours in [from, to] or have any compensation plan overlapping [from, to].
func (r *AnalyticsRepository) ListEligibleEmployees(ctx context.Context, companyID string, from, to time.Time) ([]AnalyticsEmployeeRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id::text, e.first_name || ' ' || e.last_name, e.role
		FROM employees e
		WHERE e.company_id = $1::uuid AND e.active = true
		  AND (
		    EXISTS (
		        SELECT 1 FROM worker_daily_hours wdh
		        WHERE wdh.company_id = $1::uuid AND wdh.worker_id = e.id
		          AND wdh.work_date >= $2::date AND wdh.work_date <= $3::date
		    )
		    OR EXISTS (
		        SELECT 1 FROM employee_compensation_plans ecp
		        WHERE ecp.company_id = $1::uuid AND ecp.employee_id = e.id
		          AND ecp.effective_from <= $3::date
		          AND COALESCE(ecp.effective_to, 'infinity'::date) >= $2::date
		    )
		  )
		ORDER BY e.last_name, e.first_name
	`, companyID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AnalyticsEmployeeRow
	for rows.Next() {
		var emp AnalyticsEmployeeRow
		if err := rows.Scan(&emp.ID, &emp.Name, &emp.Role); err != nil {
			return nil, err
		}
		result = append(result, emp)
	}
	return result, rows.Err()
}

// GetEmployee returns a single employee scoped to company, or error if not found.
func (r *AnalyticsRepository) GetEmployee(ctx context.Context, companyID, employeeID string) (*AnalyticsEmployeeRow, error) {
	var emp AnalyticsEmployeeRow
	err := r.db.QueryRow(ctx, `
		SELECT id::text, first_name || ' ' || last_name, role
		FROM employees
		WHERE company_id = $1::uuid AND id = $2::uuid AND active = true
	`, companyID, employeeID).Scan(&emp.ID, &emp.Name, &emp.Role)
	if err != nil {
		return nil, err
	}
	return &emp, nil
}
