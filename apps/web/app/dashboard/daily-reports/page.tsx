'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
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
    <DashboardShell
      user={user}
      employee={employee}
      title="Dnevni izvještaji"
      backHref="/dashboard"
      onLogout={logout}
      action={
        canCreate ? (
          <Link
            href="/dashboard/daily-reports/new"
            className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
          >
            + Novi izvještaj
          </Link>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div className="flex items-center justify-between gap-4">
          <h1 className="text-2xl font-bold text-white">Dnevni izvještaji</h1>
          <button
            onClick={() => void handleRefresh()}
            disabled={refreshing || isLoadingReports}
            className="shrink-0 text-sm text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-3 py-2 rounded-lg transition disabled:opacity-50"
          >
            {refreshing ? 'Osvježavanje…' : 'Osvježi'}
          </button>
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-3">
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Pretraži po projektu ili poslovođi…"
            className="flex-1 min-w-48 px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 transition"
          />
          <select
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value)}
            className="px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 transition"
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
            className="px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 transition"
          />
          <input
            type="date"
            value={dateTo}
            onChange={e => setDateTo(e.target.value)}
            title="Datum do"
            className="px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 transition"
          />
        </div>

        {error && (
          <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{error}</p>
          </div>
        )}

        {isLoadingReports ? (
          <LoadingOverlay />
        ) : filtered.length === 0 ? (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-10 text-center">
            <p className="text-slate-400 text-sm">Nema izvještaja koji odgovaraju odabranim filterima.</p>
            {canCreate && (
              <Link
                href="/dashboard/daily-reports/new"
                className="mt-4 inline-block px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium rounded-lg transition"
              >
                Kreiraj prvi izvještaj
              </Link>
            )}
          </div>
        ) : (
          <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
            <div className="overflow-x-auto overscroll-x-contain touch-pan-x table-scroll-touch">
            <table className="min-w-[760px] w-full text-sm">
              <thead>
                <tr className="border-b border-slate-800">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wide">Datum</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wide">Projekt</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wide hidden sm:table-cell">Poslovođa</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wide hidden md:table-cell">Radnici / Sati</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wide hidden md:table-cell">Aktivnosti</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wide">Status</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {filtered.map(r => (
                  <tr key={r.id} className="hover:bg-slate-800/40 transition-colors">
                    <td className="px-4 py-3 text-slate-300 whitespace-nowrap">
                      {formatDate(r.report_date)}
                      {formatTime(r.submitted_at ?? r.created_at) && (
                        <span className="block text-xs text-slate-500">{formatTime(r.submitted_at ?? r.created_at)}</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-slate-200 font-medium max-w-48 truncate">{r.project_name}</td>
                    <td className="px-4 py-3 text-slate-400 hidden sm:table-cell whitespace-nowrap">{r.poslovoda_name}</td>
                    <td className="px-4 py-3 text-slate-400 hidden md:table-cell whitespace-nowrap">
                      {r.worker_hours_count} / {r.total_hours}h
                    </td>
                    <td className="px-4 py-3 text-slate-400 hidden md:table-cell">{r.activities_count}</td>
                    <td className="px-4 py-3">
                      <DailyReportStatusBadge status={r.status} />
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        href={`/dashboard/daily-reports/${r.id}`}
                        className="text-xs text-blue-400 hover:text-blue-300 font-medium"
                      >
                        Otvori →
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  )
}
