'use client'

import { useEffect, useState } from 'react'
import apiClient from '@/lib/api-client'
import type {
  InventoryAssetItem,
  InventoryMaterialItem,
  InventoryProject,
  TransferTarget,
} from '@/lib/types/inventory'
import { ROLE_LABELS } from '@/lib/types/inventory'

type TransferSubject =
  | { kind: 'asset'; item: InventoryAssetItem }
  | { kind: 'material'; item: InventoryMaterialItem }

interface Props {
  subject: TransferSubject | null
  onClose: () => void
  onSuccess: () => void
  onNeedsRefresh?: () => void
}

const inputCls = [
  'w-full bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text text-sm',
  'placeholder:text-gi-text-subtle px-3.5 py-2.5',
  'focus:outline-none focus:border-gi-yellow transition-colors',
  'disabled:opacity-40 disabled:cursor-not-allowed',
].join(' ')

export default function TransferModal({ subject, onClose, onSuccess, onNeedsRefresh }: Props) {
  const [targets, setTargets] = useState<TransferTarget[]>([])
  const [loadingTargets, setLoadingTargets] = useState(false)
  const [toEmployeeID, setToEmployeeID] = useState('')
  const [recipientProjects, setRecipientProjects] = useState<InventoryProject[]>([])
  const [loadingProjects, setLoadingProjects] = useState(false)
  const [destinationProjectID, setDestinationProjectID] = useState('')
  const [quantity, setQuantity] = useState('')
  const [notes, setNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!subject) return
    setToEmployeeID('')
    setRecipientProjects([])
    setDestinationProjectID('')
    setQuantity('')
    setNotes('')
    setError(null)
    setLoadingTargets(true)
    const transferType = subject.kind
    apiClient.get(`/inventory/transfer-targets?transfer_type=${transferType}`)
      .then(res => setTargets(res.data.targets ?? []))
      .catch(() => setTargets([]))
      .finally(() => setLoadingTargets(false))
  }, [subject])

  useEffect(() => {
    if (!subject || subject.kind !== 'material' || !toEmployeeID) {
      setRecipientProjects([])
      setDestinationProjectID('')
      return
    }
    setLoadingProjects(true)
    setDestinationProjectID('')
    apiClient.get(`/inventory/employee-projects?employee_id=${toEmployeeID}`)
      .then(res => setRecipientProjects(res.data.projects ?? []))
      .catch(() => setRecipientProjects([]))
      .finally(() => setLoadingProjects(false))
  }, [toEmployeeID, subject])

  if (!subject) return null

  const isMaterial = subject.kind === 'material'
  const title = isMaterial
    ? `Prijenos: ${subject.item.material_name}`
    : `Prijenos: ${subject.item.name}`

  const maxQty = isMaterial ? subject.item.total_quantity : 1

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!subject) return
    if (!toEmployeeID) { setError('Odaberite primatelja.'); return }
    if (subject.kind === 'material' && (!quantity || parseFloat(quantity) <= 0)) {
      setError('Unesite količinu za prijenos.')
      return
    }
    if (subject.kind === 'material' && !destinationProjectID) {
      setError('Odaberite odredišni projekt za prijenos.')
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      const body = subject.kind === 'material'
        ? {
            to_employee_id: toEmployeeID,
            type: 'material',
            project_material_id: subject.item.project_material_id,
            destination_project_id: destinationProjectID,
            quantity: parseFloat(quantity),
            notes,
          }
        : {
            to_employee_id: toEmployeeID,
            type: 'asset',
            employee_asset_id: subject.item.id,
            notes,
          }
      await apiClient.post('/inventory/transfers', body)
      onSuccess()
      onClose()
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { error?: string } } }
      const msg = e?.response?.data?.error ?? 'Greška pri prijenosu.'
      setError(msg)
      // 409 means project quantities changed since form loaded — refresh displayed values
      if (e?.response?.status === 409) {
        onNeedsRefresh?.()
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-end md:items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="bg-gi-card border border-gi-border rounded-2xl w-full max-w-md shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gi-border">
          <h2 className="text-base font-semibold text-gi-text">{title}</h2>
          <button
            onClick={onClose}
            className="text-gi-text-subtle hover:text-gi-text transition-colors p-1 rounded"
            aria-label="Zatvori"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {/* Available quantity summary */}
          {isMaterial && (
            <div className="bg-gi-card-elevated border border-gi-border rounded-lg px-4 py-3 text-sm">
              <span className="text-gi-text-muted">Dostupno: </span>
              <span className="font-semibold text-gi-text">{subject.item.total_quantity.toFixed(2)} {subject.item.unit}</span>
              <span className="text-gi-text-subtle ml-2">· {subject.item.project_name}</span>
            </div>
          )}

          {/* Recipient */}
          <div>
            <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
              Primatelj <span className="text-red-400">*</span>
            </label>
            {loadingTargets ? (
              <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje primatelja...</p>
            ) : targets.length === 0 ? (
              <p className="text-gi-text-subtle text-sm">
                {isMaterial
                  ? 'Nema dostupnih poslovođa za prijenos materijala.'
                  : 'Nema dostupnih primatelja.'}
              </p>
            ) : (
              <select
                className={inputCls}
                value={toEmployeeID}
                onChange={e => setToEmployeeID(e.target.value)}
                disabled={submitting}
                required
              >
                <option value="">— Odaberite primatelja —</option>
                {targets.map(t => (
                  <option key={t.id} value={t.id}>
                    {t.first_name} {t.last_name} ({ROLE_LABELS[t.role] ?? t.role})
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Destination project (material transfers only) */}
          {isMaterial && toEmployeeID && (
            <div>
              <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
                Odredišni projekt <span className="text-red-400">*</span>
              </label>
              {loadingProjects ? (
                <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje projekata...</p>
              ) : recipientProjects.length === 0 ? (
                <p className="text-gi-text-subtle text-sm">Odabrani poslovođa nema aktivnih projekata.</p>
              ) : (
                <select
                  className={inputCls}
                  value={destinationProjectID}
                  onChange={e => setDestinationProjectID(e.target.value)}
                  disabled={submitting}
                  required
                >
                  <option value="">— Odaberite projekt —</option>
                  {recipientProjects.map(p => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
              )}
            </div>
          )}

          {/* Quantity (material transfers only) */}
          {isMaterial && (
            <div>
              <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
                Količina <span className="text-red-400">*</span>
              </label>
              <input
                type="number"
                inputMode="decimal"
                min="0.01"
                max={maxQty}
                step="0.01"
                placeholder="0.00"
                className={inputCls}
                value={quantity}
                onChange={e => setQuantity(e.target.value)}
                disabled={submitting}
                required
              />
              <p className="text-xs text-gi-text-subtle mt-1">Maksimalno: {maxQty.toFixed(2)} {subject.item.unit}</p>
            </div>
          )}

          {/* Notes */}
          <div>
            <label className="block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5">
              Napomena
            </label>
            <textarea
              rows={2}
              placeholder="Opcionalna napomena…"
              className={inputCls + ' resize-none'}
              value={notes}
              onChange={e => setNotes(e.target.value)}
              disabled={submitting}
            />
          </div>

          {error && (
            <div className="rounded-lg bg-red-900/30 border border-red-800/60 text-red-300 text-sm px-3 py-2.5 flex items-center gap-2">
              <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
              {error}
            </div>
          )}

          <div className="flex gap-3 pt-1">
            <button
              type="button"
              onClick={onClose}
              disabled={submitting}
              className="flex-1 py-2.5 rounded-lg border border-gi-border text-gi-text-muted text-sm hover:bg-gi-card-elevated transition-colors disabled:opacity-40"
            >
              Odustani
            </button>
            <button
              type="submit"
              disabled={submitting || targets.length === 0}
              className="flex-1 py-2.5 rounded-lg bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold transition-colors disabled:opacity-40 flex items-center justify-center gap-2"
            >
              {submitting ? (
                <>
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Prijenos…
                </>
              ) : 'Potvrdi prijenos'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
