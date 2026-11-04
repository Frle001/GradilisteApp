'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { Pencil } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import ReceiptPreview from '@/components/material-purchases/ReceiptPreview'
import apiClient from '@/lib/api-client'
import {
  type PurchaseDetail,
  type PurchaseFormMaterial,
} from '@/lib/types/material-purchases'

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('hr-HR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

interface EditItemDraft {
  _key: number
  project_material_id: string
  quantity: string
  unit: string
}

let _editKeyCounter = 0
function nextKey() { return ++_editKeyCounter }

type EditState = 'idle' | 'loading' | 'editing' | 'saving'

const inputCls = 'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3.5 py-2.5 text-sm text-gi-text focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'

export default function MaterialPurchaseDetailPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams()
  const id = params.id as string

  const [purchase, setPurchase] = useState<PurchaseDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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

  useEffect(() => { if (!id) return; loadPurchase() }, [id, loadPurchase])

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

  function cancelEdit() { setEditState('idle'); setEditError(null) }

  function unitForMaterial(matId: string) {
    return editMaterials.find(m => m.id === matId)?.unit ?? ''
  }

  function updateItemMaterial(key: number, matId: string) {
    setEditItems(prev => prev.map(item =>
      item._key === key ? { ...item, project_material_id: matId, unit: unitForMaterial(matId) } : item
    ))
  }

  function updateItemQuantity(key: number, qty: string) {
    setEditItems(prev => prev.map(item => item._key === key ? { ...item, quantity: qty } : item))
  }

  function removeItem(key: number) {
    setEditItems(prev => prev.filter(item => item._key !== key))
  }

  function addItem() {
    setEditItems(prev => [...prev, { _key: nextKey(), project_material_id: '', quantity: '', unit: '' }])
  }

  async function saveEdit() {
    setEditError(null)
    if (editItems.length === 0) { setEditError('Upis mora sadržavati najmanje jednu stavku materijala.'); return }
    for (const item of editItems) {
      if (!item.project_material_id) { setEditError('Svaka stavka mora imati odabrani materijal.'); return }
      const qty = parseFloat(item.quantity)
      if (isNaN(qty) || qty <= 0) { setEditError('Sve količine moraju biti pozitivne.'); return }
    }
    if (!editPurchasedAt) { setEditError('Datum kupnje je obavezan.'); return }

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

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const isEditing = editState === 'editing' || editState === 'saving'

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Detalji upisa"
      backHref="/dashboard/material-purchases"
      onLogout={logout}
      action={
        canEdit && editState === 'idle' && purchase ? (
          <button
            onClick={openEdit}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated border border-gi-border hover:border-gi-border/80 text-gi-text-muted hover:text-gi-text text-xs font-medium transition-colors"
          >
            <Pencil className="w-3.5 h-3.5" />Uredi
          </button>
        ) : canEdit && editState === 'loading' ? (
          <span className="text-gi-text-subtle text-sm">Učitavanje...</span>
        ) : undefined
      }
    >
      {loading && (
        <div className="flex items-center justify-center h-32 border border-gi-border rounded-xl bg-gi-card">
          <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje upisa...</p>
        </div>
      )}

      {error && (
        <div className="rounded-xl bg-red-900/30 border border-red-800/60 text-red-300 text-sm px-4 py-3">
          {error}
        </div>
      )}

      {purchase && (
        <div className="max-w-3xl space-y-5">
          <div>
            <h1 className="text-xl font-bold text-gi-text">Upis materijala</h1>
            <p className="text-sm text-gi-text-subtle mt-0.5">{purchase.project.name} · {formatDate(purchase.purchased_at)}</p>
          </div>

          {/* Summary card */}
          <div className="bg-gi-card border border-gi-border rounded-xl p-5 grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
            <div>
              <p className="text-[10px] font-medium text-gi-text-subtle uppercase tracking-wider mb-0.5">Gradilište</p>
              <p className="text-gi-text font-medium">{purchase.project.name}</p>
              {purchase.project.address && (
                <p className="text-xs text-gi-text-subtle mt-0.5">{purchase.project.address}</p>
              )}
            </div>
            <div>
              <p className="text-[10px] font-medium text-gi-text-subtle uppercase tracking-wider mb-0.5">Kupac</p>
              <p className="text-gi-text font-medium">{purchase.buyer.full_name}</p>
            </div>
            <div>
              <p className="text-[10px] font-medium text-gi-text-subtle uppercase tracking-wider mb-0.5">Datum kupnje</p>
              <p className="text-gi-text font-medium">{formatDate(purchase.purchased_at)}</p>
            </div>
            <div>
              <p className="text-[10px] font-medium text-gi-text-subtle uppercase tracking-wider mb-0.5">Broj stavki</p>
              <p className="text-gi-text font-medium">{purchase.items.length}</p>
            </div>
            <div>
              <p className="text-[10px] font-medium text-gi-text-subtle uppercase tracking-wider mb-0.5">Uneseno</p>
              <p className="text-gi-text font-medium">{formatDate(purchase.created_at)}</p>
            </div>
            {purchase.notes && (
              <div className="col-span-2 sm:col-span-3">
                <p className="text-[10px] font-medium text-gi-text-subtle uppercase tracking-wider mb-0.5">Napomena</p>
                <p className="text-gi-text-muted">{purchase.notes}</p>
              </div>
            )}
          </div>

          {/* Items or edit panel */}
          {!isEditing ? (
            <div>
              <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-3">Kupljeni materijali</h2>
              <div className="overflow-x-auto rounded-xl border border-gi-border">
                <table className="w-full text-sm text-left">
                  <thead>
                    <tr className="border-b border-gi-border">
                      <th className="px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Materijal</th>
                      <th className="px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider text-right">Količina</th>
                      <th className="px-4 py-3 text-xs font-medium text-gi-text-subtle uppercase tracking-wider">JM</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gi-border/60">
                    {purchase.items.map(item => (
                      <tr key={item.id} className="hover:bg-gi-card-elevated transition-colors">
                        <td className="px-4 py-3">
                          <p className="text-gi-text font-medium">{item.material_name}</p>
                          {item.material_code && <p className="text-xs text-gi-text-subtle">{item.material_code}</p>}
                        </td>
                        <td className="px-4 py-3 text-right font-mono font-semibold text-gi-text tabular-nums">
                          {item.quantity.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                        </td>
                        <td className="px-4 py-3 text-gi-text-muted">{item.unit}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div className="bg-gi-card border border-gi-border rounded-xl p-5 space-y-5">
              <h2 className="text-sm font-semibold text-gi-text">Uredi upis</h2>

              <div>
                <label className="block text-xs text-gi-text-muted mb-1.5">Datum kupnje</label>
                <input
                  type="date"
                  value={editPurchasedAt}
                  onChange={e => setEditPurchasedAt(e.target.value)}
                  disabled={editState === 'saving'}
                  className="w-full sm:w-48 bg-gi-card-elevated border border-gi-border rounded-lg px-3.5 py-2.5 text-sm text-gi-text focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50"
                />
              </div>

              <div>
                <p className="text-xs text-gi-text-muted mb-2">Stavke materijala</p>
                <div className="space-y-3">
                  {editItems.map((item, idx) => (
                    <div key={item._key} className="flex flex-col sm:flex-row gap-2 items-start sm:items-center">
                      <span className="text-xs text-gi-text-subtle w-5 text-right shrink-0 mt-2.5">{idx + 1}.</span>
                      <select
                        value={item.project_material_id}
                        onChange={e => updateItemMaterial(item._key, e.target.value)}
                        disabled={editState === 'saving'}
                        className="flex-1 min-w-0 bg-gi-card-elevated border border-gi-border rounded-lg px-3.5 py-2.5 text-sm text-gi-text focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50"
                      >
                        <option value="">— Odaberi materijal —</option>
                        {editMaterials.map(m => (
                          <option key={m.id} value={m.id}>
                            {m.material_name} ({m.unit})
                          </option>
                        ))}
                      </select>
                      <input
                        type="number"
                        min="0.01"
                        step="0.01"
                        value={item.quantity}
                        onChange={e => updateItemQuantity(item._key, e.target.value)}
                        disabled={editState === 'saving'}
                        placeholder="Kol."
                        className="w-28 shrink-0 bg-gi-card-elevated border border-gi-border rounded-lg px-3.5 py-2.5 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50"
                      />
                      <span className="text-sm text-gi-text-subtle w-10 shrink-0">{item.unit || '—'}</span>
                      <button
                        type="button"
                        onClick={() => removeItem(item._key)}
                        disabled={editState === 'saving' || editItems.length === 1}
                        className="shrink-0 text-red-500 hover:text-red-400 disabled:opacity-30 text-xs px-2 py-1 transition-colors"
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
                  className="mt-3 text-xs text-gi-text-muted hover:text-gi-yellow disabled:opacity-40 transition-colors"
                >
                  + Dodaj stavku
                </button>
              </div>

              <div>
                <label className="block text-xs text-gi-text-muted mb-1.5">Napomena <span className="text-gi-text-subtle">(nije obavezno)</span></label>
                <textarea
                  value={editNotes}
                  onChange={e => setEditNotes(e.target.value)}
                  disabled={editState === 'saving'}
                  rows={2}
                  placeholder="Kratka napomena..."
                  className={inputCls + ' resize-none'}
                />
              </div>

              {editError && (
                <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
                  <p className="text-red-300 text-sm">{editError}</p>
                </div>
              )}

              <div className="flex gap-3 pt-1">
                <button
                  type="button"
                  onClick={saveEdit}
                  disabled={editState === 'saving'}
                  className="flex-1 sm:flex-none px-5 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy font-semibold rounded-lg text-sm disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {editState === 'saving' ? 'Spremanje...' : 'Spremi izmjene'}
                </button>
                <button
                  type="button"
                  onClick={cancelEdit}
                  disabled={editState === 'saving'}
                  className="flex-1 sm:flex-none px-5 py-2 border border-gi-border text-gi-text-muted hover:bg-gi-card-elevated rounded-lg text-sm transition-colors disabled:opacity-40"
                >
                  Odustani
                </button>
              </div>

              <p className="text-xs text-gi-text-subtle">
                Izmjene se odnose samo na stavke i metapodatke ovog upisa. Gradilište, kupac i priloženi račun ne mogu se mijenjati.
              </p>
            </div>
          )}

          {/* Receipt */}
          <div>
            <h2 className="text-xs font-semibold text-gi-text-subtle uppercase tracking-wider mb-3">Račun</h2>
            {purchase.receipt_file_url ? (
              <ReceiptPreview purchaseId={purchase.id} originalFilename={purchase.receipt_original_filename} />
            ) : (
              <div className="rounded-xl border border-gi-border bg-gi-card px-4 py-6 text-center">
                <p className="text-gi-text-subtle text-sm">Nema priloženog računa za ovaj upis.</p>
              </div>
            )}
          </div>
        </div>
      )}
    </AppShell>
  )
}
