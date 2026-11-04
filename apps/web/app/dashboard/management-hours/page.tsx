'use client'

import { useEffect, useState, useCallback } from 'react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
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

const inputCls =
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2.5 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'
const labelCls = 'block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5'

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

  // Delete confirmation
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)

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
    setDeleteError(null)
    setDeleting(true)
    try {
      await apiClient.delete(`/management-hours/${id}`)
      setDeletingId(null)
      fetchEntries()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setDeleteError(msg ?? 'Greška pri brisanju unosa.')
    } finally {
      setDeleting(false)
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  if (!['direktor', 'inzenjer', 'administracija'].includes(user.role)) {
    return (
      <AppShell user={user} employee={employee} title="Radni sati" onLogout={logout}>
        <p className="text-red-400 text-sm">Nemate pristup ovoj stranici.</p>
      </AppShell>
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

  const totalHours = filteredEntries.reduce((sum, e) => sum + e.hours_worked, 0)

  return (
    <AppShell user={user} employee={employee} title="Radni sati" backHref="/dashboard" onLogout={logout}>
      <div className="max-w-3xl mx-auto space-y-6">

        {/* Page header */}
        <div>
          <h1 className="text-xl font-bold text-gi-text">Radni sati — Menadžment</h1>
          <p className="text-gi-text-subtle text-sm mt-0.5">
            Evidencija radnih sati za direktore, inženjere i administraciju
          </p>
        </div>

        {/* Entry form */}
        <section className="bg-gi-card border border-gi-border rounded-xl p-5">
          <h2 className="text-sm font-semibold text-gi-text mb-4">Unos radnih sati</h2>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className={labelCls}>
                  Datum <span className="text-red-400 normal-case font-normal">*</span>
                </label>
                <input
                  type="date"
                  value={formDate}
                  max={today}
                  onChange={e => setFormDate(e.target.value)}
                  required
                  className={inputCls}
                  disabled={submitting}
                />
              </div>
              <div>
                <label className={labelCls}>
                  Broj sati <span className="text-red-400 normal-case font-normal">*</span>
                </label>
                <input
                  type="number"
                  min={0}
                  max={24}
                  step="0.5"
                  inputMode="decimal"
                  value={hours}
                  onChange={e => setHours(e.target.value)}
                  placeholder="8"
                  required
                  className={inputCls}
                  disabled={submitting}
                />
              </div>
            </div>
            <div>
              <label className={labelCls}>
                Napomena <span className="text-gi-text-subtle normal-case font-normal">(nije obavezno)</span>
              </label>
              <input
                type="text"
                value={notes}
                onChange={e => setNotes(e.target.value)}
                placeholder="Opcionalno…"
                className={inputCls}
                disabled={submitting}
              />
            </div>

            {submitError && (
              <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-3 py-2.5">
                <p className="text-red-300 text-sm">{submitError}</p>
              </div>
            )}
            {submitSuccess && (
              <div className="bg-emerald-900/30 border border-emerald-700/60 rounded-lg px-3 py-2.5">
                <p className="text-emerald-300 text-sm">{submitSuccess}</p>
              </div>
            )}

            <button
              type="submit"
              disabled={submitting || !hours || !formDate}
              className="px-5 py-2.5 bg-gi-yellow hover:bg-gi-yellow-hover disabled:opacity-50 text-gi-navy text-sm font-semibold rounded-lg transition-colors"
            >
              {submitting ? 'Zapisivanje…' : 'Upiši sate'}
            </button>
          </form>
        </section>

        {/* History / list */}
        <section>
          <div className="flex items-center justify-between gap-3 mb-4">
            <div>
              <h2 className="text-sm font-semibold text-gi-text">
                {isDirector ? 'Radni sati' : 'Moji sati'}
              </h2>
              {!loadingEntries && filteredEntries.length > 0 && (
                <p className="text-xs text-gi-text-subtle mt-0.5">
                  {filteredEntries.length} unosa · <span className="text-gi-yellow font-medium">{totalHours.toFixed(1)} h</span>
                </p>
              )}
            </div>
            <button
              onClick={fetchEntries}
              disabled={loadingEntries}
              className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50"
            >
              Osvježi
            </button>
          </div>

          {/* Director filter tabs */}
          {isDirector && (
            <div className="flex gap-1.5 mb-4 flex-wrap">
              {tabs.map(tab => (
                <button
                  key={tab.key}
                  onClick={() => setFilterTab(tab.key)}
                  className={[
                    'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
                    filterTab === tab.key
                      ? 'bg-gi-yellow text-gi-navy'
                      : 'bg-gi-card border border-gi-border text-gi-text-muted hover:text-gi-text',
                  ].join(' ')}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          )}

          {fetchError && (
            <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3 mb-4">
              <p className="text-red-300 text-sm">{fetchError}</p>
            </div>
          )}

          {loadingEntries ? (
            <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center">
              <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
            </div>
          ) : filteredEntries.length === 0 ? (
            <div className="bg-gi-card border border-gi-border rounded-xl p-6 text-center">
              <p className="text-gi-text-subtle text-sm">Nema upisanih sati.</p>
            </div>
          ) : (
            <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden divide-y divide-gi-border/60">
              {filteredEntries.map(entry => {
                const isOwn = employee ? entry.employee_id === employee.id : false
                const isConfirming = deletingId === entry.id

                return (
                  <div key={entry.id}>
                    <div className="px-4 py-3 flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-medium text-gi-text capitalize">
                            {formatHrDate(entry.work_date)}
                          </span>
                          <span className="text-sm font-semibold text-gi-yellow">
                            {entry.hours_worked} h
                          </span>
                          {isDirector && (
                            <span className="text-xs text-gi-text-subtle">
                              {entry.employee_name} · {ROLE_LABELS[entry.employee_role] ?? entry.employee_role}
                            </span>
                          )}
                        </div>
                        {entry.notes && (
                          <p className="text-xs text-gi-text-muted mt-1 truncate">{entry.notes}</p>
                        )}
                        {entry.project_name && (
                          <p className="text-xs text-gi-text-subtle mt-0.5">Projekt: {entry.project_name}</p>
                        )}
                      </div>
                      {isOwn && !isConfirming && (
                        <button
                          onClick={() => { setDeleteError(null); setDeletingId(entry.id) }}
                          className="shrink-0 text-xs text-gi-text-subtle hover:text-red-400 border border-transparent hover:border-red-700/60 px-2 py-1 rounded transition-colors"
                          aria-label="Obriši unos"
                        >
                          Obriši
                        </button>
                      )}
                    </div>

                    {/* Inline delete confirmation */}
                    {isConfirming && (
                      <div className="px-4 pb-3 pt-0 border-t border-red-900/30 bg-red-900/10 space-y-2">
                        <p className="text-xs text-red-300 mt-2">
                          Obrisati {entry.hours_worked} h za {formatHrDate(entry.work_date)}?
                        </p>
                        {deleteError && (
                          <p className="text-xs text-red-400">{deleteError}</p>
                        )}
                        <div className="flex gap-2">
                          <button
                            disabled={deleting}
                            onClick={() => handleDelete(entry.id)}
                            className="px-3 py-1.5 bg-red-700 hover:bg-red-600 disabled:opacity-50 text-white text-xs font-medium rounded-lg transition-colors"
                          >
                            {deleting ? 'Brisanje…' : 'Potvrdi brisanje'}
                          </button>
                          <button
                            disabled={deleting}
                            onClick={() => { setDeletingId(null); setDeleteError(null) }}
                            className="px-3 py-1.5 bg-gi-card-elevated hover:bg-gi-border text-gi-text-muted text-xs rounded-lg transition-colors"
                          >
                            Odustani
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </section>

      </div>
    </AppShell>
  )
}
