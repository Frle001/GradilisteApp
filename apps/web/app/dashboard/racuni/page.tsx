'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import DocumentPreviewDialog from '@/components/ui/DocumentPreviewDialog'
import {
  type CompanyInvoice,
  type InvoiceType,
  INVOICE_TYPE_LABELS,
  SUPPLIERS,
  LEASING_COMPANIES,
  type R1Receipt,
  type FuelReceipt,
} from '@/lib/types/finance'

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
  'text-sm text-gi-text placeholder:text-gi-text-subtle',
  'focus:outline-none focus:border-gi-yellow',
  'disabled:opacity-40 disabled:cursor-not-allowed',
  'transition-colors',
].join(' ')

const INVOICE_TYPE_COLORS: Record<string, string> = {
  materijal: 'bg-blue-900/40 text-blue-300 border border-blue-700/50',
  gorivo:    'bg-amber-900/40 text-amber-300 border border-amber-700/50',
  leasing:   'bg-violet-900/40 text-violet-300 border border-violet-700/50',
  alati:     'bg-emerald-900/40 text-emerald-300 border border-emerald-700/50',
  oprema:    'bg-gi-card-elevated text-gi-text-muted border border-gi-border',
}

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

async function downloadBlob(url: string, filename: string) {
  const resp = await apiClient.get(url, { responseType: 'blob' })
  const blobUrl = window.URL.createObjectURL(resp.data as Blob)
  const a = document.createElement('a')
  a.href = blobUrl; a.download = filename; a.click()
  window.URL.revokeObjectURL(blobUrl)
}

// ── Upload form ───────────────────────────────────────────────────────────────

function UploadForm({ onSuccess, onCancel }: { onSuccess: () => void; onCancel: () => void }) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [invoiceType, setInvoiceType] = useState<InvoiceType>('gorivo')
  const [supplier, setSupplier] = useState('')
  const [leasingCompany, setLeasingCompany] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const file = fileRef.current?.files?.[0]
    if (!file) { setError('Odaberite datoteku'); return }
    const fd = new FormData()
    fd.append('file', file)
    fd.append('invoice_type', invoiceType)
    if (invoiceType === 'materijal' && supplier) fd.append('supplier', supplier)
    if (invoiceType === 'leasing' && leasingCompany) fd.append('leasing_company', leasingCompany)
    setSubmitting(true); setError('')
    try {
      await apiClient.post('/finance/racuni', fd)
      if (fileRef.current) fileRef.current.value = ''
      setSelectedFile(null); setSupplier(''); setLeasingCompany(''); setInvoiceType('gorivo')
      onSuccess()
    } catch (err) { setError(errMsg(err)) }
    finally { setSubmitting(false) }
  }

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card p-4">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-semibold text-gi-text">Novi račun</h2>
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
          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Vrsta računa</label>
          <select
            value={invoiceType}
            onChange={e => { setInvoiceType(e.target.value as InvoiceType); setSupplier(''); setLeasingCompany('') }}
            className={inputCls}
          >
            {(Object.entries(INVOICE_TYPE_LABELS) as [InvoiceType, string][]).map(([k, v]) => (
              <option key={k} value={k}>{v}</option>
            ))}
          </select>
        </div>

        {invoiceType === 'materijal' && (
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">
              Dobavljač <span className="text-red-400">*</span>
            </label>
            <select value={supplier} onChange={e => setSupplier(e.target.value)} className={inputCls} required>
              <option value="">— Odaberi dobavljača —</option>
              {SUPPLIERS.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
        )}

        {invoiceType === 'leasing' && (
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">
              Leasing firma <span className="text-red-400">*</span>
            </label>
            <select value={leasingCompany} onChange={e => setLeasingCompany(e.target.value)} className={inputCls} required>
              <option value="">— Odaberi firmu —</option>
              {LEASING_COMPANIES.map(lc => <option key={lc} value={lc}>{lc}</option>)}
            </select>
          </div>
        )}

        {/* File picker */}
        <div>
          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">
            Dokument <span className="text-red-400">*</span>
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
            {submitting ? 'Učitavanje…' : 'Dodaj račun'}
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

// ── Invoice row ───────────────────────────────────────────────────────────────

function InvoiceRow({ invoice }: { invoice: CompanyInvoice }) {
  const [downloading, setDownloading] = useState(false)
  const [dlErr, setDlErr] = useState('')

  async function handleDownload() {
    setDownloading(true); setDlErr('')
    try { await downloadBlob(`/finance/racuni/${invoice.id}/download`, invoice.document.original_filename) }
    catch (err) { setDlErr(errMsg(err)) }
    finally { setDownloading(false) }
  }

  const badgeCls = INVOICE_TYPE_COLORS[invoice.invoice_type] ?? INVOICE_TYPE_COLORS.oprema

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 hover:bg-gi-card-elevated/40 transition-colors">
      <div className="flex items-start gap-3">
        <div className="flex items-start gap-2.5 flex-1 min-w-0">
          <span className={`shrink-0 mt-0.5 inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold ${badgeCls}`}>
            {INVOICE_TYPE_LABELS[invoice.invoice_type]}
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-sm text-gi-text font-medium truncate">
              {invoice.supplier ?? invoice.leasing_company ?? '—'}
            </p>
            <p className="text-xs text-gi-text-subtle">
              {invoice.created_by_name} · {formatDate(invoice.created_at)}
            </p>
          </div>
        </div>
        <div className="hidden sm:flex flex-col items-end shrink-0 w-44 min-w-0">
          <p className="text-xs text-gi-text truncate max-w-full text-right" title={invoice.document.original_filename}>
            {invoice.document.original_filename}
          </p>
          <p className="text-xs text-gi-text-subtle">{formatFileSize(invoice.document.file_size)}</p>
        </div>
        <div className="flex gap-1.5 items-center shrink-0">
          <DocumentPreviewDialog
            fetchUrl={`/finance/racuni/${invoice.id}/download`}
            filename={invoice.document.original_filename}
            mimeType={invoice.document.mime_type}
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

// ── R1 row (admin view) ───────────────────────────────────────────────────────

function R1Row({ receipt }: { receipt: R1Receipt }) {
  const [downloading, setDownloading] = useState(false)
  const [dlErr, setDlErr] = useState('')

  async function handleDownload() {
    setDownloading(true); setDlErr('')
    try { await downloadBlob(`/finance/r1/${receipt.id}/download`, receipt.document.original_filename) }
    catch { setDlErr('Greška pri preuzimanju') }
    finally { setDownloading(false) }
  }

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 hover:bg-gi-card-elevated/40 transition-colors">
      <div className="flex items-center gap-3">
        <span className="shrink-0 w-28 text-base font-bold text-gi-text tabular-nums">
          {receipt.price.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €
        </span>
        <div className="flex-1 min-w-0">
          <p className="text-sm text-gi-text truncate">{receipt.submitter_name}</p>
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

// ── Gorivo row (admin view, inline confirm) ────────────────────────────────────

function GorivoRow({ receipt, onDelete }: { receipt: FuelReceipt; onDelete: (id: string) => void }) {
  const [downloading, setDownloading] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [err, setErr] = useState('')

  async function handleDownload() {
    setDownloading(true); setErr('')
    try { await downloadBlob(`/finance/gorivo/${receipt.id}/download`, receipt.document.original_filename) }
    catch { setErr('Greška pri preuzimanju') }
    finally { setDownloading(false) }
  }

  async function handleDelete() {
    setDeleting(true); setErr('')
    try {
      await apiClient.delete(`/finance/gorivo/${receipt.id}`)
      onDelete(receipt.id)
    } catch (e) {
      const cast = e as { response?: { data?: { error?: string } } }
      setErr(cast?.response?.data?.error ?? 'Greška pri brisanju')
      setConfirmDelete(false)
    } finally { setDeleting(false) }
  }

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 hover:bg-gi-card-elevated/40 transition-colors">
      <div className="flex items-start gap-3">
        <div className="shrink-0 w-28">
          <p className="text-base font-bold text-gi-text tabular-nums">
            {receipt.amount_eur.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €
          </p>
          <p className="text-xs text-gi-text-subtle">{receipt.receipt_date}</p>
        </div>
        <div className="flex-1 min-w-0 space-y-0.5">
          {(receipt.vehicle_registration || receipt.fuel_type || receipt.fuel_liters != null) && (
            <p className="text-xs text-gi-text-muted">
              {[receipt.vehicle_registration, receipt.fuel_type,
                receipt.fuel_liters != null ? `${receipt.fuel_liters} L` : undefined,
              ].filter(Boolean).join(' · ')}
            </p>
          )}
          <p className="text-xs text-gi-text-subtle">
            {receipt.submitter_name}
            <span className="mx-1 opacity-40">·</span>
            <span className="truncate" title={receipt.document.original_filename}>
              {receipt.document.original_filename}
            </span>
            <span className="mx-1 opacity-40">·</span>
            {formatFileSize(receipt.document.file_size)}
          </p>
          {receipt.notes && <p className="text-xs text-gi-text-subtle/70 italic">{receipt.notes}</p>}
        </div>
        <div className="shrink-0 flex flex-col items-end gap-1.5">
          <div className="flex gap-1.5 items-center">
            <DocumentPreviewDialog
              fetchUrl={`/finance/gorivo/${receipt.id}/download`}
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
          {!confirmDelete ? (
            <button
              onClick={() => setConfirmDelete(true)}
              className="text-xs text-red-400/60 hover:text-red-400 border border-transparent hover:border-red-800/50 px-2.5 py-1 rounded-lg transition-colors"
            >
              Obriši
            </button>
          ) : (
            <div className="flex items-center gap-1.5">
              <span className="text-xs text-gi-text-muted">Obrisati?</span>
              <button
                onClick={handleDelete} disabled={deleting}
                className="text-xs text-red-300 bg-red-950/50 hover:bg-red-900/50 border border-red-800/60 px-2 py-0.5 rounded transition-colors disabled:opacity-50"
              >
                {deleting ? '…' : 'Da'}
              </button>
              <button
                onClick={() => setConfirmDelete(false)} disabled={deleting}
                className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border px-2 py-0.5 rounded transition-colors"
              >
                Ne
              </button>
            </div>
          )}
        </div>
      </div>
      {err && <p className="text-xs text-red-400 mt-1.5">{err}</p>}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

type PageTab = 'invoices' | 'r1' | 'gorivo'

function EmptyState({ message }: { message: string }) {
  return (
    <div className="rounded-xl border border-gi-border bg-gi-card flex flex-col items-center justify-center py-16 gap-3">
      <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
      </svg>
      <p className="text-gi-text-subtle text-sm">{message}</p>
    </div>
  )
}

export default function RacuniPage() {
  const { user, employee, isLoading: authLoading, logout } = useAuth()
  const router = useRouter()
  const [tab, setTab] = useState<PageTab>('invoices')
  const [invoices, setInvoices] = useState<CompanyInvoice[]>([])
  const [r1s, setR1s] = useState<R1Receipt[]>([])
  const [gorivos, setGorivos] = useState<FuelReceipt[]>([])
  const [loading, setLoading] = useState(true)
  const [r1Loading, setR1Loading] = useState(false)
  const [gorivoLoading, setGorivoLoading] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [filterType, setFilterType] = useState<InvoiceType | ''>('')

  const loadInvoices = useCallback(async () => {
    try {
      const resp = await apiClient.get<{ invoices: CompanyInvoice[] }>('/finance/racuni')
      setInvoices(resp.data.invoices ?? [])
    } catch { /* silent */ }
    finally { setLoading(false) }
  }, [])

  const loadR1s = useCallback(async () => {
    setR1Loading(true)
    try {
      const resp = await apiClient.get<{ receipts: R1Receipt[] }>('/finance/r1')
      setR1s(resp.data.receipts ?? [])
    } catch { /* silent */ }
    finally { setR1Loading(false) }
  }, [])

  const loadGorivos = useCallback(async () => {
    setGorivoLoading(true)
    try {
      const resp = await apiClient.get<{ receipts: FuelReceipt[] }>('/finance/gorivo')
      setGorivos(resp.data.receipts ?? [])
    } catch { /* silent */ }
    finally { setGorivoLoading(false) }
  }, [])

  useEffect(() => {
    if (!authLoading && user) {
      if (user.role !== 'direktor' && user.role !== 'administracija') {
        router.replace('/dashboard'); return
      }
      loadInvoices()
    }
  }, [authLoading, user, router, loadInvoices])

  function handleTabChange(next: PageTab) {
    setTab(next)
    setShowForm(false)
    if (next === 'r1' && r1s.length === 0 && !r1Loading) loadR1s()
    if (next === 'gorivo' && gorivos.length === 0 && !gorivoLoading) loadGorivos()
  }

  if (authLoading || !user) return <LoadingScreen />

  const filtered = filterType === '' ? invoices : invoices.filter(inv => inv.invoice_type === filterType)

  const tabCls = (active: boolean) => [
    'px-4 py-2.5 text-sm font-medium border-b-2 transition-colors whitespace-nowrap',
    active
      ? 'border-gi-yellow text-gi-yellow'
      : 'border-transparent text-gi-text-muted hover:text-gi-text',
  ].join(' ')

  const filterInputCls = [
    'flex-1 bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
    'text-sm text-gi-text focus:outline-none focus:border-gi-yellow transition-colors',
  ].join(' ')

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Računi"
      backHref="/dashboard"
      onLogout={logout}
    >
      <div className="max-w-5xl space-y-5">
        {/* Page header */}
        <div>
          <h1 className="text-xl font-bold text-gi-text tracking-tight">Računi</h1>
          <p className="text-sm text-gi-text-subtle mt-0.5">Računi tvrtke, R1 računi i gorivo</p>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gi-border">
          <button className={tabCls(tab === 'invoices')} onClick={() => handleTabChange('invoices')}>
            Računi tvrtke
          </button>
          <button className={tabCls(tab === 'r1')} onClick={() => handleTabChange('r1')}>
            R1 računi
          </button>
          <button className={tabCls(tab === 'gorivo')} onClick={() => handleTabChange('gorivo')}>
            Gorivo
          </button>
        </div>

        {/* ── Računi tab ── */}
        {tab === 'invoices' && (
          <div className={showForm ? 'flex flex-col lg:flex-row gap-5 items-start' : 'space-y-3'}>
            {/* Register + filter */}
            <div className={showForm ? 'flex-1 min-w-0 space-y-3' : 'space-y-3'}>
              <div className="flex gap-2 items-center">
                <select
                  value={filterType}
                  onChange={e => setFilterType(e.target.value as InvoiceType | '')}
                  className={filterInputCls}
                >
                  <option value="">Sve vrste</option>
                  {(Object.entries(INVOICE_TYPE_LABELS) as [InvoiceType, string][]).map(([k, v]) => (
                    <option key={k} value={k}>{v}</option>
                  ))}
                </select>
                <button
                  onClick={() => setShowForm(f => !f)}
                  className="px-3 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors whitespace-nowrap shrink-0"
                >
                  {showForm ? '× Zatvori' : '+ Novi račun'}
                </button>
              </div>

              {loading ? (
                <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
                  <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje računa…</p>
                </div>
              ) : filtered.length === 0 ? (
                <EmptyState message={filterType ? 'Nema računa za odabranu vrstu.' : 'Još nema unesenih računa.'} />
              ) : (
                <div className="space-y-2">
                  {filtered.map(inv => <InvoiceRow key={inv.id} invoice={inv} />)}
                </div>
              )}
            </div>

            {/* Form panel */}
            {showForm && (
              <div className="w-full lg:w-[380px] shrink-0 lg:sticky lg:top-5">
                <UploadForm
                  onSuccess={() => { setShowForm(false); loadInvoices() }}
                  onCancel={() => setShowForm(false)}
                />
              </div>
            )}
          </div>
        )}

        {/* ── R1 tab ── */}
        {tab === 'r1' && (
          <div className="space-y-3">
            <div className="flex justify-end">
              <button
                onClick={loadR1s}
                className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors"
              >
                Osvježi
              </button>
            </div>
            {r1Loading ? (
              <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
                <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje R1 računa…</p>
              </div>
            ) : r1s.length === 0 ? (
              <EmptyState message="Još nema predanih R1 računa." />
            ) : (
              <div className="space-y-2">
                {r1s.map(rec => <R1Row key={rec.id} receipt={rec} />)}
              </div>
            )}
          </div>
        )}

        {/* ── Gorivo tab ── */}
        {tab === 'gorivo' && (
          <div className="space-y-3">
            <div className="flex justify-end">
              <button
                onClick={loadGorivos}
                className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors"
              >
                Osvježi
              </button>
            </div>
            {gorivoLoading ? (
              <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
                <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje računa za gorivo…</p>
              </div>
            ) : gorivos.length === 0 ? (
              <EmptyState message="Još nema predanih gorivo računa." />
            ) : (
              <div className="space-y-2">
                {gorivos.map(rec => (
                  <GorivoRow
                    key={rec.id}
                    receipt={rec}
                    onDelete={id => setGorivos(prev => prev.filter(r => r.id !== id))}
                  />
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </AppShell>
  )
}
