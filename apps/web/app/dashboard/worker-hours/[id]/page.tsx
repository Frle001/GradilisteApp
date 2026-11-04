'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
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

const inputCls =
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2.5 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'

export default function WorkerHoursDetailPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams()
  const router = useRouter()
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

  // Delete confirmation
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)

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

  async function handleDelete() {
    setDeleteError(null)
    setDeleting(true)
    try {
      await apiClient.delete(`/worker-hours/manager/${entryId}`)
      router.push('/dashboard/worker-hours')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      setDeleteError(msg ?? 'Greška pri brisanju unosa.')
      setDeleting(false)
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
      <AppShell user={user} employee={employee} title="Detalji sati" backHref="/dashboard/worker-hours" onLogout={logout}>
        <p className="text-red-400 text-sm">Nemate pristup ovoj stranici.</p>
      </AppShell>
    )
  }

  return (
    <AppShell user={user} employee={employee} title="Detalji sati" backHref="/dashboard/worker-hours" onLogout={logout}>
      <div className="max-w-2xl mx-auto space-y-4">

        {loading ? (
          <div className="bg-gi-card border border-gi-border rounded-xl p-8 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
          </div>
        ) : fetchError ? (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{fetchError}</p>
          </div>
        ) : detail && (
          <>
            {/* Entry info card */}
            <div className="bg-gi-card border border-gi-border rounded-xl p-5">
              {/* Worker header */}
              <div className="flex items-start justify-between gap-4 mb-4">
                <div>
                  <h1 className="text-lg font-bold text-gi-text">{detail.worker_name}</h1>
                  <p className="text-sm text-gi-text-muted">{ROLE_LABELS[detail.worker_role] ?? detail.worker_role}</p>
                </div>
                <div className="shrink-0 text-right">
                  <span className="text-3xl font-bold text-gi-yellow">{detail.hours_worked}</span>
                  <span className="text-gi-text-muted ml-1 text-lg">h</span>
                </div>
              </div>

              {/* Metadata grid */}
              <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm border-t border-gi-border pt-4">
                <div>
                  <dt className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Projekt</dt>
                  <dd className="text-gi-text font-medium">{detail.project_name}</dd>
                </div>
                <div>
                  <dt className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Datum</dt>
                  <dd className="text-gi-text font-medium capitalize">{formatDate(detail.work_date)}</dd>
                </div>
                {detail.notes && (
                  <div className="col-span-2">
                    <dt className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Napomena</dt>
                    <dd className="text-gi-text-muted">{detail.notes}</dd>
                  </div>
                )}
                {detail.work_description && (
                  <div className="col-span-2">
                    <dt className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Opis rada</dt>
                    <dd className="text-gi-text-muted">{detail.work_description}</dd>
                  </div>
                )}
              </dl>

              <p className="text-xs text-gi-text-subtle mt-4">
                Upisano: {formatDateTime(detail.created_at)}
                {detail.updated_at !== detail.created_at && ` · Ažurirano: ${formatDateTime(detail.updated_at)}`}
              </p>

              {/* Action buttons */}
              {!showCorrectionForm && !showDeleteConfirm && (
                <div className="flex gap-2 mt-4 pt-4 border-t border-gi-border">
                  <button
                    onClick={() => {
                      setShowCorrectionForm(true)
                      setNewHours(String(detail.hours_worked))
                      setCorrectionError(null)
                    }}
                    className="text-sm text-gi-text-muted hover:text-gi-yellow border border-gi-border hover:border-gi-yellow px-3 py-1.5 rounded-lg transition-colors"
                  >
                    Ispravi sate
                  </button>
                  <button
                    type="button"
                    onClick={() => { setDeleteError(null); setShowDeleteConfirm(true) }}
                    className="text-sm text-gi-text-subtle hover:text-red-400 border border-gi-border hover:border-red-700/60 px-3 py-1.5 rounded-lg transition-colors"
                  >
                    Obriši
                  </button>
                </div>
              )}

              {/* Correction form */}
              {showCorrectionForm && (
                <form onSubmit={handleCorrect} className="mt-4 border-t border-gi-border pt-4 space-y-3">
                  <h3 className="text-sm font-semibold text-gi-text">Ispravak sati</h3>
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
                      Ispravljeni broj sati
                    </label>
                    <input
                      type="number"
                      min="0"
                      max="24"
                      step="0.5"
                      inputMode="decimal"
                      value={newHours}
                      onChange={e => setNewHours(e.target.value)}
                      className={inputCls}
                      disabled={correcting}
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
                      Razlog izmjene <span className="text-red-400 normal-case font-normal">*</span>
                    </label>
                    <textarea
                      value={reason}
                      onChange={e => setReason(e.target.value)}
                      rows={2}
                      placeholder="Npr. Greška u unosu, prekoračenje dnevne norme…"
                      className={`${inputCls} resize-none`}
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
                      className="px-4 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                    >
                      {correcting ? 'Spremanje…' : 'Spremi ispravak'}
                    </button>
                    <button
                      type="button"
                      onClick={() => setShowCorrectionForm(false)}
                      className="px-4 py-2 bg-gi-card-elevated hover:bg-gi-border text-gi-text-muted text-sm rounded-lg transition-colors"
                    >
                      Odustani
                    </button>
                  </div>
                </form>
              )}

              {/* Delete confirmation */}
              {showDeleteConfirm && (
                <div className="mt-4 border-t border-red-900/40 pt-4 space-y-3">
                  <h3 className="text-sm font-semibold text-red-300">Obrisati evidenciju sati?</h3>
                  <p className="text-sm text-gi-text-muted">
                    Ovaj unos više se neće uračunavati u evidenciju radnih sati. Radnja će biti zabilježena.
                  </p>
                  {deleteError && (
                    <p className="text-sm text-red-400">{deleteError}</p>
                  )}
                  <div className="flex gap-2">
                    <button
                      type="button"
                      disabled={deleting}
                      onClick={handleDelete}
                      className="px-4 py-2 bg-red-700 hover:bg-red-600 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
                    >
                      {deleting ? 'Brisanje…' : 'Obriši evidenciju'}
                    </button>
                    <button
                      type="button"
                      disabled={deleting}
                      onClick={() => setShowDeleteConfirm(false)}
                      className="px-4 py-2 bg-gi-card-elevated hover:bg-gi-border text-gi-text-muted text-sm rounded-lg transition-colors"
                    >
                      Odustani
                    </button>
                  </div>
                </div>
              )}
            </div>

            {/* Revision history */}
            {detail.revisions.length > 0 && (
              <div className="bg-gi-card border border-gi-border rounded-xl p-5">
                <h2 className="text-sm font-semibold text-gi-text mb-4">Povijest ispravaka</h2>
                <ul className="space-y-3">
                  {detail.revisions.map(rev => (
                    <li key={rev.id} className="border-l-2 border-gi-yellow/60 pl-3">
                      <p className="text-sm text-gi-text">
                        <span className="line-through text-gi-text-subtle">{rev.previous_hours} h</span>
                        <span className="text-gi-text-muted mx-1">→</span>
                        <span className="font-semibold text-gi-yellow">{rev.new_hours} h</span>
                      </p>
                      <p className="text-xs text-gi-text-muted mt-0.5">{rev.reason}</p>
                      <p className="text-xs text-gi-text-subtle mt-0.5">
                        {rev.changed_by_name ?? 'Nepoznat'} · {formatDateTime(rev.created_at)}
                      </p>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {/* Internal comments */}
            <div className="bg-gi-card border border-gi-border rounded-xl p-5">
              <h2 className="text-sm font-semibold text-gi-text mb-1">Interni komentari</h2>
              <p className="text-xs text-gi-text-subtle mb-4">Vidljivo samo direktorima i inženjerima</p>

              {detail.comments.length === 0 ? (
                <p className="text-gi-text-subtle text-sm mb-4">Nema komentara.</p>
              ) : (
                <ul className="space-y-3 mb-4">
                  {detail.comments.map(c => (
                    <li key={c.id} className="bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2.5">
                      <p className="text-sm text-gi-text">{c.comment_text}</p>
                      <p className="text-xs text-gi-text-subtle mt-1">
                        {c.author_name ?? 'Nepoznat'} · {formatDateTime(c.created_at)}
                      </p>
                    </li>
                  ))}
                </ul>
              )}

              <form onSubmit={handleAddComment} className="space-y-2">
                <textarea
                  value={commentText}
                  onChange={e => setCommentText(e.target.value)}
                  rows={2}
                  placeholder="Dodaj interni komentar…"
                  className={`${inputCls} resize-none`}
                  disabled={addingComment}
                />
                {commentError && (
                  <p className="text-sm text-red-400">{commentError}</p>
                )}
                <button
                  type="submit"
                  disabled={addingComment || !commentText.trim()}
                  className="px-4 py-2 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted hover:text-gi-text text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                >
                  {addingComment ? 'Dodavanje…' : 'Dodaj komentar'}
                </button>
              </form>
            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}
