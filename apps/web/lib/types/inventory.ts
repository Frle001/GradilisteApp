// ── Inventory view ─────────────────────────────────────────────────────────────

export interface InventoryEmployeeSummary {
  id: string
  first_name: string
  last_name: string
  role: string
}

export interface InventoryAssetItem {
  id: string
  asset_type: string
  name: string
  quantity: number
  unit: string | null
  serial_number: string | null
  notes: string | null
  assigned_at: string
}

export interface InventoryMaterialItem {
  project_material_id: string
  material_name: string
  material_code: string | null
  project_id: string
  project_name: string
  total_quantity: number
  unit: string
}

export interface InventoryResponse {
  employee: InventoryEmployeeSummary
  assets: InventoryAssetItem[]
  materials: InventoryMaterialItem[]
}

// ── Transfer targets ───────────────────────────────────────────────────────────

export interface TransferTarget {
  id: string
  first_name: string
  last_name: string
  role: string
}

// ── Employee active projects ───────────────────────────────────────────────────

export interface InventoryProject {
  id: string
  name: string
}

// ── Create transfer ────────────────────────────────────────────────────────────

export interface CreateTransferRequest {
  to_employee_id: string
  type: 'asset' | 'material'
  employee_asset_id?: string
  project_material_id?: string
  destination_project_id?: string
  quantity?: number
  notes: string
}

// ── Transfer list ──────────────────────────────────────────────────────────────

export interface TransferListItem {
  id: string
  from_employee_id: string
  from_employee_name: string
  to_employee_id: string
  to_employee_name: string
  asset_type: string
  item_name: string
  quantity: number
  transferred_at: string
  notes: string | null
  transferred_by_name: string
}

export interface TransferPagination {
  page: number
  per_page: number
  total: number
  total_pages: number
}

export interface TransferListResponse {
  data: TransferListItem[]
  pagination: TransferPagination
}

// ── UI helpers ─────────────────────────────────────────────────────────────────

export const ASSET_TYPE_LABELS: Record<string, string> = {
  car: 'Vozilo',
  tool: 'Alat',
  equipment: 'Oprema',
  material: 'Materijal',
  other: 'Ostalo',
}

export const ROLE_LABELS: Record<string, string> = {
  direktor: 'Direktor',
  inzenjer: 'Inženjer',
  administracija: 'Administracija',
  poslovoda: 'Poslovoda',
  radnik: 'Radnik',
}

export function canViewOtherInventory(role: string): boolean {
  return ['direktor', 'inzenjer', 'administracija'].includes(role)
}

export function canTransfer(role: string): boolean {
  return ['direktor', 'inzenjer', 'administracija', 'poslovoda'].includes(role)
}

export function employeeFullName(emp: InventoryEmployeeSummary): string {
  return `${emp.first_name} ${emp.last_name}`
}
