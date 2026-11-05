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

    return () => { if (url) URL.revokeObjectURL(url) }
  }, [purchaseId])

  if (loading) {
    return (
      <div className="rounded-xl border border-gi-border bg-gi-card flex items-center justify-center h-32">
        <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje računa…</p>
      </div>
    )
  }

  if (error || !objectUrl) {
    return (
      <div className="rounded-xl border border-gi-border bg-gi-card flex items-center justify-center h-24">
        <p className="text-gi-text-subtle text-sm">Račun nije dostupan.</p>
      </div>
    )
  }

  const headerBar = (
    <div className="bg-gi-card-elevated px-4 py-2.5 border-b border-gi-border flex items-center justify-between">
      <span className="text-sm text-gi-text-muted font-medium truncate">{originalFilename ?? (isPdf ? 'racun.pdf' : 'račun')}</span>
      <a
        href={objectUrl}
        download={originalFilename ?? (isPdf ? 'racun.pdf' : 'racun')}
        className="text-xs text-gi-yellow hover:text-gi-yellow-hover font-medium flex-shrink-0 ml-3 transition-colors"
      >
        Preuzmi
      </a>
    </div>
  )

  if (isPdf) {
    return (
      <div className="rounded-xl border border-gi-border overflow-hidden">
        {headerBar}
        <iframe src={objectUrl} className="w-full h-[500px] bg-white" title="Pregled računa" />
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-gi-border overflow-hidden">
      {headerBar}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={objectUrl}
        alt="Fotografija računa"
        className="w-full max-h-[600px] object-contain bg-gi-card"
      />
    </div>
  )
}
