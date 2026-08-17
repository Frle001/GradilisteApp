export type AssetType = 'alat' | 'oprema' | 'vozilo'
export type AssetStatus = 'active' | 'inactive'
export type NotificationKind = 'warranty' | 'registration' | 'leasing_warning' | 'leasing_overdue'
export type NotificationState = 'warning' | 'urgent' | 'expired'

export const ASSET_TYPE_LABELS: Record<AssetType, string> = {
  alat:   'Alat',
  oprema: 'Oprema',
  vozilo: 'Vozilo',
}

export const ASSET_STATUS_LABELS: Record<AssetStatus, string> = {
  active:   'Aktivno',
  inactive: 'Neaktivno',
}

export const ASSET_SIZES = ['XS', 'S', 'M', 'L', 'XL', 'XXL', '3XL', 'Univerzalna'] as const
export type AssetSize = typeof ASSET_SIZES[number]

export const NOTIFICATION_KIND_LEASING_WARNING: NotificationKind = 'leasing_warning'
export const NOTIFICATION_KIND_LEASING_OVERDUE: NotificationKind = 'leasing_overdue'

export interface AssetEmployee {
  id: string
  first_name: string
  last_name: string
  role: string
}

export interface CompanyAsset {
  id: string
  company_id: string
  asset_type: AssetType
  name: string
  status: AssetStatus
  notes: string | null
  assigned_employee: AssetEmployee | null
  purchased_at: string | null
  warranty_expires_at: string | null
  size: string | null
  registration_plate: string | null
  registration_date: string | null
  registration_expires_at: string | null
  // Leasing (vozilo only)
  is_leasing: boolean
  leasing_company: string | null
  leasing_end_date: string | null
  created_at: string
  updated_at: string
}

export interface AssetNotification {
  asset_id: string
  asset_name: string
  asset_type: AssetType
  kind: NotificationKind
  state: NotificationState
  days_remaining: number
  expires_at: string
  assigned_employee: AssetEmployee | null
  message: string
  // Leasing-specific (present only for leasing_warning / leasing_overdue)
  registration_plate?: string | null
  period_month?: string
}

export interface LeasingPayment {
  id: string
  asset_id: string
  period_month: string  // "YYYY-MM-01"
  completed_at: string
  completed_by: string
  created_at: string
}

export interface NotificationsResponse {
  notifications: AssetNotification[]
  count: number
}

export interface CreateAssetPayload {
  asset_type: AssetType
  name: string
  assigned_employee_id?: string | null
  notes?: string | null
  purchased_at?: string | null
  warranty_expires_at?: string | null
  size?: string | null
  registration_plate?: string | null
  registration_date?: string | null
  registration_expires_at?: string | null
  // Leasing
  is_leasing?: boolean
  leasing_company?: string | null
  leasing_end_date?: string | null
}

export interface UpdateAssetPayload {
  name: string
  assigned_employee_id?: string | null
  status?: string
  notes?: string | null
  purchased_at?: string | null
  warranty_expires_at?: string | null
  size?: string | null
  registration_plate?: string | null
  registration_date?: string | null
  registration_expires_at?: string | null
  // Leasing
  is_leasing?: boolean
  leasing_company?: string | null
  leasing_end_date?: string | null
}
