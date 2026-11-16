export const PAY_TYPE_LABELS: Record<string, string> = {
  fixed_monthly: 'Fiksna mjesečna plaća',
  hourly: 'Plaćanje po satu',
}

export interface CompensationPlan {
  id: string
  employee_id: string
  employee_name: string
  pay_type: string
  pay_amount: number
  company_cost_amount?: number
  effective_from: string   // YYYY-MM-DD
  effective_to?: string    // YYYY-MM-DD or absent if open-ended
  created_by: string
  created_at: string
}

export interface ProjectLaborAllocation {
  project_id: string
  project_name: string
  hours: number
  percentage: number
  labor_cost: number
}

export interface EmployeeLaborCost {
  employee_id: string
  employee_name: string
  employee_role: string
  has_compensation: boolean
  pay_type?: string
  pay_amount?: number
  company_analytical_cost: number
  total_hours: number
  effective_cost_per_hour?: number
  has_mid_month_transition: boolean
  project_allocations: ProjectLaborAllocation[]
  warning?: string
  // Contribution fields
  neto: number
  contributions?: number    // undefined = not entered; 0 = explicitly zero
  bruto?: number            // undefined when neto unavailable or contributions missing
  contributions_missing: boolean
}

export interface MonthlyLaborSummary {
  year: number
  month: number
  total_known_cost: number
  total_hours: number
  avg_cost_per_hour?: number
  employees_without_compensation: number
  employees: EmployeeLaborCost[]
  // Contribution totals
  total_neto: number
  total_contributions: number
  total_bruto: number
  employees_missing_contributions: number
  bruto_complete: boolean
}

export interface EmployeeContribution {
  id: string
  employee_id: string
  payroll_month: string
  contributions_amount: number
  updated_at: string
}

export interface MonthlyPayrollEntry {
  year: number
  month: number
  neto: number
  contributions?: number
  bruto?: number
  contributions_missing: boolean
  has_compensation: boolean
  total_hours: number
  warning?: string
}

export interface EmployeePayrollHistory {
  employee_id: string
  employee_name: string
  year: number
  total_neto: number
  total_contributions: number
  total_bruto: number
  months: MonthlyPayrollEntry[]
}

export interface MonthlyPayrollAggregate {
  year: number
  month: number
  total_neto: number
  total_contributions: number
  total_bruto: number
  total_hours: number
  employee_count: number
  incomplete_count: number
  has_data: boolean
}

export interface AnnualPayrollSummary {
  year: number
  total_neto: number
  total_contributions: number
  total_bruto: number
  total_hours: number
  total_employees: number
  bruto_complete: boolean
  months_with_incomplete_data: number
  employee_filter?: string
  filtered_employee_name?: string
  months: MonthlyPayrollAggregate[]
}

export interface MonthlyProjectAllocation {
  project_id: string
  project_name: string
  total_hours: number
  labor_cost: number           // = neto_cost, kept for backward compat
  neto_cost: number
  contributions_cost: number
  bruto_cost: number
  material_cost: number
  total_direct_cost: number    // bruto_cost + material_cost
  employee_count: number
  pct_of_total: number
  contributions_incomplete: boolean
}

export interface MonthlyProjectAllocationSummary {
  year: number
  month: number
  total_hours: number
  total_labor_cost: number    // = total_neto, backward compat
  total_neto: number
  total_contributions: number
  total_bruto: number
  total_material_cost: number
  total_direct_cost: number   // total_bruto + total_material_cost
  unassigned_hours: number
  bruto_complete: boolean
  allocations: MonthlyProjectAllocation[]
}

export interface ProjectEmployeeAllocation {
  employee_id: string
  employee_name: string
  hours: number
  pct_of_project: number
  neto_cost: number
  contributions_cost?: number  // undefined = missing
  bruto_cost?: number
  contributions_missing: boolean
}

export interface ProjectMonthlyTrend {
  month: number
  hours: number
  neto: number
  contributions: number
  bruto: number
  material_cost: number
  total_cost: number          // bruto + material_cost
  has_data: boolean
  incomplete: boolean
}

export interface ProjectMaterialCostItem {
  project_material_id: string
  material_name: string
  material_code?: string
  installed_qty: number
  unit: string
  unit_price: number
  total_cost: number
  source: string
}

export interface ProjectDetailSummary {
  project_id: string
  project_name: string
  year: number
  month: number
  total_hours: number
  total_neto: number
  total_contributions: number
  total_bruto: number
  total_material_cost: number
  total_direct_cost: number   // total_bruto + total_material_cost
  bruto_complete: boolean
  employees: ProjectEmployeeAllocation[]
  materials: ProjectMaterialCostItem[]
  monthly_trend: ProjectMonthlyTrend[]
}

// ── Hladni pogon ──────────────────────────────────────────────────────────────

export interface ColdOpsCategory {
  id: string
  name: string
  description?: string
  color?: string
  is_active: boolean
  created_at: string
}

export interface ColdOpsExpense {
  id: string
  category_id?: string
  category_name?: string
  amount: number
  expense_date: string
  title: string
  notes?: string
  supplier?: string
  payment_method?: string
  expense_type: 'fixed' | 'variable'
  created_by: string
  created_at: string
}

export interface RecurringDef {
  id: string
  category_id?: string
  category_name?: string
  title: string
  frequency: 'monthly' | 'yearly'
  day_of_month: number
  month_of_year?: number
  start_date: string
  end_date?: string
  supplier?: string
  payment_method?: string
  notes?: string
  is_active: boolean
  current_amount?: number
  created_at: string
}

export interface ColdOpsOccurrence {
  id: string
  definition_id: string
  definition_title: string
  occurrence_key: string
  due_date: string
  status: 'planned' | 'confirmed' | 'skipped'
  amount?: number
  expense_id?: string
  notes?: string
  handled_by?: string
  handled_at?: string
}

export interface ColdOpsBudget {
  id: string
  category_id?: string
  category_name?: string
  year: number
  month: number
  budget_amount: number
  actual_amount: number
  utilization_pct: number
}

export interface ColdOpsPeriodStats {
  total: number
  count: number
  pct_change?: number
}

export interface ColdOpsTrendPoint {
  year: number
  month: number
  total: number
  count: number
}

export interface ColdOpsCategoryStats {
  category_id?: string
  category_name: string
  color?: string
  total: number
  count: number
  pct: number
}

export interface ColdOpsFixedVariable {
  fixed_total: number
  variable_total: number
  fixed_count: number
  variable_count: number
}

export interface ColdOpsForecastPoint {
  year: number
  month: number
  confirmed_amount: number
  forecast_amount: number
  is_projection: boolean
}

export interface ColdOpsBudgetUtil {
  category_id?: string
  category_name: string
  budget_amount: number
  actual_amount: number
  utilization_pct: number
  is_over_budget: boolean
}

export interface ColdOpsAnnualSummary {
  year: number
  total: number
  count: number
  months: ColdOpsTrendPoint[]
}

export interface ColdOpsSummary {
  current_month: ColdOpsPeriodStats
  prev_month: ColdOpsPeriodStats
  current_year: ColdOpsPeriodStats
  prev_year: ColdOpsPeriodStats
  monthly_trend: ColdOpsTrendPoint[]
  category_distribution: ColdOpsCategoryStats[]
  fixed_vs_variable: ColdOpsFixedVariable
  upcoming_occurrences: ColdOpsOccurrence[]
  forecast: ColdOpsForecastPoint[]
  budget_utilization: ColdOpsBudgetUtil[]
  annualized_commitment: number
}
