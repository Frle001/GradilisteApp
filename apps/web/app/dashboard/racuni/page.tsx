'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DocumentPreviewDialog from '@/components/ui/DocumentPreviewDialog'
import {
  CompanyInvoice,
  InvoiceType,
  INVOICE_TYPE_LABELS,
  SUPPLIERS,
  LEASING_COMPANIES,
  R1Receipt,
} from '@/lib/types/finance'

const inputCls =
  'w-full rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500'

const btnPrimary =
  'rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors'

const btnSecondary =
  'rounded-lg border border-slate-600 bg-slate-800 px-4 py-2 text-sm font-medium text-slate-300 hover:bg-slate-700 transition-colors'

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

async function downloadFile(invoiceId: string, filename: string) {
  const resp = await apiClient.get(`/finance/racuni/${invoiceId}/download`, {
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
  const [invoiceType, setInvoiceType] = useState<InvoiceType>('gorivo')
  const [supplier, setSupplier] = useState('')
  const [leasingCompany, setLeasingCompany] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const file = fileRef.current?.files?.[0]
    if (!file) {
      setError('Odaberite datoteku')
      return
    }

    const fd = new FormData()
    fd.append('file', file)
    fd.append('invoice_type', invoiceType)
    if (invoiceType === 'materijal' && supplier) fd.append('supplier', supplier)
    if (invoiceType === 'leasing' && leasingCompany) fd.append('leasing_company', leasingCompany)

    setSubmitting(true)
    setError('')
    try {
      await apiClient.post('/finance/racuni', fd)
      if (fileRef.current) fileRef.current.value = ''
      setSupplier('')
      setLeasingCompany('')
      setInvoiceType('gorivo')
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
        <label className="block text-sm font-medium text-slate-300 mb-1">Vrsta računa</label>
        <select
          value={invoiceType}
          onChange={(e) => {
            setInvoiceType(e.target.value as InvoiceType)
            setSupplier('')
            setLeasingCompany('')
          }}
          className={inputCls}
        >
          {(Object.entries(INVOICE_TYPE_LABELS) as [InvoiceType, string][]).map(([k, v]) => (
            <option key={k} value={k}>{v}</option>
          ))}
        </select>
      </div>

      {invoiceType === 'materijal' && (
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Dobavljač *</label>
          <select
            value={supplier}
            onChange={(e) => setSupplier(e.target.value)}
            className={inputCls}
            required
          >
            <option value="">-- Odaberi dobavljača --</option>
            {SUPPLIERS.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </div>
      )}

      {invoiceType === 'leasing' && (
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Leasing firma *</label>
          <select
            value={leasingCompany}
            onChange={(e) => setLeasingCompany(e.target.value)}
            className={inputCls}
            required
          >
            <option value="">-- Odaberi firmu --</option>
            {LEASING_COMPANIES.map((lc) => (
              <option key={lc} value={lc}>{lc}</option>
            ))}
          </select>
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Dokument *</label>
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
        {submitting ? 'Učitavanje...' : 'Dodaj račun'}
      </button>
    </form>
  )
}

// ── Invoice list ──────────────────────────────────────────────────────────────

interface InvoiceRowProps {
  invoice: CompanyInvoice
}

function InvoiceRow({ invoice }: InvoiceRowProps) {
  const [downloading, setDownloading] = useState(false)
  const [dlErr, setDlErr] = useState('')

  async function handleDownload() {
    setDownloading(true)
    setDlErr('')
    try {
      await downloadFile(invoice.id, invoice.document.original_filename)
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
          <span className="inline-block rounded-full bg-blue-900/60 px-2.5 py-0.5 text-xs font-medium text-blue-300 mb-1">
            {INVOICE_TYPE_LABELS[invoice.invoice_type]}
          </span>
          {invoice.supplier && (
            <p className="text-sm text-slate-300">Dobavljač: <span className="text-white">{invoice.supplier}</span></p>
          )}
          {invoice.leasing_company && (
            <p className="text-sm text-slate-300">Firma: <span className="text-white">{invoice.leasing_company}</span></p>
          )}
          <p className="text-xs text-slate-500 mt-1">
            {invoice.document.original_filename} · {formatFileSize(invoice.document.file_size)}
          </p>
          <p className="text-xs text-slate-500">
            Dodao: {invoice.created_by_name} · {formatDate(invoice.created_at)}
          </p>
        </div>
        <div className="flex gap-2 items-center shrink-0">
          <DocumentPreviewDialog
            fetchUrl={`/finance/racuni/${invoice.id}/download`}
            filename={invoice.document.original_filename}
            mimeType={invoice.document.mime_type}
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

// ── R1 receipt row (read-only, for the R1 tab in Računi page) ─────────────────

interface R1RowProps {
  receipt: R1Receipt
}

function R1Row({ receipt }: R1RowProps) {
  const [downloading, setDownloading] = useState(false)
  const [dlErr, setDlErr] = useState('')

  async function handleDownload() {
    setDownloading(true)
    setDlErr('')
    try {
      const resp = await apiClient.get(`/finance/r1/${receipt.id}/download`, { responseType: 'blob' })
      const url = window.URL.createObjectURL(resp.data as Blob)
      const a = document.createElement('a')
      a.href = url
      a.download = receipt.document.original_filename
      a.click()
      window.URL.revokeObjectURL(url)
    } catch {
      setDlErr('Greška pri preuzimanju')
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
          <p className="text-sm text-slate-300">Predao: <span className="text-white">{receipt.submitter_name}</span></p>
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

type PageTab = 'invoices' | 'r1'

export default function RacuniPage() {
  const { user, isLoading: authLoading } = useAuth()
  const router = useRouter()
  const [tab, setTab] = useState<PageTab>('invoices')
  const [invoices, setInvoices] = useState<CompanyInvoice[]>([])
  const [r1s, setR1s] = useState<R1Receipt[]>([])
  const [loading, setLoading] = useState(true)
  const [r1Loading, setR1Loading] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [filterType, setFilterType] = useState<InvoiceType | ''>('')

  const loadInvoices = useCallback(async () => {
    try {
      const resp = await apiClient.get<{ invoices: CompanyInvoice[] }>('/finance/racuni')
      setInvoices(resp.data.invoices ?? [])
    } catch {
      // handled silently
    } finally {
      setLoading(false)
    }
  }, [])

  const loadR1s = useCallback(async () => {
    setR1Loading(true)
    try {
      const resp = await apiClient.get<{ receipts: R1Receipt[] }>('/finance/r1')
      setR1s(resp.data.receipts ?? [])
    } catch {
      // handled silently
    } finally {
      setR1Loading(false)
    }
  }, [])

  useEffect(() => {
    if (!authLoading && user) {
      if (user.role !== 'direktor' && user.role !== 'administracija') {
        router.replace('/dashboard')
        return
      }
      loadInvoices()
    }
  }, [authLoading, user, router, loadInvoices])

  function handleTabChange(next: PageTab) {
    setTab(next)
    if (next === 'r1' && r1s.length === 0 && !r1Loading) {
      loadR1s()
    }
  }

  if (authLoading || !user) return <LoadingScreen />

  const filtered =
    filterType === ''
      ? invoices
      : invoices.filter((inv) => inv.invoice_type === filterType)

  const tabCls = (active: boolean) =>
    `px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
      active
        ? 'border-blue-500 text-white'
        : 'border-transparent text-slate-400 hover:text-slate-200'
    }`

  return (
    <main className="min-h-screen bg-slate-950 text-white px-4 py-6 max-w-2xl mx-auto">
      <div className="flex items-center gap-3 mb-6">
        <button onClick={() => router.back()} className="text-slate-400 hover:text-white transition-colors">
          ←
        </button>
        <h1 className="text-xl font-bold">Računi</h1>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-slate-700 mb-4">
        <button className={tabCls(tab === 'invoices')} onClick={() => handleTabChange('invoices')}>
          Računi
        </button>
        <button className={tabCls(tab === 'r1')} onClick={() => handleTabChange('r1')}>
          R1 računi
        </button>
      </div>

      {/* ── Računi tab ── */}
      {tab === 'invoices' && (
        <>
          <div className="mb-4 flex gap-2 items-center">
            <select
              value={filterType}
              onChange={(e) => setFilterType(e.target.value as InvoiceType | '')}
              className="rounded-lg border border-slate-600 bg-slate-800 px-3 py-1.5 text-sm text-white flex-1"
            >
              <option value="">Sve vrste</option>
              {(Object.entries(INVOICE_TYPE_LABELS) as [InvoiceType, string][]).map(([k, v]) => (
                <option key={k} value={k}>{v}</option>
              ))}
            </select>
            <button onClick={() => setShowForm((f) => !f)} className={btnPrimary}>
              {showForm ? 'Zatvori' : '+ Novi račun'}
            </button>
          </div>

          {showForm && (
            <div className="mb-6 rounded-xl border border-slate-700 bg-slate-900 p-4">
              <h2 className="text-base font-semibold mb-4">Dodaj račun</h2>
              <UploadForm
                onSuccess={() => {
                  setShowForm(false)
                  loadInvoices()
                }}
              />
            </div>
          )}

          {loading ? (
            <div className="text-center py-12 text-slate-400">Učitavanje...</div>
          ) : filtered.length === 0 ? (
            <div className="text-center py-12 text-slate-400">
              {filterType ? 'Nema računa za odabranu vrstu.' : 'Još nema unesenih računa.'}
            </div>
          ) : (
            <div className="space-y-3">
              {filtered.map((inv) => (
                <InvoiceRow key={inv.id} invoice={inv} />
              ))}
            </div>
          )}
        </>
      )}

      {/* ── R1 tab ── */}
      {tab === 'r1' && (
        <>
          <div className="mb-4 flex justify-end">
            <button onClick={loadR1s} className={btnSecondary + ' text-xs py-1.5'}>
              Osvježi
            </button>
          </div>
          {r1Loading ? (
            <div className="text-center py-12 text-slate-400">Učitavanje...</div>
          ) : r1s.length === 0 ? (
            <div className="text-center py-12 text-slate-400">Još nema predanih R1 računa.</div>
          ) : (
            <div className="space-y-3">
              {r1s.map((rec) => (
                <R1Row key={rec.id} receipt={rec} />
              ))}
            </div>
          )}
        </>
      )}
    </main>
  )
}
