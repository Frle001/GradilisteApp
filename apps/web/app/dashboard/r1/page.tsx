'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import DocumentPreviewDialog from '@/components/ui/DocumentPreviewDialog'
import { type R1Receipt } from '@/lib/types/finance'

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
  'text-sm text-gi-text placeholder:text-gi-text-subtle',
  'focus:outline-none focus:border-gi-yellow',
  'disabled:opacity-40 disabled:cursor-not-allowed',
  'transition-colors',
].join(' ')

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

// ── Upload form ───────────────────────────────────────────────────────────────

function UploadForm({ onSuccess, onCancel }: { onSuccess: () => void; onCancel: () => void }) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [price, setPrice] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const file = fileRef.current?.files?.[0]
    if (!file) { setError('Odaberite datoteku'); return }
    const priceNum = parseFloat(price)
    if (!price || isNaN(priceNum) || priceNum <= 0) {
      setError('Cijena mora biti pozitivan broj')
      return
    }
    const fd = new FormData()
    fd.append('file', file)
    fd.append('price', priceNum.toFixed(2))
    setSubmitting(true); setError('')
    try {
      await apiClient.post('/finance/r1', fd)
      if (fileRef.current) fileRef.current.value = ''
      setPrice(''); setSelectedFile(null)
      onSuccess()
    } catch (err) { setError(errMsg(err)) }
    finally { setSubmitting(false) }
  }

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card p-4">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-semibold text-gi-text">Novi R1 račun</h2>
        <button
          onClick={onCancel}
          className="text-gi-text-subtle hover:text-gi-text transition-colors p-1 rounded-lg"
          aria-label="Zatvori"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">
            Iznos (€) <span className="text-red-400">*</span>
          </label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            value={price}
            onChange={e => setPrice(e.target.value)}
            placeholder="npr. 125.50"
            className={inputCls}
            required
          />
        </div>

        {/* File picker */}
        <div>
          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">
            R1 račun (dokument) <span className="text-red-400">*</span>
          </label>
          <div className={[
            'relative rounded-lg border transition-colors cursor-pointer',
            selectedFile
              ? 'border-gi-yellow/60 bg-gi-yellow/5'
              : 'border-dashed border-gi-border hover:border-gi-text-subtle bg-gi-card-elevated',
          ].join(' ')}>
            <input
              ref={fileRef}
              type="file"
              accept=".pdf,.doc,.docx,.xls,.xlsx,.jpg,.jpeg,.png"
              className="absolute inset-0 opacity-0 cursor-pointer w-full h-full"
              onChange={e => setSelectedFile(e.target.files?.[0] ?? null)}
            />
            <div className="px-3 py-2.5 flex items-center gap-2.5 pointer-events-none">
              {selectedFile ? (
                <>
                  <svg className="w-4 h-4 text-gi-yellow shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
                  </svg>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm text-gi-text truncate">{selectedFile.name}</p>
                    <p className="text-xs text-gi-text-subtle">{formatFileSize(selectedFile.size)}</p>
                  </div>
                  <svg className="w-3.5 h-3.5 text-gi-yellow shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                  </svg>
                </>
              ) : (
                <>
                  <svg className="w-4 h-4 text-gi-text-subtle shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 16.5V9.75m0 0l3 3m-3-3l-3 3M6.75 19.5a4.5 4.5 0 01-1.41-8.775 5.25 5.25 0 0110.233-2.33 3 3 0 013.758 3.848A3.752 3.752 0 0118 19.5H6.75z" />
                  </svg>
                  <div>
                    <p className="text-sm text-gi-text-muted">Odaberite dokument</p>
                    <p className="text-xs text-gi-text-subtle">PDF, Word, Excel, JPG, PNG</p>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>

        {error && <p className="text-sm text-red-400">{error}</p>}

        <div className="flex gap-2 pt-1">
          <button
            type="submit" disabled={submitting}
            className="flex-1 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors disabled:opacity-50"
          >
            {submitting ? 'Učitavanje…' : 'Predaj R1'}
          </button>
          <button
            type="button" onClick={onCancel}
            className="px-4 py-2 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted text-sm rounded-lg transition-colors"
          >
            Odustani
          </button>
        </div>
      </form>
    </div>
  )
}

// ── Receipt row ───────────────────────────────────────────────────────────────

function ReceiptRow({ receipt, isDirektor }: { receipt: R1Receipt; isDirektor: boolean }) {
  const [downloading, setDownloading] = useState(false)
  const [dlErr, setDlErr] = useState('')

  async function handleDownload() {
    setDownloading(true); setDlErr('')
    try {
      const resp = await apiClient.get(`/finance/r1/${receipt.id}/download`, { responseType: 'blob' })
      const url = window.URL.createObjectURL(resp.data as Blob)
      const a = document.createElement('a')
      a.href = url; a.download = receipt.document.original_filename; a.click()
      window.URL.revokeObjectURL(url)
    } catch (err) { setDlErr(errMsg(err)) }
    finally { setDownloading(false) }
  }

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 hover:bg-gi-card-elevated/40 transition-colors">
      <div className="flex items-center gap-3">
        <span className="shrink-0 w-28 text-base font-bold text-gi-text tabular-nums">
          {receipt.price.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €
        </span>
        <div className="flex-1 min-w-0">
          {isDirektor && (
            <p className="text-sm text-gi-text truncate">{receipt.submitter_name}</p>
          )}
          <p className="text-xs text-gi-text-subtle">{formatDate(receipt.created_at)}</p>
        </div>
        <div className="hidden sm:flex flex-col items-end shrink-0 w-44 min-w-0">
          <p className="text-xs text-gi-text truncate max-w-full text-right" title={receipt.document.original_filename}>
            {receipt.document.original_filename}
          </p>
          <p className="text-xs text-gi-text-subtle">{formatFileSize(receipt.document.file_size)}</p>
        </div>
        <div className="flex gap-1.5 items-center shrink-0">
          <DocumentPreviewDialog
            fetchUrl={`/finance/r1/${receipt.id}/download`}
            filename={receipt.document.original_filename}
            mimeType={receipt.document.mime_type}
            triggerClassName="text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
          />
          <button
            onClick={handleDownload} disabled={downloading}
            className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-2.5 py-1 rounded-lg transition-colors disabled:opacity-50"
          >
            {downloading ? '…' : 'Preuzmi'}
          </button>
        </div>
      </div>
      {dlErr && <p className="text-xs text-red-400 mt-1.5">{dlErr}</p>}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function R1Page() {
  const { user, employee, isLoading: authLoading, logout } = useAuth()
  const router = useRouter()
  const [receipts, setReceipts] = useState<R1Receipt[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)

  const loadReceipts = useCallback(async () => {
    try {
      const resp = await apiClient.get<{ receipts: R1Receipt[] }>('/finance/r1')
      setReceipts(resp.data.receipts ?? [])
    } catch { /* silent */ }
    finally { setLoading(false) }
  }, [])

  useEffect(() => {
    if (!authLoading && user) {
      if (FORBIDDEN_ROLES.includes(user.role)) {
        router.replace('/dashboard'); return
      }
      loadReceipts()
    }
  }, [authLoading, user, router, loadReceipts])

  if (authLoading || !user) return <LoadingScreen />

  const isDirektor = user.role === 'direktor'
  const heading = isDirektor ? 'R1 računi' : 'Moji R1 računi'
  const emptyMsg = isDirektor ? 'Još nema predanih R1 računa.' : 'Još niste predali nijedan R1 račun.'

  return (
    <AppShell
      user={user}
      employee={employee}
      title={heading}
      backHref="/dashboard"
      onLogout={logout}
    >
      <div className="max-w-4xl space-y-5">
        {/* Page header */}
        <div>
          <h1 className="text-xl font-bold text-gi-text tracking-tight">{heading}</h1>
          <p className="text-sm text-gi-text-subtle mt-0.5">
            {isDirektor ? 'Svi predani R1 računi zaposlenika' : 'Vaši predani R1 računi'}
          </p>
        </div>

        {/* Two-column layout when form is open */}
        <div className={showForm ? 'flex flex-col lg:flex-row gap-5 items-start' : 'space-y-3'}>
          {/* Register */}
          <div className={showForm ? 'flex-1 min-w-0 space-y-3' : 'space-y-3'}>
            <div className="flex justify-end">
              <button
                onClick={() => setShowForm(f => !f)}
                className="px-3 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors whitespace-nowrap shrink-0"
              >
                {showForm ? '× Zatvori' : '+ Predaj R1'}
              </button>
            </div>

            {loading ? (
              <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
                <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje R1 računa…</p>
              </div>
            ) : receipts.length === 0 ? (
              <div className="rounded-xl border border-gi-border bg-gi-card flex flex-col items-center justify-center py-16 gap-3">
                <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
                </svg>
                <p className="text-gi-text-subtle text-sm">{emptyMsg}</p>
              </div>
            ) : (
              <div className="space-y-2">
                {receipts.map(rec => (
                  <ReceiptRow key={rec.id} receipt={rec} isDirektor={isDirektor} />
                ))}
              </div>
            )}
          </div>

          {/* Form sidebar */}
          {showForm && (
            <div className="w-full lg:w-[380px] shrink-0 lg:sticky lg:top-5">
              <UploadForm
                onSuccess={() => { setShowForm(false); loadReceipts() }}
                onCancel={() => setShowForm(false)}
              />
            </div>
          )}
        </div>
      </div>
    </AppShell>
  )
}
