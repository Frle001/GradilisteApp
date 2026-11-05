'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import DailyReportStatusBadge from '@/components/daily-reports/DailyReportStatusBadge'
import {
  type DailyReportListItem,
  type ReportStatus,
  REPORT_STATUS_LABELS,
  canCreateDailyReport,
} from '@/lib/types/daily-reports'
import apiClient from '@/lib/api-client'
import { onRefresh } from '@/lib/refresh-events'

const STATUS_FILTERS: Array<{ value: string; label: string }> = [
  { value: '', label: 'Svi statusi' },
  ...(['draft', 'submitted', 'approved', 'rejected'] as ReportStatus[]).map(s => ({
    value: s,
    label: REPORT_STATUS_LABELS[s],
  })),
]

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('hr-HR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

function formatTime(dateStr: string | null) {
  if (!dateStr) return null
  return new Date(dateStr).toLocaleTimeString('hr-HR', { hour: '2-digit', minute: '2-digit' })
}

const controlCls = 'px-3 py-2 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text placeholder:text-gi-text-subtle text-sm focus:outline-none focus:border-gi-yellow transition-colors'

export default function DailyReportsPage() {
  const { user, employee, isLoading, logout } = useAuth()

  const [reports, setReports] = useState<DailyReportListItem[]>([])
  const [isLoadingReports, setIsLoadingReports] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

  const canCreate = !!user && canCreateDailyReport(user.role)
  const hasActiveFilters = !!(search || statusFilter || dateFrom || dateTo)

  function resetFilters() {
    setSearch('')
    setStatusFilter('')
    setDateFrom('')
    setDateTo('')
  }

  const fetchReports = useCallback(() => {
    if (!user) return
    setIsLoadingReports(true)
    const params = new URLSearchParams()
    if (statusFilter) params.set('status', statusFilter)
    if (dateFrom) params.set('date_from', dateFrom)
    if (dateTo) params.set('date_to', dateTo)

    apiClient
      .get(`/daily-reports${params.toString() ? '?' + params.toString() : ''}`)
      .then(res => {
        setReports(res.data.daily_reports ?? [])
        setError(null)
      })
      .catch(() => setError('Greška pri učitavanju izvještaja.'))
      .finally(() => setIsLoadingReports(false))
  }, [user, statusFilter, dateFrom, dateTo])

  useEffect(() => { fetchReports() }, [fetchReports])

  // Refresh list when a daily-report sync completes
  useEffect(() => {
    return onRefresh((domains) => {
      if (domains.includes('daily-reports')) fetchReports()
    })
  }, [fetchReports])

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      const { runSync } = await import('@/lib/offline/sync-engine')
      await runSync({ userId: user?.id, companyId: user?.company_id })
    } catch { /* offline — just refetch */ }
    fetchReports()
    setRefreshing(false)
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const filtered = search.trim()
    ? reports.filter(r =>
        r.project_name.toLowerCase().includes(search.toLowerCase()) ||
        r.poslovoda_name.toLowerCase().includes(search.toLowerCase())
      )
    : reports

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Dnevni izvještaji"
      backHref="/dashboard"
      onLogout={logout}
      action={
        canCreate ? (
          <Link
            href="/dashboard/daily-reports/new"
            className="inline-flex items-center gap-1.5 text-sm font-semibold bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy px-4 py-2 rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            Novi izvještaj
          </Link>
        ) : undefined
      }
    >
      <div className="space-y-4">
        {/* Page heading */}
        <div>
          <h1 className="text-xl font-bold text-gi-text">Dnevni izvještaji</h1>
          <p className="text-gi-text-subtle text-xs mt-0.5">Pregled i praćenje izvještaja s gradilišta</p>
        </div>

        {/* Filter toolbar */}
        <div className="flex flex-wrap gap-2">
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Pretraži projekt ili poslovođu…"
            className={controlCls + ' flex-1 min-w-[180px]'}
          />
          <select
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value)}
            className={controlCls}
          >
            {STATUS_FILTERS.map(f => (
              <option key={f.value} value={f.value}>{f.label}</option>
            ))}
          </select>
          <input
            type="date"
            value={dateFrom}
            onChange={e => setDateFrom(e.target.value)}
            title="Datum od"
            className={controlCls}
          />
          <input
            type="date"
            value={dateTo}
            onChange={e => setDateTo(e.target.value)}
            title="Datum do"
            className={controlCls}
          />
          {hasActiveFilters && (
            <button
              onClick={resetFilters}
              className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-border/80 bg-gi-card-elevated px-3 py-2 rounded-lg transition whitespace-nowrap"
            >
              Poništi filtere
            </button>
          )}
          <button
            onClick={() => void handleRefresh()}
            disabled={refreshing || isLoadingReports}
            className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-border/80 bg-gi-card-elevated px-3 py-2 rounded-lg transition disabled:opacity-50 whitespace-nowrap"
          >
            {refreshing ? 'Osvježavanje…' : 'Osvježi'}
          </button>
        </div>

        {error && (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{error}</p>
          </div>
        )}

        {isLoadingReports ? (
          <div className="bg-gi-card border border-gi-border rounded-xl p-10 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje izvještaja…</p>
          </div>
        ) : filtered.length === 0 ? (
          <div className="bg-gi-card border border-gi-border rounded-xl p-10 text-center space-y-3">
            <p className="text-gi-text-subtle text-sm">Nema izvještaja koji odgovaraju odabranim filterima.</p>
            {canCreate && (
              <Link
                href="/dashboard/daily-reports/new"
                className="inline-flex items-center gap-1.5 px-4 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition"
              >
                Kreiraj prvi izvještaj
              </Link>
            )}
          </div>
        ) : (
          <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
            {/* Desktop table */}
            <table className="hidden sm:table w-full text-sm">
              <thead>
                <tr className="border-b border-gi-border">
                  <th className="text-left px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Datum</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Projekt</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider hidden md:table-cell">Poslovođa</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider hidden lg:table-cell">Radnici / Sati</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider hidden lg:table-cell">Aktivnosti</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Status</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-gi-border/60">
                {filtered.map(r => (
                  <tr key={r.id} className="hover:bg-gi-card-elevated/40 transition-colors">
                    <td className="px-4 py-3 text-gi-text-muted whitespace-nowrap">
                      {formatDate(r.report_date)}
                      {formatTime(r.submitted_at ?? r.created_at) && (
                        <span className="block text-xs text-gi-text-subtle">{formatTime(r.submitted_at ?? r.created_at)}</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <Link
                        href={`/dashboard/daily-reports/${r.id}`}
                        className="font-medium text-gi-text hover:text-gi-yellow transition-colors max-w-48 truncate block"
                      >
                        {r.project_name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-gi-text-muted hidden md:table-cell whitespace-nowrap">{r.poslovoda_name}</td>
                    <td className="px-4 py-3 text-gi-text-muted hidden lg:table-cell whitespace-nowrap">
                      {r.worker_hours_count} / {r.total_hours}h
                    </td>
                    <td className="px-4 py-3 text-gi-text-muted hidden lg:table-cell">{r.activities_count}</td>
                    <td className="px-4 py-3">
                      <DailyReportStatusBadge status={r.status} />
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        href={`/dashboard/daily-reports/${r.id}`}
                        className="text-xs text-gi-text-subtle hover:text-gi-yellow transition-colors font-medium"
                      >
                        Otvori →
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Mobile cards */}
            <div className="sm:hidden divide-y divide-gi-border/60">
              {filtered.map(r => (
                <Link
                  key={r.id}
                  href={`/dashboard/daily-reports/${r.id}`}
                  className="block p-4 hover:bg-gi-card-elevated/40 transition-colors"
                >
                  <div className="flex items-start justify-between gap-2 mb-1.5">
                    <p className="font-semibold text-gi-text text-sm truncate">{r.project_name}</p>
                    <DailyReportStatusBadge status={r.status} />
                  </div>
                  <p className="text-xs text-gi-text-muted mb-1">{formatDate(r.report_date)}</p>
                  <div className="flex items-center gap-3 text-xs text-gi-text-subtle flex-wrap">
                    <span>{r.poslovoda_name}</span>
                    {r.activities_count > 0 && (
                      <>
                        <span>·</span>
                        <span>{r.activities_count} aktivnosti</span>
                      </>
                    )}
                    {r.total_hours > 0 && (
                      <>
                        <span>·</span>
                        <span>{r.total_hours}h</span>
                      </>
                    )}
                  </div>
                </Link>
              ))}
            </div>
          </div>
        )}
      </div>
    </AppShell>
  )
}
