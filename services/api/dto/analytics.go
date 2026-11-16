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
	// Contributions fields — Bruto = Neto + Contributions (application-defined formula).
	Neto                 float64  `json:"neto"`
	Contributions        *float64 `json:"contributions"`          // nil = not entered; 0 = explicitly zero
	Bruto                *float64 `json:"bruto"`                  // nil when neto unavailable or contributions not entered
	ContributionsMissing bool     `json:"contributions_missing"`  // true when no contribution record exists
}

type MonthlyLaborSummary struct {
	Year                         int                 `json:"year"`
	Month                        int                 `json:"month"`
	TotalKnownCost               float64             `json:"total_known_cost"`
	TotalHours                   float64             `json:"total_hours"`
	AvgCostPerHour               *float64            `json:"avg_cost_per_hour,omitempty"`
	EmployeesWithoutCompensation int                 `json:"employees_without_compensation"`
	Employees                    []EmployeeLaborCost `json:"employees"`
	// Contribution totals
	TotalNeto                     float64 `json:"total_neto"`
	TotalContributions            float64 `json:"total_contributions"`
	TotalBruto                    float64 `json:"total_bruto"`
	EmployeesMissingContributions int     `json:"employees_missing_contributions"`
	BrutoComplete                 bool    `json:"bruto_complete"`
}

// ── Contribution types ────────────────────────────────────────────────────────

type UpsertContributionReq struct {
	ContributionsAmount float64 `json:"contributions_amount"`
}

type EmployeeContribution struct {
	ID                  string    `json:"id"`
	EmployeeID          string    `json:"employee_id"`
	PayrollMonth        string    `json:"payroll_month"` // YYYY-MM-01
	ContributionsAmount float64   `json:"contributions_amount"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type MonthlyPayrollEntry struct {
	Year                 int      `json:"year"`
	Month                int      `json:"month"`
	Neto                 float64  `json:"neto"`
	Contributions        *float64 `json:"contributions"`
	Bruto                *float64 `json:"bruto"`
	ContributionsMissing bool     `json:"contributions_missing"`
	HasCompensation      bool     `json:"has_compensation"`
	TotalHours           float64  `json:"total_hours"`
	Warning              *string  `json:"warning,omitempty"`
}

type EmployeePayrollHistory struct {
	EmployeeID   string                `json:"employee_id"`
	EmployeeName string                `json:"employee_name"`
	Year         int                   `json:"year"`
	TotalNeto    float64               `json:"total_neto"`
	TotalContrib float64               `json:"total_contributions"`
	TotalBruto   float64               `json:"total_bruto"`
	Months       []MonthlyPayrollEntry `json:"months"`
}

// ── Annual payroll summary ────────────────────────────────────────────────────

// MonthlyPayrollAggregate holds aggregated payroll data for one month across
// all (or a filtered) employees.
type MonthlyPayrollAggregate struct {
	Year             int     `json:"year"`
	Month            int     `json:"month"`
	TotalNeto        float64 `json:"total_neto"`
	TotalContributions float64 `json:"total_contributions"`
	TotalBruto       float64 `json:"total_bruto"`
	TotalHours       float64 `json:"total_hours"`
	EmployeeCount    int     `json:"employee_count"`
	IncompleteCount  int     `json:"incomplete_count"` // employees with contributions_missing
	HasData          bool    `json:"has_data"`
}

// AnnualPayrollSummary holds 12 monthly aggregates plus year-level totals.
type AnnualPayrollSummary struct {
	Year                     int                      `json:"year"`
	TotalNeto                float64                  `json:"total_neto"`
	TotalContributions       float64                  `json:"total_contributions"`
	TotalBruto               float64                  `json:"total_bruto"`
	TotalHours               float64                  `json:"total_hours"`
	TotalEmployees           int                      `json:"total_employees"`
	BrutoComplete            bool                     `json:"bruto_complete"`
	MonthsWithIncompleteData int                      `json:"months_with_incomplete_data"`
	EmployeeFilter           *string                  `json:"employee_filter,omitempty"`
	FilteredEmployeeName     *string                  `json:"filtered_employee_name,omitempty"`
	Months                   []MonthlyPayrollAggregate `json:"months"`
}

// ── Project labor allocation ──────────────────────────────────────────────────

type MonthlyProjectAllocation struct {
	ProjectID               string  `json:"project_id"`
	ProjectName             string  `json:"project_name"`
	TotalHours              float64 `json:"total_hours"`
	LaborCost               float64 `json:"labor_cost"` // = NetoCost, kept for backward compat
	NetoCost                float64 `json:"neto_cost"`
	ContributionsCost       float64 `json:"contributions_cost"`
	BrutoCost               float64 `json:"bruto_cost"`
	MaterialCost            float64 `json:"material_cost"`
	TotalDirectCost         float64 `json:"total_direct_cost"` // BrutoCost + MaterialCost
	EmployeeCount           int     `json:"employee_count"`
	PctOfTotal              float64 `json:"pct_of_total"`
	ContributionsIncomplete bool    `json:"contributions_incomplete"`
}

type MonthlyProjectAllocationSummary struct {
	Year               int                        `json:"year"`
	Month              int                        `json:"month"`
	TotalHours         float64                    `json:"total_hours"`
	TotalLaborCost     float64                    `json:"total_labor_cost"` // = TotalNeto, backward compat
	TotalNeto          float64                    `json:"total_neto"`
	TotalContributions float64                    `json:"total_contributions"`
	TotalBruto         float64                    `json:"total_bruto"`
	TotalMaterialCost  float64                    `json:"total_material_cost"`
	TotalDirectCost    float64                    `json:"total_direct_cost"` // TotalBruto + TotalMaterialCost
	UnassignedHours    float64                    `json:"unassigned_hours"`
	BrutoComplete      bool                       `json:"bruto_complete"`
	Allocations        []MonthlyProjectAllocation `json:"allocations"`
}

// ── Project detail analytics ──────────────────────────────────────────────────

type ProjectEmployeeAllocation struct {
	EmployeeID           string   `json:"employee_id"`
	EmployeeName         string   `json:"employee_name"`
	Hours                float64  `json:"hours"`
	PctOfProject         float64  `json:"pct_of_project"`
	NetoCost             float64  `json:"neto_cost"`
	ContributionsCost    *float64 `json:"contributions_cost"` // nil = missing
	BrutoCost            *float64 `json:"bruto_cost"`
	ContributionsMissing bool     `json:"contributions_missing"`
}

type ProjectMonthlyTrend struct {
	Month         int     `json:"month"`
	Hours         float64 `json:"hours"`
	Neto          float64 `json:"neto"`
	Contributions float64 `json:"contributions"`
	Bruto         float64 `json:"bruto"`
	MaterialCost  float64 `json:"material_cost"`
	TotalCost     float64 `json:"total_cost"` // Bruto + MaterialCost
	HasData       bool    `json:"has_data"`
	Incomplete    bool    `json:"incomplete"`
}

// ProjectMaterialCostItem holds per-material cost data for a project in a given month.
type ProjectMaterialCostItem struct {
	ProjectMaterialID string  `json:"project_material_id"`
	MaterialName      string  `json:"material_name"`
	MaterialCode      *string `json:"material_code"`
	InstalledQty      float64 `json:"installed_qty"`
	Unit              string  `json:"unit"`
	UnitPrice         float64 `json:"unit_price"`
	TotalCost         float64 `json:"total_cost"`
	Source            string  `json:"source"`
}

type ProjectDetailSummary struct {
	ProjectID          string                      `json:"project_id"`
	ProjectName        string                      `json:"project_name"`
	Year               int                         `json:"year"`
	Month              int                         `json:"month"`
	TotalHours         float64                     `json:"total_hours"`
	TotalNeto          float64                     `json:"total_neto"`
	TotalContributions float64                     `json:"total_contributions"`
	TotalBruto         float64                     `json:"total_bruto"`
	TotalMaterialCost  float64                     `json:"total_material_cost"`
	TotalDirectCost    float64                     `json:"total_direct_cost"` // TotalBruto + TotalMaterialCost
	BrutoComplete      bool                        `json:"bruto_complete"`
	Employees          []ProjectEmployeeAllocation `json:"employees"`
	Materials          []ProjectMaterialCostItem   `json:"materials"`
	MonthlyTrend       []ProjectMonthlyTrend       `json:"monthly_trend"`
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
