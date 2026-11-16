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
  type EmployeePayrollHistory,
  type MonthlyProjectAllocationSummary,
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

// ── Contributions editor ──────────────────────────────────────────────────────

function ContributionsEditor({
  employeeId, year, month, currentContributions, onSaved,
}: {
  employeeId: string; year: number; month: number
  currentContributions?: number
  onSaved: () => void
}) {
  const [editing, setEditing] = useState(false)
  const [amount, setAmount] = useState(currentContributions != null ? String(currentContributions) : '')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    setAmount(currentContributions != null ? String(currentContributions) : '')
    setEditing(false)
  }, [employeeId, year, month, currentContributions])

  async function handleSave() {
    const val = parseFloat(amount)
    if (isNaN(val) || val < 0) { setError('Iznos mora biti nenegativan broj'); return }
    setSubmitting(true); setError('')
    try {
      await apiClient.put(
        `/analytics/place/employees/${employeeId}/contributions?year=${year}&month=${month}`,
        { contributions_amount: val },
      )
      setEditing(false)
      onSaved()
    } catch (e) { setError(errMsg(e)) }
    finally { setSubmitting(false) }
  }

  if (!editing) {
    return (
      <div className="flex items-center gap-3">
        <span className="tabular-nums">
          {currentContributions != null
            ? <span className="text-sm text-gi-text font-medium">{formatEur(currentContributions)}</span>
            : <span className="text-xs text-amber-400">Nije uneseno</span>}
        </span>
        <button
          onClick={() => setEditing(true)}
          className="text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
        >
          {currentContributions != null ? 'Uredi' : 'Unesi'}
        </button>
      </div>
    )
  }

  return (
    <div className="flex flex-wrap items-start gap-2 mt-1">
      <input
        type="number" step="0.01" min="0"
        value={amount}
        onChange={e => setAmount(e.target.value)}
        placeholder="0.00"
        className={[
          'bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-1.5',
          'text-sm text-gi-text focus:outline-none focus:border-gi-yellow transition-colors w-36',
        ].join(' ')}
        autoFocus
      />
      <div className="flex gap-2">
        <button
          onClick={handleSave} disabled={submitting}
          className="px-3 py-1.5 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-xs font-semibold rounded-lg transition-colors disabled:opacity-50"
        >
          {submitting ? 'Sprema…' : 'Spremi'}
        </button>
        <button
          onClick={() => setEditing(false)}
          className="px-3 py-1.5 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted text-xs rounded-lg transition-colors"
        >
          Odustani
        </button>
      </div>
      {error && <p className="text-xs text-red-400 w-full">{error}</p>}
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

const CURRENT = new Date()

function EmployeeDetail({
  emp, year, month, onClose, onPlanChanged,
}: {
  emp: EmployeeLaborCost; year: number; month: number
  onClose: () => void; onPlanChanged: () => void
}) {
  const [plans, setPlans] = useState<CompensationPlan[]>([])
  const [plansLoading, setPlansLoading] = useState(false)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingPlan, setEditingPlan] = useState<CompensationPlan | null>(null)
  const [showHistory, setShowHistory] = useState(false)
  const [historyYear, setHistoryYear] = useState(year)
  const [history, setHistory] = useState<EmployeePayrollHistory | null>(null)
  const [historyLoading, setHistoryLoading] = useState(false)
  const [editingHistoryMonth, setEditingHistoryMonth] = useState<number | null>(null)

  const loadPlans = useCallback(async () => {
    setPlansLoading(true)
    try {
      const r = await apiClient.get(`/analytics/place/employees/${emp.employee_id}/compensation`)
      setPlans(r.data.plans ?? [])
    } catch { /* silent */ }
    finally { setPlansLoading(false) }
  }, [emp.employee_id])

  useEffect(() => { loadPlans() }, [loadPlans])

  const loadHistory = useCallback(async () => {
    if (!showHistory) return
    setHistoryLoading(true)
    setHistory(null)
    try {
      const r = await apiClient.get(`/analytics/place/employees/${emp.employee_id}/history?year=${historyYear}`)
      setHistory(r.data.history)
    } catch { /* silent */ }
    finally { setHistoryLoading(false) }
  }, [emp.employee_id, showHistory, historyYear])

  useEffect(() => { loadHistory() }, [loadHistory])

  function handleSuccess() {
    setShowCreateForm(false); setEditingPlan(null)
    loadPlans(); onPlanChanged()
  }

  const historyYears = Array.from({ length: 5 }, (_, i) => CURRENT.getFullYear() - 2 + i)
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
      </div>

      {/* Contributions section */}
      <div className="rounded-xl border border-gi-border bg-gi-card-elevated px-4 py-3 space-y-2.5">
        <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider">
          Doprinosi — {HR_MONTHS[month]} {year}
        </p>
        <div className="grid grid-cols-3 gap-4">
          <div>
            <p className="text-xs text-gi-text-subtle mb-0.5">Neto</p>
            <p className="text-sm font-medium text-gi-text tabular-nums">
              {emp.has_compensation ? formatEur(emp.neto) : '—'}
            </p>
          </div>
          <div>
            <p className="text-xs text-gi-text-subtle mb-0.5">Doprinosi</p>
            <ContributionsEditor
              employeeId={emp.employee_id}
              year={year}
              month={month}
              currentContributions={emp.contributions}
              onSaved={onPlanChanged}
            />
          </div>
          <div>
            <p className="text-xs text-gi-text-subtle mb-0.5">Bruto</p>
            <p className="text-sm font-semibold tabular-nums text-gi-yellow">
              {emp.bruto != null ? formatEur(emp.bruto) : '—'}
            </p>
          </div>
        </div>
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
            onClick={() => showCreateForm ? setShowCreateForm(false) : setShowCreateForm(true)}
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
                    onClick={() => editingPlan?.id === plan.id ? setEditingPlan(null) : setEditingPlan(plan)}
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

      {/* Payroll history (year breakdown) */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider">
            Godišnji pregled doprinosa
          </p>
          <button
            onClick={() => setShowHistory(h => !h)}
            className="text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
          >
            {showHistory ? 'Zatvori' : 'Prikaži'}
          </button>
        </div>

        {showHistory && (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <select
                value={historyYear}
                onChange={e => { setHistoryYear(Number(e.target.value)); setEditingHistoryMonth(null) }}
                className="bg-gi-card-elevated border border-gi-border rounded-lg px-2.5 py-1.5 text-xs text-gi-text focus:outline-none focus:border-gi-yellow"
              >
                {historyYears.map(y => <option key={y} value={y} className="bg-gi-card">{y}</option>)}
              </select>
              {history && (
                <span className="text-xs text-gi-text-subtle">
                  Bruto ukupno: {history.total_bruto > 0 ? formatEur(history.total_bruto) : '—'}
                </span>
              )}
            </div>

            {historyLoading ? (
              <p className="text-xs text-gi-text-subtle animate-pulse">Učitavanje…</p>
            ) : history && (
              <div className="rounded-xl border border-gi-border overflow-hidden">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b border-gi-border">
                      <th className="px-3 py-2 text-left text-gi-text-muted font-medium">Mj.</th>
                      <th className="px-3 py-2 text-right text-gi-text-muted font-medium">Neto</th>
                      <th className="px-3 py-2 text-right text-gi-text-muted font-medium">Doprinosi</th>
                      <th className="px-3 py-2 text-right text-gi-text-muted font-medium">Bruto</th>
                      <th className="px-3 py-2 text-right text-gi-text-muted font-medium">Sati</th>
                      <th className="px-2 py-2"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gi-border/50">
                    {history.months.map(m => (
                      <tr
                        key={m.month}
                        className={`transition-colors ${
                          historyYear === year && m.month === month
                            ? 'bg-gi-yellow/5'
                            : 'hover:bg-gi-card-elevated'
                        }`}
                      >
                        <td className="px-3 py-2 text-gi-text-muted whitespace-nowrap">
                          {HR_MONTHS[m.month]}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums text-gi-text">
                          {m.has_compensation ? formatEur(m.neto) : '—'}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {editingHistoryMonth === m.month ? (
                            <span /> /* editor rendered below in expanded row */
                          ) : m.contributions != null ? (
                            <span className="text-gi-text">{formatEur(m.contributions)}</span>
                          ) : m.has_compensation ? (
                            <span className="text-amber-400">Nije uneseno</span>
                          ) : (
                            <span className="text-gi-text-subtle">—</span>
                          )}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums text-gi-yellow font-medium">
                          {m.bruto != null ? formatEur(m.bruto) : '—'}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums text-gi-text-muted">
                          {m.total_hours.toFixed(1)} h
                        </td>
                        <td className="px-2 py-2">
                          {m.has_compensation && (
                            <button
                              onClick={() => setEditingHistoryMonth(mm => mm === m.month ? null : m.month)}
                              className="text-gi-yellow hover:text-gi-yellow-hover transition-colors whitespace-nowrap"
                            >
                              {editingHistoryMonth === m.month ? 'Zatvori' : m.contributions != null ? 'Uredi' : 'Unesi'}
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {/* Inline editor for selected history month */}
                {editingHistoryMonth != null && (
                  <div className="border-t border-gi-border px-4 py-3 bg-gi-card-elevated">
                    <p className="text-xs text-gi-text-muted mb-2">
                      Doprinosi za {HR_MONTHS[editingHistoryMonth]} {historyYear}
                    </p>
                    <ContributionsEditor
                      employeeId={emp.employee_id}
                      year={historyYear}
                      month={editingHistoryMonth}
                      currentContributions={history.months.find(m => m.month === editingHistoryMonth)?.contributions}
                      onSaved={() => { setEditingHistoryMonth(null); loadHistory(); onPlanChanged() }}
                    />
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// ── Employee row (accordion) ──────────────────────────────────────────────────

function EmployeeRow({
  emp, year, month, selected, onSelect, onPlanChanged,
}: {
  emp: EmployeeLaborCost; year: number; month: number; selected: boolean
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
              <p className="text-xs text-gi-text-subtle">Neto</p>
              <p className={`text-sm font-medium tabular-nums ${emp.has_compensation ? 'text-gi-text' : 'text-gi-text-subtle'}`}>
                {emp.has_compensation ? formatEur(emp.neto) : '—'}
              </p>
            </div>
            <div className="hidden md:block text-right">
              <p className="text-xs text-gi-text-subtle">Doprinosi</p>
              <p className={`text-sm tabular-nums ${
                emp.contributions_missing && emp.has_compensation
                  ? 'text-amber-400 text-xs'
                  : 'text-gi-text font-medium'
              }`}>
                {emp.contributions != null
                  ? formatEur(emp.contributions)
                  : emp.has_compensation ? 'Nije uneseno' : '—'}
              </p>
            </div>
            <div className="hidden md:block text-right">
              <p className="text-xs text-gi-text-subtle">Bruto</p>
              <p className="text-sm font-semibold text-gi-yellow tabular-nums">
                {emp.bruto != null ? formatEur(emp.bruto) : '—'}
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
          <EmployeeDetail
            emp={emp}
            year={year}
            month={month}
            onClose={onSelect}
            onPlanChanged={onPlanChanged}
          />
        </div>
      )}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

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
  const [showProjectAlloc, setShowProjectAlloc] = useState(false)
  const [projectAlloc, setProjectAlloc] = useState<MonthlyProjectAllocationSummary | null>(null)
  const [projectAllocLoading, setProjectAllocLoading] = useState(false)

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

  const loadProjectAlloc = useCallback(async () => {
    if (!showProjectAlloc) return
    setProjectAllocLoading(true); setProjectAlloc(null)
    try {
      const r = await apiClient.get<{ allocation: MonthlyProjectAllocationSummary }>(
        `/analytics/place/project-allocation?year=${year}&month=${month}`,
      )
      setProjectAlloc(r.data.allocation)
    } catch { /* silent */ }
    finally { setProjectAllocLoading(false) }
  }, [showProjectAlloc, year, month])

  useEffect(() => { loadSummary() }, [loadSummary])
  useEffect(() => { loadProjectAlloc() }, [loadProjectAlloc])

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
            <p className="text-sm text-gi-text-subtle mt-0.5">
              Trošak rada, doprinosi i kompenzacijski planovi
            </p>
            <a
              href="/dashboard/analytics/place/annual"
              className="inline-flex items-center gap-1 mt-1.5 text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
            >
              Godišnji pregled
              <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
              </svg>
            </a>
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
              title="Ukupno neto"
              value={formatEur(summary.total_neto)}
              sub="zbroj neto troškova"
              accentClass="border-l-gi-yellow"
            />
            <SummaryCard
              title="Ukupni doprinosi"
              value={formatEur(summary.total_contributions)}
              sub={summary.bruto_complete ? 'potpuno uneseno' : 'uneseni doprinosi'}
              accentClass="border-l-blue-500"
            />
            <SummaryCard
              title="Ukupno bruto"
              value={summary.total_bruto > 0 ? formatEur(summary.total_bruto) : '—'}
              sub={summary.bruto_complete ? 'neto + doprinosi' : 'nepotpuni podaci'}
              accentClass={summary.bruto_complete ? 'border-l-emerald-500' : 'border-l-gi-border'}
            />
            <SummaryCard
              title="Nedostaju doprinosi"
              value={String(summary.employees_missing_contributions)}
              sub={summary.employees_missing_contributions > 0
                ? 'zaposlenika bez unosa'
                : 'svi doprinosi uneseni'}
              accentClass={summary.employees_missing_contributions > 0 ? 'border-l-amber-500' : 'border-l-gi-border'}
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
                  year={year}
                  month={month}
                  selected={selectedId === emp.employee_id}
                  onSelect={() => setSelectedId(id => id === emp.employee_id ? null : emp.employee_id)}
                  onPlanChanged={loadSummary}
                />
              ))}
            </div>
          )}
        </div>

        {/* Project labor allocation */}
        <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
          <button
            type="button"
            onClick={() => setShowProjectAlloc(v => !v)}
            className="w-full flex items-center justify-between px-4 py-3 hover:bg-gi-card-elevated transition-colors"
          >
            <div>
              <p className="text-sm font-semibold text-gi-text text-left">Troškovi rada po projektima</p>
              <p className="text-xs text-gi-text-subtle mt-0.5 text-left">
                Raspodjela neto troškova rada prema evidentiranim satima
              </p>
            </div>
            <svg
              className={`w-4 h-4 text-gi-text-subtle transition-transform duration-150 shrink-0 ml-3 ${showProjectAlloc ? 'rotate-180' : ''}`}
              fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {showProjectAlloc && (
            <div className="border-t border-gi-border">
              {projectAllocLoading ? (
                <div className="px-4 py-8 text-center">
                  <p className="text-sm text-gi-text-subtle animate-pulse">Učitavanje podataka o projektima…</p>
                </div>
              ) : !projectAlloc || projectAlloc.allocations.length === 0 ? (
                <div className="px-4 py-8 text-center">
                  <p className="text-sm text-gi-text-subtle">Nema podataka o projektima za odabrani period.</p>
                </div>
              ) : (
                <div>
                  {projectAlloc.unassigned_hours > 0 && (
                    <div className="mx-4 mt-3 rounded-lg bg-amber-900/20 border border-amber-700/40 px-3 py-2 text-xs text-amber-300">
                      {projectAlloc.unassigned_hours.toFixed(1)} h evidentiranih sati nije pridruženo nijednom projektu
                    </div>
                  )}
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-gi-border/70 bg-gi-card-elevated/50">
                          <th className="px-4 py-2.5 text-left text-xs text-gi-text-muted font-medium">Projekt</th>
                          <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium">Sati</th>
                          <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium hidden sm:table-cell">%</th>
                          <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium">Trošak (neto)</th>
                          <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium hidden md:table-cell">Zaposleni</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gi-border/50">
                        {projectAlloc.allocations.map(p => (
                          <tr key={p.project_id} className="hover:bg-gi-card-elevated transition-colors">
                            <td className="px-4 py-2.5 text-gi-text font-medium truncate max-w-[200px]">{p.project_name}</td>
                            <td className="px-4 py-2.5 text-right tabular-nums text-gi-text-muted">{p.total_hours.toFixed(1)} h</td>
                            <td className="px-4 py-2.5 text-right tabular-nums text-gi-text-subtle hidden sm:table-cell">{p.pct_of_total.toFixed(1)}%</td>
                            <td className="px-4 py-2.5 text-right tabular-nums text-gi-yellow font-medium">{formatEur(p.labor_cost)}</td>
                            <td className="px-4 py-2.5 text-right tabular-nums text-gi-text-muted hidden md:table-cell">{p.employee_count}</td>
                          </tr>
                        ))}
                      </tbody>
                      <tfoot>
                        <tr className="border-t border-gi-border bg-gi-card-elevated/70">
                          <td className="px-4 py-2.5 text-xs font-semibold text-gi-text-muted">Ukupno</td>
                          <td className="px-4 py-2.5 text-right tabular-nums text-sm font-semibold text-gi-text">{projectAlloc.total_hours.toFixed(1)} h</td>
                          <td className="hidden sm:table-cell" />
                          <td className="px-4 py-2.5 text-right tabular-nums text-sm font-bold text-gi-yellow">{formatEur(projectAlloc.total_labor_cost)}</td>
                          <td className="hidden md:table-cell" />
                        </tr>
                      </tfoot>
                    </table>
                  </div>
                  <p className="px-4 py-2 text-xs text-gi-text-subtle border-t border-gi-border/50">
                    Iznosi se temelje na neto troškovima rada. Zaposleni s nepotpunim podacima nisu uključeni u trošak.
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}
