'use client'

import { useEffect, useState, useCallback } from 'react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import apiClient from '@/lib/api-client'

interface ManagementHoursEntry {
  id: string
  employee_id: string
  employee_name: string
  employee_role: string
  project_id: string | null
  project_name: string | null
  work_date: string
  hours_worked: number
  notes: string | null
  created_at: string
  updated_at: string
}

const ROLE_LABELS: Record<string, string> = {
  direktor: 'Direktor',
  inzenjer: 'Inženjer',
  administracija: 'Administracija',
}

// Filter tabs visible to direktor only
type FilterTab = 'all' | 'inzenjer' | 'administracija' | 'mine'

function localTodayStr(): string {
  return new Date().toLocaleDateString('sv') // YYYY-MM-DD
}

function formatHrDate(dateStr: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('hr-HR', {
    weekday: 'short', day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

export default function ManagementHoursPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const today = localTodayStr()

  const [entries, setEntries] = useState<ManagementHoursEntry[]>([])
  const [loadingEntries, setLoadingEntries] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)

  // Form state
  const [formDate, setFormDate] = useState(today)
  const [hours, setHours] = useState('')
  const [notes, setNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [submitSuccess, setSubmitSuccess] = useState<string | null>(null)

  // Director filter
  const [filterTab, setFilterTab] = useState<FilterTab>('all')

  const isDirector = user?.role === 'direktor'

  const fetchEntries = useCallback(() => {
    if (!user) return
    setLoadingEntries(true)
    setFetchError(null)
    apiClient.get('/management-hours')
      .then(res => setEntries(res.data.entries ?? []))
      .catch(() => setFetchError('Greška pri dohvatu sati.'))
      .finally(() => setLoadingEntries(false))
  }, [user])

  useEffect(() => {
    if (user) fetchEntries()
  }, [user, fetchEntries])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitError(null)
    setSubmitSuccess(null)

    const h = parseFloat(hours)
    if (isNaN(h) || h < 0 || h > 24) {
      setSubmitError('Unesite broj sati između 0 i 24.')
      return
    }
    if (!formDate) {
      setSubmitError('Datum je obavezan.')
      return
    }

    setSubmitting(true)
    try {
      await apiClient.post('/management-hours', {
        work_date: formDate,
        hours_worked: h,
        notes: notes.trim() || null,
      })
      setSubmitSuccess('Sati su uspješno upisani.')
      setHours('')
      setNotes('')
      fetchEntries()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setSubmitError(msg ?? 'Greška pri unosu sati.')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await apiClient.delete(`/management-hours/${id}`)
      fetchEntries()
    } catch {
      // Silently ignore; user will see stale data and can retry
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  // Only allowed roles reach this page (middleware blocks others, but guard client-side too)
  if (!['direktor', 'inzenjer', 'administracija'].includes(user.role)) {
    return (
      <DashboardShell user={user} employee={employee} title="Moji sati" onLogout={logout}>
        <p className="text-slate-400 text-sm">Nemate pristup ovoj stranici.</p>
      </DashboardShell>
    )
  }

  // Apply director filter
  const filteredEntries = isDirector
    ? entries.filter(e => {
        if (filterTab === 'mine') return e.employee_id === user.id || (employee && e.employee_id === employee.id)
        if (filterTab === 'inzenjer') return e.employee_role === 'inzenjer'
        if (filterTab === 'administracija') return e.employee_role === 'administracija'
        return true // 'all'
      })
    : entries

  const tabs: { key: FilterTab; label: string }[] = [
    { key: 'all', label: 'Svi' },
    { key: 'inzenjer', label: 'Inženjeri' },
    { key: 'administracija', label: 'Računovodstvo' },
    { key: 'mine', label: 'Moji sati' },
  ]

  return (
    <DashboardShell user={user} employee={employee} title="Moji sati" onLogout={logout}>
      <div className="space-y-8">

        {/* ── Hour entry form ─────────────────────────────────────────────── */}
        <section className="bg-slate-900 border border-slate-800 rounded-xl p-6">
          <h2 className="text-base font-semibold text-white mb-4">Unos radnih sati</h2>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs text-slate-400 mb-1">Datum *</label>
                <input
                  type="date"
                  value={formDate}
                  max={today}
                  onChange={e => setFormDate(e.target.value)}
                  required
                  className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-400 mb-1">Broj sati *</label>
                <input
                  type="number"
                  min={0}
                  max={24}
                  step="0.5"
                  value={hours}
                  onChange={e => setHours(e.target.value)}
                  placeholder="8"
                  required
                  className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
            </div>
            <div>
              <label className="block text-xs text-slate-400 mb-1">Napomena</label>
              <input
                type="text"
                value={notes}
                onChange={e => setNotes(e.target.value)}
                placeholder="Opcionalno…"
                className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>

            {submitError && (
              <p className="text-sm text-red-400">{submitError}</p>
            )}
            {submitSuccess && (
              <p className="text-sm text-emerald-400">{submitSuccess}</p>
            )}

            <button
              type="submit"
              disabled={submitting}
              className="px-5 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
            >
              {submitting ? 'Zapisivanje…' : 'Upiši sate'}
            </button>
          </form>
        </section>

        {/* ── History / list ──────────────────────────────────────────────── */}
        <section>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-white">
              {isDirector ? 'Radni sati' : 'Moji sati'}
            </h2>
            <button
              onClick={fetchEntries}
              disabled={loadingEntries}
              className="text-xs text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-3 py-1.5 rounded transition"
            >
              Osvježi
            </button>
          </div>

          {/* Director filter tabs */}
          {isDirector && (
            <div className="flex gap-1 mb-4 flex-wrap">
              {tabs.map(tab => (
                <button
                  key={tab.key}
                  onClick={() => setFilterTab(tab.key)}
                  className={[
                    'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
                    filterTab === tab.key
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-800 text-slate-400 hover:text-white',
                  ].join(' ')}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          )}

          {fetchError && (
            <p className="text-sm text-red-400 mb-4">{fetchError}</p>
          )}

          {loadingEntries ? (
            <p className="text-slate-400 text-sm">Učitavanje…</p>
          ) : filteredEntries.length === 0 ? (
            <p className="text-slate-500 text-sm">Nema upisanih sati.</p>
          ) : (
            <div className="space-y-2">
              {filteredEntries.map(entry => {
                const isOwn = employee ? entry.employee_id === employee.id : false
                return (
                  <div
                    key={entry.id}
                    className="bg-slate-900 border border-slate-800 rounded-lg px-4 py-3 flex items-start justify-between gap-3"
                  >
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-sm font-medium text-white">
                          {formatHrDate(entry.work_date)}
                        </span>
                        <span className="text-sm text-blue-400 font-semibold">
                          {entry.hours_worked} h
                        </span>
                        {isDirector && (
                          <span className="text-xs text-slate-500">
                            {entry.employee_name} · {ROLE_LABELS[entry.employee_role] ?? entry.employee_role}
                          </span>
                        )}
                      </div>
                      {entry.notes && (
                        <p className="text-xs text-slate-400 mt-1 truncate">{entry.notes}</p>
                      )}
                      {entry.project_name && (
                        <p className="text-xs text-slate-500 mt-0.5">Projekt: {entry.project_name}</p>
                      )}
                    </div>
                    {isOwn && (
                      <button
                        onClick={() => handleDelete(entry.id)}
                        className="shrink-0 text-xs text-slate-600 hover:text-red-400 transition-colors"
                        aria-label="Obriši unos"
                      >
                        ×
                      </button>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </section>
      </div>
    </DashboardShell>
  )
}
