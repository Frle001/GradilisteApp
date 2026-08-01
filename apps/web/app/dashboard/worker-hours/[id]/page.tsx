'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import apiClient from '@/lib/api-client'
import { ROLE_LABELS } from '@/lib/types/employees'

interface Revision {
  id: string
  previous_hours: number
  new_hours: number
  reason: string
  changed_by_name: string | null
  created_at: string
}

interface Comment {
  id: string
  author_name: string | null
  comment_text: string
  created_at: string
}

interface EntryDetail {
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
  created_at: string
  updated_at: string
  revisions: Revision[]
  comments: Comment[]
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('hr-HR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function formatDate(dateStr: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('hr-HR', {
    weekday: 'long', day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

export default function WorkerHoursDetailPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams()
  const entryId = params.id as string

  const [detail, setDetail] = useState<EntryDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)

  // Correction form
  const [showCorrectionForm, setShowCorrectionForm] = useState(false)
  const [newHours, setNewHours] = useState('')
  const [reason, setReason] = useState('')
  const [correcting, setCorrecting] = useState(false)
  const [correctionError, setCorrectionError] = useState<string | null>(null)

  // Comment form
  const [commentText, setCommentText] = useState('')
  const [addingComment, setAddingComment] = useState(false)
  const [commentError, setCommentError] = useState<string | null>(null)

  const fetchDetail = useCallback(() => {
    setLoading(true)
    setFetchError(null)
    apiClient.get(`/worker-hours/manager/${entryId}`)
      .then(res => setDetail(res.data.entry))
      .catch(err => {
        const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
        setFetchError(msg ?? 'Greška pri dohvatu podataka.')
      })
      .finally(() => setLoading(false))
  }, [entryId])

  useEffect(() => {
    if (!user) return
    if (user.role !== 'direktor' && user.role !== 'inzenjer') return
    fetchDetail()
  }, [user, fetchDetail])

  async function handleCorrect(e: React.FormEvent) {
    e.preventDefault()
    setCorrectionError(null)
    const h = parseFloat(newHours)
    if (isNaN(h) || h < 0 || h > 24) {
      setCorrectionError('Unesite broj sati između 0 i 24.')
      return
    }
    if (!reason.trim()) {
      setCorrectionError('Razlog ispravka je obavezan.')
      return
    }
    setCorrecting(true)
    try {
      await apiClient.post(`/worker-hours/manager/${entryId}/correct`, {
        new_hours: h,
        reason: reason.trim(),
      })
      setShowCorrectionForm(false)
      setNewHours('')
      setReason('')
      fetchDetail()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setCorrectionError(msg ?? 'Greška pri ispravku sati.')
    } finally {
      setCorrecting(false)
    }
  }

  async function handleAddComment(e: React.FormEvent) {
    e.preventDefault()
    setCommentError(null)
    if (!commentText.trim()) {
      setCommentError('Komentar ne smije biti prazan.')
      return
    }
    setAddingComment(true)
    try {
      await apiClient.post(`/worker-hours/manager/${entryId}/comments`, {
        comment_text: commentText.trim(),
      })
      setCommentText('')
      fetchDetail()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setCommentError(msg ?? 'Greška pri dodavanju komentara.')
    } finally {
      setAddingComment(false)
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null
  if (user.role !== 'direktor' && user.role !== 'inzenjer') {
    return (
      <DashboardShell user={user} employee={employee} title="Detalji sati" backHref="/dashboard/worker-hours" onLogout={logout}>
        <p className="text-red-400">Nemate pristup ovoj stranici.</p>
      </DashboardShell>
    )
  }

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Detalji sati"
      backHref="/dashboard/worker-hours"
      onLogout={logout}
    >
      {loading ? (
        <p className="text-slate-500 text-sm">Učitavanje...</p>
      ) : fetchError ? (
        <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{fetchError}</p>
        </div>
      ) : detail && (
        <div className="space-y-6">
          {/* Entry info */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
            <div className="flex items-start justify-between gap-4 mb-4">
              <div>
                <h2 className="text-lg font-bold text-white">{detail.worker_name}</h2>
                <p className="text-sm text-slate-400">{ROLE_LABELS[detail.worker_role] ?? detail.worker_role}</p>
              </div>
              <div className="text-right shrink-0">
                <span className="text-3xl font-bold text-white">{detail.hours_worked}</span>
                <span className="text-slate-400 ml-1">h</span>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wide mb-0.5">Projekt</p>
                <p className="text-white font-medium">{detail.project_name}</p>
              </div>
              <div>
                <p className="text-xs text-slate-500 uppercase tracking-wide mb-0.5">Datum</p>
                <p className="text-white font-medium capitalize">{formatDate(detail.work_date)}</p>
              </div>
              {detail.notes && (
                <div className="col-span-2">
                  <p className="text-xs text-slate-500 uppercase tracking-wide mb-0.5">Napomena</p>
                  <p className="text-slate-300">{detail.notes}</p>
                </div>
              )}
              {detail.work_description && (
                <div className="col-span-2">
                  <p className="text-xs text-slate-500 uppercase tracking-wide mb-0.5">Opis rada</p>
                  <p className="text-slate-300">{detail.work_description}</p>
                </div>
              )}
            </div>

            <p className="text-xs text-slate-600 mt-4">
              Upisano: {formatDateTime(detail.created_at)}
              {detail.updated_at !== detail.created_at && ` · Ažurirano: ${formatDateTime(detail.updated_at)}`}
            </p>

            {/* Correction button */}
            {!showCorrectionForm && (
              <button
                onClick={() => {
                  setShowCorrectionForm(true)
                  setNewHours(String(detail.hours_worked))
                  setCorrectionError(null)
                }}
                className="mt-4 text-sm text-amber-400 hover:text-amber-300 border border-amber-700 hover:border-amber-500 px-3 py-1.5 rounded-lg transition"
              >
                Ispravi sate
              </button>
            )}

            {/* Correction form */}
            {showCorrectionForm && (
              <form onSubmit={handleCorrect} className="mt-4 border-t border-slate-800 pt-4 space-y-3">
                <h3 className="text-sm font-semibold text-slate-300">Ispravak sati</h3>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">Ispravljeni broj sati</label>
                  <input
                    type="number"
                    min="0"
                    max="24"
                    step="0.5"
                    value={newHours}
                    onChange={e => setNewHours(e.target.value)}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-amber-500"
                    disabled={correcting}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-400 mb-1">
                    Razlog izmjene <span className="text-red-400">*</span>
                  </label>
                  <textarea
                    value={reason}
                    onChange={e => setReason(e.target.value)}
                    rows={2}
                    placeholder="Npr. Greška u unosu, prekoračenje dnevne norme..."
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-amber-500 resize-none"
                    disabled={correcting}
                  />
                </div>
                {correctionError && (
                  <p className="text-sm text-red-400">{correctionError}</p>
                )}
                <div className="flex gap-2">
                  <button
                    type="submit"
                    disabled={correcting}
                    className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white text-sm font-medium rounded-lg transition disabled:opacity-50"
                  >
                    {correcting ? 'Spremanje...' : 'Spremi ispravak'}
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowCorrectionForm(false)}
                    className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm rounded-lg transition"
                  >
                    Odustani
                  </button>
                </div>
              </form>
            )}
          </div>

          {/* Revision history */}
          {detail.revisions.length > 0 && (
            <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
              <h2 className="text-sm font-semibold text-slate-300 mb-4">Povijest ispravaka</h2>
              <ul className="space-y-3">
                {detail.revisions.map(rev => (
                  <li key={rev.id} className="border-l-2 border-amber-700 pl-3">
                    <p className="text-sm text-white">
                      <span className="line-through text-slate-500">{rev.previous_hours} h</span>
                      {' → '}
                      <span className="font-semibold">{rev.new_hours} h</span>
                    </p>
                    <p className="text-xs text-slate-400 mt-0.5">{rev.reason}</p>
                    <p className="text-xs text-slate-600 mt-0.5">
                      {rev.changed_by_name ?? 'Nepoznat'} · {formatDateTime(rev.created_at)}
                    </p>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Internal comments */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
            <h2 className="text-sm font-semibold text-slate-300 mb-4">
              Interni komentari
              <span className="ml-2 text-slate-600 font-normal normal-case text-xs">
                (vidljivo samo direktorima i inženjerima)
              </span>
            </h2>

            {detail.comments.length === 0 ? (
              <p className="text-slate-500 text-sm mb-4">Nema komentara.</p>
            ) : (
              <ul className="space-y-3 mb-4">
                {detail.comments.map(c => (
                  <li key={c.id} className="bg-slate-800/60 border border-slate-700/60 rounded-lg px-3 py-2.5">
                    <p className="text-sm text-slate-200">{c.comment_text}</p>
                    <p className="text-xs text-slate-500 mt-1">
                      {c.author_name ?? 'Nepoznat'} · {formatDateTime(c.created_at)}
                    </p>
                  </li>
                ))}
              </ul>
            )}

            {/* Add comment form */}
            <form onSubmit={handleAddComment} className="space-y-2">
              <textarea
                value={commentText}
                onChange={e => setCommentText(e.target.value)}
                rows={2}
                placeholder="Dodaj interni komentar..."
                className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 resize-none"
                disabled={addingComment}
              />
              {commentError && (
                <p className="text-sm text-red-400">{commentError}</p>
              )}
              <button
                type="submit"
                disabled={addingComment || !commentText.trim()}
                className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-200 text-sm font-medium rounded-lg transition disabled:opacity-50"
              >
                {addingComment ? 'Dodavanje...' : 'Dodaj komentar'}
              </button>
            </form>
          </div>
        </div>
      )}
    </DashboardShell>
  )
}
