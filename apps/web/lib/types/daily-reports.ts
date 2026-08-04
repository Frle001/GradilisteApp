// ── Enums / labels ────────────────────────────────────────────────────────────

export type ReportStatus = 'draft' | 'submitted' | 'approved' | 'rejected'
export type ActivityType = 'montaza' | 'demontaza' | 'other'

export const REPORT_STATUS_LABELS: Record<ReportStatus, string> = {
  draft: 'Nacrt',
  submitted: 'Predano',
  approved: 'Odobreno',
  rejected: 'Odbijeno',
}

export const ACTIVITY_TYPE_LABELS: Record<ActivityType, string> = {
  montaza: 'Montaža',
  demontaza: 'Demontaža',
  other: 'Ostalo',
}

// ── API response types ────────────────────────────────────────────────────────

export interface DailyReportListItem {
  id: string
  project_id: string
  project_name: string
  poslovoda_id: string
  poslovoda_name: string
  report_date: string
  status: ReportStatus
  worker_hours_count: number
  total_hours: number
  activities_count: number
  submitted_at: string | null
  created_at: string
  updated_at: string
}

export interface DailyReportProject {
  id: string
  name: string
  status: string
}

export interface DailyReportPerson {
  id: string
  first_name: string
  last_name: string
  full_name: string
}

export interface DailyReportWorkerHours {
  id: string
  worker_id: string
  worker_name: string
  hours_worked: number
  notes: string | null
}

export interface DailyReportActivity {
  id: string
  project_material_id: string | null
  material_name: string
  custom_material_name: string | null
  quantity: number
  unit: string
  activity_type: ActivityType
  is_vtk: boolean
  notes: string | null
  tracking_type: 'stock' | 'work'
}

export interface DailyReportAttachment {
  id: string
  original_name: string
  content_type: string
  file_size: number
  created_at: string
}

export interface DailyReportDetail {
  id: string
  project: DailyReportProject
  poslovoda: DailyReportPerson
  report_date: string
  status: ReportStatus
  notes: string | null
  submitted_at: string | null
  created_at: string
  updated_at: string
  worker_hours: DailyReportWorkerHours[]
  activities: DailyReportActivity[]
  attachments: DailyReportAttachment[]
}

// ── Form data types ───────────────────────────────────────────────────────────

export interface FormDataProject {
  id: string
  name: string
}

export interface FormDataWorker {
  id: string
  first_name: string
  last_name: string
  full_name: string
}

export interface FormDataMaterial {
  id: string
  material_name: string
  material_code: string | null
  available_quantity: number
  unit: string
  tracking_type: 'stock' | 'work'
}

export interface DailyReportFormData {
  projects: FormDataProject[]
  workers: FormDataWorker[]
  materials: FormDataMaterial[]
}

// ── Request payload types ─────────────────────────────────────────────────────

export interface WorkerHoursInput {
  worker_id: string
  hours_worked: number
  notes: string | null
}

export interface ActivityInput {
  project_material_id: string | null
  custom_material_name: string | null
  quantity: number
  unit: string
  activity_type: ActivityType
  is_vtk: boolean
  notes: string | null
  tracking_type?: 'stock' | 'work'
}

export interface CreateDailyReportPayload {
  project_id: string
  report_date: string
  notes: string | null
  worker_hours: WorkerHoursInput[]
  activities: ActivityInput[]
}

// ── UI-local types (used inside form components) ──────────────────────────────

/** Per-worker input row inside WorkerHoursEditor */
export interface WorkerHoursEntry {
  worker_id: string
  worker_name: string
  hours: string   // raw string input
  notes: string
}

/** Activity row inside the preview list (has a temp id for keying) */
export interface ActivityInputUI extends ActivityInput {
  _tempId: string
  _displayName: string  // resolved material name for display
  tracking_type?: 'stock' | 'work'
}

// ── Permission helpers ────────────────────────────────────────────────────────

export function canCreateDailyReport(role: string): boolean {
  return role === 'poslovoda' || role === 'direktor' || role === 'inzenjer'
}

export function canEditDailyReport(role: string, callerEmpId: string | null, poslovodaId: string, status: ReportStatus): boolean {
  if (status === 'approved') return false
  if (role === 'direktor' || role === 'inzenjer') return true
  if (role === 'poslovoda') return callerEmpId === poslovodaId
  return false
}

export function canApproveReject(role: string): boolean {
  return role === 'direktor' || role === 'inzenjer'
}
