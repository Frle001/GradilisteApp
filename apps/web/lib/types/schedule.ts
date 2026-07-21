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
  shift_date: string   // "YYYY-MM-DD"
  start_time: string   // "HH:MM"
  end_time: string     // "HH:MM"
  notes?: string
  status: 'active' | 'cancelled'
  cancelled_at?: string
  created_at: string
  assignments: ShiftAssignment[]
}

export interface ShiftOverlap {
  employee_id: string
  employee_name: string
  shift_id: string
  shift_date: string
  start_time: string
  end_time: string
}

export interface AssignResponse {
  assignments: ShiftAssignment[]
  overlaps?: ShiftOverlap[]
  requires_override: boolean
}

export interface CopyDayResponse {
  created: number
  skipped: number
  shifts: Shift[]
}
