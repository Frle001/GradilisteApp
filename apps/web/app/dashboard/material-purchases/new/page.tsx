'use client'

import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import PurchaseItemsPreview from '@/components/material-purchases/PurchaseItemsPreview'
import ReceiptUploadInput from '@/components/material-purchases/ReceiptUploadInput'
import MaterialPicker from '@/components/daily-reports/MaterialPicker'
import apiClient from '@/lib/api-client'
import {
  type PurchaseFormProject,
  type PurchaseFormMaterial,
  type PurchaseItemDraft,
} from '@/lib/types/material-purchases'

// ── Shared style tokens (match daily-reports form style) ──────────────────────
const inputCls = [
  'w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-100',
  'placeholder-slate-500',
  'focus:outline-none focus:ring-1 focus:ring-blue-500',
  'disabled:opacity-50',
].join(' ')

const labelCls = 'block text-xs text-slate-400 mb-1'

let tempIdCounter = 0
function nextTempId() {
  return `_item_${++tempIdCounter}`
}

export default function MaterialPurchasesNewPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  // ── Form state (unchanged) ──────────────────────────────────────────────────
  const [projects, setProjects] = useState<PurchaseFormProject[]>([])
  const [materials, setMaterials] = useState<PurchaseFormMaterial[]>([])
  const [projectID, setProjectID] = useState('')
  const [items, setItems] = useState<PurchaseItemDraft[]>([])
  const [receiptFile, setReceiptFile] = useState<File | null>(null)
  const [notes, setNotes] = useState('')
  const [purchasedAt, setPurchasedAt] = useState(() => new Date().toISOString().slice(0, 10))

  // ── Item entry state (unchanged) ────────────────────────────────────────────
  const [selectedMaterialID, setSelectedMaterialID] = useState('')
  const [quantity, setQuantity] = useState('')
  const [unit, setUnit] = useState('')

  // ── Submission state (unchanged) ────────────────────────────────────────────
  const [submitting, setSubmitting] = useState(false)
  const [loadingMaterials, setLoadingMaterials] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // ── Load materials when project changes (unchanged) ─────────────────────────
  const loadMaterials = useCallback(async (pid: string) => {
    if (!pid) {
      setMaterials([])
      return
    }
    setLoadingMaterials(true)
    try {
      const res = await apiClient.get(`/material-purchases/form-data?project_id=${pid}`)
      setMaterials(res.data.materials ?? [])
    } catch {
      setMaterials([])
    } finally {
      setLoadingMaterials(false)
    }
  }, [])

  // ── Load projects on mount; auto-select when only one is available (unchanged)
  useEffect(() => {
    apiClient.get('/material-purchases/form-data').then(res => {
      const projs: PurchaseFormProject[] = res.data.projects ?? []
      setProjects(projs)
      if (projs.length === 1) {
        setProjectID(projs[0].id)
        loadMaterials(projs[0].id)
      }
    }).catch(() => {})
  }, [loadMaterials])

  // ── Handlers (unchanged) ────────────────────────────────────────────────────
  function handleProjectChange(pid: string) {
    setProjectID(pid)
    setItems([])
    setSelectedMaterialID('')
    setQuantity('')
    setUnit('')
    loadMaterials(pid)
  }

  function handleMaterialChange(id: string, unit: string) {
    setSelectedMaterialID(id)
    setUnit(unit)
  }

  function handleAddItem() {
    if (!selectedMaterialID || !quantity || parseFloat(quantity) <= 0) return
    const mat = materials.find(m => m.id === selectedMaterialID)
    if (!mat) return

    const existing = items.find(i => i.project_material_id === selectedMaterialID)
    if (existing) {
      setItems(prev => prev.map(i =>
        i.project_material_id === selectedMaterialID
          ? { ...i, quantity: i.quantity + parseFloat(quantity) }
          : i
      ))
    } else {
      const draft: PurchaseItemDraft = {
        _tempId: nextTempId(),
        project_material_id: selectedMaterialID,
        material_name: mat.material_name,
        material_code: mat.material_code,
        quantity: parseFloat(quantity),
        unit: unit || mat.unit,
      }
      setItems(prev => [...prev, draft])
    }

    setSelectedMaterialID('')
    setQuantity('')
    setUnit('')
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!projectID) { setError('Odaberite gradilište prije spremanja upisa.'); return }
    if (items.length === 0) { setError('Dodajte barem jedan materijal.'); return }

    setSubmitting(true)
    setError(null)

    try {
      const formData = new FormData()
      formData.append('project_id', projectID)
      formData.append('purchased_at', purchasedAt)
      formData.append('notes', notes)
      formData.append('items', JSON.stringify(items.map(i => ({
        project_material_id: i.project_material_id,
        quantity: i.quantity,
        unit: i.unit,
      }))))
      if (receiptFile) {
        formData.append('receipt', receiptFile)
      }

      // Clear Content-Type so browser sets multipart/form-data with correct boundary.
      const res = await apiClient.post('/material-purchases', formData, {
        headers: { 'Content-Type': null },
      })
      router.push(`/dashboard/material-purchases/${res.data.id}`)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? 'Greška pri spremanju upisa.')
    } finally {
      setSubmitting(false)
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const selectedMaterial = materials.find(m => m.id === selectedMaterialID)
  const canAdd = selectedMaterialID !== '' && parseFloat(quantity) > 0

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Novi upis materijala"
      backHref="/dashboard/material-purchases"
      onLogout={logout}
    >
      <div className="max-w-3xl space-y-6">

        {/* ── Page heading ────────────────────────────────────────────────── */}
        <div>
          <h1 className="text-2xl font-bold text-white">Novi upis materijala</h1>
          <p className="text-sm text-slate-400 mt-1">Evidentirajte kupljeni materijal za gradilište</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-8">

          {/* ── Card 1: GRADILIŠTE I DATUM ──────────────────────────────── */}
          <section className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-4">
            <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">
              Gradilište i datum
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className={labelCls}>Gradilište *</label>
                <select
                  className={inputCls}
                  value={projectID}
                  onChange={e => handleProjectChange(e.target.value)}
                  disabled={submitting}
                  required
                >
                  <option value="">— Odaberite gradilište —</option>
                  {projects.map(p => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
                {projects.length === 0 && (
                  <p className="text-xs text-slate-500 mt-1">Nema aktivnih gradilišta koja su vam dodijeljena.</p>
                )}
              </div>

              <div>
                <label className={labelCls}>Datum kupnje *</label>
                <input
                  type="date"
                  className={inputCls}
                  value={purchasedAt}
                  onChange={e => setPurchasedAt(e.target.value)}
                  disabled={submitting}
                  required
                />
              </div>
            </div>
          </section>

          {/* ── Card 2: MATERIJALI (list + add form together) ──────────── */}
          <section className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-4">
            <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">
              Materijali
              <span className="ml-2 text-slate-600 font-normal normal-case text-xs">
                ({items.length} stavki)
              </span>
            </h2>

            {!projectID ? (
              <p className="text-slate-500 text-sm">Prvo odaberite gradilište.</p>
            ) : (
              <div className="space-y-4">
                {/* Items list — always visible; empty state handled inside component */}
                <PurchaseItemsPreview
                  items={items}
                  onRemove={tempId => setItems(prev => prev.filter(i => i._tempId !== tempId))}
                  disabled={submitting}
                />

                {/* Add form */}
                <div className="space-y-3">
                  <div>
                    <label className={labelCls}>Dodaj materijal</label>
                    <MaterialPicker
                      materials={materials}
                      loading={loadingMaterials}
                      value={selectedMaterialID}
                      onChange={handleMaterialChange}
                      disabled={submitting}
                      placeholder="Pretraži materijal..."
                    />
                    {selectedMaterial && (
                      <p className="text-xs text-slate-500 mt-1">
                        Dostupno: {selectedMaterial.available_quantity.toFixed(2)} {selectedMaterial.unit}
                      </p>
                    )}
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className={labelCls}>Količina</label>
                      <input
                        type="number"
                        min="0.01"
                        step="0.01"
                        placeholder="0.00"
                        className={inputCls}
                        value={quantity}
                        onChange={e => setQuantity(e.target.value)}
                        disabled={submitting || !selectedMaterialID}
                      />
                    </div>
                    <div>
                      <label className={labelCls}>Jedinica</label>
                      <input
                        type="text"
                        placeholder="kom, m, kg…"
                        className={inputCls}
                        value={unit}
                        onChange={e => setUnit(e.target.value)}
                        disabled={submitting || !selectedMaterialID}
                      />
                    </div>
                  </div>

                  <button
                    type="button"
                    onClick={handleAddItem}
                    disabled={!canAdd || submitting}
                    className="w-full py-2 rounded border border-slate-600 bg-slate-800 hover:bg-slate-700 text-slate-200 text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                    </svg>
                    Dodaj na listu
                  </button>
                </div>
              </div>
            )}
          </section>

          {/* ── Card 3: FOTOGRAFIJA RAČUNA ──────────────────────────────── */}
          <section className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-3">
            <div>
              <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">
                Fotografija računa
              </h2>
              <p className="text-xs text-slate-500 mt-0.5">Nije obavezno, ali preporučeno za evidenciju.</p>
            </div>
            <ReceiptUploadInput file={receiptFile} onChange={setReceiptFile} disabled={submitting} />
          </section>

          {/* ── Card 4: NAPOMENA ────────────────────────────────────────── */}
          <section className="bg-slate-900 border border-slate-800 rounded-xl p-5 space-y-2">
            <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wide">Napomena</h2>
            <textarea
              rows={3}
              placeholder="Dodatne napomene o kupnji…"
              className={inputCls + ' resize-none'}
              value={notes}
              onChange={e => setNotes(e.target.value)}
              disabled={submitting}
            />
          </section>

          {/* ── Error ───────────────────────────────────────────────────── */}
          {error && (
            <div className="bg-red-900/30 border border-red-800 rounded-lg px-4 py-3 text-sm text-red-300">
              {error}
            </div>
          )}

          {/* ── Submit (right-aligned, matches daily-reports style) ─────── */}
          <div className="flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={() => router.back()}
              disabled={submitting}
              className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 disabled:opacity-50 transition-colors"
            >
              Odustani
            </button>
            <button
              type="submit"
              disabled={submitting || items.length === 0 || !projectID}
              className="px-5 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors flex items-center gap-2"
            >
              {submitting ? (
                <>
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Spremanje…
                </>
              ) : 'Spremi upis materijala'}
            </button>
          </div>

        </form>
      </div>
    </DashboardShell>
  )
}
