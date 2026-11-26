'use client'

import { useCallback, useEffect, useState, Fragment } from 'react'
import {
  Package,
  TrendingDown,
  TrendingUp,
  RefreshCw,
  Search,
  Plus,
  ClipboardList,
  ArrowDownToLine,
  ArrowUpFromLine,
  History,
  BarChart3,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import apiClient from '@/lib/api-client'

// ── Types ─────────────────────────────────────────────────────────────────────

interface WarehouseOverview {
  total_items: number
  total_value: number
  low_stock_items: number
  recent_receipts: number
  recent_movements: number
}

interface StockItem {
  id: string
  material_name: string
  material_code: string | null
  unit: string
  quantity: number
  avg_cost: number
  total_value: number
  active: boolean
  updated_at: string
}

interface StockDetail extends StockItem {
  movements: MovementItem[]
}

interface MovementItem {
  id: string
  material_name: string
  unit: string
  movement_type: 'receipt' | 'issue' | 'return' | 'adjustment'
  quantity_delta: number
  quantity_before: number
  quantity_after: number
  unit_price: number
  avg_cost_before: number
  avg_cost_after: number
  project_id: string | null
  project_name: string | null
  notes: string | null
  created_by_name: string
  created_at: string
}

interface MovementListResponse {
  data: MovementItem[]
  pagination: { page: number; per_page: number; total: number; total_pages: number }
}

interface Project { id: string; name: string }

interface ProjectMaterial {
  id: string
  material_name: string
  material_code: string | null
  unit: string
  available_quantity: number
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function fmt(n: number, decimals = 2) {
  return n.toLocaleString('hr-HR', { minimumFractionDigits: 0, maximumFractionDigits: decimals })
}

function movementLabel(t: string) {
  switch (t) {
    case 'receipt':    return 'Primka'
    case 'issue':      return 'Izdavanje'
    case 'return':     return 'Povrat'
    case 'adjustment': return 'Inventura'
    default:           return t
  }
}

function movementColor(t: string) {
  switch (t) {
    case 'receipt': return 'text-green-400'
    case 'issue':   return 'text-red-400'
    case 'return':  return 'text-blue-400'
    default:        return 'text-gi-text-muted'
  }
}

function apiErr(err: unknown) {
  const e = err as { response?: { data?: { error?: string } } }
  return e?.response?.data?.error ?? 'Greška pri učitavanju podataka.'
}

// ── Tab types ─────────────────────────────────────────────────────────────────
type Tab = 'pregled' | 'materijali' | 'ulaz' | 'izlaz' | 'povijest' | 'inventura'

// ── Main page ─────────────────────────────────────────────────────────────────

export default function SkladistePage() {
  const { user, employee, isLoading, logout } = useAuth()
  const [tab, setTab] = useState<Tab>('pregled')

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const role = user.role
  const canWrite = ['direktor', 'inzenjer', 'administracija'].includes(role)
  const canIssue = ['direktor', 'inzenjer', 'administracija', 'poslovoda'].includes(role)

  const tabs: { key: Tab; label: string; icon: React.ReactNode; show: boolean }[] = [
    { key: 'pregled',    label: 'Pregled',    icon: <BarChart3 className="w-4 h-4" />,         show: true },
    { key: 'materijali', label: 'Materijali', icon: <Package className="w-4 h-4" />,            show: true },
    { key: 'ulaz',       label: 'Ulaz',       icon: <ArrowDownToLine className="w-4 h-4" />,    show: canWrite },
    { key: 'izlaz',      label: 'Izlaz',      icon: <ArrowUpFromLine className="w-4 h-4" />,    show: canIssue },
    { key: 'povijest',   label: 'Povijest',   icon: <History className="w-4 h-4" />,            show: true },
    { key: 'inventura',  label: 'Inventura',  icon: <ClipboardList className="w-4 h-4" />,      show: canWrite },
  ]

  return (
    <AppShell user={user} employee={employee} title="Skladište" backHref="/dashboard" onLogout={logout}>
      <div className="space-y-5">
        <div>
          <h1 className="text-xl font-bold text-gi-text">Skladište</h1>
          <p className="text-sm text-gi-text-subtle mt-0.5">Centralni pregled i upravljanje zalihama materijala</p>
        </div>

        {/* Tab bar */}
        <div className="flex gap-1 bg-gi-card border border-gi-border rounded-xl p-1 overflow-x-auto">
          {tabs.filter(t => t.show).map(t => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                tab === t.key
                  ? 'bg-gi-yellow text-gi-navy'
                  : 'text-gi-text-muted hover:text-gi-text hover:bg-gi-navy-hover'
              }`}
            >
              {t.icon}{t.label}
            </button>
          ))}
        </div>

        {/* Tab content */}
        {tab === 'pregled'    && <OverviewTab companyID={user.company_id} role={role} />}
        {tab === 'materijali' && <StockTab companyID={user.company_id} role={role} />}
        {tab === 'ulaz'       && canWrite  && <ReceiptTab companyID={user.company_id} role={role} />}
        {tab === 'izlaz'      && canIssue  && <IssueTab companyID={user.company_id} role={role} />}
        {tab === 'povijest'   && <MovementsTab companyID={user.company_id} role={role} />}
        {tab === 'inventura'  && canWrite  && <AdjustmentTab companyID={user.company_id} />}
      </div>
    </AppShell>
  )
}

// ── Pregled tab ───────────────────────────────────────────────────────────────

function OverviewTab({ role }: { companyID: string; role: string }) {
  const [data, setData] = useState<WarehouseOverview | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const canSeeFinancials = ['direktor', 'inzenjer'].includes(role)

  useEffect(() => {
    setLoading(true)
    apiClient.get('/warehouse/overview')
      .then(r => setData(r.data))
      .catch(e => setError(apiErr(e)))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <LoadingCard />
  if (error)   return <ErrorCard message={error} />
  if (!data)   return null

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
      <StatCard label="Artikala" value={data.total_items.toString()} icon={<Package className="w-4 h-4" />} />
      {canSeeFinancials && <StatCard label="Ukupna vrijednost" value={`${fmt(data.total_value)} €`} icon={<BarChart3 className="w-4 h-4" />} />}
      <StatCard label="Niske zalihe" value={data.low_stock_items.toString()} icon={<TrendingDown className="w-4 h-4" />} accent={data.low_stock_items > 0 ? 'text-amber-400' : undefined} />
      <StatCard label="Primke (30d)" value={data.recent_receipts.toString()} icon={<ArrowDownToLine className="w-4 h-4" />} />
      <StatCard label="Pokreta (30d)" value={data.recent_movements.toString()} icon={<History className="w-4 h-4" />} />
    </div>
  )
}

// ── Materijali tab ────────────────────────────────────────────────────────────

function StockTab({ role }: { companyID: string; role: string }) {
  const [items, setItems] = useState<StockItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [expanded, setExpanded] = useState<string | null>(null)
  const [detail, setDetail] = useState<StockDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const canSeeFinancials = ['direktor', 'inzenjer'].includes(role)

  const fetchStock = useCallback(() => {
    setLoading(true)
    setError(null)
    const params = new URLSearchParams({ active_only: 'true' })
    if (search) params.set('search', search)
    apiClient.get(`/warehouse/stock?${params}`)
      .then(r => setItems(r.data.data ?? []))
      .catch(e => setError(apiErr(e)))
      .finally(() => setLoading(false))
  }, [search])

  useEffect(() => { fetchStock() }, [fetchStock])

  async function toggleExpanded(id: string) {
    if (expanded === id) {
      setExpanded(null)
      setDetail(null)
      return
    }
    setExpanded(id)
    setDetailLoading(true)
    try {
      const r = await apiClient.get(`/warehouse/stock/${id}`)
      setDetail(r.data)
    } finally {
      setDetailLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gi-text-subtle" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Pretraži materijal..."
            className="w-full pl-9 pr-3 py-2 bg-gi-card border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50"
          />
        </div>
        <button onClick={fetchStock} className="p-2 rounded-lg bg-gi-card border border-gi-border text-gi-text-muted hover:text-gi-text transition-colors">
          <RefreshCw className="w-4 h-4" />
        </button>
      </div>

      {loading ? <LoadingCard /> : error ? <ErrorCard message={error} /> : (
        <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
          {items.length === 0 ? (
            <p className="text-gi-text-subtle text-sm p-6 text-center">Nema materijala u skladištu.</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gi-border text-gi-text-subtle text-[11px] uppercase tracking-wider">
                  <th className="text-left px-4 py-3">Materijal</th>
                  <th className="text-right px-4 py-3">Količina</th>
                  {canSeeFinancials && <th className="text-right px-4 py-3 hidden sm:table-cell">Pr. cijena</th>}
                  {canSeeFinancials && <th className="text-right px-4 py-3 hidden md:table-cell">Vrijednost</th>}
                  <th className="px-4 py-3 w-8"></th>
                </tr>
              </thead>
              <tbody>
                {items.map(item => (
                  <Fragment key={item.id}>
                    <tr
                      onClick={() => toggleExpanded(item.id)}
                      className="border-b border-gi-border/50 hover:bg-gi-navy-hover cursor-pointer transition-colors"
                    >
                      <td className="px-4 py-3">
                        <p className="font-medium text-gi-text">{item.material_name}</p>
                        {item.material_code && <p className="text-[11px] text-gi-text-subtle">{item.material_code}</p>}
                      </td>
                      <td className="px-4 py-3 text-right text-gi-text">
                        {fmt(item.quantity)} <span className="text-gi-text-subtle text-[11px]">{item.unit}</span>
                      </td>
                      {canSeeFinancials && <td className="px-4 py-3 text-right text-gi-text-muted hidden sm:table-cell">{fmt(item.avg_cost)} €</td>}
                      {canSeeFinancials && <td className="px-4 py-3 text-right font-medium text-gi-text hidden md:table-cell">{fmt(item.total_value)} €</td>}
                      <td className="px-4 py-3 text-gi-text-subtle">
                        {expanded === item.id ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                      </td>
                    </tr>
                    {expanded === item.id && (
                      <tr key={`${item.id}-detail`} className="bg-gi-workspace/50">
                        <td colSpan={canSeeFinancials ? 5 : 3} className="px-4 py-3">
                          {detailLoading ? (
                            <p className="text-gi-text-subtle text-xs animate-pulse">Učitavanje...</p>
                          ) : detail ? (
                            <div className="space-y-2">
                              <p className="text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium">Zadnjih 50 pokreta</p>
                              {detail.movements.length === 0 ? (
                                <p className="text-gi-text-subtle text-xs">Nema pokreta.</p>
                              ) : (
                                <div className="space-y-1">
                                  {detail.movements.map(m => (
                                    <div key={m.id} className="flex items-center gap-3 text-xs py-1 border-b border-gi-border/30 last:border-0">
                                      <span className={`w-20 font-medium ${movementColor(m.movement_type)}`}>{movementLabel(m.movement_type)}</span>
                                      <span className={`w-20 text-right font-mono ${m.quantity_delta > 0 ? 'text-green-400' : 'text-red-400'}`}>
                                        {m.quantity_delta > 0 ? '+' : ''}{fmt(m.quantity_delta)}
                                      </span>
                                      <span className="text-gi-text-subtle">→ {fmt(m.quantity_after)} {m.unit}</span>
                                      {m.project_name && <span className="text-gi-text-muted truncate">{m.project_name}</span>}
                                      <span className="ml-auto text-gi-text-subtle">{new Date(m.created_at).toLocaleDateString('hr-HR')}</span>
                                    </div>
                                  ))}
                                </div>
                              )}
                            </div>
                          ) : null}
                        </td>
                      </tr>
                    )}
                  </Fragment>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  )
}

// ── Ulaz tab (receipts) ───────────────────────────────────────────────────────

interface ReceiptLine { material_name: string; material_code: string; unit: string; quantity: string; unit_price: string }
const emptyLine = (): ReceiptLine => ({ material_name: '', material_code: '', unit: '', quantity: '', unit_price: '' })

function ReceiptTab({ role }: { companyID: string; role: string }) {
  const [receiptNumber, setReceiptNumber] = useState('')
  const [supplier, setSupplier] = useState('')
  const [receiptDate, setReceiptDate] = useState(new Date().toISOString().split('T')[0])
  const [notes, setNotes] = useState('')
  const [lines, setLines] = useState<ReceiptLine[]>([emptyLine()])
  const [saving, setSaving] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Phase 2: idempotency key — generated once per form lifecycle, regenerated after success
  const [iKey, setIKey] = useState(() => crypto.randomUUID())
  const canSeeFinancials = ['direktor', 'inzenjer'].includes(role)

  function updateLine(i: number, field: keyof ReceiptLine, value: string) {
    setLines(prev => prev.map((l, idx) => idx === i ? { ...l, [field]: value } : l))
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      await apiClient.post('/warehouse/receipts', {
        receipt_number: receiptNumber || null,
        supplier:       supplier || null,
        receipt_date:   receiptDate,
        notes:          notes || null,
        lines: lines.map(l => ({
          material_name: l.material_name,
          material_code: l.material_code || null,
          unit:          l.unit,
          quantity:      parseFloat(l.quantity),
          unit_price:    l.unit_price !== '' ? parseFloat(l.unit_price) : null,
        })),
      }, { headers: { 'X-Idempotency-Key': iKey } })
      setIKey(crypto.randomUUID())
      setSuccess(true)
      setReceiptNumber(''); setSupplier(''); setNotes('')
      setLines([emptyLine()])
    } catch (err) {
      setError(apiErr(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4 max-w-2xl">
      <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
        <h2 className="font-semibold text-gi-text text-sm">Nova primka</h2>

        {success && <SuccessCard message="Primka uspješno zabilježena." onClose={() => setSuccess(false)} />}
        {error && <ErrorCard message={error} />}

        <div className="grid grid-cols-2 gap-3">
          <Field label="Broj primke" value={receiptNumber} onChange={setReceiptNumber} placeholder="npr. P-2024-001" />
          <Field label="Dobavljač" value={supplier} onChange={setSupplier} placeholder="Naziv dobavljača" />
          <Field label="Datum" type="date" value={receiptDate} onChange={setReceiptDate} />
          <Field label="Napomena" value={notes} onChange={setNotes} placeholder="Opcijski komentar" />
        </div>
      </div>

      <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-gi-text text-sm">Stavke primke</h2>
          <button type="button" onClick={() => setLines(p => [...p, emptyLine()])}
            className="flex items-center gap-1.5 text-xs text-gi-yellow hover:text-gi-yellow-hover">
            <Plus className="w-3.5 h-3.5" />Dodaj stavku
          </button>
        </div>

        {lines.map((line, i) => (
          <div key={i} className="flex flex-wrap gap-2 items-start">
            <input value={line.material_name} onChange={e => updateLine(i, 'material_name', e.target.value)}
              placeholder="Naziv materijala *" required
              className="flex-[2_1_9rem] min-w-0 px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50" />
            <input value={line.material_code} onChange={e => updateLine(i, 'material_code', e.target.value)}
              placeholder="Šifra"
              className="flex-[1_1_5rem] min-w-0 px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50" />
            <input value={line.unit} onChange={e => updateLine(i, 'unit', e.target.value)}
              placeholder="Jed. *" required
              className="flex-[0_1_4rem] min-w-0 px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50" />
            <input value={line.quantity} onChange={e => updateLine(i, 'quantity', e.target.value)}
              placeholder="Kol. *" type="number" step="any" min="0" required
              className="flex-[0_1_5rem] min-w-0 px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50" />
            {canSeeFinancials && (
              <input value={line.unit_price} onChange={e => updateLine(i, 'unit_price', e.target.value)}
                placeholder="Cijena €" type="number" step="any" min="0"
                className="flex-[0_1_6rem] min-w-0 px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50" />
            )}
            {lines.length > 1 && (
              <button type="button" onClick={() => setLines(p => p.filter((_, idx) => idx !== i))}
                className="flex-shrink-0 p-2 text-gi-text-subtle hover:text-red-400 transition-colors">×</button>
            )}
          </div>
        ))}
      </div>

      <button type="submit" disabled={saving}
        className="w-full py-2.5 bg-gi-yellow text-gi-navy font-semibold rounded-xl text-sm hover:bg-gi-yellow-hover disabled:opacity-50 transition-colors">
        {saving ? 'Spremanje...' : 'Spremi primku'}
      </button>
    </form>
  )
}

// ── Izlaz tab (issues + returns) ──────────────────────────────────────────────

function IssueTab({ role }: { companyID: string; role: string }) {
  const [mode, setMode] = useState<'issue' | 'return'>('issue')
  const [stock, setStock] = useState<StockItem[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  // Phase 2: separate idempotency keys per mode so a mode switch never reuses a key
  const [issueKey, setIssueKey] = useState(() => crypto.randomUUID())
  const [returnKey, setReturnKey] = useState(() => crypto.randomUUID())

  // Issue form state
  const [balanceID, setBalanceID] = useState('')
  const [projectID, setProjectID] = useState('')
  const [issueQty, setIssueQty] = useState('')
  const [issueNotes, setIssueNotes] = useState('')

  // Return form state
  const [retProjectID, setRetProjectID] = useState('')
  const [retProjectMats, setRetProjectMats] = useState<ProjectMaterial[]>([])
  const [retProjMatID, setRetProjMatID] = useState('')
  const [retQty, setRetQty] = useState('')
  const [retNotes, setRetNotes] = useState('')
  const [retLoadingMats, setRetLoadingMats] = useState(false)
  const [retConfirm, setRetConfirm] = useState(false)

  // Shared
  const [saving, setSaving] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Initial data load (stock + projects)
  const [initLoading, setInitLoading] = useState(true)
  const [initError, setInitError] = useState<string | null>(null)
  const [initRetry, setInitRetry] = useState(0)

  // Return form — project materials error
  const [retMatError, setRetMatError] = useState<string | null>(null)
  const canSeeFinancials = ['direktor', 'inzenjer'].includes(role)

  useEffect(() => {
    setInitLoading(true)
    setInitError(null)
    Promise.all([
      apiClient.get('/warehouse/stock?active_only=true'),
      apiClient.get('/projects?status=active'),
    ]).then(([stockRes, projRes]) => {
      const stockArr = stockRes.data?.data
      if (!Array.isArray(stockArr)) throw new Error('Neočekivani format zalihe materijala.')
      const projArr = projRes.data?.projects
      if (!Array.isArray(projArr)) throw new Error('Neočekivani format popisa projekata.')
      setStock(stockArr)
      setProjects(projArr)
    }).catch(e => {
      setInitError(apiErr(e) || 'Greška pri učitavanju podataka.')
    }).finally(() => setInitLoading(false))
  }, [initRetry])

  // Fetch project materials when return project changes
  useEffect(() => {
    if (!retProjectID) { setRetProjectMats([]); setRetProjMatID(''); setRetMatError(null); return }
    setRetLoadingMats(true)
    setRetMatError(null)
    apiClient.get(`/projects/${retProjectID}/materials?available_only=true`)
      .then(r => {
        const mats = r.data?.materials
        if (!Array.isArray(mats)) throw new Error('Neočekivani format materijala projekta.')
        const filtered = (mats as ProjectMaterial[]).filter(m =>
          stock.some(s => s.material_name.toLowerCase() === m.material_name.toLowerCase() && s.unit === m.unit)
        )
        setRetProjectMats(filtered)
        setRetProjMatID('')
      })
      .catch(e => {
        setRetProjectMats([])
        setRetMatError(apiErr(e) || 'Greška pri učitavanju materijala projekta.')
      })
      .finally(() => setRetLoadingMats(false))
  }, [retProjectID, stock])

  const issueSelected = stock.find(s => s.id === balanceID)
  const retSelectedMat = retProjectMats.find(m => m.id === retProjMatID)
  const retMatchingBalance = retSelectedMat
    ? stock.find(s => s.material_name.toLowerCase() === retSelectedMat.material_name.toLowerCase() && s.unit === retSelectedMat.unit)
    : undefined
  const retProject = projects.find(p => p.id === retProjectID)

  async function submitIssue(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      await apiClient.post('/warehouse/issues', {
        balance_id: balanceID,
        project_id: projectID,
        quantity:   parseFloat(issueQty),
        notes:      issueNotes || null,
      }, { headers: { 'X-Idempotency-Key': issueKey } })
      setIssueKey(crypto.randomUUID())
      setSuccess(true)
      setBalanceID(''); setProjectID(''); setIssueQty(''); setIssueNotes('')
    } catch (err) {
      setError(apiErr(err))
    } finally {
      setSaving(false)
    }
  }

  async function submitReturn() {
    if (!retMatchingBalance || !retSelectedMat) return
    setRetConfirm(false)
    setError(null)
    setSaving(true)
    try {
      await apiClient.post('/warehouse/returns', {
        balance_id:          retMatchingBalance.id,
        project_id:          retProjectID,
        project_material_id: retProjMatID,
        quantity:            parseFloat(retQty),
        notes:               retNotes || null,
      }, { headers: { 'X-Idempotency-Key': returnKey } })
      setReturnKey(crypto.randomUUID())
      setSuccess(true)
      setRetProjectID(''); setRetProjectMats([]); setRetProjMatID(''); setRetQty(''); setRetNotes('')
    } catch (err) {
      setError(apiErr(err))
    } finally {
      setSaving(false)
    }
  }

  if (initLoading) return (
    <div className="flex items-center gap-2 text-gi-text-muted text-sm py-6">
      <span className="animate-spin text-base">⟳</span> Učitavanje podataka...
    </div>
  )
  if (initError) return (
    <div className="space-y-3 max-w-lg">
      <ErrorCard message={initError} />
      <button type="button" onClick={() => setInitRetry(r => r + 1)}
        className="w-full py-2 border border-gi-border rounded-xl text-sm text-gi-text-muted hover:text-gi-text transition-colors">
        Pokušaj ponovo
      </button>
    </div>
  )

  return (
    <div className="space-y-4 max-w-lg">
      {/* Mode toggle */}
      <div className="flex gap-1 bg-gi-card border border-gi-border rounded-xl p-1">
        {(['issue', 'return'] as const).map(m => (
          <button key={m} type="button"
            onClick={() => { setMode(m); setSuccess(false); setError(null) }}
            className={`flex-1 py-2 rounded-lg text-sm font-medium transition-colors ${
              mode === m ? 'bg-gi-yellow text-gi-navy' : 'text-gi-text-muted hover:text-gi-text'
            }`}>
            {m === 'issue' ? 'Izdavanje materijala' : 'Povrat materijala'}
          </button>
        ))}
      </div>

      {/* Issue form */}
      {mode === 'issue' && (
        <form onSubmit={submitIssue} className="space-y-4">
          <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
            <h2 className="font-semibold text-gi-text text-sm">Izdavanje materijala na projekt</h2>
            {success && <SuccessCard message="Materijal uspješno izdan." onClose={() => setSuccess(false)} />}
            {error && <ErrorCard message={error} />}
            <div className="space-y-3">
              <div>
                <label className="block text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">Materijal *</label>
                <select value={balanceID} onChange={e => setBalanceID(e.target.value)} required
                  className="w-full px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text focus:outline-none focus:border-gi-yellow/50">
                  <option value="">Odaberi materijal...</option>
                  {stock.map(s => (
                    <option key={s.id} value={s.id}>{s.material_name} — {fmt(s.quantity)} {s.unit}</option>
                  ))}
                </select>
                {issueSelected && (
                  <p className="text-[11px] text-gi-text-subtle mt-1">
                    Dostupno: {fmt(issueSelected.quantity)} {issueSelected.unit}
                    {canSeeFinancials && <span> · Pr. cijena: {fmt(issueSelected.avg_cost)} €/{issueSelected.unit}</span>}
                  </p>
                )}
              </div>
              <div>
                <label className="block text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">Projekt *</label>
                <select value={projectID} onChange={e => setProjectID(e.target.value)} required
                  className="w-full px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text focus:outline-none focus:border-gi-yellow/50">
                  <option value="">Odaberi projekt...</option>
                  {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                </select>
              </div>
              <Field label="Količina *" type="number" value={issueQty} onChange={setIssueQty} placeholder="0.000" />
              <Field label="Napomena" value={issueNotes} onChange={setIssueNotes} placeholder="Opcijski komentar" />
            </div>
          </div>
          <button type="submit" disabled={saving}
            className="w-full py-2.5 bg-gi-yellow text-gi-navy font-semibold rounded-xl text-sm hover:bg-gi-yellow-hover disabled:opacity-50 transition-colors">
            {saving ? 'Izdavanje...' : 'Izadaj materijal'}
          </button>
        </form>
      )}

      {/* Return form */}
      {mode === 'return' && (
        <>
          <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
            <h2 className="font-semibold text-gi-text text-sm">Povrat materijala s projekta</h2>
            {success && <SuccessCard message="Povrat uspješno zabilježen." onClose={() => setSuccess(false)} />}
            {error && <ErrorCard message={error} />}
            <div className="space-y-3">
              <div>
                <label className="block text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">Projekt *</label>
                <select value={retProjectID} onChange={e => setRetProjectID(e.target.value)} required
                  className="w-full px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text focus:outline-none focus:border-gi-yellow/50">
                  <option value="">Odaberi projekt...</option>
                  {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                </select>
              </div>

              <div>
                <label className="block text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">Materijal za povrat *</label>
                <select value={retProjMatID} onChange={e => setRetProjMatID(e.target.value)} required
                  disabled={!retProjectID || retLoadingMats}
                  className="w-full px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text focus:outline-none focus:border-gi-yellow/50 disabled:opacity-50">
                  <option value="">
                    {retLoadingMats ? 'Učitavanje...' : retProjectID ? 'Odaberi materijal...' : 'Prvo odaberi projekt'}
                  </option>
                  {retProjectMats.map(m => (
                    <option key={m.id} value={m.id}>
                      {m.material_name} — dostupno: {fmt(m.available_quantity)} {m.unit}
                    </option>
                  ))}
                </select>
                {retMatError && <ErrorCard message={retMatError} />}
                {retProjectID && !retLoadingMats && !retMatError && retProjectMats.length === 0 && (
                  <p className="text-[11px] text-amber-400 mt-1">Nema materijala za povrat na ovom projektu.</p>
                )}
                {retSelectedMat && (
                  <p className="text-[11px] text-gi-text-subtle mt-1">
                    Dostupno na projektu: <strong className="text-gi-text">{fmt(retSelectedMat.available_quantity)} {retSelectedMat.unit}</strong>
                    {retMatchingBalance && <span className="text-gi-text-subtle"> · Trenutno u skladištu: {fmt(retMatchingBalance.quantity)}</span>}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">Količina *</label>
                <input
                  type="number" step="any" min="0.001"
                  max={retSelectedMat?.available_quantity}
                  value={retQty} onChange={e => setRetQty(e.target.value)}
                  required disabled={!retProjMatID}
                  placeholder="0.000"
                  className="w-full px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50 disabled:opacity-50"
                />
              </div>

              <Field label="Napomena" value={retNotes} onChange={setRetNotes} placeholder="Opcijski komentar" />
            </div>
          </div>

          <button type="button"
            disabled={saving || !retProjMatID || !retQty || !retMatchingBalance}
            onClick={() => setRetConfirm(true)}
            className="w-full py-2.5 bg-gi-yellow text-gi-navy font-semibold rounded-xl text-sm hover:bg-gi-yellow-hover disabled:opacity-50 transition-colors">
            {saving ? 'Povrat u tijeku...' : 'Potvrdi povrat'}
          </button>

          {/* Confirmation dialog */}
          {retConfirm && retSelectedMat && retProject && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={() => setRetConfirm(false)}>
              <div className="bg-gi-card border border-gi-border rounded-2xl p-6 w-full max-w-sm space-y-4 shadow-xl" onClick={e => e.stopPropagation()}>
                <h3 className="font-semibold text-gi-text">Potvrda povrata</h3>
                <div className="space-y-1.5 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gi-text-muted">Materijal</span>
                    <span className="text-gi-text font-medium">{retSelectedMat.material_name}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gi-text-muted">Projekt</span>
                    <span className="text-gi-text font-medium">{retProject.name}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gi-text-muted">Količina</span>
                    <span className="text-gi-text font-medium">{fmt(parseFloat(retQty) || 0)} {retSelectedMat.unit}</span>
                  </div>
                </div>
                <p className="text-xs text-gi-text-subtle">Materijal će biti vraćen u skladište i oduzet s dostupne zalihe projekta.</p>
                <div className="flex gap-3 pt-1">
                  <button type="button" onClick={() => setRetConfirm(false)}
                    className="flex-1 py-2 border border-gi-border rounded-xl text-sm text-gi-text-muted hover:text-gi-text transition-colors">
                    Odustani
                  </button>
                  <button type="button" onClick={submitReturn}
                    className="flex-1 py-2 bg-gi-yellow text-gi-navy font-semibold rounded-xl text-sm hover:bg-gi-yellow-hover transition-colors">
                    Potvrdi
                  </button>
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}

// ── Povijest tab (movements) ──────────────────────────────────────────────────

function MovementsTab(_: { companyID: string; role: string }) {
  const [data, setData] = useState<MovementListResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [typeFilter, setTypeFilter] = useState('')

  const fetch = useCallback(() => {
    setLoading(true)
    setError(null)
    const params = new URLSearchParams({ page: page.toString(), per_page: '25' })
    if (typeFilter) params.set('type', typeFilter)
    apiClient.get(`/warehouse/movements?${params}`)
      .then(r => setData(r.data))
      .catch(e => setError(apiErr(e)))
      .finally(() => setLoading(false))
  }, [page, typeFilter])

  useEffect(() => { fetch() }, [fetch])

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <select value={typeFilter} onChange={e => { setTypeFilter(e.target.value); setPage(1) }}
          className="px-3 py-2 bg-gi-card border border-gi-border rounded-lg text-sm text-gi-text focus:outline-none focus:border-gi-yellow/50">
          <option value="">Svi tipovi</option>
          <option value="receipt">Primke</option>
          <option value="issue">Izdavanja</option>
          <option value="return">Povrati</option>
          <option value="adjustment">Inventura</option>
        </select>
        <button onClick={fetch} className="p-2 rounded-lg bg-gi-card border border-gi-border text-gi-text-muted hover:text-gi-text transition-colors">
          <RefreshCw className="w-4 h-4" />
        </button>
      </div>

      {loading ? <LoadingCard /> : error ? <ErrorCard message={error} /> : (
        <>
          <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
            {!data || data.data.length === 0 ? (
              <p className="text-gi-text-subtle text-sm p-6 text-center">Nema pokreta.</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gi-border text-gi-text-subtle text-[11px] uppercase tracking-wider">
                    <th className="text-left px-4 py-3">Materijal</th>
                    <th className="text-left px-4 py-3 hidden sm:table-cell">Tip</th>
                    <th className="text-right px-4 py-3">Delta</th>
                    <th className="text-right px-4 py-3 hidden md:table-cell">Nakon</th>
                    <th className="text-left px-4 py-3 hidden lg:table-cell">Projekt</th>
                    <th className="text-right px-4 py-3 hidden sm:table-cell">Datum</th>
                  </tr>
                </thead>
                <tbody>
                  {data.data.map(m => (
                    <tr key={m.id} className="border-b border-gi-border/50 hover:bg-gi-navy-hover transition-colors">
                      <td className="px-4 py-3">
                        <p className="font-medium text-gi-text">{m.material_name}</p>
                        <p className="text-[11px] text-gi-text-subtle">{m.created_by_name}</p>
                      </td>
                      <td className="px-4 py-3 hidden sm:table-cell">
                        <span className={`text-xs font-medium ${movementColor(m.movement_type)}`}>{movementLabel(m.movement_type)}</span>
                      </td>
                      <td className={`px-4 py-3 text-right font-mono text-sm ${m.quantity_delta > 0 ? 'text-green-400' : 'text-red-400'}`}>
                        {m.quantity_delta > 0 ? '+' : ''}{fmt(m.quantity_delta)} {m.unit}
                      </td>
                      <td className="px-4 py-3 text-right text-gi-text-muted hidden md:table-cell">{fmt(m.quantity_after)}</td>
                      <td className="px-4 py-3 text-gi-text-muted hidden lg:table-cell">{m.project_name ?? '—'}</td>
                      <td className="px-4 py-3 text-right text-gi-text-subtle hidden sm:table-cell">
                        {new Date(m.created_at).toLocaleDateString('hr-HR')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {data && data.pagination.total_pages > 1 && (
            <div className="flex items-center justify-between text-sm text-gi-text-subtle">
              <span>Stranica {data.pagination.page} od {data.pagination.total_pages} ({data.pagination.total} pokreta)</span>
              <div className="flex gap-2">
                <button disabled={page <= 1} onClick={() => setPage(p => p - 1)}
                  className="px-3 py-1.5 rounded-lg border border-gi-border disabled:opacity-40 hover:bg-gi-navy-hover transition-colors">
                  ← Prethodna
                </button>
                <button disabled={page >= data.pagination.total_pages} onClick={() => setPage(p => p + 1)}
                  className="px-3 py-1.5 rounded-lg border border-gi-border disabled:opacity-40 hover:bg-gi-navy-hover transition-colors">
                  Sljedeća →
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}

// ── Inventura tab (adjustments) ───────────────────────────────────────────────

function AdjustmentTab(_: { companyID: string }) {
  const [stock, setStock] = useState<StockItem[]>([])
  const [stockError, setStockError] = useState<string | null>(null)
  const [stockRetry, setStockRetry] = useState(0)
  const [balanceID, setBalanceID] = useState('')
  const [newQuantity, setNewQuantity] = useState('')
  const [notes, setNotes] = useState('')
  const [saving, setSaving] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Phase 2: idempotency key — generated once per form lifecycle, regenerated after success
  const [iKey, setIKey] = useState(() => crypto.randomUUID())

  useEffect(() => {
    setStockError(null)
    apiClient.get('/warehouse/stock?active_only=true')
      .then(r => {
        const arr = r.data?.data
        if (!Array.isArray(arr)) throw new Error('Neočekivani format zalihe materijala.')
        setStock(arr)
      })
      .catch(e => setStockError(apiErr(e) || 'Greška pri učitavanju materijala.'))
  }, [stockRetry])

  const selected = stock.find(s => s.id === balanceID)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      await apiClient.post('/warehouse/adjustments', {
        balance_id:   balanceID,
        new_quantity: parseFloat(newQuantity),
        notes:        notes || null,
      }, { headers: { 'X-Idempotency-Key': iKey } })
      setIKey(crypto.randomUUID())
      setSuccess(true)
      setBalanceID(''); setNewQuantity(''); setNotes('')
    } catch (err) {
      setError(apiErr(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4 max-w-lg">
      <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
        <h2 className="font-semibold text-gi-text text-sm">Korekcija zalihe</h2>
        <p className="text-xs text-gi-text-subtle">Upotrijebite ovu funkciju za usklađivanje fizičkog stanja s evidencijom. Razlika će biti zabilježena kao pokret vrste <em>Inventura</em>.</p>

        {success && <SuccessCard message="Korekcija uspješno zabilježena." onClose={() => setSuccess(false)} />}
        {error && <ErrorCard message={error} />}
        {stockError && (
          <div className="space-y-2">
            <ErrorCard message={stockError} />
            <button type="button" onClick={() => setStockRetry(r => r + 1)}
              className="w-full py-2 border border-gi-border rounded-xl text-sm text-gi-text-muted hover:text-gi-text transition-colors">
              Pokušaj ponovo učitati materijale
            </button>
          </div>
        )}

        <div className="space-y-3">
          <div>
            <label className="block text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">Materijal *</label>
            <select value={balanceID} onChange={e => setBalanceID(e.target.value)} required
              className="w-full px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text focus:outline-none focus:border-gi-yellow/50">
              <option value="">Odaberi materijal...</option>
              {stock.map(s => (
                <option key={s.id} value={s.id}>{s.material_name} — {fmt(s.quantity)} {s.unit}</option>
              ))}
            </select>
            {selected && (
              <p className="text-[11px] text-gi-text-subtle mt-1">Trenutno stanje: <strong className="text-gi-text">{fmt(selected.quantity)} {selected.unit}</strong></p>
            )}
          </div>

          <Field label="Nova količina *" type="number" value={newQuantity} onChange={setNewQuantity} placeholder="0.000" />
          <Field label="Napomena (razlog korekcije)" value={notes} onChange={setNotes} placeholder="npr. Fizičkim popisom utvrđeno stanje" />
        </div>
      </div>

      <button type="submit" disabled={saving}
        className="w-full py-2.5 bg-gi-yellow text-gi-navy font-semibold rounded-xl text-sm hover:bg-gi-yellow-hover disabled:opacity-50 transition-colors">
        {saving ? 'Spremanje...' : 'Spremi korekciju'}
      </button>
    </form>
  )
}

// ── Shared micro-components ───────────────────────────────────────────────────

function StatCard({ label, value, icon, accent }: { label: string; value: string; icon: React.ReactNode; accent?: string }) {
  return (
    <div className="bg-gi-card border border-gi-border rounded-xl p-4 space-y-2">
      <div className="flex items-center gap-2 text-gi-text-subtle">
        {icon}<span className="text-[11px] uppercase tracking-wider font-medium">{label}</span>
      </div>
      <p className={`text-xl font-bold ${accent ?? 'text-gi-text'}`}>{value}</p>
    </div>
  )
}

function Field({ label, value, onChange, placeholder, type = 'text' }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string; type?: string
}) {
  return (
    <div>
      <label className="block text-[11px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">{label}</label>
      <input type={type} value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder}
        className="w-full px-3 py-2 bg-gi-workspace border border-gi-border rounded-lg text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow/50" />
    </div>
  )
}

function SuccessCard({ message, onClose }: { message: string; onClose: () => void }) {
  return (
    <div className="bg-green-900/30 border border-green-700/50 rounded-lg px-4 py-3 text-green-300 text-sm flex items-center gap-2">
      <TrendingUp className="w-4 h-4 flex-shrink-0" />
      {message}
      <button type="button" onClick={onClose} className="ml-auto text-green-400 hover:text-green-300">×</button>
    </div>
  )
}

function LoadingCard() {
  return (
    <div className="flex items-center justify-center h-32 border border-gi-border rounded-xl bg-gi-card">
      <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje...</p>
    </div>
  )
}

function ErrorCard({ message }: { message: string }) {
  return (
    <div className="bg-red-900/20 border border-red-700/40 rounded-xl px-4 py-3 text-red-300 text-sm">{message}</div>
  )
}
