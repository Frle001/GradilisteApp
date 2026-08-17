'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
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

// ── Tab types ─────────────────────────────────────────────────────────────────

type Tab = 'pregled' | 'alat' | 'oprema' | 'vozila' | 'obavijesti'

const TABS: { id: Tab; label: string }[] = [
  { id: 'pregled',     label: 'Pregled' },
  { id: 'alat',       label: 'Alat' },
  { id: 'oprema',     label: 'Oprema' },
  { id: 'vozila',     label: 'Vozila' },
  { id: 'obavijesti', label: 'Obavijesti' },
]

// ── Form state ────────────────────────────────────────────────────────────────

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
  // Leasing
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

// ── Helpers ───────────────────────────────────────────────────────────────────

function stateBadge(state: string) {
  if (state === 'warning')
    return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-900/60 text-amber-300">Upozorenje</span>
  if (state === 'urgent')
    return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-orange-900/60 text-orange-300">Hitno / Kasni</span>
  if (state === 'expired')
    return <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-900/60 text-red-400">Isteklo</span>
  return null
}

function isLeasingKind(kind: string) {
  return kind === NOTIFICATION_KIND_LEASING_WARNING || kind === NOTIFICATION_KIND_LEASING_OVERDUE
}

function formatDate(s: string | null | undefined) {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('hr-HR')
}

function formatPeriodMonth(s: string | null | undefined) {
  if (!s) return ''
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleDateString('hr-HR', { month: 'long', year: 'numeric' })
}

function employeeLabel(e: AssetEmployee | null | undefined) {
  if (!e) return 'Nezaduženo'
  return `${e.first_name} ${e.last_name}`
}

// ── Main component ────────────────────────────────────────────────────────────

export default function ZaduzenjaPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [activeTab, setActiveTab]       = useState<Tab>('pregled')
  const [assets, setAssets]             = useState<CompanyAsset[]>([])
  const [notifications, setNotifications] = useState<AssetNotification[]>([])
  const [employees, setEmployees]       = useState<AssetEmployee[]>([])
  const [dataLoading, setDataLoading]   = useState(true)
  const [error, setError]               = useState<string | null>(null)
  const [search, setSearch]             = useState('')

  // Modal
  const [modalOpen, setModalOpen]       = useState(false)
  const [editAsset, setEditAsset]       = useState<CompanyAsset | null>(null)
  const [form, setForm]                 = useState<FormState>(emptyForm())
  const [formError, setFormError]       = useState<string | null>(null)
  const [saving, setSaving]             = useState(false)

  // Leasing history in modal
  const [leasingHistory, setLeasingHistory]     = useState<LeasingPayment[]>([])
  const [fetchingHistory, setFetchingHistory]   = useState(false)

  // Deactivate confirm
  const [confirmDeactivate, setConfirmDeactivate] = useState<CompanyAsset | null>(null)
  const [deactivating, setDeactivating]           = useState(false)

  // Leasing payment in-progress marker: "assetId:periodMonth"
  const [markingLeasing, setMarkingLeasing] = useState<string | null>(null)

  // Route guard
  useEffect(() => {
    if (!isLoading && user?.role !== 'administracija') router.replace('/dashboard')
  }, [isLoading, user, router])

  const fetchAssets = useCallback(async () => {
    setDataLoading(true)
    setError(null)
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
    } catch {
      // non-critical
    }
  }, [])

  useEffect(() => {
    if (!isLoading && user?.role === 'administracija') {
      fetchAssets()
      fetchEmployees()
    }
  }, [isLoading, user, fetchAssets, fetchEmployees])

  // Fetch leasing history when opening edit modal for a leasing vehicle
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
  if (!user || user.role !== 'administracija') return null

  // ── Computed ───────────────────────────────────────────────────────────────

  const activeAssets = assets.filter(a => a.status === 'active')
  const alatCount    = activeAssets.filter(a => a.asset_type === 'alat').length
  const opremaCount  = activeAssets.filter(a => a.asset_type === 'oprema').length
  const vozilaCount  = activeAssets.filter(a => a.asset_type === 'vozilo').length
  const paznjCount   = notifications.length

  function filteredAssets(type: AssetType) {
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

  // ── Modal helpers ──────────────────────────────────────────────────────────

  function openCreate() {
    setEditAsset(null)
    setForm(emptyForm())
    setFormError(null)
    setModalOpen(true)
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
    setFormError(null)
    setModalOpen(true)
  }

  function closeModal() {
    setModalOpen(false)
    setEditAsset(null)
  }

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
    setMarkingLeasing(key)
    setError(null)
    try {
      await apiClient.post(`/company-assets/${assetId}/leasing-payments`, { period_month: periodMonth })
      // Remove this notification locally so the UI updates without a full refresh.
      setNotifications(prev =>
        prev.filter(n => !(n.asset_id === assetId && n.period_month === periodMonth))
      )
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? 'Greška pri bilježenju plaćanja.')
    } finally {
      setMarkingLeasing(null)
    }
  }

  // ── Render helpers ─────────────────────────────────────────────────────────

  function AssetCard({ asset }: { asset: CompanyAsset }) {
    const isActive = asset.status === 'active'
    return (
      <div className={`bg-slate-900 border rounded-xl p-4 space-y-3 ${isActive ? 'border-slate-800' : 'border-slate-800/50 opacity-60'}`}>
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <p className="font-semibold text-white text-sm truncate">{asset.name}</p>
            <p className="text-xs text-slate-400 mt-0.5">{employeeLabel(asset.assigned_employee)}</p>
          </div>
          <span className={`shrink-0 text-xs px-2 py-0.5 rounded-full font-medium ${isActive ? 'bg-green-900/50 text-green-400' : 'bg-slate-800 text-slate-500'}`}>
            {isActive ? 'Aktivno' : 'Neaktivno'}
          </span>
        </div>

        {asset.asset_type === 'oprema' && asset.size && (
          <p className="text-xs text-slate-400">Veličina: <span className="text-white">{asset.size}</span></p>
        )}
        {asset.asset_type === 'vozilo' && (
          <div className="space-y-0.5">
            {asset.registration_plate && (
              <p className="text-xs text-slate-400">Tablica: <span className="text-white font-mono">{asset.registration_plate}</span></p>
            )}
            {asset.registration_expires_at && (
              <p className="text-xs text-slate-400">Registracija ističe: <span className="text-white">{formatDate(asset.registration_expires_at)}</span></p>
            )}
            {asset.is_leasing && (
              <p className="text-xs text-blue-400 font-medium">
                Leasing: Aktivan{asset.leasing_company ? ` · ${asset.leasing_company}` : ''}
              </p>
            )}
            {asset.is_leasing && asset.leasing_end_date && (
              <p className="text-xs text-slate-400">Leasing do: <span className="text-white">{formatDate(asset.leasing_end_date)}</span></p>
            )}
          </div>
        )}
        {(asset.asset_type === 'alat' || asset.asset_type === 'oprema') && asset.warranty_expires_at && (
          <p className="text-xs text-slate-400">Garancija ističe: <span className="text-white">{formatDate(asset.warranty_expires_at)}</span></p>
        )}
        {asset.notes && (
          <p className="text-xs text-slate-500 italic truncate">{asset.notes}</p>
        )}

        <div className="flex gap-2 pt-1">
          <button
            onClick={() => openEdit(asset)}
            className="flex-1 text-xs text-slate-300 border border-slate-700 hover:border-slate-500 rounded-lg py-1.5 transition"
          >
            Uredi
          </button>
          {isActive && (
            <button
              onClick={() => setConfirmDeactivate(asset)}
              className="flex-1 text-xs text-red-400 border border-red-900/50 hover:border-red-700 rounded-lg py-1.5 transition"
            >
              Deaktiviraj
            </button>
          )}
        </div>
      </div>
    )
  }

  function AssetSection({ type }: { type: AssetType }) {
    const list = filteredAssets(type)
    return (
      <div className="space-y-4">
        {/* Desktop table */}
        <div className="hidden sm:block overflow-x-auto rounded-xl border border-slate-800">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-slate-400 uppercase tracking-wider bg-slate-900 border-b border-slate-800">
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
            <tbody className="divide-y divide-slate-800 bg-slate-900/50">
              {list.length === 0 ? (
                <tr>
                  <td colSpan={9} className="px-4 py-8 text-center text-slate-500 text-sm">
                    {search ? 'Nema rezultata za traženi pojam.' : `Nema unosa za ${ASSET_TYPE_LABELS[type].toLowerCase()}.`}
                  </td>
                </tr>
              ) : list.map(asset => (
                <tr key={asset.id} className={`hover:bg-slate-800/30 transition ${asset.status === 'inactive' ? 'opacity-50' : ''}`}>
                  <td className="px-4 py-3 font-medium text-white">{asset.name}</td>
                  {type === 'oprema' && <td className="px-4 py-3 text-slate-300">{asset.size ?? '—'}</td>}
                  {type === 'vozilo' && <td className="px-4 py-3 text-slate-300 font-mono">{asset.registration_plate ?? '—'}</td>}
                  {type === 'vozilo' && <td className="px-4 py-3 text-slate-300">{formatDate(asset.registration_expires_at)}</td>}
                  {type === 'vozilo' && (
                    <td className="px-4 py-3">
                      {asset.is_leasing
                        ? <span className="text-blue-400 text-xs font-medium">Aktivan</span>
                        : <span className="text-slate-600 text-xs">—</span>}
                    </td>
                  )}
                  {(type === 'alat' || type === 'oprema') && <td className="px-4 py-3 text-slate-300">{formatDate(asset.warranty_expires_at)}</td>}
                  <td className="px-4 py-3 text-slate-300">{employeeLabel(asset.assigned_employee)}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex text-xs px-2 py-0.5 rounded-full font-medium ${asset.status === 'active' ? 'bg-green-900/50 text-green-400' : 'bg-slate-800 text-slate-500'}`}>
                      {asset.status === 'active' ? 'Aktivno' : 'Neaktivno'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      <button onClick={() => openEdit(asset)} className="text-xs text-slate-300 hover:text-white underline underline-offset-2">Uredi</button>
                      {asset.status === 'active' && (
                        <button onClick={() => setConfirmDeactivate(asset)} className="text-xs text-red-400 hover:text-red-300 underline underline-offset-2">Deaktiviraj</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {/* Mobile cards */}
        <div className="sm:hidden space-y-3">
          {list.length === 0 ? (
            <p className="text-center text-slate-500 text-sm py-8">
              {search ? 'Nema rezultata.' : `Nema unosa za ${ASSET_TYPE_LABELS[type].toLowerCase()}.`}
            </p>
          ) : list.map(asset => <AssetCard key={asset.id} asset={asset} />)}
        </div>
      </div>
    )
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Zaduženja"
      backHref="/dashboard"
      onLogout={logout}
      action={
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white font-semibold rounded-lg text-sm transition"
        >
          + Novi unos
        </button>
      }
    >
      <div className="space-y-6">

        {error && (
          <div className="bg-red-950/50 border border-red-800 rounded-xl px-4 py-3 text-sm text-red-300">
            {error}
          </div>
        )}

        {/* Overview stat cards */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[
            { label: 'Alat',            count: alatCount,   color: 'text-blue-400' },
            { label: 'Oprema',          count: opremaCount, color: 'text-purple-400' },
            { label: 'Vozila',          count: vozilaCount, color: 'text-cyan-400' },
            { label: 'Potrebna pažnja', count: paznjCount,  color: paznjCount > 0 ? 'text-amber-400' : 'text-slate-400' },
          ].map(s => (
            <div key={s.label} className="bg-slate-900 border border-slate-800 rounded-xl p-4 text-center">
              <p className={`text-2xl font-bold ${s.color}`}>{dataLoading ? '—' : s.count}</p>
              <p className="text-xs text-slate-400 mt-1">{s.label}</p>
            </div>
          ))}
        </div>

        {/* Search */}
        {activeTab !== 'pregled' && activeTab !== 'obavijesti' && (
          <div>
            <input
              type="search"
              placeholder="Pretraži po nazivu, zaposleniku..."
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full sm:max-w-xs bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-slate-500"
            />
          </div>
        )}

        {/* Tabs */}
        <div className="border-b border-slate-800">
          <nav className="-mb-px flex gap-1 overflow-x-auto">
            {TABS.map(t => (
              <button
                key={t.id}
                onClick={() => setActiveTab(t.id)}
                className={`shrink-0 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === t.id
                    ? 'border-blue-500 text-white'
                    : 'border-transparent text-slate-400 hover:text-slate-200 hover:border-slate-600'
                }`}
              >
                {t.label}
                {t.id === 'obavijesti' && paznjCount > 0 && (
                  <span className="ml-1.5 inline-flex items-center justify-center w-4 h-4 text-[10px] font-bold rounded-full bg-amber-500 text-black">
                    {paznjCount > 9 ? '9+' : paznjCount}
                  </span>
                )}
              </button>
            ))}
          </nav>
        </div>

        {/* Tab content */}
        {dataLoading ? (
          <div className="text-center py-16 text-slate-500 text-sm">Učitavanje...</div>
        ) : (
          <>
            {activeTab === 'pregled' && (
              <div className="space-y-6">
                {([
                  { type: 'alat'   as AssetType, label: 'Alat' },
                  { type: 'oprema' as AssetType, label: 'Oprema' },
                  { type: 'vozilo' as AssetType, label: 'Vozila' },
                ] as const).map(({ type, label }) => {
                  const list = assets.filter(a => a.asset_type === type && a.status === 'active').slice(0, 3)
                  return (
                    <div key={type}>
                      <div className="flex items-center justify-between mb-3">
                        <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">{label}</h2>
                        <button
                          onClick={() => setActiveTab(type === 'vozilo' ? 'vozila' : type)}
                          className="text-xs text-blue-400 hover:text-blue-300"
                        >
                          Prikaži sve →
                        </button>
                      </div>
                      {list.length === 0 ? (
                        <p className="text-slate-500 text-sm">Nema aktivnih unosa.</p>
                      ) : (
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                          {list.map(a => (
                            <div key={a.id} className="bg-slate-900 border border-slate-800 rounded-xl px-4 py-3 flex items-center justify-between gap-2">
                              <div className="min-w-0">
                                <p className="text-sm font-medium text-white truncate">{a.name}</p>
                                <p className="text-xs text-slate-400">{employeeLabel(a.assigned_employee)}</p>
                                {a.is_leasing && <p className="text-xs text-blue-400">Leasing</p>}
                              </div>
                              <button onClick={() => openEdit(a)} className="shrink-0 text-xs text-slate-400 hover:text-white">Uredi</button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )
                })}

                {notifications.length > 0 && (
                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">Potrebna pažnja</h2>
                      <button onClick={() => setActiveTab('obavijesti')} className="text-xs text-amber-400 hover:text-amber-300">
                        Prikaži sve ({notifications.length}) →
                      </button>
                    </div>
                    <div className="space-y-2">
                      {notifications.slice(0, 3).map((n, i) => (
                        <div key={i} className="bg-slate-900 border border-slate-800 rounded-xl px-4 py-3 flex items-start gap-3">
                          <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-white truncate">{n.asset_name}</p>
                            <p className="text-xs text-slate-400 mt-0.5">{n.message}</p>
                          </div>
                          {stateBadge(n.state)}
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

            {activeTab === 'obavijesti' && (
              <div className="space-y-3">
                {notifications.length === 0 ? (
                  <p className="text-center text-slate-500 text-sm py-8">Nema aktivnih obavijesti.</p>
                ) : notifications.map((n, i) => {
                  const leasingKind = isLeasingKind(n.kind)
                  const leasingKey  = `${n.asset_id}:${n.period_month ?? ''}`
                  const isMarking   = markingLeasing === leasingKey
                  return (
                    <div key={i} className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col sm:flex-row sm:items-start gap-3">
                      <div className="flex-1 min-w-0 space-y-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-semibold text-white text-sm">{n.asset_name}</span>
                          <span className="text-xs text-slate-500">{ASSET_TYPE_LABELS[n.asset_type]}</span>
                          {stateBadge(n.state)}
                        </div>
                        {n.registration_plate && (
                          <p className="text-xs text-slate-400 font-mono">{n.registration_plate}</p>
                        )}
                        <p className="text-xs text-slate-300">{n.message}</p>
                        {n.assigned_employee && (
                          <p className="text-xs text-slate-500">
                            Zadužen: {n.assigned_employee.first_name} {n.assigned_employee.last_name}
                          </p>
                        )}
                        {leasingKind && n.period_month && (
                          <p className="text-xs text-blue-400">{formatPeriodMonth(n.period_month)}</p>
                        )}
                      </div>
                      <div className="flex flex-col items-end gap-2 shrink-0">
                        <p className="text-xs text-slate-500">{leasingKind ? 'Rok' : 'Ističe'}: {formatDate(n.expires_at)}</p>
                        {!leasingKind && n.days_remaining >= 0 && (
                          <p className="text-xs text-slate-400">Za {n.days_remaining} {n.days_remaining === 1 ? 'dan' : 'dana'}</p>
                        )}
                        {leasingKind && n.period_month && (
                          <button
                            onClick={() => handleMarkLeasingComplete(n.asset_id, n.period_month!)}
                            disabled={isMarking}
                            className="text-xs font-medium text-white bg-blue-700 hover:bg-blue-600 disabled:opacity-50 px-3 py-1.5 rounded-lg transition"
                          >
                            {isMarking ? 'Bilježim...' : 'Označi riješeno'}
                          </button>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </>
        )}
      </div>

      {/* ── Create / Edit Modal ─────────────────────────────────────────────── */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-0 sm:p-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={closeModal} />
          <div className="relative w-full sm:max-w-lg bg-slate-900 border border-slate-700 rounded-t-2xl sm:rounded-2xl shadow-2xl overflow-y-auto max-h-[90dvh]">
            <div className="px-5 py-4 border-b border-slate-800 flex items-center justify-between">
              <h2 className="text-base font-semibold text-white">
                {editAsset ? 'Uredi unos' : 'Novi unos'}
              </h2>
              <button onClick={closeModal} className="text-slate-400 hover:text-white p-1 rounded-lg">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="px-5 py-5 space-y-4">
              {formError && (
                <div className="bg-red-950/50 border border-red-800 rounded-lg px-3 py-2 text-xs text-red-300">
                  {formError}
                </div>
              )}

              {/* Asset type — only on create */}
              {!editAsset && (
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1.5">Vrsta *</label>
                  <div className="flex gap-2">
                    {(['alat', 'oprema', 'vozilo'] as AssetType[]).map(t => (
                      <button
                        key={t}
                        type="button"
                        onClick={() => setField('asset_type', t)}
                        className={`flex-1 py-2 rounded-lg text-sm font-medium border transition ${
                          form.asset_type === t
                            ? 'bg-blue-600 border-blue-500 text-white'
                            : 'bg-slate-800 border-slate-700 text-slate-300 hover:border-slate-500'
                        }`}
                      >
                        {ASSET_TYPE_LABELS[t]}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Name */}
              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1.5">Naziv *</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setField('name', e.target.value)}
                  placeholder={form.asset_type === 'vozilo' ? 'npr. VW Caddy, Iveco Daily...' : 'Naziv alata / opreme'}
                  className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-slate-500"
                />
              </div>

              {/* Employee picker */}
              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1.5">Zadužen zaposlenik</label>
                <select
                  value={form.assigned_employee_id}
                  onChange={e => setField('assigned_employee_id', e.target.value)}
                  className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500"
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
                  <label className="block text-xs font-medium text-slate-400 mb-1.5">Status</label>
                  <select
                    value={form.status}
                    onChange={e => setField('status', e.target.value)}
                    className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500"
                  >
                    <option value="active">Aktivno</option>
                    <option value="inactive">Neaktivno</option>
                  </select>
                </div>
              )}

              {/* Oprema size */}
              {form.asset_type === 'oprema' && (
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1.5">Veličina</label>
                  <select
                    value={form.size}
                    onChange={e => setField('size', e.target.value)}
                    className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500"
                  >
                    {ASSET_SIZES.map(s => (
                      <option key={s} value={s}>{s}</option>
                    ))}
                  </select>
                </div>
              )}

              {/* Alat / Oprema fields */}
              {(form.asset_type === 'alat' || form.asset_type === 'oprema') && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1.5">Datum nabave</label>
                    <input type="date" value={form.purchased_at} onChange={e => setField('purchased_at', e.target.value)}
                      className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1.5">Garancija ističe</label>
                    <input type="date" value={form.warranty_expires_at} onChange={e => setField('warranty_expires_at', e.target.value)}
                      className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500" />
                  </div>
                </>
              )}

              {/* Vozilo fields */}
              {form.asset_type === 'vozilo' && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1.5">Registarska tablica</label>
                    <input type="text" value={form.registration_plate}
                      onChange={e => setField('registration_plate', e.target.value.toUpperCase())}
                      placeholder="ZG 1234 AB"
                      className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white font-mono placeholder-slate-500 focus:outline-none focus:border-slate-500" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1.5">Datum registracije</label>
                    <input type="date" value={form.registration_date} onChange={e => setField('registration_date', e.target.value)}
                      className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-slate-400 mb-1.5">Registracija ističe</label>
                    <input type="date" value={form.registration_expires_at} onChange={e => setField('registration_expires_at', e.target.value)}
                      className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500" />
                  </div>

                  {/* Leasing toggle */}
                  <div className="flex items-center justify-between py-1">
                    <div>
                      <p className="text-sm font-medium text-white">Vozilo je na leasingu</p>
                      <p className="text-xs text-slate-500">Uključuje mjesečne podsjetnike za plaćanje</p>
                    </div>
                    <button
                      type="button"
                      onClick={() => setField('is_leasing', !form.is_leasing)}
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${form.is_leasing ? 'bg-blue-600' : 'bg-slate-700'}`}
                    >
                      <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${form.is_leasing ? 'translate-x-6' : 'translate-x-1'}`} />
                    </button>
                  </div>

                  {/* Leasing fields (conditional) */}
                  {form.is_leasing && (
                    <>
                      <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1.5">Leasing kuća</label>
                        <input type="text" value={form.leasing_company}
                          onChange={e => setField('leasing_company', e.target.value)}
                          placeholder="npr. UniCredit Leasing, Erste Leasing..."
                          className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-slate-500" />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1.5">Datum završetka leasinga</label>
                        <input type="date" value={form.leasing_end_date} onChange={e => setField('leasing_end_date', e.target.value)}
                          className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-slate-500" />
                      </div>
                    </>
                  )}
                </>
              )}

              {/* Notes */}
              <div>
                <label className="block text-xs font-medium text-slate-400 mb-1.5">Napomena</label>
                <textarea value={form.notes} onChange={e => setField('notes', e.target.value)} rows={2}
                  placeholder="Opcionalna napomena..."
                  className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-slate-500 resize-none" />
              </div>

              {/* Leasing history (edit only, leasing vehicles) */}
              {editAsset?.is_leasing && (
                <div className="border-t border-slate-800 pt-4 space-y-2">
                  <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Povijest leasinga</p>
                  {fetchingHistory ? (
                    <p className="text-xs text-slate-500">Učitavanje...</p>
                  ) : leasingHistory.length === 0 ? (
                    <p className="text-xs text-slate-500">Nema zabilježenih plaćanja.</p>
                  ) : (
                    <div className="space-y-1">
                      {leasingHistory.slice(0, 6).map(p => (
                        <div key={p.id} className="flex items-center justify-between text-xs">
                          <span className="text-slate-300 capitalize">{formatPeriodMonth(p.period_month)}</span>
                          <span className="text-slate-500">Riješeno {formatDate(p.completed_at)}</span>
                        </div>
                      ))}
                      {leasingHistory.length > 6 && (
                        <p className="text-xs text-slate-600">+{leasingHistory.length - 6} starijih zapisa</p>
                      )}
                    </div>
                  )}
                </div>
              )}

              <div className="flex gap-3 pt-2">
                <button onClick={closeModal} disabled={saving}
                  className="flex-1 py-2.5 text-sm font-medium text-slate-300 border border-slate-700 hover:border-slate-500 rounded-lg transition disabled:opacity-50">
                  Odustani
                </button>
                <button onClick={handleSave} disabled={saving}
                  className="flex-1 py-2.5 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition disabled:opacity-50">
                  {saving ? 'Spremanje...' : editAsset ? 'Spremi promjene' : 'Dodaj unos'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Deactivate confirm ──────────────────────────────────────────────── */}
      {confirmDeactivate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setConfirmDeactivate(null)} />
          <div className="relative w-full max-w-sm bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl p-6 space-y-4">
            <h2 className="text-base font-semibold text-white">Deaktivacija unosa</h2>
            <p className="text-sm text-slate-300">
              Jeste li sigurni da želite deaktivirati <span className="font-semibold text-white">{confirmDeactivate.name}</span>?
              Unos će biti skriven iz pregleda, ali ostaje u sustavu.
            </p>
            <div className="flex gap-3">
              <button onClick={() => setConfirmDeactivate(null)} disabled={deactivating}
                className="flex-1 py-2.5 text-sm font-medium text-slate-300 border border-slate-700 hover:border-slate-500 rounded-lg transition disabled:opacity-50">
                Odustani
              </button>
              <button onClick={handleDeactivate} disabled={deactivating}
                className="flex-1 py-2.5 text-sm font-semibold text-white bg-red-700 hover:bg-red-600 rounded-lg transition disabled:opacity-50">
                {deactivating ? 'Deaktiviranje...' : 'Deaktiviraj'}
              </button>
            </div>
          </div>
        </div>
      )}
    </DashboardShell>
  )
}
