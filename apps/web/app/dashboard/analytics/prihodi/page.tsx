'use client'

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import AppShell from '@/components/layout/AppShell'
import AnalyticsTabNav from '@/components/analytics/AnalyticsTabNav'
import type {
  ProjectRevenueSummary,
  ProjectRevenueRow,
  ProjectRevenueDetail,
  MonthlyRevenueTrend,
  ProjectInvoice,
  ProjectContract,
} from '@/lib/types/revenue'

const FORBIDDEN_ROLES = ['administracija', 'poslovoda', 'radnik']

const MONTHS_HR = [
  'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac',
]

function eur(v: number) {
  return new Intl.NumberFormat('hr-HR', { style: 'currency', currency: 'EUR', maximumFractionDigits: 2 }).format(v)
}

// ── Status badge ──────────────────────────────────────────────────────────────

function InvoiceStatusBadge({ status, overdue }: { status: string; overdue: boolean }) {
  if (overdue && (status === 'issued' || status === 'partial')) {
    return (
      <span className="px-2 py-0.5 rounded text-xs font-medium bg-red-500/20 text-red-300 border border-red-500/30">
        Kasni
      </span>
    )
  }
  const map: Record<string, string> = {
    issued: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
    partial: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
    paid: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
    cancelled: 'bg-gi-card-elevated text-gi-text-subtle border-gi-border',
  }
  const labels: Record<string, string> = {
    issued: 'Izdano', partial: 'Djelomično', paid: 'Plaćeno', cancelled: 'Otkazano',
  }
  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium border ${map[status] ?? map.issued}`}>
      {labels[status] ?? status}
    </span>
  )
}

// ── Revenue trend chart ───────────────────────────────────────────────────────

function RevenueTrendChart({ trend }: { trend: MonthlyRevenueTrend[] }) {
  const maxVal = Math.max(...trend.map(t => Math.max(t.invoiced_gross, t.collected)), 1)

  return (
    <div className="mt-4">
      <div className="text-xs text-gi-text-muted mb-2">Godišnji trend — Fakturirano vs Naplaćeno (EUR)</div>
      <div className="flex items-end gap-1" style={{ height: 80 }}>
        {trend.map((t) => {
          const invH = (t.invoiced_gross / maxVal) * 76
          const vatH = t.invoiced_gross > 0 ? ((t.invoiced_gross - t.invoiced_net) / t.invoiced_gross) * invH : 0
          const netH = invH - vatH
          const collH = (t.collected / maxVal) * 76
          return (
            <div key={t.month} className="flex-1 flex flex-col justify-end group relative" style={{ height: 80 }}>
              {t.has_data ? (
                <div className="relative">
                  {/* invoiced bar */}
                  <div className="flex flex-col rounded-t-sm overflow-hidden" style={{ height: invH }}>
                    <div className="bg-blue-500/30" style={{ height: vatH }} />
                    <div className="bg-blue-500/60" style={{ height: netH }} />
                  </div>
                  {/* collected overlay */}
                  <div
                    className="absolute bottom-0 left-0 right-0 bg-gi-yellow/70 rounded-t-sm"
                    style={{ height: collH }}
                  />
                </div>
              ) : (
                <div className="h-1 bg-gi-border rounded" />
              )}
              {t.has_data && (
                <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 hidden group-hover:block z-10 bg-gi-card-elevated border border-gi-border rounded px-2 py-1.5 text-xs whitespace-nowrap shadow-lg">
                  <div className="text-gi-text font-medium">{MONTHS_HR[t.month - 1]}</div>
                  <div className="text-blue-300">Fakturirano (neto): {eur(t.invoiced_net)}</div>
                  <div className="text-blue-400">Fakturirano (bruto): {eur(t.invoiced_gross)}</div>
                  <div className="text-gi-yellow font-medium">Naplaćeno: {eur(t.collected)}</div>
                </div>
              )}
            </div>
          )
        })}
      </div>
      <div className="flex gap-1 mt-0.5">
        {trend.map((t) => (
          <div key={t.month} className="flex-1 text-center text-[9px] text-gi-text-subtle">
            {MONTHS_HR[t.month - 1].slice(0, 3)}
          </div>
        ))}
      </div>
      <div className="flex flex-wrap gap-3 mt-2">
        <span className="flex items-center gap-1 text-xs text-gi-text-muted">
          <span className="inline-block w-3 h-2 bg-blue-500/60 rounded-sm" />Fakturirano (neto)
        </span>
        <span className="flex items-center gap-1 text-xs text-gi-text-muted">
          <span className="inline-block w-3 h-2 bg-blue-500/30 rounded-sm" />PDV
        </span>
        <span className="flex items-center gap-1 text-xs text-gi-text-muted">
          <span className="inline-block w-3 h-2 bg-gi-yellow/70 rounded-sm" />Naplaćeno
        </span>
      </div>
    </div>
  )
}

// ── Project detail panel ──────────────────────────────────────────────────────

function ProjectDetailPanel({
  projectId,
  year,
  onClose,
}: {
  projectId: string
  year: number
  onClose: () => void
}) {
  const [detail, setDetail] = useState<ProjectRevenueDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    apiClient.get<{ detail: ProjectRevenueDetail }>(
      `/analytics/prihodi/projects/${encodeURIComponent(projectId)}?year=${year}`
    )
      .then(r => { setDetail(r.data.detail); setLoading(false) })
      .catch(() => { setError('Greška pri dohvatu podataka'); setLoading(false) })
  }, [projectId, year])

  return (
    <div className="bg-gi-workspace border border-gi-border rounded-xl p-5 mt-2">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-gi-text font-semibold">{detail?.project_name ?? 'Detalji projekta'}</h3>
        <button onClick={onClose} className="text-gi-text-muted hover:text-gi-text transition-colors text-lg leading-none">✕</button>
      </div>

      {loading && <div className="text-gi-text-muted text-sm py-4 text-center">Učitavanje…</div>}
      {error && <div className="text-red-400 text-sm py-4 text-center">{error}</div>}

      {detail && !loading && (
        <>
          {/* KPI strip */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-5">
            {[
              { label: 'Ugovorena vrijednost', value: eur(detail.contract_value) },
              { label: 'Fakturirano (neto)', value: eur(detail.invoiced_net) },
              { label: 'Naplaćeno', value: eur(detail.collected), hi: 'green' },
              { label: 'Neplaćeno', value: eur(detail.outstanding), hi: detail.outstanding > 0 ? 'amber' : '' },
            ].map(kpi => (
              <div key={kpi.label} className="bg-gi-card border border-gi-border rounded-lg p-3">
                <div className="text-xs text-gi-text-muted mb-0.5">{kpi.label}</div>
                <div className={`text-sm font-semibold ${kpi.hi === 'green' ? 'text-emerald-400' : kpi.hi === 'amber' ? 'text-amber-400' : 'text-gi-text'}`}>
                  {kpi.value}
                </div>
              </div>
            ))}
          </div>

          {/* Contracts */}
          {detail.contracts.length > 0 && (
            <div className="mb-5">
              <div className="text-xs text-gi-text-muted mb-2 uppercase tracking-wide">Ugovori</div>
              <div className="space-y-1">
                {detail.contracts.map((c: ProjectContract) => (
                  <div key={c.id} className="flex items-center justify-between bg-gi-card border border-gi-border rounded-lg px-3 py-2 text-sm">
                    <div>
                      <span className="text-gi-text">{c.contract_type === 'initial' ? 'Osnovni ugovor' : 'Izmjena'}</span>
                      {c.reference && <span className="text-gi-text-muted ml-2 text-xs">{c.reference}</span>}
                      {c.contract_date && <span className="text-gi-text-subtle ml-2 text-xs">{c.contract_date}</span>}
                    </div>
                    <span className="text-gi-yellow font-medium">{eur(c.net_value)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Invoice list */}
          <div className="mb-5">
            <div className="text-xs text-gi-text-muted mb-2 uppercase tracking-wide">Računi {year}.</div>
            {detail.invoices.length === 0 ? (
              <div className="text-gi-text-muted text-sm py-2">Nema računa za odabranu godinu.</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-gi-text-muted text-xs border-b border-gi-border">
                      <th className="text-left pb-2 font-medium">Broj</th>
                      <th className="text-left pb-2 font-medium">Datum</th>
                      <th className="text-right pb-2 font-medium">Neto</th>
                      <th className="text-right pb-2 font-medium">Bruto</th>
                      <th className="text-right pb-2 font-medium">Naplaćeno</th>
                      <th className="text-right pb-2 font-medium">Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {detail.invoices.map((inv: ProjectInvoice) => (
                      <tr key={inv.id} className="border-b border-gi-border/50 hover:bg-gi-card/30 transition-colors">
                        <td className="py-2 text-gi-text">{inv.invoice_number}</td>
                        <td className="py-2 text-gi-text-muted">{inv.invoice_date}</td>
                        <td className="py-2 text-right text-gi-text-muted">{eur(inv.net_amount)}</td>
                        <td className="py-2 text-right text-gi-text">{eur(inv.gross_amount)}</td>
                        <td className="py-2 text-right text-emerald-400">{eur(inv.paid_amount)}</td>
                        <td className="py-2 text-right">
                          <InvoiceStatusBadge status={inv.status} overdue={inv.is_overdue} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Trend chart */}
          {detail.monthly_trend.length > 0 && (
            <RevenueTrendChart trend={detail.monthly_trend} />
          )}
        </>
      )}
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function PrihodiPage() {
  const { user, employee, logout, isLoading: authLoading } = useAuth()
  const router = useRouter()

  const currentYear = new Date().getFullYear()
  const [year, setYear] = useState(currentYear)
  const [month, setMonth] = useState(0) // 0 = all year
  const [summary, setSummary] = useState<ProjectRevenueSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [expandedProject, setExpandedProject] = useState<string | null>(null)

  const fetchSummary = useCallback(() => {
    setLoading(true)
    setError(null)
    const params = new URLSearchParams({ year: String(year) })
    if (month > 0) params.set('month', String(month))
    apiClient.get<{ summary: ProjectRevenueSummary }>(`/analytics/prihodi/summary?${params}`)
      .then(r => { setSummary(r.data.summary); setLoading(false) })
      .catch(() => { setError('Greška pri dohvatu podataka'); setLoading(false) })
  }, [year, month])

  useEffect(() => {
    if (!authLoading && (!user || FORBIDDEN_ROLES.includes(user.role))) {
      router.replace('/dashboard')
    }
  }, [authLoading, user, router])

  useEffect(() => {
    if (!authLoading && user && !FORBIDDEN_ROLES.includes(user.role)) {
      fetchSummary()
    }
  }, [authLoading, user, fetchSummary])

  if (authLoading || !user || FORBIDDEN_ROLES.includes(user.role)) {
    return null
  }

  const handleToggleProject = (pid: string) => {
    setExpandedProject(prev => prev === pid ? null : pid)
  }

  return (
    <AppShell user={user} employee={employee} title="Prihodi projekata" backHref="/dashboard/analytics/pregled" onLogout={logout}>
      <div className="max-w-6xl mx-auto px-4 py-6">
        <AnalyticsTabNav />

        {/* Year / month navigator */}
        <div className="flex flex-wrap items-center gap-3 mb-6">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setYear(y => y - 1)}
              className="px-3 py-1.5 rounded-lg bg-gi-card border border-gi-border text-gi-text hover:bg-gi-card-elevated transition-colors text-sm"
            >
              ‹
            </button>
            <span className="text-gi-text font-semibold text-lg w-14 text-center">{year}</span>
            <button
              onClick={() => setYear(y => Math.min(y + 1, currentYear))}
              disabled={year >= currentYear}
              className="px-3 py-1.5 rounded-lg bg-gi-card border border-gi-border text-gi-text hover:bg-gi-card-elevated transition-colors text-sm disabled:opacity-40"
            >
              ›
            </button>
          </div>

          {/* Month selector */}
          <select
            value={month}
            onChange={e => setMonth(Number(e.target.value))}
            className="px-3 py-1.5 rounded-lg bg-gi-card border border-gi-border text-gi-text text-sm focus:outline-none focus:border-gi-yellow"
          >
            <option value={0}>Cijela godina</option>
            {MONTHS_HR.map((m, i) => (
              <option key={i + 1} value={i + 1}>{m}</option>
            ))}
          </select>
        </div>

        {error && (
          <div className="bg-red-500/10 border border-red-500/30 text-red-400 rounded-xl px-4 py-3 mb-5 text-sm">
            {error}
          </div>
        )}

        {loading && (
          <div className="text-gi-text-muted text-sm py-8 text-center">Učitavanje…</div>
        )}

        {summary && !loading && (
          <>
            {/* KPI strip */}
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-6">
              {[
                { label: 'Ugovorena vrijednost', value: eur(summary.total_contracted) },
                { label: 'Fakturirano (bruto)', value: eur(summary.total_invoiced_gross), sub: `Neto: ${eur(summary.total_invoiced_net)}` },
                { label: 'Naplaćeno', value: eur(summary.total_collected), hi: 'green' },
                { label: 'Neplaćeno', value: eur(summary.total_outstanding), hi: summary.total_outstanding > 0 ? 'amber' : '' },
                { label: 'Kasni plaćanje', value: summary.overdue_count > 0 ? `${summary.overdue_count} račun${summary.overdue_count === 1 ? '' : 'a'}` : '—', hi: summary.overdue_count > 0 ? 'red' : '' },
              ].map(kpi => (
                <div key={kpi.label} className="bg-gi-card border border-gi-border rounded-xl p-4">
                  <div className="text-xs text-gi-text-muted mb-1">{kpi.label}</div>
                  <div className={`text-base font-semibold ${kpi.hi === 'green' ? 'text-emerald-400' : kpi.hi === 'amber' ? 'text-amber-400' : kpi.hi === 'red' ? 'text-red-400' : 'text-gi-text'}`}>
                    {kpi.value}
                  </div>
                  {kpi.sub && <div className="text-xs text-gi-text-subtle mt-0.5">{kpi.sub}</div>}
                </div>
              ))}
            </div>

            {/* Monthly trend chart */}
            {summary.monthly_trend.length > 0 && month === 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-5 mb-6">
                <div className="text-sm font-medium text-gi-text mb-1">Godišnji trend prihoda {year}.</div>
                <RevenueTrendChart trend={summary.monthly_trend} />
              </div>
            )}

            {/* Project revenue table */}
            {summary.projects.length === 0 ? (
              <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center text-gi-text-muted text-sm">
                Nema evidentiranih prihoda za odabrani period.
              </div>
            ) : (
              <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
                <div className="px-5 py-4 border-b border-gi-border">
                  <h2 className="text-sm font-medium text-gi-text">Prihodi po projektima</h2>
                </div>

                {/* Desktop table */}
                <div className="hidden sm:block overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gi-border text-gi-text-muted text-xs">
                        <th className="text-left px-5 py-3 font-medium">Projekt</th>
                        <th className="text-right px-4 py-3 font-medium">Ugovor</th>
                        <th className="text-right px-4 py-3 font-medium">Fakturirano</th>
                        <th className="text-right px-4 py-3 font-medium">Naplaćeno</th>
                        <th className="text-right px-4 py-3 font-medium">Neplaćeno</th>
                        <th className="text-right px-4 py-3 font-medium">Računi</th>
                        <th className="px-4 py-3" />
                      </tr>
                    </thead>
                    <tbody>
                      {summary.projects.map((p: ProjectRevenueRow) => (
                        <>
                          <tr
                            key={p.project_id}
                            className={`border-b border-gi-border/50 hover:bg-gi-card/60 transition-colors cursor-pointer ${expandedProject === p.project_id ? 'bg-gi-card/40' : ''}`}
                            onClick={() => handleToggleProject(p.project_id)}
                          >
                            <td className="px-5 py-3">
                              <div className="text-gi-text font-medium">{p.project_name}</div>
                              {p.has_overdue && (
                                <div className="text-xs text-red-400 mt-0.5">⚠ Kasni plaćanje</div>
                              )}
                            </td>
                            <td className="px-4 py-3 text-right text-gi-text-muted">{p.contract_value > 0 ? eur(p.contract_value) : '—'}</td>
                            <td className="px-4 py-3 text-right text-gi-text">{eur(p.invoiced_gross)}</td>
                            <td className="px-4 py-3 text-right text-emerald-400">{eur(p.collected)}</td>
                            <td className={`px-4 py-3 text-right font-medium ${p.outstanding > 0 ? (p.has_overdue ? 'text-red-400' : 'text-amber-400') : 'text-gi-text-muted'}`}>
                              {p.outstanding > 0 ? eur(p.outstanding) : '—'}
                            </td>
                            <td className="px-4 py-3 text-right text-gi-text-muted">{p.invoice_count}</td>
                            <td className="px-4 py-3 text-right text-gi-text-subtle text-xs">
                              {expandedProject === p.project_id ? '▲' : '▼'}
                            </td>
                          </tr>
                          {expandedProject === p.project_id && (
                            <tr key={`${p.project_id}-detail`}>
                              <td colSpan={7} className="px-5 pb-4">
                                <ProjectDetailPanel
                                  projectId={p.project_id}
                                  year={year}
                                  onClose={() => setExpandedProject(null)}
                                />
                              </td>
                            </tr>
                          )}
                        </>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Mobile cards */}
                <div className="sm:hidden divide-y divide-gi-border">
                  {summary.projects.map((p: ProjectRevenueRow) => (
                    <div key={p.project_id}>
                      <button
                        onClick={() => handleToggleProject(p.project_id)}
                        className="w-full text-left px-5 py-4"
                      >
                        <div className="flex items-start justify-between mb-2">
                          <div>
                            <div className="text-gi-text font-medium">{p.project_name}</div>
                            {p.has_overdue && <div className="text-xs text-red-400 mt-0.5">⚠ Kasni plaćanje</div>}
                          </div>
                          <span className="text-gi-text-subtle text-xs">{expandedProject === p.project_id ? '▲' : '▼'}</span>
                        </div>
                        <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                          <div className="text-gi-text-muted">Fakturirano</div>
                          <div className="text-gi-text text-right">{eur(p.invoiced_gross)}</div>
                          <div className="text-gi-text-muted">Naplaćeno</div>
                          <div className="text-emerald-400 text-right">{eur(p.collected)}</div>
                          <div className="text-gi-text-muted">Neplaćeno</div>
                          <div className={`text-right font-medium ${p.outstanding > 0 ? (p.has_overdue ? 'text-red-400' : 'text-amber-400') : 'text-gi-text-muted'}`}>
                            {p.outstanding > 0 ? eur(p.outstanding) : '—'}
                          </div>
                        </div>
                      </button>
                      {expandedProject === p.project_id && (
                        <div className="px-5 pb-4">
                          <ProjectDetailPanel
                            projectId={p.project_id}
                            year={year}
                            onClose={() => setExpandedProject(null)}
                          />
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Outstanding receivables summary */}
            {summary.total_outstanding > 0 && (
              <div className="mt-6 bg-gi-card border border-gi-border rounded-xl p-5">
                <div className="text-sm font-medium text-gi-text mb-3">Potraživanja</div>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-sm">
                  <div>
                    <div className="text-gi-text-muted text-xs mb-1">Ukupno neplaćeno</div>
                    <div className="text-amber-400 font-semibold">{eur(summary.total_outstanding)}</div>
                  </div>
                  {summary.overdue_count > 0 && (
                    <div>
                      <div className="text-gi-text-muted text-xs mb-1">Od toga kasni</div>
                      <div className="text-red-400 font-semibold">{eur(summary.total_overdue)}</div>
                    </div>
                  )}
                  <div>
                    <div className="text-gi-text-muted text-xs mb-1">Fakturirano (bruto)</div>
                    <div className="text-gi-text font-semibold">{eur(summary.total_invoiced_gross)}</div>
                  </div>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </AppShell>
  )
}
