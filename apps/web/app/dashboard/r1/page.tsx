'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DocumentPreviewDialog from '@/components/ui/DocumentPreviewDialog'
import { R1Receipt } from '@/lib/types/finance'

const inputCls =
  'w-full rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500'

const btnPrimary =
  'rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors'

const btnSecondary =
  'rounded-lg border border-slate-600 bg-slate-800 px-4 py-2 text-sm font-medium text-slate-300 hover:bg-slate-700 transition-colors'

const FORBIDDEN_ROLES = ['administracija']

function errMsg(e: unknown): string {
  const cast = e as { response?: { data?: { error?: string } } }
  return cast?.response?.data?.error ?? (e instanceof Error ? e.message : 'Greška')
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('hr-HR')
}

async function downloadFile(receiptId: string, filename: string) {
  const resp = await apiClient.get(`/finance/r1/${receiptId}/download`, {
    responseType: 'blob',
  })
  const url = window.URL.createObjectURL(resp.data as Blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  window.URL.revokeObjectURL(url)
}

// ── Upload form ───────────────────────────────────────────────────────────────

interface UploadFormProps {
  onSuccess: () => void
}

function UploadForm({ onSuccess }: UploadFormProps) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [price, setPrice] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const file = fileRef.current?.files?.[0]
    if (!file) {
      setError('Odaberite datoteku')
      return
    }
    const priceNum = parseFloat(price)
    if (!price || isNaN(priceNum) || priceNum <= 0) {
      setError('Cijena mora biti pozitivan broj')
      return
    }

    const fd = new FormData()
    fd.append('file', file)
    fd.append('price', priceNum.toFixed(2))

    setSubmitting(true)
    setError('')
    try {
      await apiClient.post('/finance/r1', fd)
      if (fileRef.current) fileRef.current.value = ''
      setPrice('')
      onSuccess()
    } catch (err) {
      setError(errMsg(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Iznos (€) *</label>
        <input
          type="number"
          step="0.01"
          min="0.01"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          placeholder="npr. 125.50"
          className={inputCls}
          required
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">R1 račun (dokument) *</label>
        <input
          ref={fileRef}
          type="file"
          accept=".pdf,.doc,.docx,.xls,.xlsx,.jpg,.jpeg,.png"
          className={inputCls}
          required
        />
      </div>

      {error && <p className="text-sm text-red-400">{error}</p>}

      <button type="submit" className={btnPrimary} disabled={submitting}>
        {submitting ? 'Učitavanje...' : 'Predaj R1'}
      </button>
    </form>
  )
}

// ── Receipt row ───────────────────────────────────────────────────────────────

interface ReceiptRowProps {
  receipt: R1Receipt
  isDirektor: boolean
}

function ReceiptRow({ receipt, isDirektor }: ReceiptRowProps) {
  const [downloading, setDownloading] = useState(false)
  const [dlErr, setDlErr] = useState('')

  async function handleDownload() {
    setDownloading(true)
    setDlErr('')
    try {
      await downloadFile(receipt.id, receipt.document.original_filename)
    } catch (err) {
      setDlErr(errMsg(err))
    } finally {
      setDownloading(false)
    }
  }

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-800/50 p-4 space-y-2">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-lg font-semibold text-white">
            {receipt.price.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €
          </p>
          {isDirektor && (
            <p className="text-sm text-slate-300">Predao: <span className="text-white">{receipt.submitter_name}</span></p>
          )}
          <p className="text-xs text-slate-500 mt-1">
            {receipt.document.original_filename} · {formatFileSize(receipt.document.file_size)}
          </p>
          <p className="text-xs text-slate-500">{formatDate(receipt.created_at)}</p>
        </div>
        <div className="flex gap-2 items-center shrink-0">
          <DocumentPreviewDialog
            fetchUrl={`/finance/r1/${receipt.id}/download`}
            filename={receipt.document.original_filename}
            mimeType={receipt.document.mime_type}
            triggerClassName="text-xs text-blue-400 hover:underline"
          />
          <button
            onClick={handleDownload}
            disabled={downloading}
            className={btnSecondary + ' text-xs py-1.5'}
          >
            {downloading ? '...' : 'Preuzmi'}
          </button>
        </div>
      </div>
      {dlErr && <p className="text-xs text-red-400">{dlErr}</p>}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function R1Page() {
  const { user, isLoading: authLoading } = useAuth()
  const router = useRouter()
  const [receipts, setReceipts] = useState<R1Receipt[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)

  const loadReceipts = useCallback(async () => {
    try {
      const resp = await apiClient.get<{ receipts: R1Receipt[] }>('/finance/r1')
      setReceipts(resp.data.receipts ?? [])
    } catch {
      // handled silently
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!authLoading && user) {
      if (FORBIDDEN_ROLES.includes(user.role)) {
        router.replace('/dashboard')
        return
      }
      loadReceipts()
    }
  }, [authLoading, user, router, loadReceipts])

  if (authLoading || !user) return <LoadingScreen />

  const isDirektor = user.role === 'direktor'
  const heading = isDirektor ? 'R1 računi' : 'Moji R1 računi'

  return (
    <main className="min-h-screen bg-slate-950 text-white px-4 py-6 max-w-2xl mx-auto">
      <div className="flex items-center gap-3 mb-6">
        <button onClick={() => router.back()} className="text-slate-400 hover:text-white transition-colors">
          ←
        </button>
        <h1 className="text-xl font-bold">{heading}</h1>
      </div>

      <div className="mb-4 flex justify-end">
        <button
          onClick={() => setShowForm((f) => !f)}
          className={btnPrimary}
        >
          {showForm ? 'Zatvori' : '+ Predaj R1'}
        </button>
      </div>

      {showForm && (
        <div className="mb-6 rounded-xl border border-slate-700 bg-slate-900 p-4">
          <h2 className="text-base font-semibold mb-4">Novi R1 račun</h2>
          <UploadForm
            onSuccess={() => {
              setShowForm(false)
              loadReceipts()
            }}
          />
        </div>
      )}

      {loading ? (
        <div className="text-center py-12 text-slate-400">Učitavanje...</div>
      ) : receipts.length === 0 ? (
        <div className="text-center py-12 text-slate-400">
          {isDirektor ? 'Još nema predanih R1 računa.' : 'Još niste predali nijedan R1 račun.'}
        </div>
      ) : (
        <div className="space-y-3">
          {receipts.map((rec) => (
            <ReceiptRow key={rec.id} receipt={rec} isDirektor={isDirektor} />
          ))}
        </div>
      )}
    </main>
  )
}
