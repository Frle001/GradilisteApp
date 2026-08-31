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
}

export interface MonthlyLaborSummary {
  year: number
  month: number
  total_known_cost: number
  total_hours: number
  avg_cost_per_hour?: number
  employees_without_compensation: number
  employees: EmployeeLaborCost[]
}
