export interface DocEmployee {
  id: string
  first_name: string
  last_name: string
  role: string
  active: boolean
  created_at: string
}

export interface DocFile {
  id: string
  original_filename: string
  mime_type: string
  file_size: number
  uploaded_at: string
}

export interface MedicalExam {
  id: string
  employee_id: string
  completed_date: string
  expires_at: string
  document?: DocFile
  created_at: string
}

export interface SafetyRecord {
  id: string
  employee_id: string
  status: 'pending' | 'uploaded' | 'verified'
  document?: DocFile
  verified_at?: string
  created_at: string
}

export interface EmploymentContract {
  id: string
  employee_id: string
  contract_type: 'odredeno' | 'neodredeno'
  expires_at?: string
  document?: DocFile
  terminated_at?: string
  termination_document?: DocFile
  created_at: string
}

export interface MonthlyObligation {
  id: string
  employee_id: string
  year: number
  month: number
  obligation_type: 'platna_lista' | 'evidencija_radnika'
  status: 'pending' | 'completed'
  hours?: number
  document?: DocFile
  completed_at?: string
  created_at: string
}

export interface AnnualLeave {
  id: string
  employee_id: string
  date_from?: string
  date_to?: string
  document?: DocFile
  created_at: string
}

export interface WorkPermit {
  id: string
  employee_id: string
  entered_at: string
  expires_at: string
  document?: DocFile
  created_at: string
}

export interface PensionRecord {
  id: string
  employee_id: string
  status: 'pending' | 'uploaded' | 'verified'
  document?: DocFile
  verified_at?: string
  created_at: string
}

export interface HealthRecord {
  id: string
  employee_id: string
  status: 'pending' | 'uploaded' | 'verified'
  document?: DocFile
  verified_at?: string
  created_at: string
}

export interface DocNotification {
  type: string
  priority: 'low' | 'warning' | 'urgent' | 'overdue'
  employee_id: string
  employee_name: string
  record_id?: string
  message: string
  due_date?: string
  days_remaining?: number
  days_overdue?: number
  year?: number
  month?: number
  obligation_type?: string
}

export interface EmployeeDocSummary {
  employee: DocEmployee
  latest_medical?: MedicalExam
  safety_record?: SafetyRecord
  active_contract?: EmploymentContract
  latest_permit?: WorkPermit
  pension_record?: PensionRecord
  health_record?: HealthRecord
}
