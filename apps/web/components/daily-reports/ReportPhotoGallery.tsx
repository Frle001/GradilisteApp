'use client'

import { useEffect, useRef, useState } from 'react'
import Image from 'next/image'
import apiClient from '@/lib/api-client'
import { type DailyReportAttachment } from '@/lib/types/daily-reports'

const MAX_REPORT_PHOTOS = 100

interface Props {
  reportId: string
  initialAttachments: DailyReportAttachment[]
  canEdit: boolean
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// useInView returns true once the element attached to ref enters the viewport.
// Fires once then disconnects; rootMargin of 200px pre-loads just before visible.
function useInView(ref: React.RefObject<HTMLElement>): boolean {
  const [inView, setInView] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const io = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) { setInView(true); io.disconnect() }
    }, { rootMargin: '200px' })
    io.observe(el)
    return () => io.disconnect()
  }, [ref])
  return inView
}

// Fetch an attachment from the API and return a blob URL, or null on error.
// enabled must be true before the fetch starts (used to defer until visible).
function usePhotoBlobUrl(reportId: string, attachId: string, enabled: boolean): string | null {
  const [url, setUrl] = useState<string | null>(null)
  useEffect(() => {
    if (!enabled) return
    let objectUrl: string | null = null
    apiClient
      .get(`/daily-reports/${reportId}/attachments/${attachId}/download`, { responseType: 'blob' })
      .then(res => {
        objectUrl = URL.createObjectURL(res.data as Blob)
        setUrl(objectUrl)
      })
      .catch(() => {/* thumbnail unavailable, show fallback */})
    return () => {
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [reportId, attachId, enabled])
  return url
}

function PhotoThumb({
  reportId, att, onDelete, canDelete, onExpand,
}: {
  reportId: string
  att: DailyReportAttachment
  onDelete: (id: string) => void
  canDelete: boolean
  onExpand: (id: string) => void
}) {
  const containerRef = useRef<HTMLDivElement>(null)
  const inView = useInView(containerRef as React.RefObject<HTMLElement>)
  const blobUrl = usePhotoBlobUrl(reportId, att.id, inView)

  return (
    <div ref={containerRef} className="relative group rounded-lg overflow-hidden border border-gi-border bg-gi-card-elevated">
      <button
        type="button"
        className="relative w-full aspect-square block"
        onClick={() => onExpand(att.id)}
        title={att.original_name}
      >
        {blobUrl ? (
          <Image fill src={blobUrl} alt={att.original_name} className="object-cover" unoptimized />
        ) : (
          <div className="w-full h-full flex items-center justify-center">
            <svg className="w-8 h-8 text-gi-border" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3 3h18M3 21h18" />
            </svg>
          </div>
        )}
      </button>
      {canDelete && (
        <button
          type="button"
          onClick={() => onDelete(att.id)}
          className="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity bg-red-700 hover:bg-red-600 text-white rounded p-0.5"
          title="Obriši"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
      <div className="px-1.5 py-1 bg-gi-navy/80">
        <p className="text-xs text-gi-text-muted truncate">{att.original_name}</p>
        <p className="text-xs text-gi-text-subtle">{formatBytes(att.file_size)}</p>
      </div>
    </div>
  )
}

function Lightbox({
  reportId, att, onClose, onPrev, onNext, hasPrev, hasNext,
}: {
  reportId: string
  att: DailyReportAttachment
  onClose: () => void
  onPrev: () => void
  onNext: () => void
  hasPrev: boolean
  hasNext: boolean
}) {
  const blobUrl = usePhotoBlobUrl(reportId, att.id, true)

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft' && hasPrev) onPrev()
      if (e.key === 'ArrowRight' && hasNext) onNext()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, onPrev, onNext, hasPrev, hasNext])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/90" onClick={onClose}>
      <div className="relative max-w-4xl max-h-[90vh] mx-4" onClick={e => e.stopPropagation()}>
        {blobUrl ? (
          <Image
            src={blobUrl}
            alt={att.original_name}
            width={1920}
            height={1080}
            unoptimized
            className="max-h-[80vh] max-w-full object-contain rounded"
            style={{ width: 'auto', height: 'auto' }}
          />
        ) : (
          <div className="w-64 h-64 flex items-center justify-center text-gi-text-muted">Učitavanje…</div>
        )}
        <button
          type="button"
          onClick={onClose}
          className="absolute -top-3 -right-3 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text rounded-full p-1"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
        {hasPrev && (
          <button
            type="button"
            onClick={onPrev}
            className="absolute left-2 top-1/2 -translate-y-1/2 bg-black/50 hover:bg-black/70 text-white rounded-full p-2"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
            </svg>
          </button>
        )}
        {hasNext && (
          <button
            type="button"
            onClick={onNext}
            className="absolute right-2 top-1/2 -translate-y-1/2 bg-black/50 hover:bg-black/70 text-white rounded-full p-2"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
            </svg>
          </button>
        )}
        <p className="text-center text-gi-text-subtle text-xs mt-2">{att.original_name} &middot; {formatBytes(att.file_size)}</p>
      </div>
    </div>
  )
}

export default function ReportPhotoGallery({ reportId, initialAttachments, canEdit }: Props) {
  const [attachments, setAttachments] = useState<DailyReportAttachment[]>(initialAttachments)
  const [lightboxId, setLightboxId] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const lightboxIdx = lightboxId ? attachments.findIndex(a => a.id === lightboxId) : -1
  const lightboxAtt = lightboxIdx >= 0 ? attachments[lightboxIdx] : null

  async function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ''

    if (attachments.length >= MAX_REPORT_PHOTOS) {
      setUploadError(`Možete dodati najviše ${MAX_REPORT_PHOTOS} fotografija.`)
      return
    }

    setUploading(true)
    setUploadError(null)
    const form = new FormData()
    form.append('photo', file)
    try {
      const res = await apiClient.post(`/daily-reports/${reportId}/attachments`, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setAttachments(prev => [...prev, res.data.attachment as DailyReportAttachment])
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setUploadError(e?.response?.data?.error ?? 'Greška pri prijenosu slike.')
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(attachId: string) {
    if (!confirm('Obrisati ovu sliku?')) return
    setDeleteError(null)
    try {
      await apiClient.delete(`/daily-reports/${reportId}/attachments/${attachId}`)
      setAttachments(prev => prev.filter(a => a.id !== attachId))
      if (lightboxId === attachId) setLightboxId(null)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setDeleteError(e?.response?.data?.error ?? 'Greška pri brisanju slike.')
    }
  }

  return (
    <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-4">
      <div className="flex items-center justify-between gap-2">
        <div>
          <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">
            Fotografije
            <span className="ml-2 text-gi-text-subtle font-normal normal-case">
              {attachments.length} / {MAX_REPORT_PHOTOS}
            </span>
          </h2>
        </div>
        {canEdit && attachments.length < MAX_REPORT_PHOTOS && (
          <button
            type="button"
            disabled={uploading}
            onClick={() => fileInputRef.current?.click()}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted text-xs font-medium rounded-lg transition disabled:opacity-50"
          >
            {uploading ? (
              <>
                <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                </svg>
                Prijenos…
              </>
            ) : (
              <>
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                </svg>
                Dodaj sliku
              </>
            )}
          </button>
        )}
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/webp"
          className="hidden"
          onChange={handleFileChange}
        />
      </div>

      {uploadError && (
        <p className="text-sm text-red-400">{uploadError}</p>
      )}
      {deleteError && (
        <p className="text-sm text-red-400">{deleteError}</p>
      )}

      {attachments.length === 0 ? (
        canEdit ? (
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            className="w-full flex flex-col items-center gap-2 py-6 border border-dashed border-gi-border rounded-lg text-gi-text-subtle hover:border-gi-border/80 hover:text-gi-text-muted transition disabled:pointer-events-none"
          >
            <svg className="w-7 h-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6.827 6.175A2.31 2.31 0 015.186 7.23c-.38.054-.757.112-1.134.175C2.999 7.58 2.25 8.507 2.25 9.574V18a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9.574c0-1.067-.75-1.994-1.802-2.169a47.865 47.865 0 00-1.134-.175 2.31 2.31 0 01-1.64-1.055l-.822-1.316a2.192 2.192 0 00-1.736-1.039 48.774 48.774 0 00-5.232 0 2.192 2.192 0 00-1.736 1.039l-.821 1.316z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 12.75a4.5 4.5 0 11-9 0 4.5 4.5 0 019 0zM18.75 10.5h.008v.008h-.008V10.5z" />
            </svg>
            <span className="text-xs">Kliknite za dodavanje fotografija</span>
            <span className="text-xs text-gi-text-subtle">JPEG, PNG, WebP · maks. 10 MB</span>
          </button>
        ) : (
          <p className="text-sm text-gi-text-subtle">Nema fotografija.</p>
        )
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
          {attachments.map(att => (
            <PhotoThumb
              key={att.id}
              reportId={reportId}
              att={att}
              canDelete={canEdit}
              onDelete={handleDelete}
              onExpand={id => setLightboxId(id)}
            />
          ))}
        </div>
      )}

      {lightboxAtt && (
        <Lightbox
          reportId={reportId}
          att={lightboxAtt}
          onClose={() => setLightboxId(null)}
          hasPrev={lightboxIdx > 0}
          hasNext={lightboxIdx < attachments.length - 1}
          onPrev={() => setLightboxId(attachments[lightboxIdx - 1].id)}
          onNext={() => setLightboxId(attachments[lightboxIdx + 1].id)}
        />
      )}
    </div>
  )
}
