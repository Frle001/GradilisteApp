'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
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
      <DashboardShell user={user} employee={employee} title="Sati zaposlenika" backHref="/dashboard" onLogout={logout}>
        <p className="text-red-400">Nemate pristup ovoj stranici.</p>
      </DashboardShell>
    )
  }

  // Group entries by date for display
  const byDate = entries.reduce<Record<string, ManagerEntry[]>>((acc, e) => {
    if (!acc[e.work_date]) acc[e.work_date] = []
    acc[e.work_date].push(e)
    return acc
  }, {})
  const dates = Object.keys(byDate).sort((a, b) => b.localeCompare(a))

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Sati zaposlenika"
      backHref="/dashboard"
      onLogout={logout}
    >
      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Sati zaposlenika</h1>
        <p className="text-slate-400 text-sm mt-1">Pregled upisanih radnih sati radnika i poslovođa</p>
      </div>

      {/* Date filter */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 mb-6 flex items-center gap-4">
        <label className="text-sm font-medium text-slate-300 shrink-0">Datum</label>
        <input
          type="date"
          value={filterDate}
          onChange={e => setFilterDate(e.target.value)}
          className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
        />
        <button
          onClick={() => setFilterDate('')}
          className="text-xs text-slate-500 hover:text-slate-300 transition"
        >
          Prikaži sve datume
        </button>
      </div>

      {fetchError && (
        <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3 mb-6">
          <p className="text-red-300 text-sm">{fetchError}</p>
        </div>
      )}

      {loading ? (
        <p className="text-slate-500 text-sm">Učitavanje...</p>
      ) : entries.length === 0 ? (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 text-center">
          <p className="text-slate-500 text-sm">
            {filterDate ? `Nema upisanih sati za ${formatHrDate(filterDate)}.` : 'Nema upisanih sati.'}
          </p>
        </div>
      ) : (
        <div className="space-y-6">
          {dates.map(date => (
            <div key={date}>
              <h2 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3 capitalize">
                {formatHrDate(date)}
              </h2>
              <div className="space-y-2">
                {byDate[date].map(entry => (
                  <Link
                    key={entry.id}
                    href={`/dashboard/worker-hours/${entry.id}`}
                    className="block bg-slate-900 border border-slate-800 hover:border-slate-600 rounded-xl px-4 py-3 transition"
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-semibold text-white">{entry.worker_name}</span>
                          <span className="text-xs text-slate-500">
                            {ROLE_LABELS[entry.worker_role] ?? entry.worker_role}
                          </span>
                          {entry.has_revisions && (
                            <span className="text-xs bg-amber-900/60 text-amber-300 border border-amber-700 rounded px-1.5 py-0.5">
                              Ispravljeno
                            </span>
                          )}
                          {entry.has_comments && (
                            <span className="text-xs bg-blue-900/60 text-blue-300 border border-blue-700 rounded px-1.5 py-0.5">
                              Komentar
                            </span>
                          )}
                        </div>
                        <p className="text-xs text-slate-400 mt-0.5 truncate">{entry.project_name}</p>
                        {entry.notes && (
                          <p className="text-xs text-slate-500 mt-0.5 truncate">{entry.notes}</p>
                        )}
                        {entry.work_description && (
                          <p className="text-xs text-slate-500 mt-0.5 truncate italic">{entry.work_description}</p>
                        )}
                      </div>
                      <div className="shrink-0 text-right">
                        <span className="text-lg font-bold text-white">{entry.hours_worked}</span>
                        <span className="text-slate-400 text-sm ml-0.5">h</span>
                        <p className="text-xs text-slate-600 mt-0.5">Detalji →</p>
                      </div>
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </DashboardShell>
  )
}
