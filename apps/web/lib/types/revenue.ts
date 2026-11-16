export interface ProjectContract {
  id: string
  project_id: string
  project_name: string
  contract_type: 'initial' | 'amendment'
  reference?: string
  contract_date?: string // YYYY-MM-DD
  net_value: number
  notes?: string
  created_at: string
}

export interface ProjectInvoice {
  id: string
  project_id: string
  project_name: string
  invoice_number: string
  invoice_date: string // YYYY-MM-DD
  due_date?: string
  net_amount: number
  vat_amount: number
  gross_amount: number
  paid_amount: number
  outstanding_amount: number
  status: 'issued' | 'partial' | 'paid' | 'cancelled'
  is_overdue: boolean
  cancels_invoice_id?: string
  notes?: string
  created_at: string
}

export interface InvoicePayment {
  id: string
  invoice_id: string
  payment_date: string
  amount: number
  notes?: string
  created_at: string
}

export interface MonthlyRevenueTrend {
  month: number
  invoiced_net: number
  invoiced_gross: number
  collected: number
  has_data: boolean
}

export interface ProjectRevenueRow {
  project_id: string
  project_name: string
  contract_value: number
  invoiced_net: number
  invoiced_gross: number
  collected: number
  outstanding: number
  invoice_count: number
  has_overdue: boolean
}

export interface ProjectRevenueSummary {
  year: number
  month: number
  total_contracted: number
  total_invoiced_net: number
  total_invoiced_gross: number
  total_collected: number
  total_outstanding: number
  total_overdue: number
  overdue_count: number
  monthly_trend: MonthlyRevenueTrend[]
  projects: ProjectRevenueRow[]
}

export interface ProjectRevenueDetail {
  project_id: string
  project_name: string
  year: number
  contract_value: number
  invoiced_net: number
  invoiced_gross: number
  collected: number
  outstanding: number
  contracts: ProjectContract[]
  invoices: ProjectInvoice[]
  monthly_trend: MonthlyRevenueTrend[]
}
