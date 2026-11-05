'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import AnalyticsTabNav from '@/components/analytics/AnalyticsTabNav'
import {
  type MonthlyLaborSummary,
  type EmployeeLaborCost,
  type CompensationPlan,
  PAY_TYPE_LABELS,
} from '@/lib/types/analytics'

const HR_MONTHS = [
  '', 'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac',
]

const ROLE_LABELS: Record<string, string> = {
  radnik: 'Radnik', poslovoda: 'Poslovođa', inzenjer: 'Inženjer',
  direktor: 'Direktor', administracija: 'Administracija',
}

const FORBIDDEN_ROLES = ['administracija', 'poslovoda', 'radnik']

const TODAY = new Date().toISOString().slice(0, 10)

function formatEur(n: number): string {
  return n.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €'
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('hr-HR')
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

// ── Summary card ──────────────────────────────────────────────────────────────

function SummaryCard({
  title, value, sub, accentClass,
}: { title: string; value: string; sub?: string; accentClass: string }) {
  return (
    <div className={`bg-gi-card border border-gi-border rounded-xl px-4 py-3.5 border-l-4 ${accentClass}`}>
      <p className="text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1">{title}</p>
      <p className="text-xl font-bold text-gi-text tabular-nums leading-none">{value}</p>
      {sub && <p className="text-xs text-gi-text-subtle mt-1">{sub}</p>}
    </div>
  )
}

// ── Compensation form ─────────────────────────────────────────────────────────

interface CompFormInitial {
  payType: string; payAmount: string; companyCost: string
  effectiveFrom: string; effectiveTo?: string
}

function CompensationForm({
  employeeId, mode, planId, initialValues, onSuccess, onCancel,
}: {
  employeeId: string; mode: 'create' | 'edit'; planId?: string
  initialValues?: CompFormInitial; onSuccess: () => void; onCancel: () => void
}) {
  const [payType, setPayType] = useState(initialValues?.payType ?? 'fixed_monthly')
  const [payAmount, setPayAmount] = useState(initialValues?.payAmount ?? '')
  const [companyCost, setCompanyCost] = useState(initialValues?.companyCost ?? '')
  const [effectiveFrom, setEffectiveFrom] = useState(initialValues?.effectiveFrom ?? '')
  const [effectiveTo, setEffectiveTo] = useState(initialValues?.effectiveTo ?? '')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const isHistorical = mode === 'edit' && effectiveFrom !== '' && effectiveFrom < TODAY

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const amount = parseFloat(payAmount)
    if (!payAmount || isNaN(amount) || amount <= 0) {
      setError('Iznos mora biti pozitivan broj'); return
    }
    if (!effectiveFrom) {
      setError('Datum početka je obavezan'); return
    }
    if (mode === 'edit' && effectiveTo && effectiveTo < effectiveFrom) {
      setError('Datum završetka mora biti nakon datuma početka'); return
    }
    const body: Record<string, unknown> = {
      pay_type: payType,
      pay_amount: amount,
      effective_from: effectiveFrom,
    }
    if (mode === 'edit') {
      body.effective_to = effectiveTo || null
    }
    if (companyCost) {
      const cc = parseFloat(companyCost)
      if (isNaN(cc) || cc <= 0) { setError('Trošak firme mora biti pozitivan'); return }
      body.company_cost_amount = cc
    }
    setSubmitting(true); setError('')
    try {
      if (mode === 'create') {
        await apiClient.post(`/analytics/place/employees/${employeeId}/compensation`, body)
      } else {
        await apiClient.patch(`/analytics/place/compensation/${planId}`, body)
      }
      onSuccess()
    } catch (err) { setError(errMsg(err)) }
    finally { setSubmitting(false) }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3 mt-3 pt-3 border-t border-gi-border">
      {isHistorical && (
        <div className="rounded-lg bg-amber-900/30 border border-amber-700/50 px-3 py-2 text-xs text-amber-300">
          Uređujete plaću s datumom u prošlosti. Promjena može utjecati na prethodno izračunatu analitiku.
        </div>
      )}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs text-gi-text-muted mb-1">Model plaće *</label>
          <select value={payType} onChange={e => setPayType(e.target.value)} className={inputCls}>
            <option value="fixed_monthly">Fiksna mjesečna plaća</option>
            <option value="hourly">Plaćanje po satu</option>
          </select>
        </div>
        <div>
          <label className="block text-xs text-gi-text-muted mb-1">
            {payType === 'hourly' ? 'Satnica (€/h) *' : 'Mjesečna plaća (€) *'}
          </label>
          <input
            type="number" step="0.01" min="0.01"
            value={payAmount} onChange={e => setPayAmount(e.target.value)}
            placeholder={payType === 'hourly' ? 'npr. 9.50' : 'npr. 1800.00'}
            className={inputCls} required
          />
        </div>
        <div>
          <label className="block text-xs text-gi-text-muted mb-1">Trošak firme (€, neobavezno)</label>
          <input
            type="number" step="0.01" min="0.01"
            value={companyCost} onChange={e => setCompanyCost(e.target.value)}
            placeholder="Ako se razlikuje od plaće"
            className={inputCls}
          />
        </div>
        <div>
          <label className="block text-xs text-gi-text-muted mb-1">Na snazi od *</label>
          <input
            type="date" value={effectiveFrom} onChange={e => setEffectiveFrom(e.target.value)}
            className={inputCls} required
          />
        </div>
        {mode === 'edit' && (
          <div>
            <label className="block text-xs text-gi-text-muted mb-1">Na snazi do (prazno = otvoreno)</label>
            <input
              type="date" value={effectiveTo} onChange={e => setEffectiveTo(e.target.value)}
              className={inputCls}
            />
          </div>
        )}
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
      <div className="flex gap-2">
        <button
          type="submit" disabled={submitting}
          className="px-4 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors disabled:opacity-50"
        >
          {submitting ? 'Sprema…' : mode === 'edit' ? 'Spremi izmjenu' : 'Spremi plaću'}
        </button>
        <button
          type="button" onClick={onCancel}
          className="px-4 py-2 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted text-sm rounded-lg transition-colors"
        >
          Odustani
        </button>
      </div>
    </form>
  )
}

// ── Employee detail (expanded panel) ─────────────────────────────────────────

function EmployeeDetail({
  emp, onClose, onPlanChanged,
}: {
  emp: EmployeeLaborCost; onClose: () => void; onPlanChanged: () => void
}) {
  const [plans, setPlans] = useState<CompensationPlan[]>([])
  const [plansLoading, setPlansLoading] = useState(false)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingPlan, setEditingPlan] = useState<CompensationPlan | null>(null)

  const loadPlans = useCallback(async () => {
    setPlansLoading(true)
    try {
      const r = await apiClient.get(`/analytics/place/employees/${emp.employee_id}/compensation`)
      setPlans(r.data.plans ?? [])
    } catch { /* silent */ }
    finally { setPlansLoading(false) }
  }, [emp.employee_id])

  useEffect(() => { loadPlans() }, [loadPlans])

  function handleSuccess() {
    setShowCreateForm(false)
    setEditingPlan(null)
    loadPlans()
    onPlanChanged()
  }

  function openCreate() {
    setShowCreateForm(true)
    setEditingPlan(null)
  }

  function openEdit(plan: CompensationPlan) {
    setEditingPlan(plan)
    setShowCreateForm(false)
  }

  const analyticalCostLabel = emp.pay_type === 'fixed_monthly' ? 'Mjesečni trošak' : 'Ukupni trošak'

  return (
    <div className="pt-3 space-y-4">
      <div className="flex justify-end -mt-1">
        <button
          onClick={onClose}
          className="text-gi-text-subtle hover:text-gi-text transition-colors p-1 rounded-lg"
          aria-label="Zatvori detalje"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {emp.warning && (
        <div className="rounded-lg bg-amber-900/30 border border-amber-700/50 px-3 py-2 text-xs text-amber-300">
          {emp.warning}
        </div>
      )}

      {/* Key metrics */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div>
          <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Model</p>
          <p className="text-sm text-gi-text font-medium">
            {emp.pay_type ? PAY_TYPE_LABELS[emp.pay_type] ?? emp.pay_type : '—'}
          </p>
        </div>
        <div>
          <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Plaća</p>
          <p className="text-sm text-gi-text font-medium">
            {emp.pay_amount != null
              ? emp.pay_type === 'hourly'
                ? `${emp.pay_amount.toLocaleString('hr-HR', { minimumFractionDigits: 2 })} €/h`
                : formatEur(emp.pay_amount)
              : '—'}
          </p>
        </div>
        <div>
          <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">{analyticalCostLabel}</p>
          <p className="text-sm font-semibold text-gi-yellow tabular-nums">
            {emp.has_compensation ? formatEur(emp.company_analytical_cost) : '—'}
          </p>
        </div>
        <div>
          <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Evid. sati</p>
          <p className="text-sm text-gi-text font-medium tabular-nums">{emp.total_hours.toFixed(1)} h</p>
        </div>
        {emp.effective_cost_per_hour != null && (
          <div>
            <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Efektivan €/sat</p>
            <p className="text-sm text-gi-text font-medium tabular-nums">
              {emp.effective_cost_per_hour.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €/h
            </p>
          </div>
        )}
      </div>

      {/* Project allocations */}
      {emp.project_allocations.length > 0 && (
        <div>
          <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider mb-2">
            Raspodjela po projektima
          </p>
          <div className="rounded-xl border border-gi-border overflow-hidden">
            <table className="w-full text-sm">
              <tbody className="divide-y divide-gi-border/50">
                {emp.project_allocations.map(p => (
                  <tr key={p.project_id} className="hover:bg-gi-navy-hover transition-colors">
                    <td className="px-3 py-2 text-gi-text truncate max-w-[180px]">{p.project_name}</td>
                    <td className="px-3 py-2 text-gi-text-muted text-right tabular-nums whitespace-nowrap">
                      {p.hours.toFixed(1)} h
                    </td>
                    <td className="px-3 py-2 text-gi-text-subtle text-right tabular-nums whitespace-nowrap">
                      {p.percentage.toFixed(1)}%
                    </td>
                    <td className="px-3 py-2 text-gi-text text-right tabular-nums font-medium whitespace-nowrap">
                      {formatEur(p.labor_cost)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Compensation plan history */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider">
            Povijest plaća
          </p>
          <button
            onClick={() => showCreateForm ? setShowCreateForm(false) : openCreate()}
            className="text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
          >
            {showCreateForm ? 'Zatvori' : '+ Postavi novu plaću'}
          </button>
        </div>

        {showCreateForm && (
          <CompensationForm
            employeeId={emp.employee_id}
            mode="create"
            onSuccess={handleSuccess}
            onCancel={() => setShowCreateForm(false)}
          />
        )}

        {plansLoading ? (
          <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje plaća…</p>
        ) : plans.length === 0 ? (
          <p className="text-xs text-gi-text-subtle">Nema evidentiranih plaća za ovog zaposlenika.</p>
        ) : (
          <div className="space-y-1.5 mt-2">
            {plans.map(plan => (
              <div key={plan.id} className="rounded-lg border border-gi-border bg-gi-card-elevated px-3 py-2.5 text-xs">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <span className="font-medium text-gi-text">
                      {PAY_TYPE_LABELS[plan.pay_type] ?? plan.pay_type}
                    </span>
                    <span className="text-gi-text-muted ml-2">
                      {plan.pay_type === 'hourly'
                        ? `${plan.pay_amount.toLocaleString('hr-HR', { minimumFractionDigits: 2 })} €/h`
                        : `${formatEur(plan.pay_amount)}/mj`}
                    </span>
                    {plan.company_cost_amount != null && (
                      <span className="text-gi-text-subtle ml-2">
                        (trošak firme: {formatEur(plan.company_cost_amount)})
                      </span>
                    )}
                    <div className="text-gi-text-subtle mt-0.5">
                      {formatDate(plan.effective_from)}
                      {plan.effective_to ? ` → ${formatDate(plan.effective_to)}` : ' → (otvoreno)'}
                    </div>
                  </div>
                  <button
                    onClick={() => editingPlan?.id === plan.id ? setEditingPlan(null) : openEdit(plan)}
                    className="text-xs text-gi-yellow hover:text-gi-yellow-hover shrink-0 mt-0.5 transition-colors"
                  >
                    {editingPlan?.id === plan.id ? 'Zatvori' : 'Uredi'}
                  </button>
                </div>
                {editingPlan?.id === plan.id && (
                  <div className="mt-1">
                    <CompensationForm
                      employeeId={emp.employee_id}
                      mode="edit"
                      planId={plan.id}
                      initialValues={{
                        payType: plan.pay_type,
                        payAmount: String(plan.pay_amount),
                        companyCost: plan.company_cost_amount != null ? String(plan.company_cost_amount) : '',
                        effectiveFrom: plan.effective_from,
                        effectiveTo: plan.effective_to ?? '',
                      }}
                      onSuccess={handleSuccess}
                      onCancel={() => setEditingPlan(null)}
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

// ── Employee row (accordion) ──────────────────────────────────────────────────

function EmployeeRow({
  emp, selected, onSelect, onPlanChanged,
}: {
  emp: EmployeeLaborCost; selected: boolean
  onSelect: () => void; onPlanChanged: () => void
}) {
  const payBadge = emp.has_compensation
    ? emp.pay_type === 'fixed_monthly'
      ? (
        <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold bg-blue-900/40 text-blue-300 border border-blue-700/50">
          Fiksna
        </span>
      ) : (
        <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold bg-emerald-900/40 text-emerald-300 border border-emerald-700/50">
          Satnica
        </span>
      )
    : (
      <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold bg-red-900/30 text-red-400 border border-red-800/50">
        Bez plaće
      </span>
    )

  return (
    <div className={`rounded-xl border transition-colors ${
      selected
        ? 'border-gi-yellow/40 bg-gi-card-elevated'
        : 'border-gi-border bg-gi-card hover:bg-gi-card-elevated'
    }`}>
      <button type="button" onClick={onSelect} className="w-full text-left px-4 py-3.5">
        <div className="flex items-center justify-between gap-3">
          {/* Left: name + role + badge */}
          <div className="flex items-center gap-3 min-w-0 flex-wrap">
            <div className="min-w-0">
              <p className="text-sm font-semibold text-gi-text">{emp.employee_name}</p>
              <p className="text-xs text-gi-text-subtle">{ROLE_LABELS[emp.employee_role] ?? emp.employee_role}</p>
            </div>
            {payBadge}
            {emp.warning && (
              <span className="text-amber-400 text-sm" title={emp.warning}>⚠</span>
            )}
          </div>

          {/* Right: stats + chevron */}
          <div className="flex items-center gap-4 shrink-0">
            <div className="hidden sm:block text-right">
              <p className="text-xs text-gi-text-subtle">Sati</p>
              <p className="text-sm text-gi-text tabular-nums">{emp.total_hours.toFixed(1)} h</p>
            </div>
            <div className="text-right">
              <p className="text-xs text-gi-text-subtle">Trošak</p>
              <p className={`text-sm font-medium tabular-nums ${emp.has_compensation ? 'text-gi-text' : 'text-gi-text-subtle'}`}>
                {emp.has_compensation ? formatEur(emp.company_analytical_cost) : '—'}
              </p>
            </div>
            <div className="hidden sm:block text-right">
              <p className="text-xs text-gi-text-subtle">€/sat</p>
              <p className="text-sm text-gi-text tabular-nums">
                {emp.effective_cost_per_hour != null
                  ? emp.effective_cost_per_hour.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
                  : '—'}
              </p>
            </div>
            <svg
              className={`w-4 h-4 text-gi-text-subtle transition-transform duration-150 ${selected ? 'rotate-180' : ''}`}
              fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </div>
        </div>
      </button>

      {selected && (
        <div className="border-t border-gi-border px-4 pb-3">
          <EmployeeDetail emp={emp} onClose={onSelect} onPlanChanged={onPlanChanged} />
        </div>
      )}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

const CURRENT = new Date()

export default function PlacePage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()
  const [year, setYear] = useState(CURRENT.getFullYear())
  const [month, setMonth] = useState(CURRENT.getMonth() + 1)
  const [summary, setSummary] = useState<MonthlyLaborSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  useEffect(() => {
    if (!isLoading && user && FORBIDDEN_ROLES.includes(user.role)) {
      router.replace('/dashboard')
    }
  }, [isLoading, user, router])

  const loadSummary = useCallback(async () => {
    setLoading(true); setError(''); setSummary(null)
    try {
      const r = await apiClient.get(`/analytics/place/summary?year=${year}&month=${month}`)
      setSummary(r.data.summary)
    } catch (e) { setError(errMsg(e)) }
    finally { setLoading(false) }
  }, [year, month])

  useEffect(() => { loadSummary() }, [loadSummary])

  function prevMonth() {
    if (month === 1) { setYear(y => y - 1); setMonth(12) }
    else setMonth(m => m - 1)
  }
  function nextMonth() {
    if (month === 12) { setYear(y => y + 1); setMonth(1) }
    else setMonth(m => m + 1)
  }

  const filtered = summary?.employees.filter(e =>
    e.employee_name.toLowerCase().includes(search.toLowerCase())
  ) ?? []

  const years = Array.from({ length: 5 }, (_, i) => CURRENT.getFullYear() - 2 + i)

  if (isLoading || !user) return <LoadingScreen />
  if (FORBIDDEN_ROLES.includes(user.role)) return null

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Analitika"
      backHref="/dashboard"
      onLogout={logout}
    >
      <div className="space-y-5 max-w-5xl">
        <AnalyticsTabNav />
        {/* Page header + period selector */}
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <h1 className="text-xl font-bold text-gi-text tracking-tight">Plaće</h1>
            <p className="text-sm text-gi-text-subtle mt-0.5">Trošak rada i kompenzacijski planovi zaposlenika</p>
          </div>

          {/* Period selector */}
          <div className="flex items-center gap-0.5 bg-gi-card border border-gi-border rounded-xl px-1.5 py-1">
            <button
              onClick={prevMonth}
              className="w-8 h-8 flex items-center justify-center text-gi-text-muted hover:text-gi-text rounded-lg hover:bg-gi-navy-hover transition-colors text-base leading-none"
              aria-label="Prethodni mjesec"
            >
              ‹
            </button>
            <select
              value={month}
              onChange={e => setMonth(Number(e.target.value))}
              className="bg-transparent text-gi-text text-sm border-none focus:outline-none cursor-pointer px-1"
            >
              {HR_MONTHS.slice(1).map((m, i) => (
                <option key={i + 1} value={i + 1} className="bg-gi-card">{m}</option>
              ))}
            </select>
            <select
              value={year}
              onChange={e => setYear(Number(e.target.value))}
              className="bg-transparent text-gi-text text-sm border-none focus:outline-none cursor-pointer px-1"
            >
              {years.map(y => <option key={y} value={y} className="bg-gi-card">{y}</option>)}
            </select>
            <button
              onClick={nextMonth}
              className="w-8 h-8 flex items-center justify-center text-gi-text-muted hover:text-gi-text rounded-lg hover:bg-gi-navy-hover transition-colors text-base leading-none"
              aria-label="Sljedeći mjesec"
            >
              ›
            </button>
          </div>
        </div>

        {/* Error */}
        {error && (
          <div className="rounded-xl bg-red-900/30 border border-red-800/60 text-red-300 text-sm px-4 py-3 flex items-center gap-2">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            {error}
          </div>
        )}

        {/* Summary cards */}
        {summary && (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
            <SummaryCard
              title="Ukupni trošak plaća"
              value={formatEur(summary.total_known_cost)}
              sub="samo poznate plaće"
              accentClass="border-l-gi-yellow"
            />
            <SummaryCard
              title="Evidentirani sati"
              value={`${summary.total_hours.toLocaleString('hr-HR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })} h`}
              accentClass="border-l-violet-500"
            />
            <SummaryCard
              title="Prosj. €/sat"
              value={summary.avg_cost_per_hour != null
                ? `${summary.avg_cost_per_hour.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €`
                : '—'}
              accentClass="border-l-emerald-500"
            />
            <SummaryCard
              title="Bez definirane plaće"
              value={String(summary.employees_without_compensation)}
              sub={summary.employees_without_compensation > 0
                ? 'zaposlenika s evid. satima'
                : 'svi imaju kompenzacijski plan'}
              accentClass={summary.employees_without_compensation > 0 ? 'border-l-amber-500' : 'border-l-gi-border'}
            />
          </div>
        )}

        {/* Search + employee list */}
        <div className="space-y-2">
          <div className="flex items-center gap-3">
            <input
              type="text"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Pretraži zaposlenike…"
              className={inputCls}
            />
            {summary && (
              <span className="text-xs text-gi-text-subtle whitespace-nowrap shrink-0">
                {filtered.length} od {summary.employees.length}
              </span>
            )}
          </div>

          {loading ? (
            <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
              <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje plaća…</p>
            </div>
          ) : filtered.length === 0 ? (
            <div className="rounded-xl border border-gi-border bg-gi-card flex flex-col items-center justify-center py-16 gap-3">
              <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
              </svg>
              <p className="text-gi-text-subtle text-sm">
                {search ? 'Nema rezultata za pretragu.' : 'Nema podataka za odabrani period.'}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {filtered.map(emp => (
                <EmployeeRow
                  key={emp.employee_id}
                  emp={emp}
                  selected={selectedId === emp.employee_id}
                  onSelect={() => setSelectedId(id => id === emp.employee_id ? null : emp.employee_id)}
                  onPlanChanged={loadSummary}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}
