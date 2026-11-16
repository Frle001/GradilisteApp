'use client'

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import AppShell from '@/components/layout/AppShell'
import AnalyticsTabNav from '@/components/analytics/AnalyticsTabNav'
import {
  type MonthlyProjectAllocationSummary,
  type MonthlyProjectAllocation,
  type ProjectDetailSummary,
  type ProjectMonthlyTrend,
  type ProjectMaterialCostItem,
} from '@/lib/types/analytics'

const FORBIDDEN_ROLES = ['administracija', 'poslovoda', 'radnik']

// ── Helpers ───────────────────────────────────────────────────────────────────

const MONTHS_HR = [
  'Siječanj', 'Veljača', 'Ožujak', 'Travanj', 'Svibanj', 'Lipanj',
  'Srpanj', 'Kolovoz', 'Rujan', 'Listopad', 'Studeni', 'Prosinac',
]

function eur(v: number) {
  return new Intl.NumberFormat('hr-HR', { style: 'currency', currency: 'EUR', maximumFractionDigits: 2 }).format(v)
}

function h(v: number) {
  return `${v.toLocaleString('hr-HR', { maximumFractionDigits: 1 })} h`
}

function qty(v: number, unit: string) {
  return `${v.toLocaleString('hr-HR', { maximumFractionDigits: 2 })} ${unit}`
}

// ── Status badge ──────────────────────────────────────────────────────────────

function StatusBadge({ incomplete }: { incomplete: boolean }) {
  return incomplete ? (
    <span className="px-2 py-0.5 rounded text-xs font-medium bg-amber-500/20 text-amber-300 border border-amber-500/30">
      Nepotpuno
    </span>
  ) : (
    <span className="px-2 py-0.5 rounded text-xs font-medium bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
      Potpuno
    </span>
  )
}

// ── Monthly trend bar chart (12-month, 3-segment stacked) ─────────────────────

function TrendChart({ trend }: { trend: ProjectMonthlyTrend[] }) {
  const maxTotal = Math.max(...trend.map(t => t.total_cost), 1)

  return (
    <div className="mt-4">
      <div className="text-xs text-gi-text-muted mb-2">Godišnji trend direktnih troškova (EUR)</div>
      <div className="flex items-end gap-1" style={{ height: 80 }}>
        {trend.map((t) => {
          const totalH = (t.total_cost / maxTotal) * 76
          const matH = t.total_cost > 0 ? (t.material_cost / t.total_cost) * totalH : 0
          const brutoH = totalH - matH
          const netoH = t.bruto > 0 ? (t.neto / t.bruto) * brutoH : 0
          const contribH = brutoH - netoH
          return (
            <div
              key={t.month}
              className="flex-1 flex flex-col justify-end group relative"
              style={{ height: 80 }}
            >
              {t.has_data ? (
                <div className="flex flex-col rounded-t-sm overflow-hidden" style={{ height: totalH }}>
                  <div className="bg-emerald-500/70" style={{ height: matH }} />
                  <div className="bg-blue-500/60" style={{ height: contribH }} />
                  <div className="bg-gi-yellow/80" style={{ height: netoH }} />
                </div>
              ) : (
                <div className="h-1 bg-gi-border rounded" />
              )}
              {/* Tooltip */}
              {t.has_data && (
                <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 hidden group-hover:block z-10 bg-gi-card-elevated border border-gi-border rounded px-2 py-1.5 text-xs whitespace-nowrap shadow-lg">
                  <div className="text-gi-text font-medium">{MONTHS_HR[t.month - 1]}</div>
                  <div className="text-gi-text-muted">Neto: {eur(t.neto)}</div>
                  <div className="text-gi-text-muted">Doprinosi: {eur(t.contributions)}</div>
                  <div className="text-gi-text-muted">Bruto rad: {eur(t.bruto)}</div>
                  {t.material_cost > 0 && <div className="text-emerald-400">Materijal: {eur(t.material_cost)}</div>}
                  <div className="text-gi-yellow font-medium border-t border-gi-border mt-1 pt-1">Ukupno: {eur(t.total_cost)}</div>
                  {t.incomplete && <div className="text-amber-400">⚠ Nepotpuni doprinosi</div>}
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
          <span className="inline-block w-3 h-2 bg-gi-yellow/80 rounded-sm" />Neto
        </span>
        <span className="flex items-center gap-1 text-xs text-gi-text-muted">
          <span className="inline-block w-3 h-2 bg-blue-500/60 rounded-sm" />Doprinosi
        </span>
        <span className="flex items-center gap-1 text-xs text-gi-text-muted">
          <span className="inline-block w-3 h-2 bg-emerald-500/70 rounded-sm" />Materijal
        </span>
      </div>
    </div>
  )
}

// ── Materials cost table ──────────────────────────────────────────────────────

function MaterialsTable({ materials }: { materials: ProjectMaterialCostItem[] }) {
  if (materials.length === 0) {
    return (
      <div className="text-gi-text-muted text-sm py-2">
        Nema evidentiranog ugrađenog materijala s unesenom cijenom za ovaj projekt u odabranom mjesecu.
      </div>
    )
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-gi-text-muted text-xs border-b border-gi-border">
            <th className="text-left pb-2 font-medium">Materijal</th>
            <th className="text-right pb-2 font-medium">Ugrađeno</th>
            <th className="text-right pb-2 font-medium">Jed. cijena</th>
            <th className="text-right pb-2 font-medium">Ukupni trošak</th>
          </tr>
        </thead>
        <tbody>
          {materials.map(mat => (
            <tr key={mat.project_material_id} className="border-b border-gi-border/50 hover:bg-gi-card/30 transition-colors">
              <td className="py-2">
                <div className="text-gi-text">{mat.material_name}</div>
                {mat.material_code && (
                  <div className="text-xs text-gi-text-subtle">{mat.material_code}</div>
                )}
              </td>
              <td className="py-2 text-right text-gi-text-muted">{qty(mat.installed_qty, mat.unit)}</td>
              <td className="py-2 text-right text-gi-text-muted">{eur(mat.unit_price)}/{mat.unit}</td>
              <td className="py-2 text-right text-emerald-400 font-medium">{eur(mat.total_cost)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ── Project detail panel ──────────────────────────────────────────────────────

function ProjectDetailPanel({
  projectId,
  year,
  month,
  onClose,
}: {
  projectId: string
  year: number
  month: number
  onClose: () => void
}) {
  const [detail, setDetail] = useState<ProjectDetailSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    apiClient.get<{ project: ProjectDetailSummary }>(
      `/analytics/projects/${encodeURIComponent(projectId)}?year=${year}&month=${month}`
    )
      .then(r => { setDetail(r.data.project); setLoading(false) })
      .catch(() => { setError('Greška pri dohvatu podataka'); setLoading(false) })
  }, [projectId, year, month])

  return (
    <div className="bg-gi-workspace border border-gi-border rounded-xl p-5 mt-2">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-gi-text font-semibold">
          {detail?.project_name ?? 'Detalji projekta'}
        </h3>
        <button onClick={onClose} className="text-gi-text-muted hover:text-gi-text transition-colors text-lg leading-none">
          ✕
        </button>
      </div>

      {loading && (
        <div className="text-gi-text-muted text-sm py-4 text-center">Učitavanje…</div>
      )}
      {error && (
        <div className="text-red-400 text-sm py-4 text-center">{error}</div>
      )}

      {detail && !loading && (
        <>
          {/* KPI strip — 6 cards: 2×3 on mobile, 3×2 on sm, 6 inline on lg */}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-5">
            {[
              { label: 'Ukupno sati', value: h(detail.total_hours) },
              { label: 'Neto trošak', value: eur(detail.total_neto) },
              { label: 'Doprinosi', value: eur(detail.total_contributions) },
              { label: 'Bruto rad', value: eur(detail.total_bruto), highlight: 'labor' },
              { label: 'Trošak materijala', value: eur(detail.total_material_cost), highlight: 'material' },
              { label: 'Ukupni direktni', value: eur(detail.total_direct_cost), highlight: 'total' },
            ].map(kpi => (
              <div key={kpi.label} className="bg-gi-card border border-gi-border rounded-lg p-3">
                <div className="text-xs text-gi-text-muted mb-0.5">{kpi.label}</div>
                <div className={`text-sm font-semibold ${
                  kpi.highlight === 'total' ? 'text-white' :
                  kpi.highlight === 'material' ? 'text-emerald-400' :
                  kpi.highlight === 'labor' ? 'text-gi-yellow' :
                  'text-gi-text'
                }`}>
                  {kpi.value}
                </div>
              </div>
            ))}
          </div>

          {/* Employee breakdown */}
          <div className="mb-5">
            <div className="text-xs text-gi-text-muted mb-2 uppercase tracking-wide">Trošak rada – Zaposlenici u {MONTHS_HR[month - 1]}</div>
            {detail.employees.length === 0 ? (
              <div className="text-gi-text-muted text-sm py-2">Nema evidentiranih sati za ovaj projekt u odabranom mjesecu.</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-gi-text-muted text-xs border-b border-gi-border">
                      <th className="text-left pb-2 font-medium">Zaposlenik</th>
                      <th className="text-right pb-2 font-medium">Sati</th>
                      <th className="text-right pb-2 font-medium">%</th>
                      <th className="text-right pb-2 font-medium">Neto</th>
                      <th className="text-right pb-2 font-medium">Doprinosi</th>
                      <th className="text-right pb-2 font-medium">Bruto</th>
                    </tr>
                  </thead>
                  <tbody>
                    {detail.employees.map(emp => (
                      <tr key={emp.employee_id} className="border-b border-gi-border/50 hover:bg-gi-card/30 transition-colors">
                        <td className="py-2 text-gi-text">{emp.employee_name}</td>
                        <td className="py-2 text-right text-gi-text-muted">{h(emp.hours)}</td>
                        <td className="py-2 text-right text-gi-text-muted">{emp.pct_of_project.toFixed(1)}%</td>
                        <td className="py-2 text-right text-gi-text">{eur(emp.neto_cost)}</td>
                        <td className="py-2 text-right">
                          {emp.contributions_missing ? (
                            <span className="text-amber-400 text-xs">Nije uneseno</span>
                          ) : (
                            <span className="text-gi-text-muted">{eur(emp.contributions_cost ?? 0)}</span>
                          )}
                        </td>
                        <td className="py-2 text-right">
                          {emp.bruto_cost != null ? (
                            <span className="text-gi-yellow font-medium">{eur(emp.bruto_cost)}</span>
                          ) : (
                            <span className="text-gi-text-muted">—</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Material breakdown */}
          <div className="mb-5">
            <div className="text-xs text-gi-text-muted mb-2 uppercase tracking-wide">Trošak materijala – Ugradnja u {MONTHS_HR[month - 1]}</div>
            <MaterialsTable materials={detail.materials} />
          </div>

          {/* Monthly trend chart */}
          {detail.monthly_trend.length > 0 && (
            <TrendChart trend={detail.monthly_trend} />
          )}

          {!detail.bruto_complete && (
            <div className="mt-4 p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg text-xs text-amber-300">
              Napomena: doprinosi za neke zaposlenike nisu uneseni. Bruto vrijednosti su nepotpune.
            </div>
          )}
        </>
      )}
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function ProjektiPage() {
  const router = useRouter()
  const { user, employee, isLoading: authLoading, logout } = useAuth()
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)
  const [data, setData] = useState<MonthlyProjectAllocationSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expandedProjectId, setExpandedProjectId] = useState<string | null>(null)

  useEffect(() => {
    if (!authLoading && user && FORBIDDEN_ROLES.includes(user.role)) {
      router.replace('/dashboard')
    }
  }, [authLoading, user, router])

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    apiClient.get<{ allocation: MonthlyProjectAllocationSummary }>(
      `/analytics/place/project-allocation?year=${year}&month=${month}`
    )
      .then(r => { setData(r.data.allocation); setLoading(false) })
      .catch(() => { setError('Greška pri dohvatu podataka'); setLoading(false) })
  }, [year, month])

  useEffect(() => { load() }, [load])

  if (authLoading || !user || FORBIDDEN_ROLES.includes(user.role)) return null

  const prevMonth = () => {
    if (month === 1) { setMonth(12); setYear(y => y - 1) }
    else setMonth(m => m - 1)
    setExpandedProjectId(null)
  }
  const nextMonth = () => {
    if (month === 12) { setMonth(1); setYear(y => y + 1) }
    else setMonth(m => m + 1)
    setExpandedProjectId(null)
  }

  const toggleProject = (id: string) => {
    setExpandedProjectId(prev => prev === id ? null : id)
  }

  const years = Array.from({ length: 5 }, (_, i) => now.getFullYear() - 2 + i)

  return (
    <AppShell user={user} employee={employee} title="Troškovi projekata" backHref="/dashboard" onLogout={logout}>
      <AnalyticsTabNav />

      {/* Month / Year navigator */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <button
          onClick={prevMonth}
          className="p-2 bg-gi-card border border-gi-border rounded-lg text-gi-text-muted hover:text-gi-text hover:bg-gi-card-elevated transition-colors"
          aria-label="Prethodni mjesec"
        >
          ←
        </button>
        <div className="bg-gi-card border border-gi-border rounded-lg px-4 py-2 text-gi-text font-medium min-w-[160px] text-center">
          {MONTHS_HR[month - 1]} {year}
        </div>
        <button
          onClick={nextMonth}
          className="p-2 bg-gi-card border border-gi-border rounded-lg text-gi-text-muted hover:text-gi-text hover:bg-gi-card-elevated transition-colors"
          aria-label="Sljedeći mjesec"
        >
          →
        </button>
        <select
          value={year}
          onChange={e => { setYear(Number(e.target.value)); setExpandedProjectId(null) }}
          className="bg-gi-card border border-gi-border rounded-lg px-3 py-2 text-gi-text text-sm"
        >
          {years.map(y => <option key={y} value={y}>{y}</option>)}
        </select>
      </div>

      {/* KPI summary cards — 3 cols on sm, 6 on xl */}
      {data && (
        <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-3 mb-6">
          {[
            { label: 'Ukupno sati', value: h(data.total_hours), cls: 'text-gi-text' },
            { label: 'Neto troškovi', value: eur(data.total_neto), cls: 'text-gi-text' },
            { label: 'Doprinosi', value: eur(data.total_contributions), cls: 'text-gi-text' },
            { label: 'Bruto rad', value: eur(data.total_bruto), cls: 'text-gi-yellow' },
            { label: 'Trošak materijala', value: eur(data.total_material_cost), cls: 'text-emerald-400' },
            { label: 'Ukupni direktni', value: eur(data.total_direct_cost), cls: 'text-white font-bold' },
          ].map(kpi => (
            <div key={kpi.label} className="bg-gi-card border border-gi-border rounded-xl p-4">
              <div className="text-xs text-gi-text-muted mb-1">{kpi.label}</div>
              <div className={`text-lg font-bold ${kpi.cls}`}>{kpi.value}</div>
            </div>
          ))}
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-4 text-red-400 text-sm mb-6">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="text-gi-text-muted text-sm py-8 text-center">Učitavanje…</div>
      )}

      {/* Project table */}
      {!loading && data && (
        <>
          {data.allocations.length === 0 ? (
            <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center">
              <div className="text-gi-text-muted text-sm">
                Nema evidentiranih sati u odabranom periodu.
              </div>
            </div>
          ) : (
            <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
              {/* Desktop table header */}
              <div className="hidden sm:grid grid-cols-[2fr_1fr_1fr_1fr_1fr_auto_auto] gap-4 px-5 py-3 border-b border-gi-border text-xs text-gi-text-muted uppercase tracking-wide font-medium">
                <span>Projekt</span>
                <span className="text-right">Sati</span>
                <span className="text-right">Trošak rada</span>
                <span className="text-right">Materijal</span>
                <span className="text-right">Ukupni</span>
                <span className="text-center">Status</span>
                <span />
              </div>

              {data.allocations.map((proj: MonthlyProjectAllocation) => (
                <div key={proj.project_id}>
                  {/* Row */}
                  <button
                    onClick={() => toggleProject(proj.project_id)}
                    className="w-full text-left"
                  >
                    {/* Desktop row */}
                    <div className="hidden sm:grid grid-cols-[2fr_1fr_1fr_1fr_1fr_auto_auto] gap-4 px-5 py-4 border-b border-gi-border/60 hover:bg-gi-card-elevated/50 transition-colors items-center">
                      <div>
                        <div className="text-gi-text font-medium">{proj.project_name}</div>
                        <div className="text-xs text-gi-text-subtle mt-0.5">{proj.employee_count} zaposlenik(a) · {proj.pct_of_total.toFixed(1)}% ukupnih sati</div>
                      </div>
                      <div className="text-right text-gi-text-muted text-sm">{h(proj.total_hours)}</div>
                      <div className="text-right text-gi-yellow text-sm font-medium">
                        {proj.contributions_incomplete ? `${eur(proj.neto_cost)} *` : eur(proj.bruto_cost)}
                      </div>
                      <div className="text-right text-emerald-400 text-sm">
                        {proj.material_cost > 0 ? eur(proj.material_cost) : <span className="text-gi-text-subtle">—</span>}
                      </div>
                      <div className="text-right text-white font-semibold text-sm">
                        {proj.total_direct_cost > 0 ? eur(proj.total_direct_cost) : <span className="text-gi-text-subtle">—</span>}
                      </div>
                      <div className="flex justify-center">
                        <StatusBadge incomplete={proj.contributions_incomplete} />
                      </div>
                      <div className="text-gi-text-muted text-xs">
                        {expandedProjectId === proj.project_id ? '▲' : '▼'}
                      </div>
                    </div>

                    {/* Mobile card */}
                    <div className="sm:hidden px-4 py-3 border-b border-gi-border/60 hover:bg-gi-card-elevated/50 transition-colors">
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <div className="text-gi-text font-medium text-sm">{proj.project_name}</div>
                          <div className="text-xs text-gi-text-subtle mt-0.5">{proj.employee_count} zaposlenik(a)</div>
                        </div>
                        <div className="text-right">
                          <div className="text-white font-semibold text-sm">
                            {proj.total_direct_cost > 0 ? eur(proj.total_direct_cost) : eur(proj.neto_cost)}
                          </div>
                          <div className="text-xs text-gi-text-muted">{h(proj.total_hours)}</div>
                        </div>
                      </div>
                      {proj.contributions_incomplete && (
                        <div className="mt-1.5">
                          <StatusBadge incomplete={true} />
                        </div>
                      )}
                    </div>
                  </button>

                  {/* Expanded detail panel */}
                  {expandedProjectId === proj.project_id && (
                    <div className="px-4 pb-4">
                      <ProjectDetailPanel
                        projectId={proj.project_id}
                        year={year}
                        month={month}
                        onClose={() => setExpandedProjectId(null)}
                      />
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Unassigned hours warning */}
          {data.unassigned_hours > 0 && (
            <div className="mt-4 p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg text-xs text-amber-300">
              {h(data.unassigned_hours)} evidentirano bez projekta i nije uključeno u alokaciju.
            </div>
          )}

          {/* Disclaimer */}
          <div className="mt-4 text-xs text-gi-text-subtle">
            Trošak rada = Bruto (Neto + Doprinosi). Trošak materijala = ugrađene količine × jedinična cijena iz evidencije materijala.
            Hladni pogon (operativni troškovi) nisu uključeni. Materijali bez unesene cijene ne ulaze u izračun.
          </div>
        </>
      )}
    </AppShell>
  )
}
