'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
import ReceiptPreview from '@/components/material-purchases/ReceiptPreview'
import apiClient from '@/lib/api-client'
import {
  type PurchaseDetail,
  type PurchaseFormMaterial,
} from '@/lib/types/material-purchases'

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('hr-HR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

// ── Edit item draft ───────────────────────────────────────────────────────────

interface EditItemDraft {
  _key: number
  project_material_id: string
  quantity: string
  unit: string
}

let _editKeyCounter = 0
function nextKey() { return ++_editKeyCounter }

// ── Page ─────────────────────────────────────────────────────────────────────

type EditState = 'idle' | 'loading' | 'editing' | 'saving'

export default function MaterialPurchaseDetailPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams()
  const id = params.id as string

  const [purchase, setPurchase] = useState<PurchaseDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Edit state
  const [editState, setEditState] = useState<EditState>('idle')
  const [editItems, setEditItems] = useState<EditItemDraft[]>([])
  const [editNotes, setEditNotes] = useState('')
  const [editPurchasedAt, setEditPurchasedAt] = useState('')
  const [editMaterials, setEditMaterials] = useState<PurchaseFormMaterial[]>([])
  const [editError, setEditError] = useState<string | null>(null)

  const canEdit = !!user && (user.role === 'direktor' || user.role === 'inzenjer')

  const loadPurchase = useCallback(() => {
    setLoading(true)
    apiClient.get(`/material-purchases/${id}`)
      .then(res => setPurchase(res.data))
      .catch(err => {
        const e = err as { response?: { status?: number; data?: { error?: string } } }
        if (e?.response?.status === 404) setError('Upis nije pronađen.')
        else if (e?.response?.status === 403) setError('Nemate pristup ovom upisu.')
        else setError(e?.response?.data?.error ?? 'Greška pri učitavanju.')
      })
      .finally(() => setLoading(false))
  }, [id])

  useEffect(() => {
    if (!id) return
    loadPurchase()
  }, [id, loadPurchase])

  // ── Open edit ───────────────────────────────────────────────────────────────

  async function openEdit() {
    if (!purchase) return
    setEditState('loading')
    setEditError(null)
    try {
      const res = await apiClient.get(`/material-purchases/form-data?project_id=${purchase.project.id}`)
      const mats: PurchaseFormMaterial[] = res.data.materials ?? []
      setEditMaterials(mats)
      setEditItems(purchase.items.map(item => ({
        _key: nextKey(),
        project_material_id: item.project_material_id,
        quantity: String(item.quantity),
        unit: item.unit,
      })))
      setEditNotes(purchase.notes ?? '')
      setEditPurchasedAt(purchase.purchased_at.slice(0, 10))
      setEditState('editing')
    } catch {
      setEditError('Greška pri učitavanju podataka za uređivanje.')
      setEditState('idle')
    }
  }

  function cancelEdit() {
    setEditState('idle')
    setEditError(null)
  }

  // ── Item helpers ────────────────────────────────────────────────────────────

  function unitForMaterial(matId: string) {
    return editMaterials.find(m => m.id === matId)?.unit ?? ''
  }

  function updateItemMaterial(key: number, matId: string) {
    setEditItems(prev => prev.map(item =>
      item._key === key
        ? { ...item, project_material_id: matId, unit: unitForMaterial(matId) }
        : item
    ))
  }

  function updateItemQuantity(key: number, qty: string) {
    setEditItems(prev => prev.map(item =>
      item._key === key ? { ...item, quantity: qty } : item
    ))
  }

  function removeItem(key: number) {
    setEditItems(prev => prev.filter(item => item._key !== key))
  }

  function addItem() {
    setEditItems(prev => [...prev, { _key: nextKey(), project_material_id: '', quantity: '', unit: '' }])
  }

  // ── Save edit ───────────────────────────────────────────────────────────────

  async function saveEdit() {
    setEditError(null)

    if (editItems.length === 0) {
      setEditError('Upis mora sadržavati najmanje jednu stavku materijala.')
      return
    }
    for (const item of editItems) {
      if (!item.project_material_id) {
        setEditError('Svaka stavka mora imati odabrani materijal.')
        return
      }
      const qty = parseFloat(item.quantity)
      if (isNaN(qty) || qty <= 0) {
        setEditError('Sve količine moraju biti pozitivne.')
        return
      }
    }
    if (!editPurchasedAt) {
      setEditError('Datum kupnje je obavezan.')
      return
    }

    setEditState('saving')
    try {
      const payload = {
        purchased_at: editPurchasedAt,
        notes: editNotes.trim() || null,
        items: editItems.map(item => ({
          project_material_id: item.project_material_id,
          quantity: parseFloat(item.quantity),
          unit: item.unit,
        })),
      }
      const res = await apiClient.patch(`/material-purchases/${id}`, payload)
      setPurchase(res.data)
      setEditState('idle')
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setEditError(e?.response?.data?.error ?? 'Greška pri spremanju izmjena.')
      setEditState('editing')
    }
  }

  // ── Render ──────────────────────────────────────────────────────────────────

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const isEditing = editState === 'editing' || editState === 'saving'

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Detalji upisa"
      backHref="/dashboard/material-purchases"
      onLogout={logout}
      action={
        canEdit && editState === 'idle' && purchase ? (
          <button
            onClick={openEdit}
            className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
          >
            Uredi
          </button>
        ) : canEdit && editState === 'loading' ? (
          <span className="text-slate-400 text-sm">Učitavanje...</span>
        ) : undefined
      }
    >
      {loading && <LoadingOverlay />}

      {error && (
        <div className="rounded-xl bg-red-950 border border-red-800 text-red-300 text-sm px-4 py-3">
          {error}
        </div>
      )}

      {purchase && (
        <div className="max-w-3xl space-y-6">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight">Upis materijala</h1>
            <p className="text-sm text-slate-400 mt-1">{purchase.project.name} · {formatDate(purchase.purchased_at)}</p>
          </div>

          {/* Summary card */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Gradilište</p>
              <p className="text-slate-100 font-medium mt-0.5">{purchase.project.name}</p>
              {purchase.project.address && (
                <p className="text-xs text-slate-500 mt-0.5">{purchase.project.address}</p>
              )}
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Kupac</p>
              <p className="text-slate-100 font-medium mt-0.5">{purchase.buyer.full_name}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Datum kupnje</p>
              <p className="text-slate-100 font-medium mt-0.5">{formatDate(purchase.purchased_at)}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Broj stavki</p>
              <p className="text-slate-100 font-medium mt-0.5">{purchase.items.length}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Uneseno</p>
              <p className="text-slate-100 font-medium mt-0.5">{formatDate(purchase.created_at)}</p>
            </div>
            {purchase.notes && (
              <div className="col-span-2 sm:col-span-3">
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Napomena</p>
                <p className="text-slate-300 mt-0.5">{purchase.notes}</p>
              </div>
            )}
          </div>

          {/* Items — read-only view or edit panel */}
          {!isEditing ? (
            <div>
              <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">Kupljeni materijali</h2>
              <div className="overflow-x-auto rounded-xl border border-slate-700">
                <table className="w-full text-sm text-left">
                  <thead>
                    <tr className="bg-slate-800/80 border-b border-slate-700">
                      <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Materijal</th>
                      <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider text-right">Količina</th>
                      <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Jed.</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800">
                    {purchase.items.map(item => (
                      <tr key={item.id} className="hover:bg-slate-800/40 transition-colors">
                        <td className="px-4 py-3">
                          <p className="text-slate-100 font-medium">{item.material_name}</p>
                          {item.material_code && <p className="text-xs text-slate-500">{item.material_code}</p>}
                        </td>
                        <td className="px-4 py-3 text-right font-mono font-semibold text-slate-100 tabular-nums">
                          {item.quantity.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                        </td>
                        <td className="px-4 py-3 text-slate-400">{item.unit}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            /* ── Edit panel ── */
            <div className="bg-slate-900 border border-slate-700 rounded-xl p-5 space-y-5">
              <h2 className="text-sm font-semibold text-slate-200">Uredi upis</h2>

              {/* Date */}
              <div>
                <label className="block text-xs text-slate-400 mb-1">Datum kupnje</label>
                <input
                  type="date"
                  value={editPurchasedAt}
                  onChange={e => setEditPurchasedAt(e.target.value)}
                  disabled={editState === 'saving'}
                  className="w-full sm:w-48 bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
                />
              </div>

              {/* Items */}
              <div>
                <p className="text-xs text-slate-400 mb-2">Stavke materijala</p>
                <div className="space-y-3">
                  {editItems.map((item, idx) => (
                    <div key={item._key} className="flex flex-col sm:flex-row gap-2 items-start sm:items-center">
                      <span className="text-xs text-slate-600 w-5 text-right shrink-0 mt-2.5">{idx + 1}.</span>
                      {/* Material select */}
                      <select
                        value={item.project_material_id}
                        onChange={e => updateItemMaterial(item._key, e.target.value)}
                        disabled={editState === 'saving'}
                        className="flex-1 min-w-0 bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
                      >
                        <option value="">— Odaberi materijal —</option>
                        {editMaterials.map(m => (
                          <option key={m.id} value={m.id}>
                            {m.material_name} ({m.unit})
                          </option>
                        ))}
                      </select>
                      {/* Quantity */}
                      <input
                        type="number"
                        min="0.01"
                        step="0.01"
                        value={item.quantity}
                        onChange={e => updateItemQuantity(item._key, e.target.value)}
                        disabled={editState === 'saving'}
                        placeholder="Kol."
                        className="w-28 shrink-0 bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
                      />
                      {/* Unit */}
                      <span className="text-sm text-slate-500 w-10 shrink-0">{item.unit || '—'}</span>
                      {/* Remove */}
                      <button
                        type="button"
                        onClick={() => removeItem(item._key)}
                        disabled={editState === 'saving' || editItems.length === 1}
                        className="shrink-0 text-red-500 hover:text-red-400 disabled:opacity-30 text-xs px-2 py-1 transition"
                        aria-label="Ukloni stavku"
                      >
                        Ukloni
                      </button>
                    </div>
                  ))}
                </div>
                <button
                  type="button"
                  onClick={addItem}
                  disabled={editState === 'saving'}
                  className="mt-3 text-xs text-blue-400 hover:text-blue-300 disabled:opacity-40 transition"
                >
                  + Dodaj stavku
                </button>
              </div>

              {/* Notes */}
              <div>
                <label className="block text-xs text-slate-400 mb-1">Napomena <span className="text-slate-600">(nije obavezno)</span></label>
                <textarea
                  value={editNotes}
                  onChange={e => setEditNotes(e.target.value)}
                  disabled={editState === 'saving'}
                  rows={2}
                  placeholder="Kratka napomena..."
                  className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-1 focus:ring-blue-500 resize-none disabled:opacity-50"
                />
              </div>

              {editError && (
                <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
                  <p className="text-red-300 text-sm">{editError}</p>
                </div>
              )}

              <div className="flex gap-3 pt-1">
                <button
                  type="button"
                  onClick={saveEdit}
                  disabled={editState === 'saving'}
                  className="flex-1 sm:flex-none px-5 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed transition"
                >
                  {editState === 'saving' ? 'Spremanje...' : 'Spremi izmjene'}
                </button>
                <button
                  type="button"
                  onClick={cancelEdit}
                  disabled={editState === 'saving'}
                  className="flex-1 sm:flex-none px-5 py-2 border border-slate-700 text-slate-400 hover:bg-slate-800 rounded-lg text-sm transition disabled:opacity-40"
                >
                  Odustani
                </button>
              </div>

              <p className="text-xs text-slate-600">
                Napomena: izmjene se odnose samo na stavke i metapodatke ovog upisa.
                Gradilište, kupac i priloženi račun ne mogu se mijenjati.
              </p>
            </div>
          )}

          {/* Receipt — always visible */}
          <div>
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">Račun</h2>
            {purchase.receipt_file_url ? (
              <ReceiptPreview purchaseId={purchase.id} originalFilename={purchase.receipt_original_filename} />
            ) : (
              <div className="rounded-xl border border-slate-700 bg-slate-800/30 px-4 py-6 text-center">
                <p className="text-slate-500 text-sm">Nema priloženog računa za ovaj upis.</p>
              </div>
            )}
          </div>
        </div>
      )}
    </DashboardShell>
  )
}
