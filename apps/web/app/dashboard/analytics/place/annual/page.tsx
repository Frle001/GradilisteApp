'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import AnalyticsTabNav from '@/components/analytics/AnalyticsTabNav'
import {
  type AnnualPayrollSummary,
  type MonthlyLaborSummary,
} from '@/lib/types/analytics'

const HR_MONTHS = [
  '', 'Sij', 'Velj', 'Ožu', 'Tra', 'Svi', 'Lip',
  'Srp', 'Kol', 'Ruj', 'Lis', 'Stu', 'Pro',
]

const HR_MONTHS_FULL = [
  '', 'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac',
]

const FORBIDDEN_ROLES = ['administracija', 'poslovoda', 'radnik']

const CURRENT = new Date()

function formatEur(n: number): string {
  return n.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €'
}

function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Greška'
}

// ── KPI card ──────────────────────────────────────────────────────────────────

function KpiCard({
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

// ── Stacked monthly bar chart ─────────────────────────────────────────────────

function MonthlyBarChart({ summary }: { summary: AnnualPayrollSummary }) {
  const maxBruto = Math.max(...summary.months.map(m => m.total_bruto), 1)

  return (
    <div className="bg-gi-card border border-gi-border rounded-xl px-4 pt-4 pb-6">
      <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider mb-4">
        Mjesečni razvoj plaća
      </p>
      <div className="flex items-end gap-1 h-40">
        {summary.months.map(m => {
          const bruto = m.total_bruto
          const totalH = 160 // px
          const brutoH = bruto > 0 ? Math.round((bruto / maxBruto) * totalH) : 0
          const netoFrac = bruto > 0 ? m.total_neto / bruto : 0
          const netoH = Math.round(brutoH * netoFrac)
          const contribH = brutoH - netoH
          const incomplete = m.incomplete_count > 0 || !m.has_data

          return (
            <div key={m.month} className="flex flex-col items-center flex-1 min-w-0 group relative">
              {/* Tooltip */}
              <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 z-10 hidden group-hover:flex flex-col items-center pointer-events-none">
                <div className="bg-gi-card-elevated border border-gi-border rounded-lg px-2.5 py-2 text-xs whitespace-nowrap shadow-lg">
                  <p className="font-semibold text-gi-text mb-1">{HR_MONTHS_FULL[m.month]}</p>
                  {m.has_data ? (
                    <>
                      <p className="text-gi-text-muted">Neto: <span className="text-gi-text tabular-nums">{formatEur(m.total_neto)}</span></p>
                      <p className="text-gi-text-muted">Doprinosi: <span className="text-gi-text tabular-nums">{m.incomplete_count > 0 ? 'nepotpuno' : formatEur(m.total_contributions)}</span></p>
                      {m.total_bruto > 0 && <p className="text-gi-yellow">Bruto: <span className="tabular-nums">{formatEur(m.total_bruto)}</span></p>}
                    </>
                  ) : (
                    <p className="text-gi-text-subtle">Nema podataka</p>
                  )}
                </div>
                <div className="w-2 h-2 bg-gi-card-elevated border-r border-b border-gi-border rotate-45 -mt-1" />
              </div>

              {/* Bar */}
              <div className="w-full flex flex-col-reverse rounded-t overflow-hidden" style={{ height: `${brutoH || 2}px` }}>
                {netoH > 0 && (
                  <div
                    className="w-full bg-gi-yellow"
                    style={{ height: `${netoH}px` }}
                  />
                )}
                {contribH > 0 && (
                  <div
                    className="w-full bg-blue-500"
                    style={{ height: `${contribH}px` }}
                  />
                )}
                {bruto === 0 && (
                  <div className={`w-full ${incomplete ? 'bg-amber-800/40' : 'bg-gi-border/40'}`} style={{ height: '2px' }} />
                )}
              </div>

              {/* Month label */}
              <p className={`text-[10px] mt-1.5 tabular-nums leading-none ${incomplete && m.has_data ? 'text-amber-400' : 'text-gi-text-subtle'}`}>
                {HR_MONTHS[m.month]}
                {incomplete && m.has_data && ' ⚠'}
              </p>
            </div>
          )
        })}
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 mt-3 justify-center">
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-gi-yellow inline-block" />
          <span className="text-xs text-gi-text-subtle">Neto</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-blue-500 inline-block" />
          <span className="text-xs text-gi-text-subtle">Doprinosi</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-amber-400 text-xs">⚠</span>
          <span className="text-xs text-gi-text-subtle">Nepotpuni unos</span>
        </div>
      </div>
    </div>
  )
}

// ── Annual composition bar ────────────────────────────────────────────────────

function CompositionBar({ summary }: { summary: AnnualPayrollSummary }) {
  const total = summary.total_neto + summary.total_contributions
  if (total === 0) return null
  const netoP = Math.round((summary.total_neto / total) * 100)
  const contribP = 100 - netoP

  return (
    <div className="bg-gi-card border border-gi-border rounded-xl px-4 py-4">
      <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider mb-3">
        Godišnja struktura plaće
      </p>
      <div className="flex rounded-lg overflow-hidden h-6">
        <div
          className="bg-gi-yellow flex items-center justify-center text-gi-navy text-xs font-bold"
          style={{ width: `${netoP}%` }}
        >
          {netoP >= 15 && `${netoP}%`}
        </div>
        <div
          className="bg-blue-500 flex items-center justify-center text-white text-xs font-bold"
          style={{ width: `${contribP}%` }}
        >
          {contribP >= 10 && `${contribP}%`}
        </div>
      </div>
      <div className="flex justify-between mt-2 text-xs text-gi-text-subtle">
        <span>Neto {formatEur(summary.total_neto)}</span>
        <span>Doprinosi {summary.bruto_complete ? formatEur(summary.total_contributions) : 'nepotpuno'}</span>
      </div>
    </div>
  )
}

// ── Monthly breakdown table ───────────────────────────────────────────────────

function MonthlyTable({ summary }: { summary: AnnualPayrollSummary }) {
  return (
    <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
      <div className="px-4 py-3 border-b border-gi-border">
        <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider">
          Pregled po mjesecima
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gi-border/70 bg-gi-card-elevated/50">
              <th className="px-4 py-2.5 text-left text-xs text-gi-text-muted font-medium">Mj.</th>
              <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium">Neto</th>
              <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium">Doprinosi</th>
              <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium">Bruto</th>
              <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium hidden sm:table-cell">Sati</th>
              {!summary.employee_filter && (
                <th className="px-4 py-2.5 text-right text-xs text-gi-text-muted font-medium hidden md:table-cell">Zaposleni</th>
              )}
            </tr>
          </thead>
          <tbody className="divide-y divide-gi-border/50">
            {summary.months.map(m => {
              const incomplete = m.incomplete_count > 0
              const noData = !m.has_data
              return (
                <tr
                  key={m.month}
                  className={`transition-colors ${noData ? 'opacity-40' : 'hover:bg-gi-card-elevated'}`}
                >
                  <td className="px-4 py-2.5 text-gi-text-muted whitespace-nowrap">
                    {HR_MONTHS_FULL[m.month]}
                  </td>
                  <td className="px-4 py-2.5 text-right tabular-nums text-gi-text">
                    {m.has_data ? formatEur(m.total_neto) : '—'}
                  </td>
                  <td className="px-4 py-2.5 text-right tabular-nums">
                    {noData ? (
                      <span className="text-gi-text-subtle">—</span>
                    ) : incomplete ? (
                      <span className="text-amber-400 text-xs">
                        Nije uneseno ({m.incomplete_count})
                      </span>
                    ) : (
                      <span className="text-gi-text">{formatEur(m.total_contributions)}</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-right tabular-nums text-gi-yellow font-medium">
                    {m.total_bruto > 0 ? formatEur(m.total_bruto) : '—'}
                  </td>
                  <td className="px-4 py-2.5 text-right tabular-nums text-gi-text-muted hidden sm:table-cell">
                    {m.has_data ? `${m.total_hours.toFixed(1)} h` : '—'}
                  </td>
                  {!summary.employee_filter && (
                    <td className="px-4 py-2.5 text-right tabular-nums text-gi-text-muted hidden md:table-cell">
                      {m.has_data ? m.employee_count : '—'}
                    </td>
                  )}
                </tr>
              )
            })}
          </tbody>
          <tfoot>
            <tr className="border-t border-gi-border bg-gi-card-elevated/70">
              <td className="px-4 py-2.5 text-xs font-semibold text-gi-text-muted">Godišnje</td>
              <td className="px-4 py-2.5 text-right tabular-nums text-sm font-semibold text-gi-text">
                {formatEur(summary.total_neto)}
              </td>
              <td className="px-4 py-2.5 text-right tabular-nums text-sm font-semibold">
                {summary.bruto_complete
                  ? <span className="text-gi-text">{formatEur(summary.total_contributions)}</span>
                  : <span className="text-amber-400 text-xs">Nepotpuno</span>}
              </td>
              <td className="px-4 py-2.5 text-right tabular-nums text-sm font-bold text-gi-yellow">
                {summary.total_bruto > 0 ? formatEur(summary.total_bruto) : '—'}
              </td>
              <td className="px-4 py-2.5 text-right tabular-nums text-sm text-gi-text-muted hidden sm:table-cell">
                {summary.total_hours.toFixed(1)} h
              </td>
              {!summary.employee_filter && (
                <td className="px-4 py-2.5 text-right tabular-nums text-sm text-gi-text-muted hidden md:table-cell">
                  {summary.total_employees}
                </td>
              )}
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

interface EmployeeOption {
  id: string
  name: string
}

export default function AnnualPayrollPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()
  const [year, setYear] = useState(CURRENT.getFullYear())
  const [employeeId, setEmployeeId] = useState('')
  const [summary, setSummary] = useState<AnnualPayrollSummary | null>(null)
  const [employees, setEmployees] = useState<EmployeeOption[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!isLoading && user && FORBIDDEN_ROLES.includes(user.role)) {
      router.replace('/dashboard')
    }
  }, [isLoading, user, router])

  // Load employee list for the filter dropdown (use current month summary)
  useEffect(() => {
    apiClient
      .get<{ summary: MonthlyLaborSummary }>(
        `/analytics/place/summary?year=${year}&month=${CURRENT.getMonth() + 1}`,
      )
      .then(r => {
        setEmployees(
          r.data.summary.employees.map(e => ({ id: e.employee_id, name: e.employee_name })),
        )
      })
      .catch(() => { /* silent — filter just stays empty */ })
  }, [year])

  const loadSummary = useCallback(async () => {
    setLoading(true); setError(''); setSummary(null)
    try {
      const params = new URLSearchParams({ year: String(year) })
      if (employeeId) params.set('employee_id', employeeId)
      const r = await apiClient.get<{ summary: AnnualPayrollSummary }>(
        `/analytics/place/annual?${params}`,
      )
      setSummary(r.data.summary)
    } catch (e) { setError(errMsg(e)) }
    finally { setLoading(false) }
  }, [year, employeeId])

  useEffect(() => { loadSummary() }, [loadSummary])

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

        {/* Page header */}
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <div className="flex items-center gap-2">
              <a
                href="/dashboard/analytics/place"
                className="text-gi-text-subtle hover:text-gi-text transition-colors text-sm"
              >
                Plaće
              </a>
              <span className="text-gi-text-subtle">/</span>
              <h1 className="text-xl font-bold text-gi-text tracking-tight">Godišnji pregled</h1>
            </div>
            <p className="text-sm text-gi-text-subtle mt-0.5">
              Godišnja analiza plaća, doprinosa i bruto troškova
            </p>
          </div>

          {/* Controls */}
          <div className="flex items-center gap-2 flex-wrap">
            {employees.length > 0 && (
              <select
                value={employeeId}
                onChange={e => setEmployeeId(e.target.value)}
                className="bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text focus:outline-none focus:border-gi-yellow"
              >
                <option value="" className="bg-gi-card">Svi zaposlenici</option>
                {employees.map(e => (
                  <option key={e.id} value={e.id} className="bg-gi-card">{e.name}</option>
                ))}
              </select>
            )}
            <select
              value={year}
              onChange={e => setYear(Number(e.target.value))}
              className="bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text focus:outline-none focus:border-gi-yellow"
            >
              {years.map(y => <option key={y} value={y} className="bg-gi-card">{y}</option>)}
            </select>
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

        {/* Loading */}
        {loading && (
          <div className="rounded-xl border border-gi-border bg-gi-card py-20 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje godišnjih podataka…</p>
          </div>
        )}

        {/* Incomplete data notice */}
        {!loading && summary && !summary.bruto_complete && summary.months_with_incomplete_data > 0 && (
          <div className="rounded-xl bg-amber-900/20 border border-amber-700/50 px-4 py-3 text-xs text-amber-300 flex items-start gap-2">
            <svg className="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126z" />
            </svg>
            <span>
              Bruto iznosi nisu potpuni — doprinosi nisu uneseni za {summary.months_with_incomplete_data}{' '}
              {summary.months_with_incomplete_data === 1 ? 'mjesec' : 'mjeseca/i'}. Prikazani bruto su parcijalni.
            </span>
          </div>
        )}

        {!loading && summary && (
          <>
            {/* KPI cards */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
              <KpiCard
                title="Godišnji neto"
                value={formatEur(summary.total_neto)}
                sub={summary.employee_filter ? summary.filtered_employee_name : `${summary.total_employees} zaposleni`}
                accentClass="border-l-gi-yellow"
              />
              <KpiCard
                title="Godišnji doprinosi"
                value={summary.total_contributions > 0 ? formatEur(summary.total_contributions) : '—'}
                sub={summary.bruto_complete ? 'potpuno uneseno' : 'nepotpuni podaci'}
                accentClass="border-l-blue-500"
              />
              <KpiCard
                title="Godišnji bruto"
                value={summary.total_bruto > 0 ? formatEur(summary.total_bruto) : '—'}
                sub={summary.bruto_complete ? 'neto + doprinosi' : 'nepotpuni podaci'}
                accentClass={summary.bruto_complete ? 'border-l-emerald-500' : 'border-l-gi-border'}
              />
              <KpiCard
                title="Nepotpuni mj."
                value={String(summary.months_with_incomplete_data)}
                sub={summary.months_with_incomplete_data > 0 ? 'nedostaju doprinosi' : 'svi doprinosi uneseni'}
                accentClass={summary.months_with_incomplete_data > 0 ? 'border-l-amber-500' : 'border-l-gi-border'}
              />
            </div>

            {/* Charts */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
              <div className="lg:col-span-2">
                <MonthlyBarChart summary={summary} />
              </div>
              <div>
                <CompositionBar summary={summary} />
              </div>
            </div>

            {/* Monthly table */}
            <MonthlyTable summary={summary} />
          </>
        )}
      </div>
    </AppShell>
  )
}
