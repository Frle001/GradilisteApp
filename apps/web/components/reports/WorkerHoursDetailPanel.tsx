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
        className="relative w-full max-w-lg h-full bg-slate-950 border-l border-slate-800 overflow-y-auto shadow-2xl"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="sticky top-0 z-10 bg-slate-900 border-b border-slate-800 px-5 py-4 flex items-center justify-between gap-3">
          <h2 className="text-base font-semibold text-white">Detalji upisanih sati</h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded transition"
            aria-label="Zatvori"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="p-5 space-y-6">
          {loading ? (
            <p className="text-slate-500 text-sm">Učitavanje...</p>
          ) : fetchError ? (
            <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
              <p className="text-red-300 text-sm">{fetchError}</p>
            </div>
          ) : detail && (
            <>
              {/* Main info */}
              <div className="space-y-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-lg font-bold text-white">{detail.worker_name}</p>
                    <p className="text-sm text-slate-400">{ROLE_LABELS[detail.worker_role] ?? detail.worker_role}</p>
                  </div>
                  <div className="text-right shrink-0">
                    <span className="text-3xl font-bold text-white">{detail.hours_worked}</span>
                    <span className="text-slate-400 ml-1 text-sm">h</span>
                    {originalHours !== null && originalHours !== detail.hours_worked && (
                      <p className="text-xs text-amber-400 mt-0.5">
                        Originalno: <span className="line-through">{originalHours} h</span>
                      </p>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <p className="text-xs text-slate-500 uppercase tracking-wide mb-0.5">Projekt</p>
                    <p className="text-white">{detail.project_name}</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500 uppercase tracking-wide mb-0.5">Datum</p>
                    <p className="text-white capitalize">{fmtDate(detail.work_date)}</p>
                  </div>

                  {detail.notes && (
                    <div className="col-span-2">
                      <p className="text-xs text-slate-500 uppercase tracking-wide mb-1">Napomena</p>
                      <p className="text-slate-300 whitespace-pre-wrap break-words">{detail.notes}</p>
                    </div>
                  )}

                  {detail.work_description && (
                    <div className="col-span-2">
                      <p className="text-xs text-slate-500 uppercase tracking-wide mb-1">Opis rada</p>
                      <p className="text-slate-300 whitespace-pre-wrap break-words">{detail.work_description}</p>
                    </div>
                  )}
                </div>

                <p className="text-xs text-slate-600">
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
                    className="text-sm text-amber-400 hover:text-amber-300 border border-amber-700/60 hover:border-amber-500 px-3 py-1.5 rounded-lg transition"
                  >
                    Ispravi sate
                  </button>
                )}

                {/* Correction form */}
                {showCorrectionForm && (
                  <form onSubmit={handleCorrect} className="border border-amber-800/50 rounded-xl p-4 space-y-3 bg-amber-950/20">
                    <p className="text-xs font-semibold text-amber-300 uppercase tracking-wide">Ispravak sati</p>
                    <div>
                      <label className="block text-xs text-slate-400 mb-1">Ispravljeni sati</label>
                      <input
                        type="number" min="0" max="24" step="0.5"
                        value={newHours}
                        onChange={e => setNewHours(e.target.value)}
                        className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-amber-500"
                        disabled={correcting}
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-slate-400 mb-1">
                        Razlog izmjene <span className="text-red-400">*</span>
                      </label>
                      <textarea
                        value={reason}
                        onChange={e => setReason(e.target.value)}
                        rows={2}
                        placeholder="Npr. Greška u unosu..."
                        className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-amber-500 resize-none"
                        disabled={correcting}
                      />
                    </div>
                    {correctionError && <p className="text-sm text-red-400">{correctionError}</p>}
                    <div className="flex gap-2">
                      <button
                        type="submit"
                        disabled={correcting}
                        className="px-3 py-1.5 bg-amber-600 hover:bg-amber-500 text-white text-sm font-medium rounded-lg transition disabled:opacity-50"
                      >
                        {correcting ? 'Spremanje...' : 'Spremi'}
                      </button>
                      <button
                        type="button"
                        onClick={() => setShowCorrectionForm(false)}
                        className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm rounded-lg transition"
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
                  <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-3">
                    Povijest ispravaka
                  </p>
                  <ul className="space-y-3">
                    {detail.revisions.map(rev => (
                      <li key={rev.id} className="border-l-2 border-amber-700 pl-3">
                        <p className="text-sm text-white">
                          <span className="line-through text-slate-500">{rev.previous_hours} h</span>
                          {' → '}
                          <span className="font-semibold">{rev.new_hours} h</span>
                        </p>
                        <p className="text-xs text-slate-400 mt-0.5 whitespace-pre-wrap break-words">{rev.reason}</p>
                        <p className="text-xs text-slate-600 mt-0.5">
                          {rev.changed_by_name ?? 'Nepoznat'} · {fmt(rev.created_at)}
                        </p>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Internal comments */}
              <div>
                <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-3">
                  Interni komentari
                  <span className="ml-2 text-slate-600 font-normal normal-case">
                    (samo direktor / inženjer)
                  </span>
                </p>

                {detail.comments.length > 0 && (
                  <ul className="space-y-2 mb-3">
                    {detail.comments.map(c => (
                      <li key={c.id} className="bg-slate-800/60 border border-slate-700/60 rounded-lg px-3 py-2.5">
                        <p className="text-sm text-slate-200 whitespace-pre-wrap break-words">{c.comment_text}</p>
                        <p className="text-xs text-slate-500 mt-1">
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
                    placeholder="Dodaj interni komentar..."
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 resize-none"
                    disabled={addingComment}
                  />
                  {commentError && <p className="text-sm text-red-400">{commentError}</p>}
                  <button
                    type="submit"
                    disabled={addingComment || !commentText.trim()}
                    className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 text-sm font-medium rounded-lg transition disabled:opacity-50"
                  >
                    {addingComment ? 'Dodavanje...' : 'Dodaj komentar'}
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
