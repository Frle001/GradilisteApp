'use client'

import { useEffect, useState, useCallback, useRef } from 'react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import apiClient from '@/lib/api-client'
import { enqueue, getEntry, findPendingEntry } from '@/lib/offline/outbox'
import { trySyncEntry } from '@/lib/offline/sync-engine'
import { onRefresh } from '@/lib/refresh-events'
import type { WorkerHoursPayload } from '@/lib/offline/types'

interface WorkerProject {
  id: string
  name: string
}

interface HoursEntry {
  id: string
  project_id: string
  project_name: string
  work_date: string
  hours_worked: number
  notes: string | null
  work_description: string | null
  updated_at: string
}

function localTodayStr(): string {
  return new Date().toLocaleDateString('sv') // YYYY-MM-DD
}

function formatHrDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('hr-HR', {
    weekday: 'long', day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

function formatHistoryDate(dateStr: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('hr-HR', {
    weekday: 'short', day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

const inputCls =
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2.5 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'
const labelCls = 'block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5'

export default function MyHoursPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const today = localTodayStr()
  const formRef = useRef<HTMLElement>(null)

  const [projects, setProjects] = useState<WorkerProject[]>([])
  const [entries, setEntries] = useState<HoursEntry[]>([])
  const [history, setHistory] = useState<HoursEntry[]>([])
  const [loadingProjects, setLoadingProjects] = useState(true)
  const [loadingEntries, setLoadingEntries] = useState(true)
  const [loadingHistory, setLoadingHistory] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)
  const [historyError, setHistoryError] = useState<string | null>(null)

  // Form state
  const [selectedProject, setSelectedProject] = useState('')
  const [hours, setHours] = useState('')
  const [notes, setNotes] = useState('')
  const [workDescription, setWorkDescription] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [submitSuccess, setSubmitSuccess] = useState<string | null>(null)

  const fetchProjects = useCallback(() => {
    setLoadingProjects(true)
    apiClient.get('/worker-hours/projects')
      .then(res => setProjects(res.data.projects ?? []))
      .catch(() => setFetchError('Greška pri dohvatu projekata.'))
      .finally(() => setLoadingProjects(false))
  }, [])

  const fetchEntries = useCallback(() => {
    setLoadingEntries(true)
    apiClient.get(`/worker-hours/my?date=${today}`)
      .then(res => setEntries(res.data.entries ?? []))
      .catch(() => {})
      .finally(() => setLoadingEntries(false))
  }, [today])

  const fetchHistory = useCallback(() => {
    setLoadingHistory(true)
    setHistoryError(null)
    apiClient.get('/worker-hours/history')
      .then(res => setHistory(res.data.entries ?? []))
      .catch(() => setHistoryError('Greška pri dohvatu prethodnih unosa.'))
      .finally(() => setLoadingHistory(false))
  }, [])

  useEffect(() => {
    if (!user) return
    fetchProjects()
    fetchEntries()
    fetchHistory()
  }, [user, fetchProjects, fetchEntries, fetchHistory])

  // Refresh when sync completes
  useEffect(() => {
    return onRefresh((domains) => {
      if (domains.includes('worker-hours')) {
        fetchEntries()
        fetchHistory()
      }
    })
  }, [fetchEntries, fetchHistory])

  // Pre-fill form when editing an existing entry
  useEffect(() => {
    if (!selectedProject) return
    const existing = entries.find(e => e.project_id === selectedProject)
    if (existing) {
      setHours(String(existing.hours_worked))
      setNotes(existing.notes ?? '')
      setWorkDescription(existing.work_description ?? '')
    } else {
      setHours('')
      setNotes('')
      setWorkDescription('')
    }
    setSubmitError(null)
    setSubmitSuccess(null)
  }, [selectedProject, entries])

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      const { runSync } = await import('@/lib/offline/sync-engine')
      await runSync({ userId: user?.id, companyId: user?.company_id })
    } catch { /* offline — just refetch */ }
    await Promise.all([fetchEntries(), fetchHistory()])
    setRefreshing(false)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitError(null)
    setSubmitSuccess(null)

    const h = parseFloat(hours)
    if (isNaN(h) || h < 0 || h > 24) {
      setSubmitError('Unesite broj sati između 0 i 24.')
      return
    }
    if (!selectedProject) {
      setSubmitError('Odaberite projekt.')
      return
    }

    setSubmitting(true)
    const payload: WorkerHoursPayload = {
      project_id: selectedProject,
      work_date: today,
      hours_worked: h,
      notes: notes.trim() || null,
      work_description: user?.role === 'poslovoda' ? (workDescription.trim() || null) : null,
    }
    let submissionId = crypto.randomUUID()
    let useDirectFallback = false
    try {
      const existingEntry = await findPendingEntry('worker-hours', e => {
        const p = e.payload as WorkerHoursPayload
        return p.project_id === payload.project_id && p.work_date === payload.work_date
      })
      if (existingEntry) {
        submissionId = existingEntry.id
      }
      await enqueue({
        id: submissionId,
        type: 'worker-hours',
        payload,
        ownerId: user?.id,
        companyId: user?.company_id,
      })
    } catch {
      useDirectFallback = true
    }

    if (useDirectFallback) {
      try {
        await apiClient.post('/worker-hours', payload)
        setSubmitSuccess('Sati su uspješno upisani.')
        fetchEntries()
      } catch (err: unknown) {
        const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
        setSubmitError(msg ?? 'Greška pri unosu sati.')
      }
      setSubmitting(false)
      return
    }

    await trySyncEntry(submissionId)
    const entry = await getEntry(submissionId)

    if (entry?.status === 'synced') {
      setSubmitSuccess('Sati su uspješno upisani.')
      fetchEntries()
    } else if (entry?.status === 'failed') {
      setSubmitError(entry.lastError ?? 'Greška pri unosu sati.')
    } else {
      setSubmitSuccess('Sati su lokalno snimljeni i bit će poslani kada se povežete.')
    }
    setSubmitting(false)
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const existingForSelected = entries.find(e => e.project_id === selectedProject)
  const isUpdate = !!existingForSelected
  const totalToday = entries.reduce((sum, e) => sum + e.hours_worked, 0)

  const otherProjectsTotal = entries
    .filter(e => e.project_id !== selectedProject)
    .reduce((sum, e) => sum + e.hours_worked, 0)
  const projectedTotal = otherProjectsTotal + (parseFloat(hours) || 0)
  const wouldExceed24 = projectedTotal > 24

  // Group history by date, most recent first
  const historyByDate = history.reduce<Record<string, HoursEntry[]>>((acc, e) => {
    if (!acc[e.work_date]) acc[e.work_date] = []
    acc[e.work_date].push(e)
    return acc
  }, {})
  const historyDates = Object.keys(historyByDate).sort((a, b) => b.localeCompare(a))

  // Weekly total from history (last 7 days)
  const sevenDaysAgo = new Date()
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6)
  const weekTotal = history.reduce((sum, e) => {
    const [y, m, d] = e.work_date.split('-').map(Number)
    const date = new Date(y, m - 1, d)
    const cutoff = new Date(sevenDaysAgo.getFullYear(), sevenDaysAgo.getMonth(), sevenDaysAgo.getDate())
    return date >= cutoff ? sum + e.hours_worked : sum
  }, 0)

  return (
    <AppShell user={user} employee={employee} title="Moji sati" backHref="/dashboard" onLogout={logout}>
      <div className="max-w-2xl mx-auto space-y-6">

        {/* Page header */}
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-gi-text">Moji radni sati</h1>
            <p className="text-gi-text-subtle text-sm mt-0.5 capitalize">{formatHrDate(today)}</p>
          </div>
          <button
            onClick={() => void handleRefresh()}
            disabled={refreshing || submitting}
            className="shrink-0 text-sm text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-2 rounded-lg transition-colors disabled:opacity-50"
          >
            {refreshing ? 'Osvježavanje…' : 'Osvježi'}
          </button>
        </div>

        {fetchError && (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{fetchError}</p>
          </div>
        )}

        {/* Entry form */}
        <section ref={formRef} className="bg-gi-card border border-gi-border rounded-xl p-5">
          <h2 className="text-sm font-semibold text-gi-text mb-4">
            Upiši sate za <span className="text-gi-yellow">{today}</span>
          </h2>

          {loadingProjects ? (
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje projekata…</p>
          ) : projects.length === 0 ? (
            <p className="text-gi-text-subtle text-sm">Nema dostupnih aktivnih projekata za unos sati.</p>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className={labelCls}>
                  Projekt <span className="text-red-400 normal-case">*</span>
                </label>
                <select
                  value={selectedProject}
                  onChange={e => setSelectedProject(e.target.value)}
                  className={inputCls}
                  disabled={submitting}
                >
                  <option value="">— Odaberi projekt —</option>
                  {projects.map(p => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className={labelCls}>
                  Broj sati <span className="text-red-400 normal-case">*</span>
                </label>
                <input
                  type="number"
                  min="0"
                  max="24"
                  step="0.5"
                  inputMode="decimal"
                  value={hours}
                  onChange={e => setHours(e.target.value)}
                  placeholder="npr. 8"
                  className={inputCls}
                  disabled={submitting}
                />
              </div>

              {/* Opis radova — always visible; payload only sent for poslovoda (server-enforced) */}
              <div>
                <label className={labelCls}>
                  Opis radova <span className="text-gi-text-subtle normal-case font-normal">(nije obavezno)</span>
                </label>
                <textarea
                  value={workDescription}
                  onChange={e => setWorkDescription(e.target.value)}
                  rows={3}
                  placeholder="Kratki opis izvedenih radova…"
                  className={`${inputCls} resize-none`}
                  disabled={submitting}
                />
              </div>

              <div>
                <label className={labelCls}>
                  Napomena <span className="text-gi-text-subtle normal-case font-normal">(nije obavezno)</span>
                </label>
                <textarea
                  value={notes}
                  onChange={e => setNotes(e.target.value)}
                  rows={2}
                  placeholder="Kratka napomena o radu…"
                  className={`${inputCls} resize-none`}
                  disabled={submitting}
                />
              </div>

              {isUpdate && selectedProject && (
                <div className="flex items-start gap-2 bg-amber-900/20 border border-amber-700/40 rounded-lg px-3 py-2.5">
                  <svg className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M12 3a9 9 0 110 18A9 9 0 0112 3z" />
                  </svg>
                  <p className="text-xs text-amber-300">
                    Već ste upisali sate za ovaj projekt danas — potvrdom ćete ažurirati postojeći unos.
                  </p>
                </div>
              )}

              {wouldExceed24 && selectedProject && hours && (
                <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-3 py-2.5">
                  <p className="text-red-300 text-sm">
                    Ukupni sati za danas bi premašili 24 h (na ostalim projektima: {otherProjectsTotal.toFixed(1)} h).
                  </p>
                </div>
              )}

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
                disabled={submitting || !selectedProject || !hours || wouldExceed24}
                className="w-full py-2.5 px-4 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy font-semibold rounded-lg text-sm disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {submitting ? 'Upisivanje…' : isUpdate ? 'Ažuriraj sate' : 'Upiši sate'}
              </button>
            </form>
          )}
        </section>

        {/* Today's submitted entries */}
        <section className="bg-gi-card border border-gi-border rounded-xl p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-gi-text">Upisani sati za danas</h2>
            {entries.length > 0 && (
              <span className="text-xs text-gi-text-muted">
                Ukupno: <span className="text-gi-yellow font-semibold">{totalToday.toFixed(1)} h</span>
              </span>
            )}
          </div>

          {loadingEntries ? (
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
          ) : entries.length === 0 ? (
            <p className="text-gi-text-subtle text-sm">Nema upisanih sati za danas.</p>
          ) : (
            <ul className="divide-y divide-gi-border">
              {entries.map(entry => (
                <li key={entry.id} className="flex items-start justify-between gap-4 py-3 first:pt-0 last:pb-0">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-gi-text truncate">{entry.project_name}</p>
                    {entry.work_description && (
                      <p className="text-xs text-gi-text-muted mt-0.5 line-clamp-2">{entry.work_description}</p>
                    )}
                    {entry.notes && (
                      <p className="text-xs text-gi-text-subtle mt-0.5 truncate">{entry.notes}</p>
                    )}
                  </div>
                  <div className="shrink-0 text-right">
                    <span className="text-lg font-bold text-gi-text">{entry.hours_worked}</span>
                    <span className="text-gi-text-subtle text-sm ml-0.5">h</span>
                    <button
                      onClick={() => {
                        setSelectedProject(entry.project_id)
                        setSubmitSuccess(null)
                        setSubmitError(null)
                        formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
                      }}
                      className="block text-xs text-gi-text-subtle hover:text-gi-yellow mt-0.5 transition-colors"
                    >
                      Izmijeni
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* History — grouped by date */}
        <section className="bg-gi-card border border-gi-border rounded-xl p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-gi-text">Prethodni unosi</h2>
            {weekTotal > 0 && (
              <span className="text-xs text-gi-text-muted">
                Ovaj tjedan: <span className="text-gi-yellow font-semibold">{weekTotal.toFixed(1)} h</span>
              </span>
            )}
          </div>

          {loadingHistory ? (
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
          ) : historyError ? (
            <p className="text-red-400 text-sm">{historyError}</p>
          ) : history.length === 0 ? (
            <p className="text-gi-text-subtle text-sm">Još nema prethodnih unosa.</p>
          ) : (
            <div className="space-y-5">
              {historyDates.map(date => {
                const dateEntries = historyByDate[date]
                const dateTotal = dateEntries.reduce((s, e) => s + e.hours_worked, 0)
                return (
                  <div key={date}>
                    <div className="flex items-center justify-between mb-2">
                      <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wider capitalize">
                        {formatHistoryDate(date)}
                      </p>
                      <span className="text-xs text-gi-text-subtle">{dateTotal.toFixed(1)} h</span>
                    </div>
                    <ul className="divide-y divide-gi-border/60">
                      {dateEntries.map(entry => (
                        <li key={entry.id} className="flex items-start justify-between gap-3 py-2.5 first:pt-0 last:pb-0">
                          <div className="min-w-0">
                            <p className="text-sm font-medium text-gi-text truncate">{entry.project_name}</p>
                            {entry.work_description && (
                              <p className="text-xs text-gi-text-muted mt-0.5 line-clamp-2">{entry.work_description}</p>
                            )}
                            {entry.notes && (
                              <p className="text-xs text-gi-text-subtle mt-0.5 truncate">{entry.notes}</p>
                            )}
                          </div>
                          <span className="shrink-0 text-sm font-semibold text-gi-text">
                            {entry.hours_worked} <span className="font-normal text-gi-text-subtle">h</span>
                          </span>
                        </li>
                      ))}
                    </ul>
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
