package dto

import "time"

const (
	PayTypeFixed  = "fixed_monthly"
	PayTypeHourly = "hourly"
)

// ── Compensation plan ─────────────────────────────────────────────────────────

type CompensationPlan struct {
	ID                string    `json:"id"`
	EmployeeID        string    `json:"employee_id"`
	EmployeeName      string    `json:"employee_name"`
	PayType           string    `json:"pay_type"`
	PayAmount         float64   `json:"pay_amount"`
	CompanyCostAmount *float64  `json:"company_cost_amount,omitempty"`
	EffectiveFrom     string    `json:"effective_from"` // YYYY-MM-DD
	EffectiveTo       *string   `json:"effective_to,omitempty"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
}

// ── Labor cost analytics ──────────────────────────────────────────────────────

type ProjectLaborAllocation struct {
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	Hours       float64 `json:"hours"`
	Percentage  float64 `json:"percentage"`
	LaborCost   float64 `json:"labor_cost"`
}

type EmployeeLaborCost struct {
	EmployeeID            string                   `json:"employee_id"`
	EmployeeName          string                   `json:"employee_name"`
	EmployeeRole          string                   `json:"employee_role"`
	HasCompensation       bool                     `json:"has_compensation"`
	PayType               *string                  `json:"pay_type,omitempty"`
	PayAmount             *float64                 `json:"pay_amount,omitempty"`
	CompanyAnalyticalCost float64                  `json:"company_analytical_cost"`
	TotalHours            float64                  `json:"total_hours"`
	EffectiveCostPerHour  *float64                 `json:"effective_cost_per_hour,omitempty"`
	HasMidMonthTransition bool                     `json:"has_mid_month_transition"`
	ProjectAllocations    []ProjectLaborAllocation `json:"project_allocations"`
	Warning               *string                  `json:"warning,omitempty"`
}

type MonthlyLaborSummary struct {
	Year                         int                 `json:"year"`
	Month                        int                 `json:"month"`
	TotalKnownCost               float64             `json:"total_known_cost"`
	TotalHours                   float64             `json:"total_hours"`
	AvgCostPerHour               *float64            `json:"avg_cost_per_hour,omitempty"`
	EmployeesWithoutCompensation int                 `json:"employees_without_compensation"`
	Employees                    []EmployeeLaborCost `json:"employees"`
}

// ── Request types ─────────────────────────────────────────────────────────────

type CreateCompensationPlanReq struct {
	PayType           string   `json:"pay_type"`
	PayAmount         float64  `json:"pay_amount"`
	CompanyCostAmount *float64 `json:"company_cost_amount"`
	EffectiveFrom     string   `json:"effective_from"` // YYYY-MM-DD
}

type UpdateCompensationPlanReq struct {
	PayType           string   `json:"pay_type"`
	PayAmount         float64  `json:"pay_amount"`
	CompanyCostAmount *float64 `json:"company_cost_amount"`
	EffectiveFrom     string   `json:"effective_from"` // YYYY-MM-DD
	EffectiveTo       *string  `json:"effective_to"`   // YYYY-MM-DD or null
}
