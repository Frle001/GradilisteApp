export type ProjectStatus = 'active' | 'closed' | 'archived'

export interface ProjectListItem {
  id: string
  name: string
  address: string | null
  description: string | null
  status: ProjectStatus
  start_date: string | null
  end_date: string | null
  closed_at: string | null
  created_at: string
  updated_at: string
  primary_poslovoda_id: string | null
  primary_poslovoda_name: string | null
  assignment_count: number
}

export interface ProjectAssignment {
  id: string
  employee_id: string
  employee_name: string
  employee_role: string
  role_on_project: string
  active: boolean
  assigned_at: string
}

export interface ProjectDetail extends ProjectListItem {
  company_id: string
  created_by: string | null
  closed_by: string | null
  materials_count: number
  daily_reports_count: number
  assignments: ProjectAssignment[]
}

export interface CreateProjectPayload {
  name: string
  address?: string | null
  description?: string | null
  start_date?: string | null
  end_date?: string | null
  primary_poslovoda_id?: string | null
}

export interface UpdateProjectPayload {
  name: string
  address?: string | null
  description?: string | null
  start_date?: string | null
  end_date?: string | null
}

export const STATUS_LABELS: Record<ProjectStatus, string> = {
  active: 'Aktivno',
  closed: 'Zatvoreno',
  archived: 'Arhivirano',
}

export function canManageProjects(role: string): boolean {
  return role === 'direktor' || role === 'inzenjer'
}

export function canViewProjects(role: string): boolean {
  return ['direktor', 'inzenjer', 'poslovoda', 'radnik'].includes(role)
}

export function canViewArchive(role: string): boolean {
  return ['direktor', 'inzenjer'].includes(role)
}

// ── Archive list ──────────────────────────────────────────────────────────────

export interface ArchiveListItem {
  id: string
  name: string
  address: string | null
  status: ProjectStatus
  start_date: string | null
  end_date: string | null
  closed_at: string | null
  closed_by_name: string | null
  primary_poslovoda_name: string | null
  daily_reports_count: number
  material_purchases_count: number
  project_materials_count: number
}

export interface ArchiveListResponse {
  projects: ArchiveListItem[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

// ── Archive summary ───────────────────────────────────────────────────────────

export interface ArchiveSummaryAssignee {
  id: string
  name: string
  active: boolean
}

export interface ArchiveSummary {
  id: string
  name: string
  address: string | null
  status: ProjectStatus
  start_date: string | null
  end_date: string | null
  closed_at: string | null
  closed_by_name: string | null
  daily_reports_count: number
  worker_hours_total: number
  activities_count: number
  vtk_count: number
  purchase_sessions_count: number
  purchase_items_count: number
  project_materials_count: number
  material_responsibility_count: number
  transfers_count: number
  assigned_poslovode: ArchiveSummaryAssignee[]
}
