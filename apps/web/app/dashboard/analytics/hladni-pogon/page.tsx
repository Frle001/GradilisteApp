'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import AnalyticsTabNav from '@/components/analytics/AnalyticsTabNav'
import type {
  ColdOpsSummary,
  ColdOpsCategory,
  ColdOpsExpense,
  RecurringDef,
  ColdOpsOccurrence,
  ColdOpsBudget,
} from '@/lib/types/analytics'

const FORBIDDEN_ROLES = ['administracija', 'poslovoda', 'radnik']

const HR_MONTHS = [
  '', 'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac',
]

const PAYMENT_LABELS: Record<string, string> = {
  cash: 'Gotovina', card: 'Kartica', transfer: 'Transakcija', other: 'Ostalo',
}

function formatEur(n: number): string {
  return n.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €'
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('hr-HR')
}

function pctLabel(pct?: number): string {
  if (pct == null) return '—'
  const sign = pct >= 0 ? '+' : ''
  return sign + pct.toFixed(1) + '%'
}

function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Greška'
}

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
  'text-sm text-gi-text placeholder:text-gi-text-subtle',
  'focus:outline-none focus:border-gi-yellow',
  'disabled:opacity-40 disabled:cursor-not-allowed',
  'transition-colors',
].join(' ')

const btnPrimary = 'bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy font-semibold text-sm px-4 py-2 rounded-lg transition-colors disabled:opacity-50'
const btnSecondary = 'bg-gi-card-elevated hover:bg-gi-border text-gi-text text-sm px-3 py-2 rounded-lg border border-gi-border transition-colors'
const btnDanger = 'bg-red-600/10 hover:bg-red-600/20 text-red-400 text-sm px-3 py-2 rounded-lg border border-red-600/20 transition-colors'

type Tab = 'summary' | 'expenses' | 'recurring' | 'budgets'

// ── Stat card ─────────────────────────────────────────────────────────────────

function StatCard({ title, value, sub, accent }: { title: string; value: string; sub?: string; accent: string }) {
  return (
    <div className={`bg-gi-card border border-gi-border rounded-xl px-4 py-3.5 border-l-4 ${accent}`}>
      <p className="text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1">{title}</p>
      <p className="text-xl font-bold text-gi-text tabular-nums leading-none">{value}</p>
      {sub && <p className="text-xs text-gi-text-subtle mt-1">{sub}</p>}
    </div>
  )
}

// ── Trend bar chart (CSS-only) ────────────────────────────────────────────────

function TrendBar({ label, value, max }: { label: string; value: number; max: number }) {
  const pct = max > 0 ? Math.round((value / max) * 100) : 0
  return (
    <div className="flex items-center gap-2">
      <span className="w-14 text-xs text-gi-text-muted text-right shrink-0">{label}</span>
      <div className="flex-1 h-5 bg-gi-card-elevated rounded overflow-hidden">
        <div
          className="h-full bg-gi-yellow/60 rounded transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="w-20 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">{formatEur(value)}</span>
    </div>
  )
}

// ── Predefined color swatches ─────────────────────────────────────────────────

const COLOR_SWATCHES = [
  { hex: '#f5c500', label: 'Žuta' },
  { hex: '#3b82f6', label: 'Plava' },
  { hex: '#22c55e', label: 'Zelena' },
  { hex: '#ef4444', label: 'Crvena' },
  { hex: '#a855f7', label: 'Ljubičasta' },
  { hex: '#f97316', label: 'Narančasta' },
  { hex: '#14b8a6', label: 'Tirkizna' },
  { hex: '#6b7280', label: 'Siva' },
]

// ── Category badge ────────────────────────────────────────────────────────────

function CategoryBadge({ name, color }: { name: string; color?: string }) {
  // Always use light text (readable on dark theme); convey color via dot + tinted bg + border.
  const style: React.CSSProperties = color
    ? { background: color + '22', borderColor: color + '70' }
    : {}
  return (
    <span
      className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full border border-gi-border text-gi-text"
      style={style}
    >
      {color && <span aria-hidden="true" className="w-2 h-2 rounded-full shrink-0 inline-block" style={{ background: color }} />}
      {name}
    </span>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function HladniPogonPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()
  const [tab, setTab] = useState<Tab>('summary')

  const [summary, setSummary] = useState<ColdOpsSummary | null>(null)
  const [categories, setCategories] = useState<ColdOpsCategory[]>([])
  const [expenses, setExpenses] = useState<ColdOpsExpense[]>([])
  const [recurring, setRecurring] = useState<RecurringDef[]>([])
  const [budgets, setBudgets] = useState<ColdOpsBudget[]>([])

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [toast, setToast] = useState('')

  // ── Modals ──────────────────────────────────────────────────────────────────
  const [showCatForm, setShowCatForm] = useState(false)
  const [editCat, setEditCat] = useState<ColdOpsCategory | null>(null)
  const [catName, setCatName] = useState('')
  const [catDesc, setCatDesc] = useState('')
  const [catColor, setCatColor] = useState('')
  const [catActive, setCatActive] = useState(true)
  const [catSaving, setCatSaving] = useState(false)

  const [showExpenseForm, setShowExpenseForm] = useState(false)
  const [editExp, setEditExp] = useState<ColdOpsExpense | null>(null)
  const [expTitle, setExpTitle] = useState('')
  const [expAmount, setExpAmount] = useState('')
  const [expDate, setExpDate] = useState('')
  const [expType, setExpType] = useState('variable')
  const [expCat, setExpCat] = useState('')
  const [expNotes, setExpNotes] = useState('')
  const [expSupplier, setExpSupplier] = useState('')
  const [expPayment, setExpPayment] = useState('')
  const [expSaving, setExpSaving] = useState(false)

  const [showRecurForm, setShowRecurForm] = useState(false)
  const [editRecur, setEditRecur] = useState<RecurringDef | null>(null)
  const [recurTitle, setRecurTitle] = useState('')
  const [recurFreq, setRecurFreq] = useState('monthly')
  const [recurDay, setRecurDay] = useState('1')
  const [recurMonth, setRecurMonth] = useState('')
  const [recurStart, setRecurStart] = useState('')
  const [recurEnd, setRecurEnd] = useState('')
  const [recurAmount, setRecurAmount] = useState('')
  const [recurCat, setRecurCat] = useState('')
  const [recurActive, setRecurActive] = useState(true)
  const [recurSaving, setRecurSaving] = useState(false)

  const [occs, setOccs] = useState<ColdOpsOccurrence[]>([])
  const [occsDefId, setOccsDefId] = useState<string | null>(null)
  const [occsSaving, setOccsSaving] = useState('')

  const [showBudgetForm, setShowBudgetForm] = useState(false)
  const [budgetYear, setBudgetYear] = useState(String(new Date().getFullYear()))
  const [budgetMonth, setBudgetMonth] = useState(String(new Date().getMonth() + 1))
  const [budgetAmount, setBudgetAmount] = useState('')
  const [budgetCat, setBudgetCat] = useState('')
  const [budgetSaving, setBudgetSaving] = useState(false)

  // ── Auth guard ──────────────────────────────────────────────────────────────
  useEffect(() => {
    if (!isLoading && user && FORBIDDEN_ROLES.includes(user.role)) {
      router.replace('/dashboard')
    }
  }, [isLoading, user, router])

  // ── Data loaders ────────────────────────────────────────────────────────────
  const loadSummary = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await apiClient.get('/analytics/hladni-pogon/summary')
      setSummary(res.data.summary)
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadCategories = useCallback(async () => {
    const res = await apiClient.get('/analytics/hladni-pogon/categories')
    setCategories(res.data.categories ?? [])
  }, [])

  const loadExpenses = useCallback(async () => {
    setLoading(true)
    try {
      const res = await apiClient.get('/analytics/hladni-pogon/expenses')
      setExpenses(res.data.expenses ?? [])
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadRecurring = useCallback(async () => {
    setLoading(true)
    try {
      const res = await apiClient.get('/analytics/hladni-pogon/recurring')
      setRecurring(res.data.definitions ?? [])
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadBudgets = useCallback(async () => {
    const y = new Date().getFullYear()
    const m = new Date().getMonth() + 1
    const res = await apiClient.get(`/analytics/hladni-pogon/budgets?year=${y}&month=${m}`)
    setBudgets(res.data.budgets ?? [])
  }, [])

  const loadOccurrences = useCallback(async (defId: string) => {
    const res = await apiClient.get(`/analytics/hladni-pogon/recurring/${defId}/occurrences`)
    setOccs(res.data.occurrences ?? [])
    setOccsDefId(defId)
  }, [])

  useEffect(() => {
    if (!user || FORBIDDEN_ROLES.includes(user.role)) return
    if (tab === 'summary') {
      loadSummary()
      loadCategories()
    }
    if (tab === 'expenses') {
      loadExpenses()
      loadCategories()
    }
    if (tab === 'recurring') loadRecurring()
    if (tab === 'budgets') {
      loadBudgets()
      loadCategories()
    }
  }, [tab, user, loadSummary, loadExpenses, loadRecurring, loadBudgets, loadCategories])

  // ── Toast helper ────────────────────────────────────────────────────────────
  function showToast(msg: string) {
    setToast(msg)
    setTimeout(() => setToast(''), 3000)
  }

  // ── Category CRUD ───────────────────────────────────────────────────────────
  function openCatForm(cat?: ColdOpsCategory) {
    setEditCat(cat ?? null)
    setCatName(cat?.name ?? '')
    setCatDesc(cat?.description ?? '')
    setCatColor(cat?.color ?? '')
    setCatActive(cat?.is_active ?? true)
    setShowCatForm(true)
  }

  async function saveCat(e: React.FormEvent) {
    e.preventDefault()
    setCatSaving(true)
    try {
      if (editCat) {
        await apiClient.put(`/analytics/hladni-pogon/categories/${editCat.id}`, {
          name: catName, description: catDesc || null, color: catColor || null, is_active: catActive,
        })
      } else {
        await apiClient.post('/analytics/hladni-pogon/categories', {
          name: catName, description: catDesc || null, color: catColor || null,
        })
      }
      await loadCategories()
      setShowCatForm(false)
      showToast('Kategorija spremljena')
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setCatSaving(false)
    }
  }

  async function deleteCat(id: string) {
    if (!confirm('Obrisati kategoriju?')) return
    try {
      await apiClient.delete(`/analytics/hladni-pogon/categories/${id}`)
      await loadCategories()
      showToast('Kategorija uklonjena')
    } catch (e) {
      setError(errMsg(e))
    }
  }

  // ── Expense CRUD ─────────────────────────────────────────────────────────────
  function openExpForm(exp?: ColdOpsExpense) {
    setEditExp(exp ?? null)
    setExpTitle(exp?.title ?? '')
    setExpAmount(exp ? String(exp.amount) : '')
    setExpDate(exp?.expense_date ?? new Date().toISOString().slice(0, 10))
    setExpType(exp?.expense_type ?? 'variable')
    setExpCat(exp?.category_id ?? '')
    setExpNotes(exp?.notes ?? '')
    setExpSupplier(exp?.supplier ?? '')
    setExpPayment(exp?.payment_method ?? '')
    setShowExpenseForm(true)
  }

  async function saveExp(e: React.FormEvent) {
    e.preventDefault()
    setExpSaving(true)
    try {
      const payload = {
        title: expTitle,
        amount: parseFloat(expAmount),
        expense_date: expDate,
        expense_type: expType,
        category_id: expCat || null,
        notes: expNotes || null,
        supplier: expSupplier || null,
        payment_method: expPayment || null,
      }
      if (editExp) {
        await apiClient.put(`/analytics/hladni-pogon/expenses/${editExp.id}`, payload)
      } else {
        await apiClient.post('/analytics/hladni-pogon/expenses', payload)
      }
      await loadExpenses()
      setShowExpenseForm(false)
      showToast('Trošak spremljen')
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setExpSaving(false)
    }
  }

  async function deleteExp(id: string) {
    if (!confirm('Obrisati trošak?')) return
    try {
      await apiClient.delete(`/analytics/hladni-pogon/expenses/${id}`)
      await loadExpenses()
      showToast('Trošak obrisan')
    } catch (e) {
      setError(errMsg(e))
    }
  }

  // ── Recurring CRUD ──────────────────────────────────────────────────────────
  function openRecurForm(def?: RecurringDef) {
    setEditRecur(def ?? null)
    setRecurTitle(def?.title ?? '')
    setRecurFreq(def?.frequency ?? 'monthly')
    setRecurDay(String(def?.day_of_month ?? 1))
    setRecurMonth(def?.month_of_year != null ? String(def.month_of_year) : '')
    setRecurStart(def?.start_date ?? new Date().toISOString().slice(0, 10))
    setRecurEnd(def?.end_date ?? '')
    setRecurAmount(def?.current_amount != null ? String(def.current_amount) : '')
    setRecurCat(def?.category_id ?? '')
    setRecurActive(def?.is_active ?? true)
    setShowRecurForm(true)
  }

  async function saveRecur(e: React.FormEvent) {
    e.preventDefault()
    setRecurSaving(true)
    try {
      if (editRecur) {
        await apiClient.put(`/analytics/hladni-pogon/recurring/${editRecur.id}`, {
          title: recurTitle,
          category_id: recurCat || null,
          day_of_month: parseInt(recurDay),
          month_of_year: recurFreq === 'yearly' ? parseInt(recurMonth) : null,
          end_date: recurEnd || null,
          new_amount: recurAmount ? parseFloat(recurAmount) : null,
          is_active: recurActive,
        })
      } else {
        await apiClient.post('/analytics/hladni-pogon/recurring', {
          title: recurTitle,
          frequency: recurFreq,
          category_id: recurCat || null,
          day_of_month: parseInt(recurDay),
          month_of_year: recurFreq === 'yearly' ? parseInt(recurMonth) : null,
          start_date: recurStart,
          end_date: recurEnd || null,
          amount: parseFloat(recurAmount),
        })
      }
      await loadRecurring()
      setShowRecurForm(false)
      showToast('Definicija spremljena')
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setRecurSaving(false)
    }
  }

  async function confirmOcc(id: string) {
    const amountStr = prompt('Unesite iznos za potvrdu (€):')
    if (!amountStr) return
    const amount = parseFloat(amountStr)
    if (isNaN(amount) || amount <= 0) return
    setOccsSaving(id)
    try {
      await apiClient.post(`/analytics/hladni-pogon/occurrences/${id}/confirm`, { amount })
      if (occsDefId) await loadOccurrences(occsDefId)
      showToast('Plaćanje potvrđeno')
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setOccsSaving('')
    }
  }

  async function skipOcc(id: string) {
    if (!confirm('Preskočiti ovu obvezu?')) return
    setOccsSaving(id)
    try {
      await apiClient.post(`/analytics/hladni-pogon/occurrences/${id}/skip`, {})
      if (occsDefId) await loadOccurrences(occsDefId)
      showToast('Obveza preskočena')
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setOccsSaving('')
    }
  }

  // ── Budget CRUD ─────────────────────────────────────────────────────────────
  async function saveBudget(e: React.FormEvent) {
    e.preventDefault()
    setBudgetSaving(true)
    try {
      await apiClient.put('/analytics/hladni-pogon/budgets', {
        year: parseInt(budgetYear),
        month: parseInt(budgetMonth),
        budget_amount: parseFloat(budgetAmount),
        category_id: budgetCat || null,
      })
      await loadBudgets()
      setShowBudgetForm(false)
      showToast('Budžet postavljen')
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setBudgetSaving(false)
    }
  }

  // ── Guard ───────────────────────────────────────────────────────────────────
  if (isLoading) return <LoadingScreen />
  if (!user || FORBIDDEN_ROLES.includes(user.role)) return null

  const now = new Date()

  // ── Render ──────────────────────────────────────────────────────────────────
  return (
    <AppShell user={user} employee={employee} title="Analitika" backHref="/dashboard" onLogout={logout}>
      <div className="max-w-6xl mx-auto px-4 py-6 space-y-6">

        <AnalyticsTabNav />

        {/* Header */}
        <div>
          <h1 className="text-2xl font-bold text-gi-text">Hladni pogon</h1>
          <p className="text-sm text-gi-text-subtle mt-1">Operativni troškovi poduzeća</p>
        </div>

        {/* Toast */}
        {toast && (
          <div className="bg-gi-yellow/10 border border-gi-yellow/40 text-gi-yellow text-sm rounded-lg px-4 py-2">
            {toast}
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="bg-red-600/10 border border-red-600/30 text-red-400 text-sm rounded-lg px-4 py-2 flex justify-between">
            <span>{error}</span>
            <button onClick={() => setError('')} className="ml-4 hover:text-red-300">×</button>
          </div>
        )}

        {/* Tabs */}
        <div className="flex gap-1 bg-gi-card border border-gi-border rounded-xl p-1 w-fit">
          {(['summary', 'expenses', 'recurring', 'budgets'] as Tab[]).map(t => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={[
                'px-4 py-1.5 rounded-lg text-sm font-medium transition-colors',
                tab === t
                  ? 'bg-gi-yellow text-gi-navy'
                  : 'text-gi-text-muted hover:text-gi-text',
              ].join(' ')}
            >
              {{ summary: 'Pregled', expenses: 'Troškovi', recurring: 'Ponavljajući', budgets: 'Budžet' }[t]}
            </button>
          ))}
        </div>

        {loading && <p className="text-sm text-gi-text-subtle">Učitavanje…</p>}

        {/* ── Summary tab ────────────────────────────────────────────────────── */}
        {tab === 'summary' && summary && (
          <div className="space-y-6">
            {/* KPI cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <StatCard
                title="Ovaj mjesec"
                value={formatEur(summary.current_month.total)}
                sub={pctLabel(summary.current_month.pct_change) + ' vs. prethodni'}
                accent="border-gi-yellow"
              />
              <StatCard
                title="Prošli mjesec"
                value={formatEur(summary.prev_month.total)}
                sub={`${summary.prev_month.count} troška`}
                accent="border-gi-border"
              />
              <StatCard
                title="Ova godina"
                value={formatEur(summary.current_year.total)}
                sub={pctLabel(summary.current_year.pct_change) + ' vs. prošla'}
                accent="border-blue-500"
              />
              <StatCard
                title="Godišnja obveza"
                value={formatEur(summary.annualized_commitment)}
                sub="Aktivni ponavljajući"
                accent="border-purple-500"
              />
            </div>

            {/* Fixed vs variable */}
            <div className="bg-gi-card border border-gi-border rounded-xl p-4">
              <h2 className="text-sm font-semibold text-gi-text mb-3">Fiksni vs. varijabilni (ova godina)</h2>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-xs text-gi-text-muted uppercase tracking-wide">Fiksni</p>
                  <p className="text-lg font-bold text-gi-text tabular-nums">{formatEur(summary.fixed_vs_variable.fixed_total)}</p>
                  <p className="text-xs text-gi-text-subtle">{summary.fixed_vs_variable.fixed_count} troška</p>
                </div>
                <div>
                  <p className="text-xs text-gi-text-muted uppercase tracking-wide">Varijabilni</p>
                  <p className="text-lg font-bold text-gi-text tabular-nums">{formatEur(summary.fixed_vs_variable.variable_total)}</p>
                  <p className="text-xs text-gi-text-subtle">{summary.fixed_vs_variable.variable_count} troška</p>
                </div>
              </div>
            </div>

            {/* Monthly trend */}
            {summary.monthly_trend.length > 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                <h2 className="text-sm font-semibold text-gi-text mb-3">Trend po mjesecima</h2>
                <div className="space-y-2">
                  {(() => {
                    const max = Math.max(...summary.monthly_trend.map(t => t.total))
                    return summary.monthly_trend.map(t => (
                      <TrendBar
                        key={`${t.year}-${t.month}`}
                        label={`${HR_MONTHS[t.month]?.slice(0, 3)} '${String(t.year).slice(2)}`}
                        value={t.total}
                        max={max}
                      />
                    ))
                  })()}
                </div>
              </div>
            )}

            {/* Category distribution */}
            {summary.category_distribution.length > 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                <h2 className="text-sm font-semibold text-gi-text mb-3">Po kategorijama (ova godina)</h2>
                <div className="space-y-2">
                  {summary.category_distribution.map((c, i) => (
                    <div key={c.category_id ?? i} className="flex items-center gap-3">
                      <span className="w-32 text-xs text-gi-text truncate">{c.category_name}</span>
                      <div className="flex-1 h-4 bg-gi-card-elevated rounded overflow-hidden">
                        <div
                          className="h-full rounded transition-all"
                          style={{
                            width: `${c.pct}%`,
                            background: c.color ?? 'rgba(var(--gi-yellow-rgb,245,197,0),0.6)',
                          }}
                        />
                      </div>
                      <span className="w-10 text-xs text-gi-text-subtle text-right">{c.pct.toFixed(0)}%</span>
                      <span className="w-24 text-xs text-gi-text-subtle text-right tabular-nums">{formatEur(c.total)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Upcoming occurrences */}
            {summary.upcoming_occurrences.length > 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                <h2 className="text-sm font-semibold text-gi-text mb-3">Nadolazeće obveze (30 dana)</h2>
                <div className="divide-y divide-gi-border/60">
                  {summary.upcoming_occurrences.map(o => (
                    <div key={o.id} className="py-2.5 flex items-center justify-between gap-4">
                      <div>
                        <p className="text-sm text-gi-text">{o.definition_title}</p>
                        <p className="text-xs text-gi-text-subtle">{formatDate(o.due_date)}</p>
                      </div>
                      {o.amount && (
                        <span className="text-sm font-medium text-gi-text tabular-nums">{formatEur(o.amount)}</span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Budget utilization */}
            {summary.budget_utilization.length > 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                <h2 className="text-sm font-semibold text-gi-text mb-3">Iskorištenost budžeta ({HR_MONTHS[now.getMonth() + 1]})</h2>
                <div className="space-y-3">
                  {summary.budget_utilization.map((b, i) => (
                    <div key={b.category_id ?? i}>
                      <div className="flex justify-between text-xs text-gi-text-muted mb-1">
                        <span>{b.category_name}</span>
                        <span className={b.is_over_budget ? 'text-red-400' : ''}>
                          {formatEur(b.actual_amount)} / {formatEur(b.budget_amount)}
                          {b.is_over_budget && ' ⚠'}
                        </span>
                      </div>
                      <div className="h-2 bg-gi-card-elevated rounded overflow-hidden">
                        <div
                          className={`h-full rounded transition-all ${b.is_over_budget ? 'bg-red-500' : 'bg-gi-yellow/70'}`}
                          style={{ width: `${Math.min(b.utilization_pct, 100)}%` }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Categories management */}
            <div className="bg-gi-card border border-gi-border rounded-xl p-4">
              <div className="flex items-center justify-between mb-3">
                <h2 className="text-sm font-semibold text-gi-text">Kategorije</h2>
                <button onClick={() => openCatForm()} className={btnPrimary}>+ Nova</button>
              </div>
              {categories.length === 0 ? (
                <p className="text-sm text-gi-text-subtle">Nema kategorija.</p>
              ) : (
                <div className="divide-y divide-gi-border/60">
                  {categories.map(c => (
                    <div key={c.id} className="py-2.5 flex items-center justify-between gap-3">
                      <CategoryBadge name={c.name} color={c.color} />
                      {!c.is_active && <span className="text-xs text-gi-text-subtle">(neaktivna)</span>}
                      <div className="flex gap-2 ml-auto">
                        <button onClick={() => openCatForm(c)} className={btnSecondary}>Uredi</button>
                        <button onClick={() => deleteCat(c.id)} className={btnDanger}>Briši</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* ── Expenses tab ──────────────────────────────────────────────────── */}
        {tab === 'expenses' && (
          <div className="space-y-4">
            <div className="flex justify-end">
              <button onClick={() => openExpForm()} className={btnPrimary}>+ Novi trošak</button>
            </div>
            {expenses.length === 0 ? (
              <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center text-sm text-gi-text-subtle">
                Nema troškova. Dodajte prvi trošak.
              </div>
            ) : (
              <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gi-border">
                      <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide">Datum</th>
                      <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide">Naziv</th>
                      <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide hidden md:table-cell">Kategorija</th>
                      <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide hidden md:table-cell">Tip</th>
                      <th className="text-right px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide">Iznos</th>
                      <th className="px-4 py-2.5" />
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gi-border/60">
                    {expenses.map(e => (
                      <tr key={e.id} className="hover:bg-gi-card-elevated/50 transition-colors">
                        <td className="px-4 py-2.5 text-gi-text-subtle whitespace-nowrap">{formatDate(e.expense_date)}</td>
                        <td className="px-4 py-2.5 text-gi-text">
                          <p>{e.title}</p>
                          {e.supplier && <p className="text-xs text-gi-text-subtle">{e.supplier}</p>}
                        </td>
                        <td className="px-4 py-2.5 hidden md:table-cell">
                          {e.category_name
                            ? <CategoryBadge name={e.category_name} />
                            : <span className="text-gi-text-subtle">—</span>}
                        </td>
                        <td className="px-4 py-2.5 hidden md:table-cell">
                          <span className={`text-xs px-2 py-0.5 rounded-full border ${e.expense_type === 'fixed' ? 'border-blue-500/30 text-blue-400' : 'border-gi-border text-gi-text-subtle'}`}>
                            {e.expense_type === 'fixed' ? 'Fiksni' : 'Varijabilni'}
                          </span>
                        </td>
                        <td className="px-4 py-2.5 text-right font-medium text-gi-text tabular-nums whitespace-nowrap">
                          {formatEur(e.amount)}
                        </td>
                        <td className="px-4 py-2.5">
                          <div className="flex gap-1 justify-end">
                            <button onClick={() => openExpForm(e)} className={btnSecondary}>Uredi</button>
                            <button onClick={() => deleteExp(e.id)} className={btnDanger}>×</button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* ── Recurring tab ─────────────────────────────────────────────────── */}
        {tab === 'recurring' && (
          <div className="space-y-4">
            <div className="flex justify-end">
              <button onClick={() => openRecurForm()} className={btnPrimary}>+ Nova definicija</button>
            </div>
            {recurring.length === 0 ? (
              <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center text-sm text-gi-text-subtle">
                Nema ponavljajućih troškova.
              </div>
            ) : (
              <div className="space-y-3">
                {recurring.map(d => (
                  <div key={d.id} className="bg-gi-card border border-gi-border rounded-xl p-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <p className="font-medium text-gi-text">{d.title}</p>
                          {!d.is_active && <span className="text-xs text-gi-text-subtle">(neaktivna)</span>}
                          {d.category_name && <CategoryBadge name={d.category_name} />}
                        </div>
                        <p className="text-sm text-gi-text-subtle mt-0.5">
                          {d.frequency === 'monthly' ? 'Mjesečno' : 'Godišnje'}
                          {d.frequency === 'monthly' && ` · ${d.day_of_month}. u mjesecu`}
                          {d.frequency === 'yearly' && d.month_of_year && ` · ${d.day_of_month}. ${HR_MONTHS[d.month_of_year]}`}
                        </p>
                        {d.current_amount != null && (
                          <p className="text-sm font-medium text-gi-text tabular-nums mt-1">{formatEur(d.current_amount)}</p>
                        )}
                      </div>
                      <div className="flex gap-2 shrink-0">
                        <button onClick={() => openRecurForm(d)} className={btnSecondary}>Uredi</button>
                        <button
                          onClick={() => {
                            setOccsDefId(d.id)
                            loadOccurrences(d.id)
                          }}
                          className={btnSecondary}
                        >
                          Obveze
                        </button>
                      </div>
                    </div>

                    {/* Occurrences panel */}
                    {occsDefId === d.id && (
                      <div className="mt-4 border-t border-gi-border pt-4">
                        <h3 className="text-xs font-medium text-gi-text-muted uppercase tracking-wide mb-2">Generirane obveze</h3>
                        {occs.length === 0 ? (
                          <p className="text-sm text-gi-text-subtle">Nema obveza za ovaj period.</p>
                        ) : (
                          <div className="divide-y divide-gi-border/60">
                            {occs.map(o => (
                              <div key={o.id} className="py-2 flex items-center justify-between gap-3">
                                <div>
                                  <p className="text-sm text-gi-text">{formatDate(o.due_date)}</p>
                                  {o.amount && <p className="text-xs text-gi-text-subtle">{formatEur(o.amount)}</p>}
                                </div>
                                <div className="flex items-center gap-2">
                                  <span className={`text-xs px-2 py-0.5 rounded-full border ${
                                    o.status === 'confirmed' ? 'border-green-500/30 text-green-400' :
                                    o.status === 'skipped' ? 'border-gi-border text-gi-text-subtle' :
                                    'border-gi-yellow/30 text-gi-yellow'
                                  }`}>
                                    {o.status === 'confirmed' ? 'Plaćeno' : o.status === 'skipped' ? 'Preskočeno' : 'Planirano'}
                                  </span>
                                  {o.status === 'planned' && (
                                    <>
                                      <button
                                        disabled={occsSaving === o.id}
                                        onClick={() => confirmOcc(o.id)}
                                        className={btnPrimary + ' py-1 text-xs'}
                                      >
                                        Potvrdi
                                      </button>
                                      <button
                                        disabled={occsSaving === o.id}
                                        onClick={() => skipOcc(o.id)}
                                        className={btnSecondary + ' py-1 text-xs'}
                                      >
                                        Preskoči
                                      </button>
                                    </>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        )}
                        <button
                          onClick={() => setOccsDefId(null)}
                          className="mt-2 text-xs text-gi-text-subtle hover:text-gi-text"
                        >
                          Zatvori
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* ── Budgets tab ───────────────────────────────────────────────────── */}
        {tab === 'budgets' && (
          <div className="space-y-4">
            <div className="flex justify-end">
              <button onClick={() => setShowBudgetForm(true)} className={btnPrimary}>+ Postavi budžet</button>
            </div>
            {budgets.length === 0 ? (
              <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center text-sm text-gi-text-subtle">
                Nema budžeta za ovaj mjesec.
              </div>
            ) : (
              <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gi-border">
                      <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide">Kategorija</th>
                      <th className="text-right px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide">Budžet</th>
                      <th className="text-right px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide">Potrošeno</th>
                      <th className="text-right px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wide">Iskorištenost</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gi-border/60">
                    {budgets.map(b => (
                      <tr key={b.id} className="hover:bg-gi-card-elevated/50 transition-colors">
                        <td className="px-4 py-2.5 text-gi-text">
                          {b.category_name ?? <span className="text-gi-text-subtle">Ukupno</span>}
                        </td>
                        <td className="px-4 py-2.5 text-right tabular-nums text-gi-text">{formatEur(b.budget_amount)}</td>
                        <td className="px-4 py-2.5 text-right tabular-nums text-gi-text">{formatEur(b.actual_amount)}</td>
                        <td className="px-4 py-2.5 text-right">
                          <span className={b.utilization_pct > 100 ? 'text-red-400 font-medium' : 'text-gi-text-subtle'}>
                            {b.utilization_pct.toFixed(0)}%
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* ── Category form modal ────────────────────────────────────────────── */}
        {showCatForm && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-gi-workspace border border-gi-border rounded-2xl w-full max-w-md p-6">
              <h2 className="text-lg font-bold text-gi-text mb-4">
                {editCat ? 'Uredi kategoriju' : 'Nova kategorija'}
              </h2>
              <form onSubmit={saveCat} className="space-y-3">
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Naziv *</label>
                  <input className={inputCls} value={catName} onChange={e => setCatName(e.target.value)} required />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Opis</label>
                  <input className={inputCls} value={catDesc} onChange={e => setCatDesc(e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-2">Boja</label>
                  <div className="flex flex-wrap gap-2" role="radiogroup" aria-label="Odabir boje kategorije">
                    {COLOR_SWATCHES.map(s => {
                      const selected = catColor === s.hex
                      return (
                        <button
                          key={s.hex}
                          type="button"
                          role="radio"
                          aria-checked={selected}
                          aria-label={s.label}
                          title={s.label}
                          onClick={() => setCatColor(s.hex)}
                          onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setCatColor(s.hex) } }}
                          className={[
                            'w-8 h-8 rounded-full border-2 transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-gi-yellow focus-visible:ring-offset-2 focus-visible:ring-offset-gi-workspace',
                            selected ? 'border-white scale-110' : 'border-transparent hover:border-white/50',
                          ].join(' ')}
                          style={{ background: s.hex }}
                        >
                          {selected && (
                            <span className="flex items-center justify-center w-full h-full" aria-hidden="true">
                              <svg viewBox="0 0 12 12" className="w-3 h-3" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                <path d="M2 6l3 3 5-5" />
                              </svg>
                            </span>
                          )}
                        </button>
                      )
                    })}
                    {catColor && !COLOR_SWATCHES.some(s => s.hex === catColor) && (
                      <span
                        className="w-8 h-8 rounded-full border-2 border-white/60 flex items-center justify-center"
                        style={{ background: catColor }}
                        title={catColor}
                      >
                        <svg viewBox="0 0 12 12" className="w-3 h-3" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                          <path d="M2 6l3 3 5-5" />
                        </svg>
                      </span>
                    )}
                    {catColor && (
                      <button
                        type="button"
                        onClick={() => setCatColor('')}
                        className="w-8 h-8 rounded-full border-2 border-dashed border-gi-border hover:border-gi-text-subtle flex items-center justify-center transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-gi-yellow"
                        aria-label="Ukloni boju"
                        title="Bez boje"
                      >
                        <svg viewBox="0 0 12 12" className="w-3 h-3 text-gi-text-subtle" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                          <path d="M2 2l8 8M10 2l-8 8" />
                        </svg>
                      </button>
                    )}
                  </div>
                  {catName && (
                    <div className="mt-3 flex items-center gap-2">
                      <span className="text-xs text-gi-text-muted">Pregled:</span>
                      <CategoryBadge name={catName} color={catColor || undefined} />
                    </div>
                  )}
                </div>
                {editCat && (
                  <label className="flex items-center gap-2 text-sm text-gi-text cursor-pointer">
                    <input type="checkbox" checked={catActive} onChange={e => setCatActive(e.target.checked)} />
                    Aktivna
                  </label>
                )}
                <div className="flex gap-2 pt-2">
                  <button type="submit" disabled={catSaving} className={btnPrimary}>
                    {catSaving ? 'Sprema…' : 'Spremi'}
                  </button>
                  <button type="button" onClick={() => setShowCatForm(false)} className={btnSecondary}>
                    Odustani
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ── Expense form modal ─────────────────────────────────────────────── */}
        {showExpenseForm && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-gi-workspace border border-gi-border rounded-2xl w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
              <h2 className="text-lg font-bold text-gi-text mb-4">
                {editExp ? 'Uredi trošak' : 'Novi trošak'}
              </h2>
              <form onSubmit={saveExp} className="space-y-3">
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Naziv *</label>
                  <input className={inputCls} value={expTitle} onChange={e => setExpTitle(e.target.value)} required />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted mb-1">Iznos (€) *</label>
                    <input className={inputCls} type="number" step="0.01" min="0.01" value={expAmount} onChange={e => setExpAmount(e.target.value)} required />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted mb-1">Datum *</label>
                    <input className={inputCls} type="date" value={expDate} onChange={e => setExpDate(e.target.value)} required />
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Tip *</label>
                  <select className={inputCls} value={expType} onChange={e => setExpType(e.target.value)}>
                    <option value="variable">Varijabilni</option>
                    <option value="fixed">Fiksni</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Kategorija</label>
                  <select className={inputCls} value={expCat} onChange={e => setExpCat(e.target.value)}>
                    <option value="">Bez kategorije</option>
                    {categories.filter(c => c.is_active).map(c => (
                      <option key={c.id} value={c.id}>{c.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Dobavljač</label>
                  <input className={inputCls} value={expSupplier} onChange={e => setExpSupplier(e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Način plaćanja</label>
                  <select className={inputCls} value={expPayment} onChange={e => setExpPayment(e.target.value)}>
                    <option value="">—</option>
                    {Object.entries(PAYMENT_LABELS).map(([k, v]) => (
                      <option key={k} value={k}>{v}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Napomena</label>
                  <textarea className={inputCls} rows={2} value={expNotes} onChange={e => setExpNotes(e.target.value)} />
                </div>
                <div className="flex gap-2 pt-2">
                  <button type="submit" disabled={expSaving} className={btnPrimary}>
                    {expSaving ? 'Sprema…' : 'Spremi'}
                  </button>
                  <button type="button" onClick={() => setShowExpenseForm(false)} className={btnSecondary}>
                    Odustani
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ── Recurring form modal ───────────────────────────────────────────── */}
        {showRecurForm && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-gi-workspace border border-gi-border rounded-2xl w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
              <h2 className="text-lg font-bold text-gi-text mb-4">
                {editRecur ? 'Uredi ponavljajući' : 'Nova ponavljajuća obveza'}
              </h2>
              <form onSubmit={saveRecur} className="space-y-3">
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Naziv *</label>
                  <input className={inputCls} value={recurTitle} onChange={e => setRecurTitle(e.target.value)} required />
                </div>
                {!editRecur && (
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted mb-1">Frekvencija *</label>
                    <select className={inputCls} value={recurFreq} onChange={e => setRecurFreq(e.target.value)}>
                      <option value="monthly">Mjesečno</option>
                      <option value="yearly">Godišnje</option>
                    </select>
                  </div>
                )}
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted mb-1">Dan u mjesecu *</label>
                    <input className={inputCls} type="number" min="1" max="31" value={recurDay} onChange={e => setRecurDay(e.target.value)} required />
                  </div>
                  {recurFreq === 'yearly' && (
                    <div>
                      <label className="block text-xs font-medium text-gi-text-muted mb-1">Mjesec *</label>
                      <select className={inputCls} value={recurMonth} onChange={e => setRecurMonth(e.target.value)} required>
                        <option value="">—</option>
                        {HR_MONTHS.slice(1).map((m, i) => (
                          <option key={i + 1} value={i + 1}>{m}</option>
                        ))}
                      </select>
                    </div>
                  )}
                </div>
                {!editRecur && (
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted mb-1">Početni datum *</label>
                    <input className={inputCls} type="date" value={recurStart} onChange={e => setRecurStart(e.target.value)} required />
                  </div>
                )}
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Završni datum</label>
                  <input className={inputCls} type="date" value={recurEnd} onChange={e => setRecurEnd(e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">
                    {editRecur ? 'Novi iznos (€)' : 'Iznos (€) *'}
                  </label>
                  <input
                    className={inputCls}
                    type="number" step="0.01" min="0.01"
                    value={recurAmount}
                    onChange={e => setRecurAmount(e.target.value)}
                    required={!editRecur}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Kategorija</label>
                  <select className={inputCls} value={recurCat} onChange={e => setRecurCat(e.target.value)}>
                    <option value="">Bez kategorije</option>
                    {categories.filter(c => c.is_active).map(c => (
                      <option key={c.id} value={c.id}>{c.name}</option>
                    ))}
                  </select>
                </div>
                {editRecur && (
                  <label className="flex items-center gap-2 text-sm text-gi-text cursor-pointer">
                    <input type="checkbox" checked={recurActive} onChange={e => setRecurActive(e.target.checked)} />
                    Aktivna
                  </label>
                )}
                <div className="flex gap-2 pt-2">
                  <button type="submit" disabled={recurSaving} className={btnPrimary}>
                    {recurSaving ? 'Sprema…' : 'Spremi'}
                  </button>
                  <button type="button" onClick={() => setShowRecurForm(false)} className={btnSecondary}>
                    Odustani
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* ── Budget form modal ─────────────────────────────────────────────── */}
        {showBudgetForm && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-gi-workspace border border-gi-border rounded-2xl w-full max-w-md p-6">
              <h2 className="text-lg font-bold text-gi-text mb-4">Postavi budžet</h2>
              <form onSubmit={saveBudget} className="space-y-3">
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted mb-1">Godina *</label>
                    <input className={inputCls} type="number" value={budgetYear} onChange={e => setBudgetYear(e.target.value)} required />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted mb-1">Mjesec *</label>
                    <select className={inputCls} value={budgetMonth} onChange={e => setBudgetMonth(e.target.value)}>
                      {HR_MONTHS.slice(1).map((m, i) => (
                        <option key={i + 1} value={i + 1}>{m}</option>
                      ))}
                    </select>
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Iznos budžeta (€) *</label>
                  <input className={inputCls} type="number" step="0.01" min="0.01" value={budgetAmount} onChange={e => setBudgetAmount(e.target.value)} required />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gi-text-muted mb-1">Kategorija</label>
                  <select className={inputCls} value={budgetCat} onChange={e => setBudgetCat(e.target.value)}>
                    <option value="">Ukupni budžet</option>
                    {categories.filter(c => c.is_active).map(c => (
                      <option key={c.id} value={c.id}>{c.name}</option>
                    ))}
                  </select>
                </div>
                <div className="flex gap-2 pt-2">
                  <button type="submit" disabled={budgetSaving} className={btnPrimary}>
                    {budgetSaving ? 'Sprema…' : 'Spremi'}
                  </button>
                  <button type="button" onClick={() => setShowBudgetForm(false)} className={btnSecondary}>
                    Odustani
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

      </div>
    </AppShell>
  )
}
