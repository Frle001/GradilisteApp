'use client'

import { useEffect, useState, useCallback } from 'react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import apiClient from '@/lib/api-client'
import { enqueue, getEntry, findPendingEntry } from '@/lib/offline/outbox'
import { trySyncEntry } from '@/lib/offline/sync-engine'
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

// Parse YYYY-MM-DD as local date to avoid UTC-midnight timezone shift.
function formatHistoryDate(dateStr: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('hr-HR', {
    weekday: 'short', day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

export default function MyHoursPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const today = localTodayStr()

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
    // Reuse the existing outbox ID for this project/date if one is pending,
    // so every retry sends the same client_submission_id to the server.
    // db.put with the same id updates the payload in place.
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
      await enqueue({ id: submissionId, type: 'worker-hours', payload, ownerId: user?.id, companyId: user?.company_id })
    } catch {
      useDirectFallback = true
    }

    if (useDirectFallback) {
      // IDB unavailable — fall back to a direct network call.
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

    // Attempt immediate sync; entry stays pending and retries automatically if offline.
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

  // Hours on other projects today (excluding the currently selected one)
  const otherProjectsTotal = entries
    .filter(e => e.project_id !== selectedProject)
    .reduce((sum, e) => sum + e.hours_worked, 0)
  const projectedTotal = otherProjectsTotal + (parseFloat(hours) || 0)
  const wouldExceed24 = projectedTotal > 24

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Moji sati"
      backHref="/dashboard"
      onLogout={logout}
    >
      {/* Date header */}
      <div className="mb-6">
        <h1 className="text-xl font-bold text-white">Moji radni sati</h1>
        <p className="text-slate-400 text-sm mt-1 capitalize">{formatHrDate(today)}</p>
      </div>

      {fetchError && (
        <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3 mb-6">
          <p className="text-red-300 text-sm">{fetchError}</p>
        </div>
      )}

      {/* Entry form */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 mb-6">
        <h2 className="text-sm font-semibold text-slate-300 mb-4">Upiši sate za danas</h2>

        {loadingProjects ? (
          <p className="text-slate-500 text-sm">Učitavanje projekata...</p>
        ) : projects.length === 0 ? (
          <p className="text-slate-500 text-sm">
            Nema dostupnih aktivnih projekata za unos sati.
          </p>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Project select */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1.5">
                Projekt <span className="text-red-400">*</span>
              </label>
              <select
                value={selectedProject}
                onChange={e => setSelectedProject(e.target.value)}
                className="w-full px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
                disabled={submitting}
              >
                <option value="">— Odaberi projekt —</option>
                {projects.map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>

            {/* Hours */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1.5">
                Broj sati <span className="text-red-400">*</span>
              </label>
              <input
                type="number"
                min="0"
                max="24"
                step="0.5"
                value={hours}
                onChange={e => setHours(e.target.value)}
                placeholder="npr. 8"
                className="w-full px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
                disabled={submitting}
              />
            </div>

            {/* Notes */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1.5">
                Napomena <span className="text-slate-500">(nije obavezno)</span>
              </label>
              <textarea
                value={notes}
                onChange={e => setNotes(e.target.value)}
                rows={2}
                placeholder="Kratka napomena o radu..."
                className="w-full px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 resize-none"
                disabled={submitting}
              />
            </div>

            {/* Opis rada — only visible to poslovoda */}
            {user?.role === 'poslovoda' && (
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1.5">
                  Opis rada <span className="text-slate-500">(nije obavezno)</span>
                </label>
                <textarea
                  value={workDescription}
                  onChange={e => setWorkDescription(e.target.value)}
                  rows={3}
                  placeholder="Opis obavljenog rada na projektu..."
                  className="w-full px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 resize-none"
                  disabled={submitting}
                />
              </div>
            )}

            {isUpdate && selectedProject && (
              <p className="text-xs text-amber-400">
                Već ste upisali sate za ovaj projekt danas — potvrdom ćete ažurirati postojeći unos.
              </p>
            )}

            {wouldExceed24 && selectedProject && hours && (
              <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
                <p className="text-red-300 text-sm">
                  Ukupni sati za danas bi premašili 24 sata (već upisano na ostalim projektima: {otherProjectsTotal.toFixed(1)} h).
                </p>
              </div>
            )}

            {submitError && (
              <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
                <p className="text-red-300 text-sm">{submitError}</p>
              </div>
            )}

            {submitSuccess && (
              <div className="bg-green-950 border border-green-800 rounded-lg px-4 py-3">
                <p className="text-green-300 text-sm">{submitSuccess}</p>
              </div>
            )}

            <button
              type="submit"
              disabled={submitting || !selectedProject || !hours || wouldExceed24}
              className="w-full py-2.5 px-4 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed transition"
            >
              {submitting ? 'Upisivanje...' : isUpdate ? 'Ažuriraj sate' : 'Upiši sate'}
            </button>
          </form>
        )}
      </div>

      {/* Today's submitted entries */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-slate-300">Upisani sati za danas</h2>
          {entries.length > 0 && (
            <span className="text-xs text-slate-400">
              Ukupno: <span className="text-white font-semibold">{totalToday.toFixed(1)} h</span>
            </span>
          )}
        </div>

        {loadingEntries ? (
          <p className="text-slate-500 text-sm">Učitavanje...</p>
        ) : entries.length === 0 ? (
          <p className="text-slate-500 text-sm">Nema upisanih sati za danas.</p>
        ) : (
          <ul className="space-y-3">
            {entries.map(entry => (
              <li
                key={entry.id}
                className="flex items-start justify-between gap-4 py-3 border-b border-slate-800 last:border-0"
              >
                <div className="min-w-0">
                  <p className="text-sm font-medium text-white truncate">{entry.project_name}</p>
                  {entry.notes && (
                    <p className="text-xs text-slate-400 mt-0.5 truncate">{entry.notes}</p>
                  )}
                  {user?.role === 'poslovoda' && entry.work_description && (
                    <p className="text-xs text-slate-500 mt-0.5 truncate italic">{entry.work_description}</p>
                  )}
                </div>
                <div className="shrink-0 text-right">
                  <span className="text-lg font-bold text-white">{entry.hours_worked}</span>
                  <span className="text-slate-400 text-sm ml-0.5">h</span>
                  <button
                    onClick={() => {
                      setSelectedProject(entry.project_id)
                      setSubmitSuccess(null)
                      setSubmitError(null)
                      window.scrollTo({ top: 0, behavior: 'smooth' })
                    }}
                    className="block text-xs text-slate-500 hover:text-slate-300 mt-0.5 transition"
                  >
                    Izmijeni
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
      {/* Previous entries — read-only */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 mt-6">
        <h2 className="text-sm font-semibold text-slate-300 mb-4">Prethodni unosi</h2>

        {loadingHistory ? (
          <p className="text-slate-500 text-sm">Učitavanje...</p>
        ) : historyError ? (
          <p className="text-red-400 text-sm">{historyError}</p>
        ) : history.length === 0 ? (
          <p className="text-slate-500 text-sm">Još nema prethodnih unosa.</p>
        ) : (
          <ul className="space-y-3">
            {history.map(entry => (
              <li
                key={entry.id}
                className="bg-slate-800/50 border border-slate-700/60 rounded-lg px-4 py-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-xs text-slate-500 mb-0.5 capitalize">
                      {formatHistoryDate(entry.work_date)}
                    </p>
                    <p className="text-sm font-medium text-white truncate">{entry.project_name}</p>
                    {entry.notes && (
                      <p className="text-xs text-slate-400 mt-1">{entry.notes}</p>
                    )}
                    {user?.role === 'poslovoda' && entry.work_description && (
                      <p className="text-xs text-slate-500 mt-0.5 italic">{entry.work_description}</p>
                    )}
                  </div>
                  <div className="shrink-0 text-right">
                    <span className="text-lg font-bold text-white">{entry.hours_worked}</span>
                    <span className="text-slate-400 text-sm ml-0.5">h</span>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </DashboardShell>
  )
}
