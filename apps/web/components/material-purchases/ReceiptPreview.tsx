'use client'

import { useEffect, useState } from 'react'
import apiClient from '@/lib/api-client'

interface Props {
  purchaseId: string
  originalFilename: string | null
}

export default function ReceiptPreview({ purchaseId, originalFilename }: Props) {
  const [objectUrl, setObjectUrl] = useState<string | null>(null)
  const [isPdf, setIsPdf] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    let url: string | null = null
    setLoading(true)
    setError(false)

    apiClient
      .get(`/material-purchases/${purchaseId}/receipt`, { responseType: 'blob' })
      .then(res => {
        const blob: Blob = res.data
        setIsPdf(blob.type === 'application/pdf')
        url = URL.createObjectURL(blob)
        setObjectUrl(url)
      })
      .catch(() => setError(true))
      .finally(() => setLoading(false))

    return () => {
      if (url) URL.revokeObjectURL(url)
    }
  }, [purchaseId])

  if (loading) {
    return (
      <div className="rounded-xl border border-slate-700 bg-slate-800/30 flex items-center justify-center h-48">
        <p className="text-slate-500 text-sm">Učitavanje računa…</p>
      </div>
    )
  }

  if (error || !objectUrl) {
    return (
      <div className="rounded-xl border border-slate-700 bg-slate-800/30 flex items-center justify-center h-32">
        <p className="text-slate-500 text-sm">Račun nije dostupan.</p>
      </div>
    )
  }

  if (isPdf) {
    return (
      <div className="rounded-xl border border-slate-700 overflow-hidden">
        <div className="bg-slate-800 px-4 py-2.5 border-b border-slate-700 flex items-center justify-between">
          <span className="text-sm text-slate-300 font-medium truncate">{originalFilename ?? 'racun.pdf'}</span>
          <a
            href={objectUrl}
            download={originalFilename ?? 'racun.pdf'}
            className="text-xs text-blue-400 hover:text-blue-300 font-medium flex-shrink-0 ml-3"
          >
            Preuzmi
          </a>
        </div>
        <iframe src={objectUrl} className="w-full h-[500px] bg-white" title="Pregled računa" />
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-slate-700 overflow-hidden">
      <div className="bg-slate-800 px-4 py-2.5 border-b border-slate-700 flex items-center justify-between">
        <span className="text-sm text-slate-300 font-medium truncate">{originalFilename ?? 'račun'}</span>
        <a
          href={objectUrl}
          download={originalFilename ?? 'racun'}
          className="text-xs text-blue-400 hover:text-blue-300 font-medium flex-shrink-0 ml-3"
        >
          Preuzmi
        </a>
      </div>
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={objectUrl}
        alt="Fotografija računa"
        className="w-full max-h-[600px] object-contain bg-slate-900"
      />
    </div>
  )
}
