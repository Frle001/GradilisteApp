'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import apiClient from '@/lib/api-client'
import {
  type CompanyAsset,
  type AssetEmployee,
  type AssetNotification,
  type AssetType,
  type LeasingPayment,
  type CreateAssetPayload,
  type UpdateAssetPayload,
  ASSET_TYPE_LABELS,
  ASSET_SIZES,
  NOTIFICATION_KIND_LEASING_WARNING,
  NOTIFICATION_KIND_LEASING_OVERDUE,
} from '@/lib/types/company-assets'

type Tab = 'pregled' | 'alat' | 'oprema' | 'vozila' | 'obavijesti'

const TABS: { id: Tab; label: string }[] = [
  { id: 'pregled',     label: 'Pregled' },
  { id: 'alat',       label: 'Alat' },
  { id: 'oprema',     label: 'Oprema' },
  { id: 'vozila',     label: 'Vozila' },
  { id: 'obavijesti', label: 'Obavijesti' },
]

interface FormState {
  asset_type: AssetType
  name: string
  assigned_employee_id: string
  notes: string
  purchased_at: string
  warranty_expires_at: string
  size: string
  registration_plate: string
  registration_date: string
  registration_expires_at: string
  status: string
  is_leasing: boolean
  leasing_company: string
  leasing_end_date: string
}

const emptyForm = (): FormState => ({
  asset_type: 'alat',
  name: '',
  assigned_employee_id: '',
  notes: '',
  purchased_at: '',
  warranty_expires_at: '',
  size: 'M',
  registration_plate: '',
  registration_date: '',
  registration_expires_at: '',
  status: 'active',
  is_leasing: false,
  leasing_company: '',
  leasing_end_date: '',
})

function addOneYear(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  d.setFullYear(d.getFullYear() + 1)
  return d.toISOString().split('T')[0]
}

// ── Date / expiry helpers ─────────────────────────────────────────────────────

function formatDate(s: string | null | undefined): string {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('hr-HR')
}

function formatPeriodMonth(s: string | null | undefined): string {
  if (!s) return ''
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('hr-HR', { month: 'long', year: 'numeric' })
}

function employeeLabel(e: AssetEmployee | null | undefined): string {
  if (!e) return 'Nezaduženo'
  return `${e.first_name} ${e.last_name}`
}

function isLeasingKind(kind: string): boolean {
  return kind === NOTIFICATION_KIND_LEASING_WARNING || kind === NOTIFICATION_KIND_LEASING_OVERDUE
}

type ExpiryStatus = 'expired' | 'soon' | 'ok' | 'none'

function expiryStatus(dateStr: string | null | undefined): ExpiryStatus {
  if (!dateStr) return 'none'
  const expiry = new Date(dateStr)
  if (isNaN(expiry.getTime())) return 'none'
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const diffDays = Math.floor((expiry.getTime() - today.getTime()) / 86400000)
  if (diffDays < 0) return 'expired'
  if (diffDays <= 10) return 'soon'
  return 'ok'
}

// ── Visual components ─────────────────────────────────────────────────────────

function ExpiryBadge({ dateStr, label }: { dateStr: string | null | undefined; label?: string }) {
  const status = expiryStatus(dateStr)
  if (status === 'none') return <span className="text-gi-text-subtle">—</span>

  const display = formatDate(dateStr)
  if (status === 'ok') {
    return (
      <span className="text-gi-text-muted text-xs">
        {label ? `${label}: ` : ''}{display}
      </span>
    )
  }
  if (status === 'soon') {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-amber-300">
        <span className="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0" />
        {display}
      </span>
    )
  }
  // expired
  return (
    <span className="inline-flex items-center gap-1 text-xs text-red-400">
      <span className="w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />
      {display}
    </span>
  )
}

function NotificationStateBadge({ state }: { state: string }) {
  if (state === 'warning')
    return <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-amber-900/40 text-amber-300 border border-amber-700/40">Upozorenje</span>
  if (state === 'urgent')
    return <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-orange-900/40 text-orange-300 border border-orange-700/40">Hitno</span>
  if (state === 'expired')
    return <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-red-950/50 text-red-400 border border-red-800/50">Isteklo</span>
  return null
}

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
  'text-sm text-gi-text placeholder:text-gi-text-subtle',
  'focus:outline-none focus:border-gi-yellow transition-colors',
].join(' ')

// ── Main component ────────────────────────────────────────────────────────────

export default function ZaduzenjaPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [activeTab, setActiveTab]           = useState<Tab>('pregled')
  const [assets, setAssets]                 = useState<CompanyAsset[]>([])
  const [notifications, setNotifications]   = useState<AssetNotification[]>([])
  const [employees, setEmployees]           = useState<AssetEmployee[]>([])
  const [dataLoading, setDataLoading]       = useState(true)
  const [error, setError]                   = useState<string | null>(null)
  const [search, setSearch]                 = useState('')

  const [modalOpen, setModalOpen]           = useState(false)
  const [editAsset, setEditAsset]           = useState<CompanyAsset | null>(null)
  const [form, setForm]                     = useState<FormState>(emptyForm())
  const [formError, setFormError]           = useState<string | null>(null)
  const [saving, setSaving]                 = useState(false)

  const [leasingHistory, setLeasingHistory]   = useState<LeasingPayment[]>([])
  const [fetchingHistory, setFetchingHistory] = useState(false)

  const [confirmDeactivate, setConfirmDeactivate] = useState<CompanyAsset | null>(null)
  const [deactivating, setDeactivating]           = useState(false)
  const [markingLeasing, setMarkingLeasing]       = useState<string | null>(null)

  useEffect(() => {
    if (!isLoading && user?.role !== 'administracija' && user?.role !== 'direktor') router.replace('/dashboard')
  }, [isLoading, user, router])

  const fetchAssets = useCallback(async () => {
    setDataLoading(true); setError(null)
    try {
      const [assetsRes, notifRes] = await Promise.all([
        apiClient.get('/company-assets'),
        apiClient.get('/company-assets/notifications'),
      ])
      setAssets(assetsRes.data.assets ?? [])
      setNotifications(notifRes.data.notifications ?? [])
    } catch {
      setError('Greška pri učitavanju zaduženja.')
    } finally {
      setDataLoading(false)
    }
  }, [])

  const fetchEmployees = useCallback(async () => {
    try {
      const res = await apiClient.get('/company-assets/employees')
      setEmployees(res.data.employees ?? [])
    } catch { /* non-critical */ }
  }, [])

  useEffect(() => {
    if (!isLoading && (user?.role === 'administracija' || user?.role === 'direktor')) {
      fetchAssets()
      fetchEmployees()
    }
  }, [isLoading, user, fetchAssets, fetchEmployees])

  useEffect(() => {
    if (modalOpen && editAsset?.is_leasing) {
      setFetchingHistory(true)
      apiClient.get(`/company-assets/${editAsset.id}/leasing-payments`)
        .then(res => setLeasingHistory(res.data.payments ?? []))
        .catch(() => setLeasingHistory([]))
        .finally(() => setFetchingHistory(false))
    } else {
      setLeasingHistory([])
    }
  }, [modalOpen, editAsset?.id, editAsset?.is_leasing])

  if (isLoading) return <LoadingScreen />
  if (!user || (user.role !== 'administracija' && user.role !== 'direktor')) return null

  // ── Computed ──────────────────────────────────────────────────────────────

  const activeAssets  = assets.filter(a => a.status === 'active')
  const alatCount     = activeAssets.filter(a => a.asset_type === 'alat').length
  const opremaCount   = activeAssets.filter(a => a.asset_type === 'oprema').length
  const vozilaCount   = activeAssets.filter(a => a.asset_type === 'vozilo').length
  const paznjCount    = notifications.length

  function filteredAssets(type: AssetType): CompanyAsset[] {
    const lc = search.toLowerCase()
    return assets.filter(a => {
      if (a.asset_type !== type) return false
      if (!lc) return true
      return (
        a.name.toLowerCase().includes(lc) ||
        employeeLabel(a.assigned_employee).toLowerCase().includes(lc) ||
        (a.registration_plate ?? '').toLowerCase().includes(lc)
      )
    })
  }

  // ── Modal helpers ─────────────────────────────────────────────────────────

  function openCreate() {
    setEditAsset(null); setForm(emptyForm()); setFormError(null); setModalOpen(true)
  }

  function openEdit(asset: CompanyAsset) {
    setEditAsset(asset)
    setForm({
      asset_type: asset.asset_type,
      name: asset.name,
      assigned_employee_id: asset.assigned_employee?.id ?? '',
      notes: asset.notes ?? '',
      purchased_at: asset.purchased_at ?? '',
      warranty_expires_at: asset.warranty_expires_at ?? '',
      size: asset.size ?? 'M',
      registration_plate: asset.registration_plate ?? '',
      registration_date: asset.registration_date ?? '',
      registration_expires_at: asset.registration_expires_at ?? '',
      status: asset.status,
      is_leasing: asset.is_leasing,
      leasing_company: asset.leasing_company ?? '',
      leasing_end_date: asset.leasing_end_date ?? '',
    })
    setFormError(null); setModalOpen(true)
  }

  function closeModal() { setModalOpen(false); setEditAsset(null) }

  function setField<K extends keyof FormState>(k: K, v: FormState[K]) {
    setForm(prev => {
      const next = { ...prev, [k]: v }
      if (k === 'registration_date' && !editAsset) {
        next.registration_expires_at = addOneYear(v as string)
      }
      return next
    })
  }

  async function handleSave() {
    setFormError(null)
    if (!form.name.trim()) { setFormError('Naziv je obavezan.'); return }
    setSaving(true)
    try {
      if (editAsset) {
        const payload: UpdateAssetPayload = {
          name: form.name.trim(),
          assigned_employee_id: form.assigned_employee_id || null,
          status: form.status,
          notes: form.notes || null,
          purchased_at: form.purchased_at || null,
          warranty_expires_at: form.warranty_expires_at || null,
          size: form.size || null,
          registration_plate: form.registration_plate || null,
          registration_date: form.registration_date || null,
          registration_expires_at: form.registration_expires_at || null,
          is_leasing: form.is_leasing,
          leasing_company: form.leasing_company || null,
          leasing_end_date: form.leasing_end_date || null,
        }
        await apiClient.patch(`/company-assets/${editAsset.id}`, payload)
      } else {
        const payload: CreateAssetPayload = {
          asset_type: form.asset_type,
          name: form.name.trim(),
          assigned_employee_id: form.assigned_employee_id || null,
          notes: form.notes || null,
          purchased_at: form.purchased_at || null,
          warranty_expires_at: form.warranty_expires_at || null,
          size: form.size || null,
          registration_plate: form.registration_plate || null,
          registration_date: form.registration_date || null,
          registration_expires_at: form.registration_expires_at || null,
          is_leasing: form.is_leasing,
          leasing_company: form.leasing_company || null,
          leasing_end_date: form.leasing_end_date || null,
        }
        await apiClient.post('/company-assets', payload)
      }
      closeModal()
      await fetchAssets()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setFormError(e?.response?.data?.error ?? 'Greška pri spremanju.')
    } finally {
      setSaving(false)
    }
  }

  async function handleDeactivate() {
    if (!confirmDeactivate) return
    setDeactivating(true)
    try {
      await apiClient.delete(`/company-assets/${confirmDeactivate.id}`)
      setConfirmDeactivate(null)
      await fetchAssets()
    } catch {
      setError('Greška pri deaktivaciji.')
      setConfirmDeactivate(null)
    } finally {
      setDeactivating(false)
    }
  }

  async function handleMarkLeasingComplete(assetId: string, periodMonth: string) {
    const key = `${assetId}:${periodMonth}`
    setMarkingLeasing(key); setError(null)
    try {
      await apiClient.post(`/company-assets/${assetId}/leasing-payments`, { period_month: periodMonth })
      setNotifications(prev =>
        prev.filter(n => !(n.asset_id === assetId && n.period_month === periodMonth))
      )
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? 'Greška pri bilježenju plaćanja.')
    } finally {
      setMarkingLeasing(null) }
  }

  // ── Asset mobile card ──────────────────────────────────────────────────────

  function AssetCard({ asset }: { asset: CompanyAsset }) {
    const isActive = asset.status === 'active'
    const warrantyStatus = expiryStatus(asset.warranty_expires_at)
    const regStatus = expiryStatus(asset.registration_expires_at)
    const hasDeadlineAlert = warrantyStatus === 'expired' || warrantyStatus === 'soon'
      || regStatus === 'expired' || regStatus === 'soon'

    return (
      <div className={[
        'rounded-xl border bg-gi-card px-4 py-3 space-y-2.5 transition-colors',
        hasDeadlineAlert && isActive ? 'border-amber-700/40' : 'border-gi-border',
        !isActive ? 'opacity-50' : '',
      ].join(' ')}>
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <p className="font-semibold text-gi-text text-sm truncate">{asset.name}</p>
            <p className="text-xs text-gi-text-muted mt-0.5">{employeeLabel(asset.assigned_employee)}</p>
          </div>
          <span className={`shrink-0 text-xs px-2 py-0.5 rounded-md font-medium border ${
            isActive
              ? 'bg-emerald-950/40 text-emerald-400 border-emerald-800/40'
              : 'bg-gi-card-elevated text-gi-text-subtle border-gi-border'
          }`}>
            {isActive ? 'Aktivno' : 'Neaktivno'}
          </span>
        </div>

        {asset.asset_type === 'oprema' && asset.size && (
          <p className="text-xs text-gi-text-muted">Veličina: <span className="text-gi-text">{asset.size}</span></p>
        )}

        {asset.asset_type === 'vozilo' && (
          <div className="space-y-1">
            {asset.registration_plate && (
              <p className="text-xs text-gi-text-muted">
                Tablica: <span className="text-gi-text font-mono tracking-wide">{asset.registration_plate}</span>
              </p>
            )}
            {asset.registration_expires_at && (
              <div className="flex items-center gap-1.5 text-xs text-gi-text-muted">
                Reg. ističe:&nbsp;<ExpiryBadge dateStr={asset.registration_expires_at} />
              </div>
            )}
            {asset.is_leasing && (
              <div className="flex items-center gap-1.5">
                <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-950/40 text-blue-300 border border-blue-800/40">
                  Leasing{asset.leasing_company ? ` · ${asset.leasing_company}` : ''}
                </span>
              </div>
            )}
            {asset.is_leasing && asset.leasing_end_date && (
              <p className="text-xs text-gi-text-muted">
                Leasing do: <span className="text-gi-text">{formatDate(asset.leasing_end_date)}</span>
              </p>
            )}
          </div>
        )}

        {(asset.asset_type === 'alat' || asset.asset_type === 'oprema') && asset.warranty_expires_at && (
          <div className="flex items-center gap-1.5 text-xs text-gi-text-muted">
            Garancija ističe:&nbsp;<ExpiryBadge dateStr={asset.warranty_expires_at} />
          </div>
        )}

        {asset.notes && (
          <p className="text-xs text-gi-text-subtle italic truncate">{asset.notes}</p>
        )}

        <div className="flex gap-2 pt-1">
          <button
            onClick={() => openEdit(asset)}
            className="flex-1 text-xs text-gi-text-muted border border-gi-border hover:border-gi-text-subtle hover:text-gi-text rounded-lg py-1.5 transition-colors"
          >
            Uredi
          </button>
          {isActive && (
            <button
              onClick={() => setConfirmDeactivate(asset)}
              className="flex-1 text-xs text-red-400/70 border border-red-900/40 hover:border-red-700/60 hover:text-red-400 rounded-lg py-1.5 transition-colors"
            >
              Deaktiviraj
            </button>
          )}
        </div>
      </div>
    )
  }

  // ── Asset section (table + cards) ──────────────────────────────────────────

  function AssetSection({ type }: { type: AssetType }) {
    const list = filteredAssets(type)
    const colCount = type === 'vozilo' ? 7 : 5

    return (
      <div className="space-y-3">
        {/* Desktop table */}
        <div className="hidden sm:block overflow-x-auto rounded-xl border border-gi-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-gi-text-subtle bg-gi-card border-b border-gi-border">
                <th className="px-4 py-3 font-medium">Naziv</th>
                {type === 'oprema' && <th className="px-4 py-3 font-medium">Veličina</th>}
                {type === 'vozilo' && <th className="px-4 py-3 font-medium">Tablica</th>}
                {type === 'vozilo' && <th className="px-4 py-3 font-medium">Reg. ističe</th>}
                {type === 'vozilo' && <th className="px-4 py-3 font-medium">Leasing</th>}
                {(type === 'alat' || type === 'oprema') && <th className="px-4 py-3 font-medium">Garancija ističe</th>}
                <th className="px-4 py-3 font-medium">Zadužen</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium w-32">Akcije</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gi-border bg-gi-card/50">
              {list.length === 0 ? (
                <tr>
                  <td colSpan={colCount} className="px-4 py-10 text-center text-gi-text-subtle text-sm">
                    {search
                      ? 'Nema rezultata za traženi pojam.'
                      : `Nema unosa za ${ASSET_TYPE_LABELS[type].toLowerCase()}.`}
                  </td>
                </tr>
              ) : list.map(asset => (
                <tr
                  key={asset.id}
                  className={`hover:bg-gi-card-elevated/50 transition-colors ${asset.status === 'inactive' ? 'opacity-40' : ''}`}
                >
                  <td className="px-4 py-3 font-medium text-gi-text">{asset.name}</td>
                  {type === 'oprema' && (
                    <td className="px-4 py-3 text-gi-text-muted">{asset.size ?? '—'}</td>
                  )}
                  {type === 'vozilo' && (
                    <td className="px-4 py-3 text-gi-text font-mono tracking-wide text-xs">
                      {asset.registration_plate ?? '—'}
                    </td>
                  )}
                  {type === 'vozilo' && (
                    <td className="px-4 py-3">
                      <ExpiryBadge dateStr={asset.registration_expires_at} />
                    </td>
                  )}
                  {type === 'vozilo' && (
                    <td className="px-4 py-3">
                      {asset.is_leasing
                        ? <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-blue-950/40 text-blue-300 border border-blue-800/40">
                            {asset.leasing_company ?? 'Leasing'}
                          </span>
                        : <span className="text-gi-text-subtle text-xs">—</span>}
                    </td>
                  )}
                  {(type === 'alat' || type === 'oprema') && (
                    <td className="px-4 py-3">
                      <ExpiryBadge dateStr={asset.warranty_expires_at} />
                    </td>
                  )}
                  <td className="px-4 py-3 text-gi-text-muted text-sm">
                    {employeeLabel(asset.assigned_employee)}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex text-xs px-2 py-0.5 rounded-md font-medium border ${
                      asset.status === 'active'
                        ? 'bg-emerald-950/40 text-emerald-400 border-emerald-800/40'
                        : 'bg-gi-card-elevated text-gi-text-subtle border-gi-border'
                    }`}>
                      {asset.status === 'active' ? 'Aktivno' : 'Neaktivno'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-3">
                      <button
                        onClick={() => openEdit(asset)}
                        className="text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
                      >
                        Uredi
                      </button>
                      {asset.status === 'active' && (
                        <button
                          onClick={() => setConfirmDeactivate(asset)}
                          className="text-xs text-red-400/60 hover:text-red-400 transition-colors"
                        >
                          Deaktiviraj
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Mobile cards */}
        <div className="sm:hidden space-y-2.5">
          {list.length === 0 ? (
            <div className="rounded-xl border border-gi-border bg-gi-card py-10 text-center">
              <p className="text-gi-text-subtle text-sm">
                {search ? 'Nema rezultata.' : `Nema unosa za ${ASSET_TYPE_LABELS[type].toLowerCase()}.`}
              </p>
            </div>
          ) : list.map(asset => <AssetCard key={asset.id} asset={asset} />)}
        </div>
      </div>
    )
  }

  // ── Notifications tab ─────────────────────────────────────────────────────

  const expiryNotifs  = notifications.filter(n => !isLeasingKind(n.kind))
  const leasingNotifs = notifications.filter(n => isLeasingKind(n.kind))

  // ── Render ────────────────────────────────────────────────────────────────

  const tabCls = (active: boolean) => [
    'shrink-0 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors',
    active
      ? 'border-gi-yellow text-gi-yellow'
      : 'border-transparent text-gi-text-muted hover:text-gi-text',
  ].join(' ')

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Zaduženja"
      backHref="/dashboard"
      onLogout={logout}
      action={
        <button
          onClick={openCreate}
          className="px-3 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors whitespace-nowrap"
        >
          + Novo zaduženje
        </button>
      }
    >
      <div className="max-w-6xl space-y-5">
        {/* Page header */}
        <div>
          <h1 className="text-xl font-bold text-gi-text tracking-tight">Zaduženja</h1>
          <p className="text-sm text-gi-text-subtle mt-0.5">Upravljanje alatima, opremom i vozilima tvrtke</p>
        </div>

        {error && (
          <div className="rounded-xl border border-red-900/50 bg-red-950/20 px-4 py-3 text-sm text-red-300">
            {error}
          </div>
        )}

        {/* Overview stat cards */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[
            {
              label: 'Alat', count: alatCount,
              icon: (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z" />
                </svg>
              ),
              tab: 'alat' as Tab, accent: 'text-gi-yellow',
            },
            {
              label: 'Oprema', count: opremaCount,
              icon: (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25z" />
                </svg>
              ),
              tab: 'oprema' as Tab, accent: 'text-gi-text-muted',
            },
            {
              label: 'Vozila', count: vozilaCount,
              icon: (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 18.75a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h6m-9 0H3.375a1.125 1.125 0 01-1.125-1.125V14.25m17.25 4.5a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m3 0h1.125c.621 0 1.129-.504 1.09-1.124a17.902 17.902 0 00-3.213-9.193 2.056 2.056 0 00-1.58-.86H14.25M16.5 18.75h-2.25m0-11.177v-.958c0-.568-.422-1.048-.987-1.106a48.554 48.554 0 00-10.026 0 1.106 1.106 0 00-.987 1.106v7.635m12-6.677v6.677m0 4.5v-4.5m0 0h-12" />
                </svg>
              ),
              tab: 'vozila' as Tab, accent: 'text-gi-text-muted',
            },
            {
              label: 'Potrebna pažnja', count: paznjCount,
              icon: (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
              ),
              tab: 'obavijesti' as Tab,
              accent: paznjCount > 0 ? 'text-amber-300' : 'text-gi-text-subtle',
            },
          ].map(s => (
            <button
              key={s.label}
              onClick={() => setActiveTab(s.tab)}
              className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 text-left hover:bg-gi-card-elevated/60 hover:border-gi-text-subtle transition-colors group"
            >
              <div className={`${s.accent} mb-1.5 group-hover:scale-110 transition-transform origin-left`}>{s.icon}</div>
              <p className={`text-xl font-bold ${s.accent}`}>{dataLoading ? '—' : s.count}</p>
              <p className="text-xs text-gi-text-subtle mt-0.5">{s.label}</p>
            </button>
          ))}
        </div>

        {/* Search (shown on asset tabs) */}
        {(activeTab === 'alat' || activeTab === 'oprema' || activeTab === 'vozila') && (
          <div>
            <input
              type="search"
              placeholder="Pretraži po nazivu, zaposleniku, tablici..."
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full sm:max-w-sm bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors"
            />
          </div>
        )}

        {/* Tabs */}
        <div className="flex border-b border-gi-border overflow-x-auto">
          {TABS.map(t => (
            <button key={t.id} onClick={() => { setActiveTab(t.id); setSearch('') }} className={tabCls(activeTab === t.id)}>
              {t.label}
              {t.id === 'obavijesti' && paznjCount > 0 && (
                <span className="ml-1.5 inline-flex items-center justify-center w-4 h-4 text-[10px] font-bold rounded-full bg-amber-500 text-gi-navy">
                  {paznjCount > 9 ? '9+' : paznjCount}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Tab content */}
        {dataLoading ? (
          <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje zaduženja…</p>
          </div>
        ) : (
          <>
            {/* ── Pregled tab ── */}
            {activeTab === 'pregled' && (
              <div className="space-y-6">
                {(['alat', 'oprema', 'vozilo'] as const).map(type => {
                  const list = assets.filter(a => a.asset_type === type && a.status === 'active').slice(0, 4)
                  const tabTarget: Tab = type === 'vozilo' ? 'vozila' : type
                  return (
                    <div key={type}>
                      <div className="flex items-center justify-between mb-3">
                        <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider">
                          {ASSET_TYPE_LABELS[type]}
                        </h2>
                        <button
                          onClick={() => setActiveTab(tabTarget)}
                          className="text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
                        >
                          Prikaži sve →
                        </button>
                      </div>
                      {list.length === 0 ? (
                        <p className="text-gi-text-subtle text-sm">Nema aktivnih unosa.</p>
                      ) : (
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2.5">
                          {list.map(a => {
                            const expDate = a.registration_expires_at ?? a.warranty_expires_at
                            const es = expiryStatus(expDate)
                            return (
                              <div
                                key={a.id}
                                className={[
                                  'rounded-xl border bg-gi-card px-4 py-3 hover:bg-gi-card-elevated/60 transition-colors cursor-pointer',
                                  es === 'expired' ? 'border-red-800/50' : es === 'soon' ? 'border-amber-700/40' : 'border-gi-border',
                                ].join(' ')}
                                onClick={() => openEdit(a)}
                              >
                                <p className="text-sm font-medium text-gi-text truncate">{a.name}</p>
                                <p className="text-xs text-gi-text-muted mt-0.5 truncate">{employeeLabel(a.assigned_employee)}</p>
                                {a.registration_plate && (
                                  <p className="text-xs text-gi-text-subtle font-mono mt-0.5">{a.registration_plate}</p>
                                )}
                                {a.is_leasing && (
                                  <span className="mt-1 inline-flex text-xs text-blue-300">Leasing</span>
                                )}
                                {expDate && es !== 'ok' && (
                                  <div className="mt-1.5">
                                    <ExpiryBadge dateStr={expDate} />
                                  </div>
                                )}
                              </div>
                            )
                          })}
                        </div>
                      )}
                    </div>
                  )
                })}

                {notifications.length > 0 && (
                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider">Potrebna pažnja</h2>
                      <button onClick={() => setActiveTab('obavijesti')} className="text-xs text-amber-400 hover:text-amber-300 transition-colors">
                        Prikaži sve ({notifications.length}) →
                      </button>
                    </div>
                    <div className="space-y-2">
                      {notifications.slice(0, 3).map((n, i) => (
                        <div key={i} className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 flex items-start gap-3">
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-gi-text truncate">{n.asset_name}</p>
                            <p className="text-xs text-gi-text-muted mt-0.5">{n.message}</p>
                          </div>
                          <NotificationStateBadge state={n.state} />
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'alat'   && <AssetSection type="alat" />}
            {activeTab === 'oprema' && <AssetSection type="oprema" />}
            {activeTab === 'vozila' && <AssetSection type="vozilo" />}

            {/* ── Obavijesti tab ── */}
            {activeTab === 'obavijesti' && (
              <div className="space-y-5">
                {notifications.length === 0 ? (
                  <div className="rounded-xl border border-gi-border bg-gi-card py-14 flex flex-col items-center gap-3">
                    <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <p className="text-gi-text-subtle text-sm">Nema aktivnih obavijesti.</p>
                  </div>
                ) : (
                  <>
                    {/* Expiry notifications */}
                    {expiryNotifs.length > 0 && (
                      <div>
                        <h3 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-2.5">
                          Registracije i garancije ({expiryNotifs.length})
                        </h3>
                        <div className="space-y-2">
                          {expiryNotifs.map((n, i) => (
                            <div key={i} className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 flex flex-col sm:flex-row sm:items-start gap-3">
                              <div className="flex-1 min-w-0 space-y-0.5">
                                <div className="flex flex-wrap items-center gap-2">
                                  <span className="font-medium text-gi-text text-sm">{n.asset_name}</span>
                                  <span className="text-xs text-gi-text-subtle">{ASSET_TYPE_LABELS[n.asset_type]}</span>
                                  <NotificationStateBadge state={n.state} />
                                </div>
                                {n.registration_plate && (
                                  <p className="text-xs text-gi-text-subtle font-mono">{n.registration_plate}</p>
                                )}
                                <p className="text-xs text-gi-text-muted">{n.message}</p>
                                {n.assigned_employee && (
                                  <p className="text-xs text-gi-text-subtle">
                                    Zadužen: {n.assigned_employee.first_name} {n.assigned_employee.last_name}
                                  </p>
                                )}
                              </div>
                              <div className="shrink-0 flex flex-col items-end gap-1">
                                <p className="text-xs text-gi-text-muted">Ističe {formatDate(n.expires_at)}</p>
                                {n.days_remaining >= 0 && (
                                  <p className="text-xs text-gi-text-subtle">Za {n.days_remaining} {n.days_remaining === 1 ? 'dan' : 'dana'}</p>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Leasing notifications */}
                    {leasingNotifs.length > 0 && (
                      <div>
                        <h3 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-2.5">
                          Leasing obveze ({leasingNotifs.length})
                        </h3>
                        <div className="space-y-2">
                          {leasingNotifs.map((n, i) => {
                            const leasingKey = `${n.asset_id}:${n.period_month ?? ''}`
                            const isMarking = markingLeasing === leasingKey
                            return (
                              <div key={i} className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 flex flex-col sm:flex-row sm:items-start gap-3">
                                <div className="flex-1 min-w-0 space-y-0.5">
                                  <div className="flex flex-wrap items-center gap-2">
                                    <span className="font-medium text-gi-text text-sm">{n.asset_name}</span>
                                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-blue-950/40 text-blue-300 border border-blue-800/40">
                                      Leasing
                                    </span>
                                    <NotificationStateBadge state={n.state} />
                                  </div>
                                  {n.registration_plate && (
                                    <p className="text-xs text-gi-text-subtle font-mono">{n.registration_plate}</p>
                                  )}
                                  <p className="text-xs text-gi-text-muted">{n.message}</p>
                                  {n.period_month && (
                                    <p className="text-xs text-gi-yellow/80">{formatPeriodMonth(n.period_month)}</p>
                                  )}
                                  {n.assigned_employee && (
                                    <p className="text-xs text-gi-text-subtle">
                                      Zadužen: {n.assigned_employee.first_name} {n.assigned_employee.last_name}
                                    </p>
                                  )}
                                </div>
                                <div className="shrink-0 flex flex-col items-end gap-2">
                                  <p className="text-xs text-gi-text-muted">Rok: {formatDate(n.expires_at)}</p>
                                  {n.period_month && (
                                    <button
                                      onClick={() => handleMarkLeasingComplete(n.asset_id, n.period_month!)}
                                      disabled={isMarking}
                                      className="text-xs font-medium text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover disabled:opacity-50 px-3 py-1.5 rounded-lg transition-colors whitespace-nowrap"
                                    >
                                      {isMarking ? 'Bilježim…' : 'Označi riješeno'}
                                    </button>
                                  )}
                                </div>
                              </div>
                            )
                          })}
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            )}
          </>
        )}
      </div>

      {/* ── Create / Edit Modal ─────────────────────────────────────────────── */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-0 sm:p-4">
          <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={closeModal} />
          <div className="relative w-full sm:max-w-lg bg-gi-workspace border border-gi-border rounded-t-2xl sm:rounded-2xl shadow-2xl overflow-y-auto max-h-[92dvh]">

            {/* Modal header */}
            <div className="px-5 py-4 border-b border-gi-border flex items-center justify-between sticky top-0 bg-gi-workspace z-10">
              <div>
                <h2 className="text-base font-semibold text-gi-text">
                  {editAsset ? `Uredi: ${editAsset.name}` : 'Novo zaduženje'}
                </h2>
                {editAsset && (
                  <p className="text-xs text-gi-text-subtle mt-0.5">{ASSET_TYPE_LABELS[editAsset.asset_type]}</p>
                )}
              </div>
              <button onClick={closeModal} className="text-gi-text-subtle hover:text-gi-text p-1 rounded-lg transition-colors">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="px-5 py-5 space-y-4">
              {formError && (
                <div className="rounded-lg border border-red-800/50 bg-red-950/30 px-3 py-2 text-xs text-red-300">
                  {formError}
                </div>
              )}

              {/* Asset type — create only */}
              {!editAsset && (
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-2">Vrsta *</label>
                  <div className="flex gap-2">
                    {(['alat', 'oprema', 'vozilo'] as AssetType[]).map(t => (
                      <button
                        key={t} type="button"
                        onClick={() => setField('asset_type', t)}
                        className={[
                          'flex-1 py-2 rounded-lg text-sm font-medium border transition-colors',
                          form.asset_type === t
                            ? 'bg-gi-yellow text-gi-navy border-gi-yellow'
                            : 'bg-gi-card-elevated border-gi-border text-gi-text-muted hover:border-gi-text-subtle',
                        ].join(' ')}
                      >
                        {ASSET_TYPE_LABELS[t]}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Name */}
              <div>
                <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Naziv *</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setField('name', e.target.value)}
                  placeholder={form.asset_type === 'vozilo' ? 'npr. VW Caddy, Iveco Daily…' : 'Naziv alata / opreme'}
                  className={inputCls}
                />
              </div>

              {/* Employee */}
              <div>
                <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Zadužen zaposlenik</label>
                <select
                  value={form.assigned_employee_id}
                  onChange={e => setField('assigned_employee_id', e.target.value)}
                  className={inputCls}
                >
                  <option value="">Nezaduženo</option>
                  {employees.map(e => (
                    <option key={e.id} value={e.id}>{e.first_name} {e.last_name}</option>
                  ))}
                </select>
              </div>

              {/* Status — edit only */}
              {editAsset && (
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Status</label>
                  <select value={form.status} onChange={e => setField('status', e.target.value)} className={inputCls}>
                    <option value="active">Aktivno</option>
                    <option value="inactive">Neaktivno</option>
                  </select>
                </div>
              )}

              {/* Oprema size */}
              {form.asset_type === 'oprema' && (
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Veličina</label>
                  <select value={form.size} onChange={e => setField('size', e.target.value)} className={inputCls}>
                    {ASSET_SIZES.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
              )}

              {/* Alat / Oprema fields */}
              {(form.asset_type === 'alat' || form.asset_type === 'oprema') && (
                <div className="space-y-3 pt-1 border-t border-gi-border/60">
                  <p className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider pt-1">Datumi</p>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum nabave</label>
                      <input type="date" value={form.purchased_at} onChange={e => setField('purchased_at', e.target.value)} className={inputCls} />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Garancija ističe</label>
                      <input type="date" value={form.warranty_expires_at} onChange={e => setField('warranty_expires_at', e.target.value)} className={inputCls} />
                    </div>
                  </div>
                </div>
              )}

              {/* Vozilo fields */}
              {form.asset_type === 'vozilo' && (
                <>
                  {/* Registration group */}
                  <div className="space-y-3 pt-1 border-t border-gi-border/60">
                    <p className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider pt-1">Registracija</p>
                    <div>
                      <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Registarska tablica</label>
                      <input
                        type="text"
                        value={form.registration_plate}
                        onChange={e => setField('registration_plate', e.target.value.toUpperCase())}
                        placeholder="ZG 1234 AB"
                        className={inputCls + ' font-mono tracking-wider'}
                      />
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum registracije</label>
                        <input type="date" value={form.registration_date} onChange={e => setField('registration_date', e.target.value)} className={inputCls} />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Registracija ističe</label>
                        <input type="date" value={form.registration_expires_at} onChange={e => setField('registration_expires_at', e.target.value)} className={inputCls} />
                      </div>
                    </div>
                  </div>

                  {/* Leasing toggle */}
                  <div className="border-t border-gi-border/60 pt-3">
                    <div className="flex items-center justify-between py-1">
                      <div>
                        <p className="text-sm font-medium text-gi-text">Vozilo je na leasingu</p>
                        <p className="text-xs text-gi-text-subtle">Uključuje mjesečne podsjetnike za plaćanje</p>
                      </div>
                      <button
                        type="button"
                        onClick={() => setField('is_leasing', !form.is_leasing)}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${form.is_leasing ? 'bg-gi-yellow' : 'bg-gi-border'}`}
                        role="switch"
                        aria-checked={form.is_leasing}
                      >
                        <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${form.is_leasing ? 'translate-x-6' : 'translate-x-1'}`} />
                      </button>
                    </div>

                    {form.is_leasing && (
                      <div className="mt-3 space-y-3 pl-0 border-l-2 border-gi-yellow/30 pl-3">
                        <div>
                          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Leasing kuća</label>
                          <input
                            type="text"
                            value={form.leasing_company}
                            onChange={e => setField('leasing_company', e.target.value)}
                            placeholder="npr. UniCredit Leasing, Erste Leasing…"
                            className={inputCls}
                          />
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Datum završetka leasinga</label>
                          <input type="date" value={form.leasing_end_date} onChange={e => setField('leasing_end_date', e.target.value)} className={inputCls} />
                        </div>
                      </div>
                    )}
                  </div>
                </>
              )}

              {/* Notes */}
              <div>
                <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Napomena</label>
                <textarea
                  value={form.notes}
                  onChange={e => setField('notes', e.target.value)}
                  rows={2}
                  placeholder="Opcionalna napomena…"
                  className={inputCls + ' resize-none'}
                />
              </div>

              {/* Leasing history — edit only */}
              {editAsset?.is_leasing && (
                <div className="border-t border-gi-border pt-4 space-y-2">
                  <p className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Povijest plaćanja leasinga</p>
                  {fetchingHistory ? (
                    <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>
                  ) : leasingHistory.length === 0 ? (
                    <p className="text-xs text-gi-text-subtle">Nema zabilježenih plaćanja.</p>
                  ) : (
                    <div className="space-y-1.5">
                      {leasingHistory.slice(0, 6).map(p => (
                        <div key={p.id} className="flex items-center justify-between text-xs">
                          <span className="text-gi-text-muted capitalize">{formatPeriodMonth(p.period_month)}</span>
                          <span className="text-gi-text-subtle">Riješeno {formatDate(p.completed_at)}</span>
                        </div>
                      ))}
                      {leasingHistory.length > 6 && (
                        <p className="text-xs text-gi-text-subtle">+{leasingHistory.length - 6} starijih zapisa</p>
                      )}
                    </div>
                  )}
                </div>
              )}

              <div className="flex gap-3 pt-2">
                <button
                  onClick={closeModal} disabled={saving}
                  className="flex-1 py-2.5 text-sm font-medium text-gi-text-muted border border-gi-border hover:border-gi-text-subtle rounded-lg transition-colors disabled:opacity-50"
                >
                  Odustani
                </button>
                <button
                  onClick={handleSave} disabled={saving}
                  className="flex-1 py-2.5 text-sm font-semibold text-gi-navy bg-gi-yellow hover:bg-gi-yellow-hover rounded-lg transition-colors disabled:opacity-50"
                >
                  {saving ? 'Sprema…' : editAsset ? 'Spremi promjene' : 'Dodaj unos'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Deactivate confirm ──────────────────────────────────────────────── */}
      {confirmDeactivate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={() => setConfirmDeactivate(null)} />
          <div className="relative w-full max-w-sm bg-gi-workspace border border-gi-border rounded-2xl shadow-2xl p-6 space-y-4">
            <h2 className="text-base font-semibold text-gi-text">Deaktivacija unosa</h2>
            <p className="text-sm text-gi-text-muted">
              Jeste li sigurni da želite deaktivirati{' '}
              <span className="font-semibold text-gi-text">{confirmDeactivate.name}</span>?
              {' '}Unos će biti skriven iz pregleda, ali ostaje u sustavu.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setConfirmDeactivate(null)} disabled={deactivating}
                className="flex-1 py-2.5 text-sm font-medium text-gi-text-muted border border-gi-border hover:border-gi-text-subtle rounded-lg transition-colors disabled:opacity-50"
              >
                Odustani
              </button>
              <button
                onClick={handleDeactivate} disabled={deactivating}
                className="flex-1 py-2.5 text-sm font-semibold text-white bg-red-700 hover:bg-red-600 rounded-lg transition-colors disabled:opacity-50"
              >
                {deactivating ? 'Deaktiviranje…' : 'Deaktiviraj'}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  )
}
