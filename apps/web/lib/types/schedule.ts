export interface ShiftAssignment {
  id: string
  employee_id: string
  employee_name: string
  overlap_overridden: boolean
}

export interface Shift {
  id: string
  project_id: string
  project_name: string
  shift_date: string    // "YYYY-MM-DD"
  start_time?: string   // "HH:MM" — optional; may be absent after migration 018
  end_time?: string     // "HH:MM" — optional; may be absent after migration 018
  notes?: string
  status: 'active' | 'cancelled'
  cancelled_at?: string
  created_at: string
  assignments: ShiftAssignment[]
}

export interface ShiftConflict {
  employee_id: string
  employee_name: string
  shift_id: string
  project_name: string
}

export interface AssignResponse {
  assignments: ShiftAssignment[]
  requires_override: boolean
}

export interface CopyDayResponse {
  created: number
  skipped: number
  shifts: Shift[]
}

export interface EmployeeForDate {
  id: string
  name: string
  role: string
  assigned: boolean
  shift_id?: string
  project_name?: string
}
