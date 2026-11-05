'use client'

import { useEffect, useState, useCallback } from 'react'
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
  worker_name: string
  worker_role: string
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

function fmt(iso: string): string {
  return new Date(iso).toLocaleString('hr-HR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function fmtDate(dateStr: string): string {
  const [y, m, d] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('hr-HR', {
    weekday: 'long', day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

const inputCls =
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'

interface Props {
  entryId: string
  onClose: () => void
}

export default function WorkerHoursDetailPanel({ entryId, onClose }: Props) {
  const [detail, setDetail] = useState<EntryDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState<string | null>(null)

  const [showCorrectionForm, setShowCorrectionForm] = useState(false)
  const [newHours, setNewHours] = useState('')
  const [reason, setReason] = useState('')
  const [correcting, setCorrecting] = useState(false)
  const [correctionError, setCorrectionError] = useState<string | null>(null)

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

  useEffect(() => { fetchDetail() }, [fetchDetail])

  // Close on Escape
  useEffect(() => {
    function onKey(e: KeyboardEvent) { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

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

  const originalHours = detail?.revisions.length
    ? detail.revisions[0].previous_hours
    : null

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-end" onClick={onClose}>
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60" />

      {/* Panel */}
      <div
        className="relative w-full max-w-lg h-full bg-gi-navy border-l border-gi-border overflow-y-auto shadow-2xl"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="sticky top-0 z-10 bg-gi-workspace border-b border-gi-border px-5 py-3 flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold text-gi-text">Detalji upisanih sati</h2>
            {detail && (
              <p className="text-xs text-gi-text-subtle mt-0.5 truncate">
                {detail.worker_name}
                <span className="mx-1 text-gi-border">·</span>
                <span className="text-gi-yellow font-semibold">{detail.hours_worked} h</span>
                <span className="mx-1 text-gi-border">·</span>
                {fmtDate(detail.work_date)}
              </p>
            )}
          </div>
          <button
            onClick={onClose}
            className="text-gi-text-muted hover:text-gi-text p-1 rounded transition-colors shrink-0 mt-0.5"
            aria-label="Zatvori"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-5 space-y-6">
          {loading ? (
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
          ) : fetchError ? (
            <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
              <p className="text-red-300 text-sm">{fetchError}</p>
            </div>
          ) : detail && (
            <>
              {/* Main info */}
              <div className="space-y-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-lg font-bold text-gi-text">{detail.worker_name}</p>
                    <p className="text-sm text-gi-text-muted">{ROLE_LABELS[detail.worker_role] ?? detail.worker_role}</p>
                  </div>
                  <div className="text-right shrink-0">
                    <span className="text-3xl font-bold text-gi-yellow">{detail.hours_worked}</span>
                    <span className="text-gi-text-muted ml-1 text-sm">h</span>
                    {originalHours !== null && originalHours !== detail.hours_worked && (
                      <p className="text-xs text-amber-400 mt-0.5">
                        Originalno: <span className="line-through">{originalHours} h</span>
                      </p>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Projekt</p>
                    <p className="text-gi-text">{detail.project_name}</p>
                  </div>
                  <div>
                    <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-0.5">Datum</p>
                    <p className="text-gi-text capitalize">{fmtDate(detail.work_date)}</p>
                  </div>

                  {detail.notes && (
                    <div className="col-span-2">
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-1">Napomena</p>
                      <p className="text-gi-text-muted whitespace-pre-wrap break-words">{detail.notes}</p>
                    </div>
                  )}

                  {detail.work_description && (
                    <div className="col-span-2">
                      <p className="text-xs text-gi-text-subtle uppercase tracking-wide mb-1">Opis rada</p>
                      <p className="text-gi-text-muted whitespace-pre-wrap break-words">{detail.work_description}</p>
                    </div>
                  )}
                </div>

                <p className="text-xs text-gi-text-subtle">
                  Upisano: {fmt(detail.created_at)}
                  {detail.updated_at !== detail.created_at && (
                    <> · Ažurirano: {fmt(detail.updated_at)}</>
                  )}
                </p>

                {/* Correction button */}
                {!showCorrectionForm && (
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
                )}

                {/* Correction form */}
                {showCorrectionForm && (
                  <form onSubmit={handleCorrect} className="border border-amber-800/50 rounded-xl p-4 space-y-3 bg-amber-950/20">
                    <p className="text-xs font-semibold text-amber-300 uppercase tracking-wide">Ispravak sati</p>
                    <div>
                      <label className="block text-xs text-gi-text-muted mb-1">Ispravljeni sati</label>
                      <input
                        type="number" min="0" max="24" step="0.5"
                        inputMode="decimal"
                        value={newHours}
                        onChange={e => setNewHours(e.target.value)}
                        className={inputCls}
                        disabled={correcting}
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-gi-text-muted mb-1">
                        Razlog izmjene <span className="text-red-400">*</span>
                      </label>
                      <textarea
                        value={reason}
                        onChange={e => setReason(e.target.value)}
                        rows={2}
                        placeholder="Npr. Greška u unosu…"
                        className={`${inputCls} resize-none`}
                        disabled={correcting}
                      />
                    </div>
                    {correctionError && <p className="text-sm text-red-400">{correctionError}</p>}
                    <div className="flex gap-2">
                      <button
                        type="submit"
                        disabled={correcting}
                        className="px-3 py-1.5 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                      >
                        {correcting ? 'Spremanje…' : 'Spremi'}
                      </button>
                      <button
                        type="button"
                        onClick={() => setShowCorrectionForm(false)}
                        className="px-3 py-1.5 bg-gi-card-elevated hover:bg-gi-border text-gi-text-muted text-sm rounded-lg transition-colors"
                      >
                        Odustani
                      </button>
                    </div>
                  </form>
                )}
              </div>

              {/* Revision history */}
              {detail.revisions.length > 0 && (
                <div>
                  <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wide mb-3">
                    Povijest ispravaka
                  </p>
                  <ul className="space-y-3">
                    {detail.revisions.map(rev => (
                      <li key={rev.id} className="border-l-2 border-gi-yellow/60 pl-3">
                        <p className="text-sm text-gi-text">
                          <span className="line-through text-gi-text-subtle">{rev.previous_hours} h</span>
                          <span className="text-gi-text-muted mx-1">→</span>
                          <span className="font-semibold text-gi-yellow">{rev.new_hours} h</span>
                        </p>
                        <p className="text-xs text-gi-text-muted mt-0.5 whitespace-pre-wrap break-words">{rev.reason}</p>
                        <p className="text-xs text-gi-text-subtle mt-0.5">
                          {rev.changed_by_name ?? 'Nepoznat'} · {fmt(rev.created_at)}
                        </p>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Internal comments */}
              <div>
                <p className="text-xs font-semibold text-gi-text-muted uppercase tracking-wide mb-1">
                  Interni komentari
                </p>
                <p className="text-xs text-gi-text-subtle mb-3">(samo direktor / inženjer)</p>

                {detail.comments.length > 0 && (
                  <ul className="space-y-2 mb-3">
                    {detail.comments.map(c => (
                      <li key={c.id} className="bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2.5">
                        <p className="text-sm text-gi-text whitespace-pre-wrap break-words">{c.comment_text}</p>
                        <p className="text-xs text-gi-text-subtle mt-1">
                          {c.author_name ?? 'Nepoznat'} · {fmt(c.created_at)}
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
                  {commentError && <p className="text-sm text-red-400">{commentError}</p>}
                  <button
                    type="submit"
                    disabled={addingComment || !commentText.trim()}
                    className="px-3 py-1.5 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted hover:text-gi-text text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                  >
                    {addingComment ? 'Dodavanje…' : 'Dodaj komentar'}
                  </button>
                </form>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
