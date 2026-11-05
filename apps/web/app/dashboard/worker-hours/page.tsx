'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import apiClient from '@/lib/api-client'
import { ROLE_LABELS } from '@/lib/types/employees'

interface ManagerEntry {
  id: string
  worker_id: string
  worker_name: string
  worker_role: string
  project_id: string
  project_name: string
  work_date: string
  hours_worked: number
  notes: string | null
  work_description: string | null
  has_revisions: boolean
  has_comments: boolean
  updated_at: string
}

function localTodayStr(): string {
  return new Date().toLocaleDateString('sv') // YYYY-MM-DD
}

function formatHrDate(dateStr: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('hr-HR', {
    weekday: 'short', day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

export default function WorkerHoursManagerPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const today = localTodayStr()

  const [entries, setEntries] = useState<ManagerEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)
  const [filterDate, setFilterDate] = useState(today)

  const fetchEntries = useCallback((date: string) => {
    setLoading(true)
    setFetchError(null)
    const params = date ? `?work_date=${date}` : ''
    apiClient.get(`/worker-hours/manager${params}`)
      .then(res => setEntries(res.data.entries ?? []))
      .catch(() => setFetchError('Greška pri dohvatu podataka.'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!user) return
    if (user.role !== 'direktor' && user.role !== 'inzenjer') return
    fetchEntries(filterDate)
  }, [user, filterDate, fetchEntries])

  if (isLoading) return <LoadingScreen />
  if (!user) return null
  if (user.role !== 'direktor' && user.role !== 'inzenjer') {
    return (
      <AppShell user={user} employee={employee} title="Sati zaposlenika" backHref="/dashboard" onLogout={logout}>
        <p className="text-red-400 text-sm">Nemate pristup ovoj stranici.</p>
      </AppShell>
    )
  }

  // Group entries by date for display
  const byDate = entries.reduce<Record<string, ManagerEntry[]>>((acc, e) => {
    if (!acc[e.work_date]) acc[e.work_date] = []
    acc[e.work_date].push(e)
    return acc
  }, {})
  const dates = Object.keys(byDate).sort((a, b) => b.localeCompare(a))
  const totalHours = entries.reduce((sum, e) => sum + e.hours_worked, 0)

  return (
    <AppShell user={user} employee={employee} title="Sati zaposlenika" backHref="/dashboard" onLogout={logout}>
      <div className="space-y-5">

        {/* Page header */}
        <div>
          <h1 className="text-xl font-bold text-gi-text">Sati zaposlenika</h1>
          <p className="text-gi-text-subtle text-sm mt-0.5">Pregled upisanih radnih sati radnika i poslovođa</p>
        </div>

        {/* Filter toolbar */}
        <div className="bg-gi-card border border-gi-border rounded-xl px-4 py-3 flex flex-wrap items-center gap-3">
          <label className="text-xs font-medium text-gi-text-muted uppercase tracking-wider shrink-0">Datum</label>
          <input
            type="date"
            value={filterDate}
            onChange={e => setFilterDate(e.target.value)}
            className="px-3 py-2 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text text-sm focus:outline-none focus:border-gi-yellow transition-colors"
          />
          <div className="flex gap-2">
            <button
              onClick={() => setFilterDate(today)}
              disabled={filterDate === today}
              className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-2 rounded-lg transition-colors disabled:opacity-40"
            >
              Danas
            </button>
            <button
              onClick={() => setFilterDate('')}
              disabled={filterDate === ''}
              className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-2 rounded-lg transition-colors disabled:opacity-40"
            >
              Svi datumi
            </button>
          </div>

          {/* Summary pill */}
          {!loading && entries.length > 0 && (
            <div className="ml-auto flex items-center gap-3">
              <span className="text-xs text-gi-text-muted">{entries.length} unosa</span>
              <span className="text-xs font-semibold text-gi-yellow">{totalHours.toFixed(1)} h ukupno</span>
            </div>
          )}
        </div>

        {fetchError && (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{fetchError}</p>
          </div>
        )}

        {loading ? (
          <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
          </div>
        ) : entries.length === 0 ? (
          <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center">
            <p className="text-gi-text-subtle text-sm">
              {filterDate ? `Nema upisanih sati za ${formatHrDate(filterDate)}.` : 'Nema upisanih sati.'}
            </p>
          </div>
        ) : (
          <div className="space-y-6">
            {dates.map(date => {
              const dateEntries = byDate[date]
              const dateTotal = dateEntries.reduce((s, e) => s + e.hours_worked, 0)
              return (
                <div key={date}>
                  <div className="flex items-center justify-between mb-2 px-0.5">
                    <h2 className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider capitalize">
                      {formatHrDate(date)}
                    </h2>
                    <span className="text-xs text-gi-text-subtle">{dateTotal.toFixed(1)} h</span>
                  </div>

                  {/* Desktop: table */}
                  <div className="hidden sm:block bg-gi-card border border-gi-border rounded-xl overflow-hidden">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-gi-border">
                          <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wider">Zaposlenik</th>
                          <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wider">Projekt</th>
                          <th className="text-left px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wider">Bilješka</th>
                          <th className="text-right px-4 py-2.5 text-xs font-medium text-gi-text-muted uppercase tracking-wider">Sati</th>
                          <th className="w-8" />
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gi-border/60">
                        {dateEntries.map(entry => (
                          <tr
                            key={entry.id}
                            className="hover:bg-gi-navy-hover transition-colors cursor-pointer"
                            onClick={() => window.location.href = `/dashboard/worker-hours/${entry.id}`}
                          >
                            <td className="px-4 py-3">
                              <div className="flex items-center gap-2 flex-wrap">
                                <span className="font-medium text-gi-text">{entry.worker_name}</span>
                                <span className="text-xs text-gi-text-subtle">
                                  {ROLE_LABELS[entry.worker_role] ?? entry.worker_role}
                                </span>
                                {entry.has_revisions && (
                                  <span className="text-[10px] bg-amber-900/50 text-amber-300 border border-amber-700/60 rounded px-1.5 py-0.5">
                                    Ispravljeno
                                  </span>
                                )}
                                {entry.has_comments && (
                                  <span className="text-[10px] bg-blue-900/50 text-blue-300 border border-blue-700/60 rounded px-1.5 py-0.5">
                                    Komentar
                                  </span>
                                )}
                              </div>
                            </td>
                            <td className="px-4 py-3 text-gi-text-muted truncate max-w-[200px]">{entry.project_name}</td>
                            <td className="px-4 py-3 text-gi-text-subtle text-xs truncate max-w-[180px]">
                              {entry.notes ?? (entry.work_description ? <em>{entry.work_description}</em> : '—')}
                            </td>
                            <td className="px-4 py-3 text-right font-bold text-gi-text">
                              {entry.hours_worked} <span className="text-gi-text-subtle font-normal text-xs">h</span>
                            </td>
                            <td className="px-4 py-3 text-gi-text-subtle text-xs text-right">→</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Mobile: cards */}
                  <div className="sm:hidden space-y-2">
                    {dateEntries.map(entry => (
                      <Link
                        key={entry.id}
                        href={`/dashboard/worker-hours/${entry.id}`}
                        className="block bg-gi-card border border-gi-border hover:border-gi-text-subtle rounded-xl px-4 py-3 transition-colors"
                      >
                        <div className="flex items-start justify-between gap-4">
                          <div className="min-w-0">
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="text-sm font-semibold text-gi-text">{entry.worker_name}</span>
                              <span className="text-xs text-gi-text-subtle">
                                {ROLE_LABELS[entry.worker_role] ?? entry.worker_role}
                              </span>
                              {entry.has_revisions && (
                                <span className="text-[10px] bg-amber-900/50 text-amber-300 border border-amber-700/60 rounded px-1.5 py-0.5">
                                  Ispravljeno
                                </span>
                              )}
                              {entry.has_comments && (
                                <span className="text-[10px] bg-blue-900/50 text-blue-300 border border-blue-700/60 rounded px-1.5 py-0.5">
                                  Komentar
                                </span>
                              )}
                            </div>
                            <p className="text-xs text-gi-text-muted mt-0.5 truncate">{entry.project_name}</p>
                            {entry.notes && (
                              <p className="text-xs text-gi-text-subtle mt-0.5 truncate">{entry.notes}</p>
                            )}
                          </div>
                          <div className="shrink-0 text-right">
                            <span className="text-lg font-bold text-gi-text">{entry.hours_worked}</span>
                            <span className="text-gi-text-subtle text-sm ml-0.5">h</span>
                            <p className="text-xs text-gi-text-subtle mt-0.5">Detalji →</p>
                          </div>
                        </div>
                      </Link>
                    ))}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </AppShell>
  )
}
