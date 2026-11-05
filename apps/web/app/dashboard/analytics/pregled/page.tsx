'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import AnalyticsTabNav from '@/components/analytics/AnalyticsTabNav'
import type { MonthlyLaborSummary, ColdOpsSummary } from '@/lib/types/analytics'

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

// ── Page ──────────────────────────────────────────────────────────────────────

export default function AnalyticsPregledPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [year, setYear] = useState(CURRENT.getFullYear())
  const [month, setMonth] = useState(CURRENT.getMonth() + 1)

  const [payrollSummary, setPayrollSummary] = useState<MonthlyLaborSummary | null>(null)
  const [coldOpsSummary, setColdOpsSummary] = useState<ColdOpsSummary | null>(null)
  const [loadingPayroll, setLoadingPayroll] = useState(false)
  const [loadingColdOps, setLoadingColdOps] = useState(false)
  const [errorPayroll, setErrorPayroll] = useState('')
  const [errorColdOps, setErrorColdOps] = useState('')

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

  useEffect(() => { loadPayroll() }, [loadPayroll])
  useEffect(() => { loadColdOps() }, [loadColdOps])

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

  // Find cold ops data for the selected month from the trend, fall back to current_month
  const isCurrentMonth = year === CURRENT.getFullYear() && month === (CURRENT.getMonth() + 1)
  const trendEntry = coldOpsSummary?.monthly_trend.find(t => t.year === year && t.month === month)
  const coldOpsTotal = trendEntry?.total ?? (isCurrentMonth ? (coldOpsSummary?.current_month.total ?? 0) : null)

  const combined = payrollTotal + (coldOpsTotal ?? 0)
  const years = Array.from({ length: 5 }, (_, i) => CURRENT.getFullYear() - 2 + i)

  const trend = coldOpsSummary?.monthly_trend ?? []
  const trendMax = Math.max(...trend.map(t => t.total), 1)

  return (
    <AppShell user={user} employee={employee} title="Analitika" backHref="/dashboard" onLogout={logout}>
      <div className="space-y-5 max-w-5xl">
        <AnalyticsTabNav />

        {/* Header + period selector */}
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <h1 className="text-xl font-bold text-gi-text tracking-tight">Pregled analitike</h1>
            <p className="text-sm text-gi-text-subtle mt-0.5">Kombinirani prikaz troškova rada i hladnog pogona</p>
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
        </div>

        {/* Errors */}
        {errorPayroll && (
          <p className="text-sm text-red-400">{errorPayroll}</p>
        )}
        {errorColdOps && (
          <p className="text-sm text-red-400">{errorColdOps}</p>
        )}

        {/* KPI cards */}
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

        {/* Distribution: payroll vs cold ops */}
        {!loadingPayroll && !loadingColdOps && coldOpsTotal != null && combined > 0 && (
          <div className="bg-gi-card border border-gi-border rounded-xl p-4">
            <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-widest mb-3">
              Distribucija troškova — {HR_MONTHS[month]} {year}
            </h2>
            <div className="space-y-2.5">
              <div className="flex items-center gap-3">
                <span className="w-28 text-sm text-gi-text-muted shrink-0">Troškovi rada</span>
                <div className="flex-1 h-6 bg-gi-card-elevated rounded overflow-hidden">
                  <div
                    className="h-full bg-blue-500/60 rounded transition-all"
                    style={{ width: `${(payrollTotal / combined) * 100}%` }}
                  />
                </div>
                <span className="w-20 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">
                  {pct(payrollTotal, combined)}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <span className="w-28 text-sm text-gi-text-muted shrink-0">Hladni pogon</span>
                <div className="flex-1 h-6 bg-gi-card-elevated rounded overflow-hidden">
                  <div
                    className="h-full bg-gi-yellow/60 rounded transition-all"
                    style={{ width: `${(coldOpsTotal / combined) * 100}%` }}
                  />
                </div>
                <span className="w-20 text-xs text-gi-text-subtle text-right tabular-nums shrink-0">
                  {pct(coldOpsTotal, combined)}
                </span>
              </div>
            </div>
            <div className="mt-3 pt-3 border-t border-gi-border flex justify-between text-sm">
              <span className="text-gi-text-muted">Ukupno</span>
              <span className="font-semibold text-gi-text tabular-nums">{formatEur(combined)}</span>
            </div>
          </div>
        )}

        {/* Cold ops monthly trend chart */}
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
                  color={
                    t.year === year && t.month === month
                      ? 'bg-gi-yellow'
                      : 'bg-gi-yellow/40'
                  }
                />
              ))}
            </div>
          </div>
        )}

        {/* Additional payroll detail */}
        {!loadingPayroll && payrollSummary && (
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
        )}

      </div>
    </AppShell>
  )
}
