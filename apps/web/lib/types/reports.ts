// ── Shared ────────────────────────────────────────────────────────────────────

export interface ReportPagination {
  page: number
  per_page: number
  total: number
  total_pages: number
}

export interface ReportFilter {
  project_id?: string
  poslovoda_id?: string
  worker_id?: string
  material_id?: string
  activity_type?: string
  is_vtk?: boolean
  date_from?: string
  date_to?: string
  status?: string
  search?: string
  page?: number
  per_page?: number
}

// ── Građevinski dnevnik ───────────────────────────────────────────────────────

export interface GradevinskiDnevnikRow {
  report_id: string
  report_date: string
  project_id: string
  project_name: string
  project_address: string | null
  poslovoda_id: string
  poslovoda_name: string
  worker_id: string
  worker_name: string
  hours_worked: number
  notes: string | null
  status: string
}

export interface GradevinskiDnevnikSummary {
  total_hours: number
  workers_count: number
  projects_count: number
}

export interface GradevinskiDnevnikResponse {
  data: GradevinskiDnevnikRow[]
  pagination: ReportPagination
  summary: GradevinskiDnevnikSummary
}

// ── Građevinska knjiga ────────────────────────────────────────────────────────

export interface GradevinskaKnjigaRow {
  report_id: string
  report_date: string
  project_id: string
  project_name: string
  project_address: string | null
  poslovoda_id: string
  poslovoda_name: string
  activity_id: string
  material_id: string | null
  material_name: string
  material_code: string | null
  custom_material_name: string | null
  quantity: number
  unit: string
  activity_type: string
  is_vtk: boolean
  notes: string | null
  status: string
}

export interface GradevinskaKnjigaSummary {
  total_quantity: number
  activities_count: number
  vtk_count: number
  projects_count: number
}

export interface GradevinskaKnjigaResponse {
  data: GradevinskaKnjigaRow[]
  pagination: ReportPagination
  summary: GradevinskaKnjigaSummary
}

// ── Filter options ────────────────────────────────────────────────────────────

export interface FilterOptionProject {
  id: string
  name: string
  status: string
}

export interface FilterOptionPerson {
  id: string
  full_name: string
  supervisor_id?: string
}

export interface FilterOptionMaterial {
  id: string
  material_name: string
  project_id: string
}

export interface ReportFilterOptions {
  projects: FilterOptionProject[]
  poslovode: FilterOptionPerson[]
  workers: FilterOptionPerson[]
  materials: FilterOptionMaterial[]
  activity_types: string[]
  statuses: string[]
}
