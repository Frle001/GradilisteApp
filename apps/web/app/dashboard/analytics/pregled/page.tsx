'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import AnalyticsTabNav from '@/components/analytics/AnalyticsTabNav'
import type { MonthlyLaborSummary, ColdOpsSummary, AnnualPayrollSummary, ColdOpsAnnualSummary } from '@/lib/types/analytics'

const FORBIDDEN_ROLES = ['administracija', 'poslovoda', 'radnik']

const HR_MONTHS = [
  '', 'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac',
]

const CURRENT = new Date()

function formatEur(n: number): string {
  return n.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €'
}

function pct(a: number, total: number): string {
  if (total <= 0) return '—'
  return ((a / total) * 100).toFixed(1) + '%'
}

function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Greška pri učitavanju'
}

// ── KPI card ──────────────────────────────────────────────────────────────────

function KpiCard({ title, value, sub, accent }: { title: string; value: string; sub?: string; accent: string }) {
  return (
    <div className={`bg-gi-card border border-gi-border rounded-xl px-4 py-3.5 border-l-4 ${accent}`}>
      <p className="text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1">{title}</p>
      <p className="text-xl font-bold text-gi-text tabular-nums leading-none">{value}</p>
      {sub && <p className="text-xs text-gi-text-subtle mt-1">{sub}</p>}
    </div>
  )
}

// ── Trend bar (CSS-only) ──────────────────────────────────────────────────────

function TrendBar({ label, value, max, color }: { label: string; value: number; max: number; color: string }) {
  const widthPct = max > 0 ? Math.round((value / max) * 100) : 0
  return (
    <div className="flex items-center gap-2">
      <span className="w-14 text-xs text-gi-text-muted text-right shrink-0">{label}</span>
      <div className="flex-1 h-5 bg-gi-card-elevated rounded overflow-hidden">
        <div className={`h-full rounded transition-all ${color}`} style={{ width: `${widthPct}%` }} />
      </div>
      <span className="w-20 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">{formatEur(value)}</span>
    </div>
  )
}

// ── Annual combined chart ─────────────────────────────────────────────────────

const HR_MONTHS_SHORT = [
  '', 'Sij', 'Velj', 'Ožu', 'Tra', 'Svi', 'Lip',
  'Srp', 'Kol', 'Ruj', 'Lis', 'Stu', 'Pro',
]

function AnnualCombinedChart({
  payroll, coldOps,
}: { payroll: AnnualPayrollSummary; coldOps: ColdOpsAnnualSummary }) {
  const coldMap = new Map(coldOps.months.map(m => [m.month, m.total]))
  const maxVal = Math.max(
    ...payroll.months.map(m => m.total_bruto + (coldMap.get(m.month) ?? 0)),
    1,
  )
  const H = 128

  return (
    <div className="bg-gi-card border border-gi-border rounded-xl px-4 pt-4 pb-6">
      <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider mb-4">
        Operativni troškovi po mjesecima — {payroll.year}
      </p>
      <div className="flex items-end gap-1" style={{ height: `${H}px` }}>
        {payroll.months.map(m => {
          const payBruto = m.total_bruto
          const cold = coldMap.get(m.month) ?? 0
          const total = payBruto + cold
          const totalH = total > 0 ? Math.round((total / maxVal) * H) : 2
          const payH = total > 0 ? Math.round((payBruto / total) * totalH) : 0
          const coldH = totalH - payH

          return (
            <div key={m.month} className="flex flex-col items-center flex-1 min-w-0 group relative">
              <div className="absolute bottom-full mb-1.5 left-1/2 -translate-x-1/2 z-10 hidden group-hover:flex flex-col items-center pointer-events-none">
                <div className="bg-gi-card-elevated border border-gi-border rounded-lg px-2.5 py-2 text-xs whitespace-nowrap shadow-lg">
                  <p className="font-semibold text-gi-text mb-1">{HR_MONTHS[m.month]}</p>
                  <p className="text-gi-text-muted">Plaće: <span className="text-gi-yellow tabular-nums">{payBruto > 0 ? formatEur(payBruto) : '—'}</span></p>
                  <p className="text-gi-text-muted">Pogon: <span className="text-gi-text tabular-nums">{cold > 0 ? formatEur(cold) : '—'}</span></p>
                  {total > 0 && <p className="text-emerald-400 mt-0.5">Ukupno: <span className="tabular-nums">{formatEur(total)}</span></p>}
                </div>
                <div className="w-2 h-2 bg-gi-card-elevated border-r border-b border-gi-border rotate-45 -mt-1" />
              </div>
              <div className="w-full flex flex-col-reverse rounded-t overflow-hidden" style={{ height: `${totalH}px` }}>
                {payH > 0 && <div className="w-full bg-blue-500" style={{ height: `${payH}px` }} />}
                {coldH > 0 && <div className="w-full bg-gi-yellow" style={{ height: `${coldH}px` }} />}
              </div>
              <p className="text-[10px] mt-1.5 text-gi-text-subtle">{HR_MONTHS_SHORT[m.month]}</p>
            </div>
          )
        })}
      </div>
      <div className="flex items-center gap-4 mt-3 justify-center">
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-blue-500 inline-block" />
          <span className="text-xs text-gi-text-subtle">Plaće (bruto)</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-gi-yellow inline-block" />
          <span className="text-xs text-gi-text-subtle">Hladni pogon</span>
        </div>
      </div>
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function AnalyticsPregledPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [mode, setMode] = useState<'monthly' | 'annual'>('monthly')
  const [year, setYear] = useState(CURRENT.getFullYear())
  const [month, setMonth] = useState(CURRENT.getMonth() + 1)

  const [payrollSummary, setPayrollSummary] = useState<MonthlyLaborSummary | null>(null)
  const [coldOpsSummary, setColdOpsSummary] = useState<ColdOpsSummary | null>(null)
  const [loadingPayroll, setLoadingPayroll] = useState(false)
  const [loadingColdOps, setLoadingColdOps] = useState(false)
  const [errorPayroll, setErrorPayroll] = useState('')
  const [errorColdOps, setErrorColdOps] = useState('')

  // Annual mode state
  const [annualPayroll, setAnnualPayroll] = useState<AnnualPayrollSummary | null>(null)
  const [annualColdOps, setAnnualColdOps] = useState<ColdOpsAnnualSummary | null>(null)
  const [loadingAnnual, setLoadingAnnual] = useState(false)
  const [errorAnnual, setErrorAnnual] = useState('')

  useEffect(() => {
    if (!isLoading && user && FORBIDDEN_ROLES.includes(user.role)) {
      router.replace('/dashboard')
    }
  }, [isLoading, user, router])

  const loadPayroll = useCallback(async () => {
    setLoadingPayroll(true); setErrorPayroll(''); setPayrollSummary(null)
    try {
      const r = await apiClient.get(`/analytics/place/summary?year=${year}&month=${month}`)
      setPayrollSummary(r.data.summary)
    } catch (e) { setErrorPayroll(errMsg(e)) }
    finally { setLoadingPayroll(false) }
  }, [year, month])

  const loadColdOps = useCallback(async () => {
    setLoadingColdOps(true); setErrorColdOps('')
    try {
      const r = await apiClient.get('/analytics/hladni-pogon/summary')
      setColdOpsSummary(r.data.summary)
    } catch (e) { setErrorColdOps(errMsg(e)) }
    finally { setLoadingColdOps(false) }
  }, [])

  const loadAnnual = useCallback(async () => {
    setLoadingAnnual(true); setErrorAnnual(''); setAnnualPayroll(null); setAnnualColdOps(null)
    try {
      const [rp, rc] = await Promise.all([
        apiClient.get<{ summary: AnnualPayrollSummary }>(`/analytics/place/annual?year=${year}`),
        apiClient.get<{ summary: ColdOpsAnnualSummary }>(`/analytics/hladni-pogon/annual?year=${year}`),
      ])
      setAnnualPayroll(rp.data.summary)
      setAnnualColdOps(rc.data.summary)
    } catch (e) { setErrorAnnual(errMsg(e)) }
    finally { setLoadingAnnual(false) }
  }, [year])

  useEffect(() => {
    if (mode === 'monthly') { loadPayroll(); loadColdOps() }
  }, [mode, loadPayroll, loadColdOps])

  useEffect(() => {
    if (mode === 'annual') loadAnnual()
  }, [mode, loadAnnual])

  function prevMonth() {
    if (month === 1) { setYear(y => y - 1); setMonth(12) }
    else setMonth(m => m - 1)
  }
  function nextMonth() {
    if (month === 12) { setYear(y => y + 1); setMonth(1) }
    else setMonth(m => m + 1)
  }

  if (isLoading || !user) return <LoadingScreen />
  if (FORBIDDEN_ROLES.includes(user.role)) return null

  const payrollTotal = payrollSummary?.total_known_cost ?? 0
  const isCurrentMonth = year === CURRENT.getFullYear() && month === (CURRENT.getMonth() + 1)
  const trendEntry = coldOpsSummary?.monthly_trend.find(t => t.year === year && t.month === month)
  const coldOpsTotal = trendEntry?.total ?? (isCurrentMonth ? (coldOpsSummary?.current_month.total ?? 0) : null)
  const combined = payrollTotal + (coldOpsTotal ?? 0)
  const years = Array.from({ length: 5 }, (_, i) => CURRENT.getFullYear() - 2 + i)
  const trend = coldOpsSummary?.monthly_trend ?? []
  const trendMax = Math.max(...trend.map(t => t.total), 1)

  // Annual combined totals
  const annualPayrollTotal = annualPayroll?.total_bruto ?? 0
  const annualColdOpsTotal = annualColdOps?.total ?? 0
  const annualCombined = annualPayrollTotal + annualColdOpsTotal

  return (
    <AppShell user={user} employee={employee} title="Analitika" backHref="/dashboard" onLogout={logout}>
      <div className="space-y-5 max-w-5xl">
        <AnalyticsTabNav />

        {/* Header + controls */}
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <h1 className="text-xl font-bold text-gi-text tracking-tight">Pregled analitike</h1>
            <p className="text-sm text-gi-text-subtle mt-0.5">Kombinirani prikaz troškova rada i hladnog pogona</p>
          </div>

          <div className="flex items-center gap-2 flex-wrap">
            {/* Mode toggle */}
            <div className="flex gap-1 bg-gi-card border border-gi-border rounded-xl p-1">
              <button
                onClick={() => setMode('monthly')}
                className={[
                  'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                  mode === 'monthly' ? 'bg-gi-yellow text-gi-navy' : 'text-gi-text-muted hover:text-gi-text',
                ].join(' ')}
              >
                Mjesečni
              </button>
              <button
                onClick={() => setMode('annual')}
                className={[
                  'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                  mode === 'annual' ? 'bg-gi-yellow text-gi-navy' : 'text-gi-text-muted hover:text-gi-text',
                ].join(' ')}
              >
                Godišnji
              </button>
            </div>

            {/* Year / month selector */}
            {mode === 'monthly' ? (
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
                  className="bg-transparent text-gi-text text-sm px-1 py-1 focus:outline-none cursor-pointer"
                >
                  {HR_MONTHS.slice(1).map((name, i) => (
                    <option key={i + 1} value={i + 1}>{name}</option>
                  ))}
                </select>
                <select
                  value={year}
                  onChange={e => setYear(Number(e.target.value))}
                  className="bg-transparent text-gi-text text-sm px-1 py-1 focus:outline-none cursor-pointer"
                >
                  {years.map(y => <option key={y} value={y}>{y}</option>)}
                </select>
                <button
                  onClick={nextMonth}
                  className="w-8 h-8 flex items-center justify-center text-gi-text-muted hover:text-gi-text rounded-lg hover:bg-gi-navy-hover transition-colors text-base leading-none"
                  aria-label="Sljedeći mjesec"
                >
                  ›
                </button>
              </div>
            ) : (
              <select
                value={year}
                onChange={e => setYear(Number(e.target.value))}
                className="bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text focus:outline-none focus:border-gi-yellow"
              >
                {years.map(y => <option key={y} value={y} className="bg-gi-card">{y}</option>)}
              </select>
            )}
          </div>
        </div>

        {/* ── MONTHLY MODE ── */}
        {mode === 'monthly' && (
          <>
            {errorPayroll && <p className="text-sm text-red-400">{errorPayroll}</p>}
            {errorColdOps && <p className="text-sm text-red-400">{errorColdOps}</p>}

            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <KpiCard
                title="Troškovi rada"
                value={loadingPayroll ? '…' : formatEur(payrollTotal)}
                sub={`${HR_MONTHS[month]} ${year}`}
                accent="border-blue-500"
              />
              <KpiCard
                title="Hladni pogon"
                value={loadingColdOps ? '…' : (coldOpsTotal != null ? formatEur(coldOpsTotal) : '—')}
                sub={coldOpsTotal != null ? `${HR_MONTHS[month]} ${year}` : 'Nema podataka'}
                accent="border-gi-yellow"
              />
              <KpiCard
                title="Ukupno"
                value={(loadingPayroll || loadingColdOps) ? '…' : (coldOpsTotal != null ? formatEur(combined) : '—')}
                sub="Troškovi rada + hladni pogon"
                accent="border-emerald-500"
              />
              <KpiCard
                title="Udio hladnog pogona"
                value={(coldOpsTotal != null && combined > 0) ? pct(coldOpsTotal, combined) : '—'}
                sub={coldOpsTotal != null && combined > 0
                  ? `Rad: ${pct(payrollTotal, combined)}`
                  : 'Nema podataka'}
                accent="border-violet-500"
              />
            </div>

            {!loadingPayroll && !loadingColdOps && coldOpsTotal != null && combined > 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-widest mb-3">
                  Distribucija troškova — {HR_MONTHS[month]} {year}
                </h2>
                <div className="space-y-2.5">
                  <div className="flex items-center gap-3">
                    <span className="w-28 text-sm text-gi-text-muted shrink-0">Troškovi rada</span>
                    <div className="flex-1 h-6 bg-gi-card-elevated rounded overflow-hidden">
                      <div className="h-full bg-blue-500/60 rounded transition-all" style={{ width: `${(payrollTotal / combined) * 100}%` }} />
                    </div>
                    <span className="w-20 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">{pct(payrollTotal, combined)}</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="w-28 text-sm text-gi-text-muted shrink-0">Hladni pogon</span>
                    <div className="flex-1 h-6 bg-gi-card-elevated rounded overflow-hidden">
                      <div className="h-full bg-gi-yellow/60 rounded transition-all" style={{ width: `${(coldOpsTotal / combined) * 100}%` }} />
                    </div>
                    <span className="w-20 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">{pct(coldOpsTotal, combined)}</span>
                  </div>
                </div>
                <div className="mt-3 pt-3 border-t border-gi-border flex justify-between text-sm">
                  <span className="text-gi-text-muted">Ukupno</span>
                  <span className="font-semibold text-gi-text tabular-nums">{formatEur(combined)}</span>
                </div>
              </div>
            )}

            {trend.length > 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-widest mb-3">
                  Trend hladnog pogona — zadnjih {trend.length} mjeseci
                </h2>
                <div className="space-y-1.5">
                  {trend.map(t => (
                    <TrendBar
                      key={`${t.year}-${t.month}`}
                      label={`${HR_MONTHS[t.month]?.slice(0, 3)} ${t.year}`}
                      value={t.total}
                      max={trendMax}
                      color={t.year === year && t.month === month ? 'bg-gi-yellow' : 'bg-gi-yellow/40'}
                    />
                  ))}
                </div>
              </div>
            )}

            {!loadingPayroll && payrollSummary && (
              <>
                {/* Payroll detail stats */}
                <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                  <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-widest mb-3">
                    Detalji plaća — {HR_MONTHS[month]} {year}
                  </h2>
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                    <div>
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Zaposlenici</p>
                      <p className="text-lg font-bold text-gi-text tabular-nums">{payrollSummary.employees.length}</p>
                    </div>
                    <div>
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Ukupni sati</p>
                      <p className="text-lg font-bold text-gi-text tabular-nums">{payrollSummary.total_hours.toFixed(1)} h</p>
                    </div>
                    <div>
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Bez kompenzacije</p>
                      <p className={`text-lg font-bold tabular-nums ${payrollSummary.employees_without_compensation > 0 ? 'text-amber-400' : 'text-gi-text'}`}>
                        {payrollSummary.employees_without_compensation}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Payroll structure: neto / doprinosi / bruto */}
                <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                  <div className="flex items-center justify-between mb-3">
                    <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-widest">
                      Struktura plaće — {HR_MONTHS[month]} {year}
                    </h2>
                    {!payrollSummary.bruto_complete && payrollSummary.employees_missing_contributions > 0 && (
                      <span className="text-xs text-amber-400">
                        ⚠ {payrollSummary.employees_missing_contributions} zap. bez doprinosa
                      </span>
                    )}
                  </div>
                  <div className="grid grid-cols-3 gap-4 mb-4">
                    <div>
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Neto</p>
                      <p className="text-base font-bold text-gi-text tabular-nums">{formatEur(payrollSummary.total_neto)}</p>
                      {payrollSummary.total_bruto > 0 && (
                        <p className="text-xs text-gi-text-subtle mt-0.5">{pct(payrollSummary.total_neto, payrollSummary.total_bruto)}</p>
                      )}
                    </div>
                    <div>
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Doprinosi</p>
                      {payrollSummary.total_contributions > 0 ? (
                        <>
                          <p className="text-base font-bold text-gi-text tabular-nums">{formatEur(payrollSummary.total_contributions)}</p>
                          {payrollSummary.total_bruto > 0 && (
                            <p className="text-xs text-gi-text-subtle mt-0.5">{pct(payrollSummary.total_contributions, payrollSummary.total_bruto)}</p>
                          )}
                        </>
                      ) : (
                        <p className="text-sm text-amber-400">Nije uneseno</p>
                      )}
                    </div>
                    <div>
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Bruto</p>
                      {payrollSummary.bruto_complete ? (
                        <p className="text-base font-bold text-gi-yellow tabular-nums">{formatEur(payrollSummary.total_bruto)}</p>
                      ) : (
                        <p className="text-sm text-gi-text-subtle">—</p>
                      )}
                      <p className="text-xs text-gi-text-subtle mt-0.5">
                        {payrollSummary.bruto_complete ? 'neto + doprinosi' : 'nepotpuno'}
                      </p>
                    </div>
                  </div>
                  {/* Stacked neto/contributions bar */}
                  {payrollSummary.total_bruto > 0 && (
                    <div>
                      <div className="flex rounded-lg overflow-hidden h-5">
                        <div
                          className="bg-gi-yellow"
                          style={{ width: `${(payrollSummary.total_neto / payrollSummary.total_bruto) * 100}%` }}
                        />
                        <div
                          className="bg-blue-500"
                          style={{ width: `${(payrollSummary.total_contributions / payrollSummary.total_bruto) * 100}%` }}
                        />
                      </div>
                      <div className="flex gap-4 mt-2 justify-center">
                        <div className="flex items-center gap-1.5">
                          <span className="w-2.5 h-2.5 rounded-sm bg-gi-yellow inline-block" />
                          <span className="text-xs text-gi-text-subtle">Neto</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <span className="w-2.5 h-2.5 rounded-sm bg-blue-500 inline-block" />
                          <span className="text-xs text-gi-text-subtle">Doprinosi</span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </>
            )}
          </>
        )}

        {/* ── ANNUAL MODE ── */}
        {mode === 'annual' && (
          <>
            {errorAnnual && <p className="text-sm text-red-400">{errorAnnual}</p>}

            {loadingAnnual && (
              <div className="rounded-xl border border-gi-border bg-gi-card py-20 text-center">
                <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje godišnjih podataka…</p>
              </div>
            )}

            {!loadingAnnual && annualPayroll && annualColdOps && (
              <>
                {/* Annual KPI cards */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  <KpiCard
                    title="Godišnje plaće"
                    value={annualPayrollTotal > 0 ? formatEur(annualPayrollTotal) : '—'}
                    sub={annualPayroll.bruto_complete ? 'bruto (potpuno)' : 'bruto (nepotpuno)'}
                    accent="border-blue-500"
                  />
                  <KpiCard
                    title="Hladni pogon"
                    value={annualColdOpsTotal > 0 ? formatEur(annualColdOpsTotal) : '—'}
                    sub={`${annualColdOps.count} troška`}
                    accent="border-gi-yellow"
                  />
                  <KpiCard
                    title="Ukupni operativni"
                    value={annualCombined > 0 ? formatEur(annualCombined) : '—'}
                    sub={`${year}. godina`}
                    accent="border-emerald-500"
                  />
                  <KpiCard
                    title="Udio plaća"
                    value={annualCombined > 0 ? pct(annualPayrollTotal, annualCombined) : '—'}
                    sub={annualCombined > 0 ? `Pogon: ${pct(annualColdOpsTotal, annualCombined)}` : ''}
                    accent="border-violet-500"
                  />
                </div>

                {/* Combined monthly chart */}
                <AnnualCombinedChart payroll={annualPayroll} coldOps={annualColdOps} />

                {/* Annual cost breakdown */}
                {annualCombined > 0 && (
                  <div className="bg-gi-card border border-gi-border rounded-xl p-4">
                    <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-widest mb-3">
                      Godišnja struktura operativnih troškova — {year}
                    </h2>
                    <div className="space-y-2.5">
                      <div className="flex items-center gap-3">
                        <span className="w-28 text-sm text-gi-text-muted shrink-0">Plaće (bruto)</span>
                        <div className="flex-1 h-6 bg-gi-card-elevated rounded overflow-hidden">
                          <div
                            className="h-full bg-blue-500/60 rounded transition-all"
                            style={{ width: `${(annualPayrollTotal / annualCombined) * 100}%` }}
                          />
                        </div>
                        <span className="w-24 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">
                          {pct(annualPayrollTotal, annualCombined)}
                        </span>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="w-28 text-sm text-gi-text-muted shrink-0">Hladni pogon</span>
                        <div className="flex-1 h-6 bg-gi-card-elevated rounded overflow-hidden">
                          <div
                            className="h-full bg-gi-yellow/60 rounded transition-all"
                            style={{ width: `${(annualColdOpsTotal / annualCombined) * 100}%` }}
                          />
                        </div>
                        <span className="w-24 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">
                          {pct(annualColdOpsTotal, annualCombined)}
                        </span>
                      </div>
                    </div>
                    <div className="mt-3 pt-3 border-t border-gi-border flex justify-between text-sm">
                      <span className="text-gi-text-muted">Ukupno {year}</span>
                      <span className="font-semibold text-gi-text tabular-nums">{formatEur(annualCombined)}</span>
                    </div>
                    {!annualPayroll.bruto_complete && (
                      <p className="text-xs text-amber-400 mt-2">
                        ⚠ Bruto plaće su nepotpune — {annualPayroll.months_with_incomplete_data} mj. bez doprinosa
                      </p>
                    )}
                  </div>
                )}
              </>
            )}
          </>
        )}

      </div>
    </AppShell>
  )
}
