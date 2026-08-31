'use client'

import { useCallback, useEffect, useState } from 'react'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import {
  MonthlyLaborSummary,
  EmployeeLaborCost,
  CompensationPlan,
  PAY_TYPE_LABELS,
} from '@/lib/types/analytics'

const HR_MONTHS = ['', 'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac']

const ROLE_LABELS: Record<string, string> = {
  radnik: 'Radnik', poslovoda: 'Poslovođa', inzenjer: 'Inženjer',
  direktor: 'Direktor', administracija: 'Administracija',
}

function formatEur(n: number): string {
  return n.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €'
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('hr-HR')
}

function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Greška'
}

const TODAY = new Date().toISOString().slice(0, 10)

const inputCls = 'w-full rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500'
const btnPrimary = 'rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 transition-colors'
const btnSecondary = 'rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-300 hover:bg-slate-700 transition-colors'

// ── Summary card ──────────────────────────────────────────────────────────────

function SummaryCard({ title, value, sub }: { title: string; value: string; sub?: string }) {
  return (
    <div className="rounded-xl border border-slate-700 bg-slate-800/50 p-4">
      <p className="text-xs text-slate-500 mb-1">{title}</p>
      <p className="text-xl font-bold text-white">{value}</p>
      {sub && <p className="text-xs text-slate-500 mt-0.5">{sub}</p>}
    </div>
  )
}

// ── Compensation form (create + edit) ─────────────────────────────────────────

interface CompFormInitial {
  payType: string
  payAmount: string
  companyCost: string
  effectiveFrom: string
  effectiveTo?: string
}

function CompensationForm({
  employeeId,
  mode,
  planId,
  initialValues,
  onSuccess,
  onCancel,
}: {
  employeeId: string
  mode: 'create' | 'edit'
  planId?: string
  initialValues?: CompFormInitial
  onSuccess: () => void
  onCancel: () => void
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
    const body: Record<string, unknown> = { pay_type: payType, pay_amount: amount, effective_from: effectiveFrom }
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
    <form onSubmit={handleSubmit} className="space-y-3 mt-3 border-t border-slate-700 pt-3">
      {isHistorical && (
        <div className="rounded-lg bg-amber-900/30 border border-amber-700/50 px-3 py-2 text-xs text-amber-300">
          ⚠ Uređujete plaću s datumom u prošlosti. Promjena može utjecati na prethodno izračunatu analitiku.
        </div>
      )}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs text-slate-400 mb-1">Model plaće *</label>
          <select value={payType} onChange={e => setPayType(e.target.value)} className={inputCls}>
            <option value="fixed_monthly">Fiksna mjesečna plaća</option>
            <option value="hourly">Plaćanje po satu</option>
          </select>
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">
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
          <label className="block text-xs text-slate-400 mb-1">Trošak firme (€, neobavezno)</label>
          <input
            type="number" step="0.01" min="0.01"
            value={companyCost} onChange={e => setCompanyCost(e.target.value)}
            placeholder="Ako se razlikuje od plaće"
            className={inputCls}
          />
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">Na snazi od *</label>
          <input
            type="date" value={effectiveFrom} onChange={e => setEffectiveFrom(e.target.value)}
            className={inputCls} required
          />
        </div>
        {mode === 'edit' && (
          <div>
            <label className="block text-xs text-slate-400 mb-1">Na snazi do (prazno = otvoreno)</label>
            <input
              type="date" value={effectiveTo} onChange={e => setEffectiveTo(e.target.value)}
              className={inputCls}
            />
          </div>
        )}
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
      <div className="flex gap-2">
        <button type="submit" disabled={submitting} className={btnPrimary}>
          {submitting ? 'Sprema...' : mode === 'edit' ? 'Spremi izmjenu' : 'Spremi plaću'}
        </button>
        <button type="button" onClick={onCancel} className={btnSecondary}>Odustani</button>
      </div>
    </form>
  )
}

// ── Employee detail panel ─────────────────────────────────────────────────────

function EmployeeDetail({
  emp,
  onClose,
  onPlanChanged,
}: {
  emp: EmployeeLaborCost
  onClose: () => void
  onPlanChanged: () => void
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

  function openEdit(plan: CompensationPlan) {
    setEditingPlan(plan)
    setShowCreateForm(false)
  }

  function openCreate() {
    setShowCreateForm(true)
    setEditingPlan(null)
  }

  const analyticalCostLabel = emp.pay_type === 'fixed_monthly' ? 'Mjesečni trošak' : 'Ukupni trošak'

  return (
    <div className="mt-2 rounded-xl border border-slate-700 bg-slate-900 p-4 space-y-4">
      <div className="flex items-start justify-between">
        <div>
          <p className="font-semibold text-white text-base">{emp.employee_name}</p>
          <p className="text-xs text-slate-400">{ROLE_LABELS[emp.employee_role] ?? emp.employee_role}</p>
        </div>
        <button onClick={onClose} className="text-slate-500 hover:text-white text-xs">✕ Zatvori</button>
      </div>

      {emp.warning && (
        <div className="rounded-lg bg-amber-900/30 border border-amber-700/50 px-3 py-2 text-xs text-amber-300">
          ⚠ {emp.warning}
        </div>
      )}

      {/* Key metrics */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div>
          <p className="text-xs text-slate-500">Model</p>
          <p className="text-sm text-white font-medium">
            {emp.pay_type ? PAY_TYPE_LABELS[emp.pay_type] ?? emp.pay_type : '—'}
          </p>
        </div>
        <div>
          <p className="text-xs text-slate-500">Plaća</p>
          <p className="text-sm text-white font-medium">
            {emp.pay_amount != null
              ? emp.pay_type === 'hourly'
                ? `${emp.pay_amount.toLocaleString('hr-HR', { minimumFractionDigits: 2 })} €/h`
                : `${formatEur(emp.pay_amount)}`
              : '—'}
          </p>
        </div>
        <div>
          <p className="text-xs text-slate-500">{analyticalCostLabel}</p>
          <p className="text-sm text-white font-medium">
            {emp.has_compensation ? formatEur(emp.company_analytical_cost) : '—'}
          </p>
        </div>
        <div>
          <p className="text-xs text-slate-500">Evid. sati</p>
          <p className="text-sm text-white font-medium">{emp.total_hours.toFixed(1)} h</p>
        </div>
        {emp.effective_cost_per_hour != null && (
          <div>
            <p className="text-xs text-slate-500">Efektivni trošak/sat</p>
            <p className="text-sm text-white font-medium">
              {emp.effective_cost_per_hour.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €/h
            </p>
          </div>
        )}
      </div>

      {/* Project allocations */}
      {emp.project_allocations.length > 0 && (
        <div>
          <p className="text-xs font-semibold text-slate-400 mb-2 uppercase tracking-wider">Raspodjela po projektima</p>
          <div className="space-y-1.5">
            {emp.project_allocations.map((p) => (
              <div key={p.project_id} className="flex items-center gap-3 text-sm">
                <span className="flex-1 text-slate-200 truncate">{p.project_name}</span>
                <span className="text-slate-400 w-14 text-right">{p.hours.toFixed(1)} h</span>
                <span className="text-slate-500 w-10 text-right">{p.percentage.toFixed(1)}%</span>
                <span className="text-white w-20 text-right">{formatEur(p.labor_cost)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Compensation history */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Povijest plaća</p>
          <button
            onClick={() => showCreateForm ? setShowCreateForm(false) : openCreate()}
            className="text-xs text-blue-400 hover:underline"
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
          <p className="text-xs text-slate-500">Učitavanje...</p>
        ) : plans.length === 0 ? (
          <p className="text-xs text-slate-500">Nema evidentiranih plaća</p>
        ) : (
          <div className="space-y-1.5 mt-2">
            {plans.map((plan) => (
              <div key={plan.id} className="rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-xs">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <span className="font-medium text-slate-200">
                      {PAY_TYPE_LABELS[plan.pay_type] ?? plan.pay_type}
                    </span>
                    <span className="text-slate-400 ml-2">
                      {plan.pay_type === 'hourly'
                        ? `${plan.pay_amount.toLocaleString('hr-HR', { minimumFractionDigits: 2 })} €/h`
                        : `${formatEur(plan.pay_amount)}/mj`}
                    </span>
                    {plan.company_cost_amount != null && (
                      <span className="text-slate-500 ml-2">
                        (trošak: {formatEur(plan.company_cost_amount)})
                      </span>
                    )}
                    <div className="text-slate-500 mt-0.5">
                      {formatDate(plan.effective_from)}
                      {plan.effective_to ? ` → ${formatDate(plan.effective_to)}` : ' →'}
                    </div>
                  </div>
                  <button
                    onClick={() => editingPlan?.id === plan.id ? setEditingPlan(null) : openEdit(plan)}
                    className="text-xs text-blue-400 hover:underline shrink-0 mt-0.5"
                  >
                    {editingPlan?.id === plan.id ? 'Zatvori' : 'Uredi'}
                  </button>
                </div>
                {editingPlan?.id === plan.id && (
                  <div className="mt-1">
                    <p className="text-xs font-semibold text-slate-300 mb-1">Uredi plaću</p>
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

// ── Employee row ──────────────────────────────────────────────────────────────

function EmployeeRow({
  emp,
  selected,
  onSelect,
  onPlanChanged,
}: {
  emp: EmployeeLaborCost
  selected: boolean
  onSelect: () => void
  onPlanChanged: () => void
}) {
  const modelBadge = emp.has_compensation
    ? emp.pay_type === 'fixed_monthly' ? (
      <span className="inline-block rounded-full bg-blue-900/50 px-2 py-0.5 text-xs text-blue-300">Fiksna</span>
    ) : (
      <span className="inline-block rounded-full bg-green-900/50 px-2 py-0.5 text-xs text-green-300">Po satu</span>
    )
    : (
      <span className="inline-block rounded-full bg-red-900/40 px-2 py-0.5 text-xs text-red-400">Nedefinirana</span>
    )

  return (
    <div className="rounded-xl border border-slate-700 bg-slate-800/50 hover:bg-slate-800 transition-colors">
      <button
        type="button"
        onClick={onSelect}
        className="w-full text-left px-4 py-3"
      >
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div className="flex items-center gap-3 flex-wrap">
            <div>
              <p className="text-sm font-medium text-white">{emp.employee_name}</p>
              <p className="text-xs text-slate-500">{ROLE_LABELS[emp.employee_role] ?? emp.employee_role}</p>
            </div>
            {modelBadge}
            {emp.warning && <span className="text-xs text-amber-400">⚠</span>}
          </div>
          <div className="flex items-center gap-4 text-sm">
            <div className="text-right">
              <p className="text-slate-400 text-xs">Sati</p>
              <p className="text-white">{emp.total_hours.toFixed(1)} h</p>
            </div>
            <div className="text-right">
              <p className="text-slate-400 text-xs">Trošak</p>
              <p className="text-white">
                {emp.has_compensation ? formatEur(emp.company_analytical_cost) : '—'}
              </p>
            </div>
            <div className="text-right">
              <p className="text-slate-400 text-xs">€/h</p>
              <p className="text-white">
                {emp.effective_cost_per_hour != null
                  ? `${emp.effective_cost_per_hour.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
                  : '—'}
              </p>
            </div>
            <span className="text-slate-600">{selected ? '▲' : '▼'}</span>
          </div>
        </div>
      </button>
      {selected && (
        <div className="px-4 pb-4">
          <EmployeeDetail emp={emp} onClose={onSelect} onPlanChanged={onPlanChanged} />
        </div>
      )}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

const CURRENT = new Date()

export default function PlacePage() {
  const { user } = useAuth()
  const [year, setYear] = useState(CURRENT.getFullYear())
  const [month, setMonth] = useState(CURRENT.getMonth() + 1)
  const [summary, setSummary] = useState<MonthlyLaborSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  void user

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

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold text-white">Plaće</h1>

        {/* Period selector */}
        <div className="flex items-center gap-2">
          <button onClick={prevMonth} className="text-slate-400 hover:text-white px-2 py-1 rounded transition-colors">
            ‹
          </button>
          <select
            value={month}
            onChange={e => setMonth(Number(e.target.value))}
            className="rounded-lg border border-slate-600 bg-slate-800 px-2 py-1.5 text-sm text-white focus:outline-none"
          >
            {HR_MONTHS.slice(1).map((m, i) => (
              <option key={i + 1} value={i + 1}>{m}</option>
            ))}
          </select>
          <select
            value={year}
            onChange={e => setYear(Number(e.target.value))}
            className="rounded-lg border border-slate-600 bg-slate-800 px-2 py-1.5 text-sm text-white focus:outline-none"
          >
            {years.map(y => <option key={y} value={y}>{y}</option>)}
          </select>
          <button onClick={nextMonth} className="text-slate-400 hover:text-white px-2 py-1 rounded transition-colors">
            ›
          </button>
        </div>
      </div>

      {/* Error */}
      {error && <p className="text-sm text-red-400">{error}</p>}

      {/* Summary cards */}
      {summary && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
          <SummaryCard
            title="Ukupni trošak plaća"
            value={formatEur(summary.total_known_cost)}
            sub="poznate plaće"
          />
          <SummaryCard
            title="Evidentirani sati"
            value={`${summary.total_hours.toFixed(1)} h`}
          />
          <SummaryCard
            title="Prosj. trošak/sat"
            value={summary.avg_cost_per_hour != null
              ? `${summary.avg_cost_per_hour.toLocaleString('hr-HR', { minimumFractionDigits: 2 })} €`
              : '—'}
          />
          <SummaryCard
            title="Bez definirane plaće"
            value={String(summary.employees_without_compensation)}
            sub={summary.employees_without_compensation > 0 ? 'evidentiranih sati' : ''}
          />
        </div>
      )}

      {/* Employee list */}
      <div className="space-y-2">
        <input
          type="text"
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder="Pretraži zaposlenike..."
          className="w-full rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />

        {loading ? (
          <div className="text-center py-12 text-slate-400">Učitavanje...</div>
        ) : filtered.length === 0 && !loading ? (
          <div className="text-center py-12 text-slate-400">
            {search ? 'Nema rezultata za pretragu.' : 'Nema podataka za odabrani period.'}
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
  )
}
