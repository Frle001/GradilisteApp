'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import DocumentPreviewDialog from '@/components/ui/DocumentPreviewDialog'
import { type FuelReceipt } from '@/lib/types/finance'

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2',
  'text-sm text-gi-text placeholder:text-gi-text-subtle',
  'focus:outline-none focus:border-gi-yellow',
  'disabled:opacity-40 disabled:cursor-not-allowed',
  'transition-colors',
].join(' ')

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

function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

// ── Upload form ───────────────────────────────────────────────────────────────

function UploadForm({ onSuccess, onCancel }: { onSuccess: () => void; onCancel: () => void }) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [receiptDate, setReceiptDate] = useState(todayISO())
  const [amountEur, setAmountEur] = useState('')
  const [notes, setNotes] = useState('')
  const [vehicleReg, setVehicleReg] = useState('')
  const [fuelType, setFuelType] = useState('')
  const [fuelLiters, setFuelLiters] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const file = fileRef.current?.files?.[0]
    if (!file) { setError('Odaberite dokument'); return }
    const amount = parseFloat(amountEur)
    if (!amountEur || isNaN(amount) || amount <= 0) {
      setError('Iznos mora biti pozitivan broj'); return
    }
    const fd = new FormData()
    fd.append('file', file)
    fd.append('receipt_date', receiptDate)
    fd.append('amount_eur', amount.toFixed(2))
    if (notes) fd.append('notes', notes)
    if (vehicleReg) fd.append('vehicle_registration', vehicleReg)
    if (fuelType) fd.append('fuel_type', fuelType)
    if (fuelLiters) fd.append('fuel_liters', fuelLiters)
    setSubmitting(true); setError('')
    try {
      await apiClient.post('/finance/gorivo', fd)
      if (fileRef.current) fileRef.current.value = ''
      setSelectedFile(null)
      setAmountEur(''); setNotes(''); setVehicleReg(''); setFuelType(''); setFuelLiters('')
      setReceiptDate(todayISO())
      onSuccess()
    } catch (err) { setError(errMsg(err)) }
    finally { setSubmitting(false) }
  }

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card p-4">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-semibold text-gi-text">Novi račun za gorivo</h2>
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
        {/* Date + Amount */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">
              Datum računa <span className="text-red-400">*</span>
            </label>
            <input
              type="date" value={receiptDate} onChange={e => setReceiptDate(e.target.value)}
              className={inputCls} required
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">
              Iznos (€) <span className="text-red-400">*</span>
            </label>
            <input
              type="number" step="0.01" min="0.01"
              value={amountEur} onChange={e => setAmountEur(e.target.value)}
              placeholder="npr. 85.00" className={inputCls} required
            />
          </div>
        </div>

        {/* Vehicle + Fuel type */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Registracija vozila</label>
            <input
              type="text" value={vehicleReg} onChange={e => setVehicleReg(e.target.value)}
              placeholder="npr. ZG-1234-AB" className={inputCls}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Vrsta goriva</label>
            <input
              type="text" value={fuelType} onChange={e => setFuelType(e.target.value)}
              placeholder="npr. Diesel" className={inputCls}
            />
          </div>
        </div>

        {/* Liters */}
        <div>
          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Količina (L)</label>
          <input
            type="number" step="0.01" min="0.01"
            value={fuelLiters} onChange={e => setFuelLiters(e.target.value)}
            placeholder="npr. 45.00" className={inputCls}
          />
        </div>

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
              accept=".pdf,.jpg,.jpeg,.png,.doc,.docx"
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
                    <p className="text-xs text-gi-text-subtle">PDF, JPG, PNG, DOC</p>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Notes */}
        <div>
          <label className="block text-xs font-medium text-gi-text-muted mb-1.5">Napomena</label>
          <textarea
            value={notes} onChange={e => setNotes(e.target.value)}
            placeholder="Opcionalna napomena…"
            rows={2}
            className={inputCls + ' resize-none'}
          />
        </div>

        {error && <p className="text-sm text-red-400">{error}</p>}

        <div className="flex gap-2 pt-1">
          <button
            type="submit" disabled={submitting}
            className="flex-1 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors disabled:opacity-50"
          >
            {submitting ? 'Učitavanje…' : 'Predaj račun za gorivo'}
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

function ReceiptRow({ receipt, isAdmin, onDelete }: {
  receipt: FuelReceipt; isAdmin: boolean; onDelete?: (id: string) => void
}) {
  const [downloading, setDownloading] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [err, setErr] = useState('')

  async function handleDownload() {
    setDownloading(true); setErr('')
    try {
      const resp = await apiClient.get(`/finance/gorivo/${receipt.id}/download`, { responseType: 'blob' })
      const url = window.URL.createObjectURL(resp.data as Blob)
      const a = document.createElement('a'); a.href = url; a.download = receipt.document.original_filename; a.click()
      window.URL.revokeObjectURL(url)
    } catch (e) { setErr(errMsg(e)) }
    finally { setDownloading(false) }
  }

  async function handleDelete() {
    setDeleting(true); setErr('')
    try {
      await apiClient.delete(`/finance/gorivo/${receipt.id}`)
      onDelete?.(receipt.id)
    } catch (e) {
      setErr(errMsg(e))
      setConfirmDelete(false)
    } finally { setDeleting(false) }
  }

  return (
    <div className="rounded-xl border border-gi-border bg-gi-card px-4 py-3 hover:bg-gi-card-elevated/40 transition-colors">
      <div className="flex items-start gap-3">
        {/* Amount + date (fixed width) */}
        <div className="shrink-0 w-28 sm:w-32">
          <p className="text-base font-bold text-gi-text tabular-nums">
            {receipt.amount_eur.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €
          </p>
          <p className="text-xs text-gi-text-subtle">{receipt.receipt_date}</p>
        </div>

        {/* Vehicle + meta */}
        <div className="flex-1 min-w-0 space-y-0.5">
          {(receipt.vehicle_registration || receipt.fuel_type || receipt.fuel_liters != null) && (
            <p className="text-xs text-gi-text-muted">
              {[
                receipt.vehicle_registration,
                receipt.fuel_type,
                receipt.fuel_liters != null ? `${receipt.fuel_liters} L` : undefined,
              ].filter(Boolean).join(' · ')}
            </p>
          )}
          {isAdmin && (
            <p className="text-xs text-gi-text-subtle">Predao: {receipt.submitter_name}</p>
          )}
          {receipt.notes && (
            <p className="text-xs text-gi-text-subtle/70 italic">{receipt.notes}</p>
          )}
          <p className="text-xs text-gi-text-subtle truncate" title={receipt.document.original_filename}>
            {receipt.document.original_filename}
            <span className="mx-1 opacity-40">·</span>
            {formatFileSize(receipt.document.file_size)}
            <span className="mx-1 opacity-40">·</span>
            {formatDate(receipt.created_at)}
          </p>
        </div>

        {/* Actions */}
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

          {isAdmin && (
            !confirmDelete ? (
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
            )
          )}
        </div>
      </div>
      {err && <p className="text-xs text-red-400 mt-1.5">{err}</p>}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function GorivoPage() {
  const { user, employee, isLoading: authLoading, logout } = useAuth()
  const [receipts, setReceipts] = useState<FuelReceipt[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [showForm, setShowForm] = useState(false)

  const isAdmin = user?.role === 'direktor' || user?.role === 'administracija'

  const loadReceipts = useCallback(async () => {
    setLoading(true); setLoadError('')
    try {
      const resp = await apiClient.get<{ receipts: FuelReceipt[] }>('/finance/gorivo')
      setReceipts(resp.data.receipts ?? [])
    } catch (e) {
      setLoadError(errMsg(e))
    } finally { setLoading(false) }
  }, [])

  useEffect(() => {
    if (!authLoading && user) loadReceipts()
  }, [authLoading, user, loadReceipts])

  if (authLoading || !user) return <LoadingScreen />

  const heading = isAdmin ? 'Gorivo računi' : 'Moji gorivo računi'
  const subtitle = isAdmin
    ? 'Pregled svih predanih gorivo računa'
    : 'Vaši predani računi za gorivo'

  return (
    <AppShell
      user={user}
      employee={employee}
      title={heading}
      backHref="/dashboard"
      onLogout={logout}
    >
      <div className="max-w-5xl space-y-5">
        {/* Header */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-gi-text tracking-tight">{heading}</h1>
            <p className="text-sm text-gi-text-subtle mt-0.5">{subtitle}</p>
          </div>
          <button
            onClick={() => setShowForm(f => !f)}
            className="px-3 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors whitespace-nowrap shrink-0"
          >
            {showForm ? '× Zatvori' : '+ Predaj račun'}
          </button>
        </div>

        {/* Two-col layout on wide screens when form is open */}
        <div className={showForm ? 'flex flex-col lg:flex-row gap-5 items-start' : ''}>
          {/* Receipt list */}
          <div className={showForm ? 'flex-1 min-w-0' : 'w-full'}>
            {loading ? (
              <div className="rounded-xl border border-gi-border bg-gi-card py-16 text-center">
                <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje gorivo računa…</p>
              </div>
            ) : loadError ? (
              <div className="rounded-xl border border-red-900/50 bg-red-950/20 flex flex-col items-center justify-center py-16 gap-3">
                <svg className="w-8 h-8 text-red-400/70" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
                <div className="text-center space-y-2">
                  <p className="text-sm font-medium text-red-300">Greška pri učitavanju</p>
                  <p className="text-xs text-red-400/80 max-w-xs">{loadError}</p>
                  <button
                    onClick={loadReceipts}
                    className="mt-1 text-xs text-gi-yellow hover:text-gi-yellow-hover border border-gi-border hover:border-gi-text-subtle px-3 py-1.5 rounded-lg transition-colors"
                  >
                    Pokušaj ponovo
                  </button>
                </div>
              </div>
            ) : receipts.length === 0 ? (
              <div className="rounded-xl border border-gi-border bg-gi-card flex flex-col items-center justify-center py-16 gap-3">
                <svg className="w-10 h-10 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 3c2.755 0 5.455.232 8.083.678.533.09.917.556.917 1.096v1.044a2.25 2.25 0 01-.659 1.591l-5.432 5.432a2.25 2.25 0 00-.659 1.591v2.927a2.25 2.25 0 01-1.244 2.013L9.75 21v-6.568a2.25 2.25 0 00-.659-1.591L3.659 7.409A2.25 2.25 0 013 5.818V4.774c0-.54.384-1.006.917-1.096A48.32 48.32 0 0112 3z" />
                </svg>
                <div className="text-center">
                  <p className="text-gi-text-subtle text-sm font-medium">
                    {isAdmin ? 'Još nema predanih gorivo računa.' : 'Još niste predali nijedan gorivo račun.'}
                  </p>
                  {!isAdmin && !showForm && (
                    <button
                      onClick={() => setShowForm(true)}
                      className="mt-2 text-xs text-gi-yellow hover:text-gi-yellow-hover transition-colors"
                    >
                      Predaj prvi račun →
                    </button>
                  )}
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                {receipts.map(rec => (
                  <ReceiptRow
                    key={rec.id}
                    receipt={rec}
                    isAdmin={isAdmin}
                    onDelete={id => setReceipts(prev => prev.filter(r => r.id !== id))}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Form panel */}
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
