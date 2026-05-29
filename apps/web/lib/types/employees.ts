export interface Employee {
  id: string
  first_name: string
  last_name: string
  role: string
  email: string | null
  phone: string | null
  active: boolean
  supervisor_id: string | null
  created_at: string
}

export interface SupervisorRef {
  id: string
  first_name: string
  last_name: string
  role: string
}

export interface EmployeeDetail extends Employee {
  company_id: string
  supervisor: SupervisorRef | null
  updated_at: string
}

export interface Asset {
  id: string
  employee_id: string
  asset_type: 'car' | 'tool' | 'equipment' | 'other'
  name: string
  quantity: number
  unit: string | null
  serial_number: string | null
  notes: string | null
  active: boolean
  assigned_at: string
}

export interface CreateEmployeePayload {
  first_name: string
  last_name: string
  role: string
  email?: string | null
  phone?: string | null
  supervisor_id?: string | null
}

export interface UpdateEmployeePayload extends CreateEmployeePayload {}

export interface CreateAssetPayload {
  asset_type: string
  name: string
  quantity: number
  unit?: string | null
  serial_number?: string | null
  notes?: string | null
}

export interface UpdateAssetPayload extends CreateAssetPayload {}

export const ROLE_LABELS: Record<string, string> = {
  direktor: 'Direktor',
  inzenjer: 'Inženjer',
  administracija: 'Administracija',
  poslovoda: 'Poslovođa',
  radnik: 'Radnik',
}

export const ASSET_TYPE_LABELS: Record<string, string> = {
  car: 'Vozilo',
  tool: 'Alat',
  equipment: 'Oprema',
  other: 'Ostalo',
}

export const VALID_ROLES = ['direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik']
export const VALID_ASSET_TYPES = ['car', 'tool', 'equipment', 'other']

export function canManageEmployees(role: string): boolean {
  return ['direktor', 'inzenjer', 'administracija'].includes(role)
}

export function fullName(e: { first_name: string; last_name: string }): string {
  return `${e.first_name} ${e.last_name}`
}
