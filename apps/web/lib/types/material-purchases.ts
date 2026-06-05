// ── List ──────────────────────────────────────────────────────────────────────

export interface PurchaseListItem {
  id: string
  project_id: string
  project_name: string
  buyer_id: string
  buyer_name: string
  purchased_at: string
  items_count: number
  receipt_file_url: string | null
  receipt_original_filename: string | null
  notes: string | null
  created_at: string
}

export interface PurchasePagination {
  page: number
  per_page: number
  total: number
  total_pages: number
}

export interface PurchaseListResponse {
  data: PurchaseListItem[]
  pagination: PurchasePagination
}

// ── Detail ────────────────────────────────────────────────────────────────────

export interface PurchaseItemDetail {
  id: string
  project_material_id: string
  material_name: string
  material_code: string | null
  quantity: number
  unit: string
}

export interface PurchaseDetail {
  id: string
  project: { id: string; name: string; address?: string }
  buyer: { id: string; full_name: string }
  purchased_at: string
  receipt_file_url: string | null
  receipt_original_filename: string | null
  notes: string | null
  items: PurchaseItemDetail[]
  created_at: string
}

// ── Form data ─────────────────────────────────────────────────────────────────

export interface PurchaseFormProject {
  id: string
  name: string
}

export interface PurchaseFormMaterial {
  id: string
  material_name: string
  material_code: string | null
  available_quantity: number
  unit: string
}

export interface PurchaseFormDataResponse {
  projects: PurchaseFormProject[]
  materials: PurchaseFormMaterial[]
}

// ── Local form state ──────────────────────────────────────────────────────────

export interface PurchaseItemDraft {
  _tempId: string
  project_material_id: string
  material_name: string
  material_code: string | null
  quantity: number
  unit: string
}

// ── Filter ────────────────────────────────────────────────────────────────────

export interface PurchaseFilter {
  project_id?: string
  buyer_id?: string
  date_from?: string
  date_to?: string
  search?: string
  page?: number
  per_page?: number
}
