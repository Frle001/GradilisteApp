// ── Existing material CRUD types ──────────────────────────────────────────────

export type TrackingType = 'stock' | 'work'

export const TRACKING_TYPE_LABELS: Record<TrackingType, string> = {
  stock: 'Materijal',
  work:  'Radna aktivnost',
}

export interface MaterialListItem {
  id: string
  material_name: string
  material_code: string | null
  planned_quantity: number
  used_quantity: number
  available_quantity: number
  unit: string
  source: string | null
  active: boolean
  tracking_type: TrackingType
  created_at: string
  updated_at: string
}

export interface CreateMaterialPayload {
  material_name: string
  material_code?: string | null
  planned_quantity: number
  unit: string
  tracking_type: TrackingType
}

export interface UpdateMaterialPayload {
  material_name: string
  material_code?: string | null
  planned_quantity: number
  unit: string
  tracking_type: TrackingType
}

// ── Phase 6.5 wizard import types ────────────────────────────────────────────

/** Internal field names the backend understands. */
export type ImportField =
  | 'material_name'
  | 'quantity'
  | 'unit'
  | 'unit_price'
  | 'total_price'
  | 'category'
  | 'note'
  | 'position'
  | 'ignore'

/** Human-readable labels for each internal field. */
export const FIELD_LABELS: Record<ImportField, string> = {
  material_name: 'Naziv materijala',
  quantity: 'Količina',
  unit: 'Jedinica mjere',
  unit_price: 'Jedinična cijena',
  total_price: 'Ukupna cijena',
  category: 'Kategorija',
  note: 'Napomena',
  position: 'Pozicija / Rbr.',
  ignore: 'Ignoriraj',
}

/** Confidence thresholds for badge display — matches backend thresholds. */
export function confidenceLabel(conf: number): 'high' | 'medium' | 'low' | 'none' {
  if (conf >= 0.85) return 'high'
  if (conf >= 0.60) return 'medium'
  if (conf > 0) return 'low'
  return 'none'
}

// Step 1 — analyze response

export interface AnalyzeHeaderSuggestion {
  index: number
  original: string
  normalized: string
  suggested_field: ImportField | null
  confidence: number
  confidence_label: 'high' | 'medium' | 'low' | 'none'
  reasons: string[]
}

export interface AnalyzeSampleRow {
  row_number: number
  values: string[]
}

export interface ImportAnalyzeResponse {
  import_job_id: string
  sheet_name: string
  header_row_number: number
  headers: AnalyzeHeaderSuggestion[]
  sample_rows: AnalyzeSampleRow[]
}

// Step 2 — wizard preview

/** User-defined mapping: {originalExcelHeader: ImportField} */
export type ImportMapping = Record<string, ImportField>

export interface WizardPreviewRow {
  row_number: number
  material_name: string
  quantity: number
  unit: string
  unit_price: number | null
  total_price: number | null
  category: string | null
  note: string | null
  position: string | null
  source_row_numbers: number[]
  row_type: string
  warnings: string[]
  errors: string[]
  status: 'valid' | 'invalid'
}

export interface WizardPreviewSummary {
  total_rows: number
  valid_rows: number
  invalid_rows: number
  total_scanned: number
  skipped_categories: number
  skipped_continuations: number
  skipped_subtotals: number
}

export interface WizardPreviewResponse {
  rows: WizardPreviewRow[]
  summary: WizardPreviewSummary
}

// Step 3 — confirm

export interface ConfirmRow {
  row_number: number
  position?: string | null
  material_name: string
  quantity: number
  unit: string
  unit_price?: number | null
  total_price?: number | null
  category?: string | null
  note?: string | null
  include: boolean
}

export interface WizardConfirmRowError {
  row_number: number
  errors: string[]
}

export interface WizardConfirmResponse {
  imported_count: number
  skipped_count: number
  failed_count: number
  errors: WizardConfirmRowError[]
}

// ── Role helpers ──────────────────────────────────────────────────────────────

export function canImportMaterials(role: string): boolean {
  return role === 'direktor' || role === 'inzenjer'
}

export function canManageMaterials(role: string): boolean {
  return role === 'direktor' || role === 'inzenjer'
}
