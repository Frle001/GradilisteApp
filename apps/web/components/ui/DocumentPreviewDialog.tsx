'use client'

import { useEffect, useState } from 'react'
import apiClient from '@/lib/api-client'

interface DocumentPreviewDialogProps {
  fetchUrl?: string
  file?: File
  filename: string
  mimeType: string
  triggerLabel?: string
  triggerClassName?: string
}

export default function DocumentPreviewDialog({
  fetchUrl,
  file,
  filename,
  mimeType,
  triggerLabel = 'Pregledaj',
  triggerClassName,
}: DocumentPreviewDialogProps) {
  const [open, setOpen] = useState(false)
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function cleanup() {
    if (blobUrl) URL.revokeObjectURL(blobUrl)
    setBlobUrl(null)
  }

  function handleClose() {
    setOpen(false)
    cleanup()
  }

  async function handleOpen() {
    setOpen(true)
    setError(null)
    if (file) {
      setBlobUrl(URL.createObjectURL(file))
      return
    }
    if (!fetchUrl) return
    setLoading(true)
    try {
      const resp = await apiClient.get(fetchUrl, { responseType: 'blob' })
      setBlobUrl(URL.createObjectURL(resp.data as Blob))
    } catch {
      setError('Greška pri učitavanju datoteke')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    return () => {
      if (blobUrl) URL.revokeObjectURL(blobUrl)
    }
  }, [blobUrl])

  const isImage = mimeType.startsWith('image/')
  const isPdf = mimeType === 'application/pdf'

  const defaultTriggerCls =
    'text-xs text-violet-400 hover:underline disabled:opacity-50 disabled:cursor-not-allowed'

  return (
    <>
      <button
        type="button"
        onClick={handleOpen}
        className={triggerClassName ?? defaultTriggerCls}
      >
        {triggerLabel}
      </button>

      {open && (
        <div
          className="fixed inset-0 z-50 bg-black/70 flex items-center justify-center p-4"
          onClick={handleClose}
        >
          <div
            className="bg-zinc-900 rounded-xl border border-zinc-700 max-w-3xl w-full flex flex-col"
            style={{ maxHeight: '90vh' }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-700 shrink-0">
              <span className="text-sm font-medium text-zinc-200 truncate mr-3">{filename}</span>
              <button
                onClick={handleClose}
                className="text-zinc-500 hover:text-white transition-colors shrink-0"
              >
                ✕
              </button>
            </div>

            <div className="flex-1 overflow-auto min-h-0" style={{ minHeight: '60vh' }}>
              {loading && (
                <p className="text-center text-zinc-400 py-12">Učitavanje...</p>
              )}
              {error && (
                <p className="text-center text-red-400 py-12">{error}</p>
              )}
              {blobUrl && isPdf && (
                <iframe
                  src={blobUrl}
                  className="w-full border-0"
                  style={{ height: '70vh' }}
                  title={filename}
                />
              )}
              {blobUrl && isImage && (
                <div className="flex items-center justify-center p-4 h-full">
                  <img
                    src={blobUrl}
                    alt={filename}
                    className="max-w-full max-h-full object-contain"
                    style={{ maxHeight: '65vh' }}
                  />
                </div>
              )}
              {blobUrl && !isPdf && !isImage && (
                <div className="p-8 text-center">
                  <p className="text-zinc-300 mb-4">
                    Pregled nije dostupan za ovaj tip datoteke.
                  </p>
                  <a
                    href={blobUrl}
                    download={filename}
                    className="px-3 py-1.5 rounded-lg bg-violet-600 hover:bg-violet-700 text-white text-xs font-medium transition"
                  >
                    Preuzmi
                  </a>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  )
}
