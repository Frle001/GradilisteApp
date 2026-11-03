'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import apiClient from '@/lib/api-client'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DocumentPreviewDialog from '@/components/ui/DocumentPreviewDialog'
import { FuelReceipt } from '@/lib/types/finance'

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

function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

// ── Upload form ───────────────────────────────────────────────────────────────

interface UploadFormProps {
  onSuccess: () => void
}

function UploadForm({ onSuccess }: UploadFormProps) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [receiptDate, setReceiptDate] = useState(todayISO())
  const [amountEur, setAmountEur] = useState('')
  const [notes, setNotes] = useState('')
  const [vehicleReg, setVehicleReg] = useState('')
  const [fuelType, setFuelType] = useState('')
  const [fuelLiters, setFuelLiters] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const file = fileRef.current?.files?.[0]
    if (!file) {
      setError('Odaberite dokument')
      return
    }
    const amount = parseFloat(amountEur)
    if (!amountEur || isNaN(amount) || amount <= 0) {
      setError('Iznos mora biti pozitivan broj')
      return
    }

    const fd = new FormData()
    fd.append('file', file)
    fd.append('receipt_date', receiptDate)
    fd.append('amount_eur', amount.toFixed(2))
    if (notes) fd.append('notes', notes)
    if (vehicleReg) fd.append('vehicle_registration', vehicleReg)
    if (fuelType) fd.append('fuel_type', fuelType)
    if (fuelLiters) fd.append('fuel_liters', fuelLiters)

    setSubmitting(true)
    setError('')
    try {
      await apiClient.post('/finance/gorivo', fd)
      if (fileRef.current) fileRef.current.value = ''
      setAmountEur('')
      setNotes('')
      setVehicleReg('')
      setFuelType('')
      setFuelLiters('')
      setReceiptDate(todayISO())
      onSuccess()
    } catch (err) {
      setError(errMsg(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Datum računa *</label>
          <input
            type="date"
            value={receiptDate}
            onChange={(e) => setReceiptDate(e.target.value)}
            className={inputCls}
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Iznos (€) *</label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            value={amountEur}
            onChange={(e) => setAmountEur(e.target.value)}
            placeholder="npr. 85.00"
            className={inputCls}
            required
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Dokument (slika/PDF) *</label>
        <input
          ref={fileRef}
          type="file"
          accept=".pdf,.jpg,.jpeg,.png,.doc,.docx"
          className={inputCls}
          required
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Registracija vozila</label>
          <input
            type="text"
            value={vehicleReg}
            onChange={(e) => setVehicleReg(e.target.value)}
            placeholder="npr. ZG-1234-AB"
            className={inputCls}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Vrsta goriva</label>
          <input
            type="text"
            value={fuelType}
            onChange={(e) => setFuelType(e.target.value)}
            placeholder="npr. Diesel"
            className={inputCls}
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Količina (L)</label>
        <input
          type="number"
          step="0.01"
          min="0.01"
          value={fuelLiters}
          onChange={(e) => setFuelLiters(e.target.value)}
          placeholder="npr. 45.00"
          className={inputCls}
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Napomena</label>
        <textarea
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          placeholder="Opcionalna napomena..."
          rows={2}
          className={inputCls}
        />
      </div>

      {error && <p className="text-sm text-red-400">{error}</p>}

      <button type="submit" className={btnPrimary} disabled={submitting}>
        {submitting ? 'Učitavanje...' : 'Predaj račun za gorivo'}
      </button>
    </form>
  )
}

// ── Receipt row ───────────────────────────────────────────────────────────────

interface ReceiptRowProps {
  receipt: FuelReceipt
  isAdmin: boolean
  onDelete?: (id: string) => void
}

function ReceiptRow({ receipt, isAdmin, onDelete }: ReceiptRowProps) {
  const [downloading, setDownloading] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [err, setErr] = useState('')

  async function handleDownload() {
    setDownloading(true)
    setErr('')
    try {
      const resp = await apiClient.get(`/finance/gorivo/${receipt.id}/download`, { responseType: 'blob' })
      const url = window.URL.createObjectURL(resp.data as Blob)
      const a = document.createElement('a')
      a.href = url
      a.download = receipt.document.original_filename
      a.click()
      window.URL.revokeObjectURL(url)
    } catch (e) {
      setErr(errMsg(e))
    } finally {
      setDownloading(false)
    }
  }

  async function handleDelete() {
    if (!confirm('Obrisati ovaj račun za gorivo?')) return
    setDeleting(true)
    setErr('')
    try {
      await apiClient.delete(`/finance/gorivo/${receipt.id}`)
      onDelete?.(receipt.id)
    } catch (e) {
      setErr(errMsg(e))
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-800/50 p-4 space-y-2">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <p className="text-lg font-semibold text-white">
              {receipt.amount_eur.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €
            </p>
            <span className="text-xs text-slate-400">{receipt.receipt_date}</span>
          </div>
          {isAdmin && (
            <p className="text-sm text-slate-300">Predao: <span className="text-white">{receipt.submitter_name}</span></p>
          )}
          {receipt.vehicle_registration && (
            <p className="text-xs text-slate-400">Vozilo: {receipt.vehicle_registration}
              {receipt.fuel_type && ` · ${receipt.fuel_type}`}
              {receipt.fuel_liters != null && ` · ${receipt.fuel_liters} L`}
            </p>
          )}
          {receipt.notes && (
            <p className="text-xs text-slate-500 mt-0.5">{receipt.notes}</p>
          )}
          <p className="text-xs text-slate-500 mt-1">
            {receipt.document.original_filename} · {formatFileSize(receipt.document.file_size)}
          </p>
          <p className="text-xs text-slate-500">{formatDate(receipt.created_at)}</p>
        </div>
        <div className="flex gap-2 items-center shrink-0 flex-wrap justify-end">
          <DocumentPreviewDialog
            fetchUrl={`/finance/gorivo/${receipt.id}/download`}
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
          {isAdmin && (
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="rounded-lg border border-red-800 bg-red-950/40 px-3 py-1.5 text-xs font-medium text-red-400 hover:bg-red-900/50 transition-colors disabled:opacity-50"
            >
              {deleting ? '...' : 'Obriši'}
            </button>
          )}
        </div>
      </div>
      {err && <p className="text-xs text-red-400">{err}</p>}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function GorivoPage() {
  const { user, isLoading: authLoading } = useAuth()
  const router = useRouter()
  const [receipts, setReceipts] = useState<FuelReceipt[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)

  const isAdmin = user?.role === 'direktor' || user?.role === 'administracija'

  const loadReceipts = useCallback(async () => {
    try {
      const resp = await apiClient.get<{ receipts: FuelReceipt[] }>('/finance/gorivo')
      setReceipts(resp.data.receipts ?? [])
    } catch {
      // handled silently
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!authLoading && user) {
      loadReceipts()
    }
  }, [authLoading, user, loadReceipts])

  if (authLoading || !user) return <LoadingScreen />

  const heading = isAdmin ? 'Gorivo računi' : 'Moji gorivo računi'

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
          {showForm ? 'Zatvori' : '+ Predaj račun'}
        </button>
      </div>

      {showForm && (
        <div className="mb-6 rounded-xl border border-slate-700 bg-slate-900 p-4">
          <h2 className="text-base font-semibold mb-4">Novi račun za gorivo</h2>
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
          {isAdmin ? 'Još nema predanih gorivo računa.' : 'Još niste predali nijedan gorivo račun.'}
        </div>
      ) : (
        <div className="space-y-3">
          {receipts.map((rec) => (
            <ReceiptRow
              key={rec.id}
              receipt={rec}
              isAdmin={isAdmin}
              onDelete={(id) => setReceipts((prev) => prev.filter((r) => r.id !== id))}
            />
          ))}
        </div>
      )}
    </main>
  )
}
