'use client'

import { useState } from 'react'
import {
  type ActivityType,
  type ActivityInputUI,
  type FormDataMaterial,
  ACTIVITY_TYPE_LABELS,
} from '@/lib/types/daily-reports'
import MaterialPicker from './MaterialPicker'
import CreateMaterialModal from './CreateMaterialModal'

interface Props {
  materials: FormDataMaterial[]
  materialsLoading?: boolean
  onAdd: (activity: ActivityInputUI) => void
  disabled?: boolean
  projectId?: string
  onMaterialAdded?: (material: FormDataMaterial) => void
}

const ACTIVITY_TYPES: ActivityType[] = ['montaza', 'demontaza', 'other']

export default function ActivityEntryForm({ materials, materialsLoading = false, onAdd, disabled = false, projectId, onMaterialAdded }: Props) {
  const [isVtk, setIsVtk] = useState(false)
  const [materialId, setMaterialId] = useState<string>('')
  const [customName, setCustomName] = useState('')
  const [quantity, setQuantity] = useState('')
  const [unit, setUnit] = useState('')
  const [activityType, setActivityType] = useState<ActivityType>('montaza')
  const [notes, setNotes] = useState('')
  const [errors, setErrors] = useState<string[]>([])
  const [createModalName, setCreateModalName] = useState<string | null>(null)

  const selectedMat = !isVtk && materialId ? materials.find(m => m.id === materialId) ?? null : null
  const isWorkItem = selectedMat?.tracking_type === 'work'

  function validate(): string[] {
    const errs: string[] = []

    // Contradictory state: should never occur via UI, but defend anyway
    if (isVtk && materialId) {
      errs.push('Nije moguće odabrati projekt-materijal i VTR način istovremeno.')
    }

    if (isVtk) {
      if (!customName.trim()) errs.push('Naziv materijala je obavezan za VTR aktivnost.')
    } else {
      if (!materialId) errs.push('Odaberite materijal.')
      // Stock check only for stock-type montaža; work items have no stock pool.
      if (materialId && activityType === 'montaza' && !isWorkItem) {
        if (selectedMat && selectedMat.available_quantity === 0) {
          errs.push('Nije moguće utrošiti novi materijal bez raspoložive količine.')
        }
      }
    }
    const qty = parseFloat(quantity)
    if (!quantity || isNaN(qty) || qty <= 0) errs.push('Količina mora biti pozitivan broj.')
    if (!unit.trim()) errs.push('Jedinica mjere je obavezna.')
    return errs
  }

  function handleVtkToggle() {
    if (disabled) return
    if (!isVtk) {
      // Enabling VTK: clear any selected project material so state is never contradictory.
      setMaterialId('')
      setUnit('')
    }
    setIsVtk(v => !v)
  }

  function handleCreateNew(name: string) {
    setCreateModalName(name)
  }

  function resolveDisplayName(): string {
    if (isVtk) return customName.trim()
    return selectedMat ? selectedMat.material_name : ''
  }

  function resolveUnit(): string {
    if (unit.trim()) return unit.trim()
    return selectedMat?.unit ?? ''
  }

  // unit comes directly from the picker so no materials.find() needed
  function handleMaterialChange(id: string, pickerUnit: string) {
    setMaterialId(id)
    setUnit(id ? pickerUnit : '')
  }

  function handleAdd() {
    const errs = validate()
    if (errs.length) { setErrors(errs); return }
    setErrors([])

    const activity: ActivityInputUI = {
      _tempId: crypto.randomUUID(),
      _displayName: resolveDisplayName(),
      project_material_id: isVtk ? null : (materialId || null),
      custom_material_name: isVtk ? customName.trim() : null,
      quantity: parseFloat(quantity),
      unit: resolveUnit(),
      activity_type: activityType,
      is_vtk: isVtk,
      notes: notes.trim() || null,
      tracking_type: isVtk ? undefined : (isWorkItem ? 'work' : 'stock'),
    }
    onAdd(activity)

    // reset
    setMaterialId('')
    setCustomName('')
    setQuantity('')
    setUnit('')
    setActivityType('montaza')
    setNotes('')
    setIsVtk(false)
  }

  return (
    <div className="bg-slate-800/50 border border-slate-700 rounded-lg p-4 space-y-4">
      <div className="flex items-center gap-3">
        <h4 className="text-sm font-medium text-slate-300 flex-1">Dodaj aktivnost</h4>
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <span className="text-xs text-slate-400">VTR aktivnost</span>
          <div
            role="checkbox"
            aria-checked={isVtk}
            onClick={handleVtkToggle}
            className={`relative w-9 h-5 rounded-full transition-colors cursor-pointer ${isVtk ? 'bg-amber-500' : 'bg-slate-600'}`}
          >
            <span
              className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${isVtk ? 'translate-x-4' : ''}`}
            />
          </div>
        </label>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {/* Material / Custom Name */}
        {isVtk ? (
          <div className="sm:col-span-2">
            <label className="block text-xs text-slate-400 mb-1">Naziv materijala (VTR) *</label>
            <input
              type="text"
              value={customName}
              disabled={disabled}
              onChange={e => setCustomName(e.target.value)}
              placeholder="Unesi naziv…"
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            />
          </div>
        ) : (
          <div className="sm:col-span-2">
            <label className="block text-xs text-slate-400 mb-1">Materijal *</label>
            <MaterialPicker
              materials={materials}
              loading={materialsLoading}
              value={materialId}
              onChange={handleMaterialChange}
              disabled={disabled}
              onCreateNew={!disabled && projectId ? handleCreateNew : undefined}
            />
            {isWorkItem && (
              <p className="mt-1.5 text-xs text-emerald-400">
                Upisana količina evidentira izvedeni rad i ne oduzima se sa zalihe.
              </p>
            )}
          </div>
        )}

        {/* Quantity */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Količina *</label>
          <input
            type="number"
            min={0}
            step="any"
            value={quantity}
            disabled={disabled}
            onChange={e => setQuantity(e.target.value)}
            placeholder="0"
            className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        {/* Unit */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Jedinica mjere *</label>
          <input
            type="text"
            value={unit}
            disabled={disabled}
            onChange={e => setUnit(e.target.value)}
            placeholder="kom, m, m², kg…"
            className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>

        {/* Activity Type */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Vrsta aktivnosti *</label>
          <select
            value={activityType}
            disabled={disabled}
            onChange={e => setActivityType(e.target.value as ActivityType)}
            className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
          >
            {ACTIVITY_TYPES.map(t => (
              <option key={t} value={t}>{ACTIVITY_TYPE_LABELS[t]}</option>
            ))}
          </select>
        </div>

        {/* Notes */}
        <div>
          <label className="block text-xs text-slate-400 mb-1">Napomena</label>
          <input
            type="text"
            value={notes}
            disabled={disabled}
            onChange={e => setNotes(e.target.value)}
            placeholder="Opcionalno…"
            className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
          />
        </div>
      </div>

      {errors.length > 0 && (
        <ul className="space-y-0.5">
          {errors.map((e, i) => (
            <li key={i} className="text-xs text-red-400">{e}</li>
          ))}
        </ul>
      )}

      <button
        type="button"
        disabled={disabled}
        onClick={handleAdd}
        className="w-full sm:w-auto px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium rounded transition-colors"
      >
        + Dodaj aktivnost
      </button>

      {createModalName !== null && projectId && (
        <CreateMaterialModal
          projectId={projectId}
          initialName={createModalName}
          onCreated={m => {
            onMaterialAdded?.(m)
            handleMaterialChange(m.id, m.unit)
            setCreateModalName(null)
          }}
          onClose={() => setCreateModalName(null)}
        />
      )}
    </div>
  )
}
