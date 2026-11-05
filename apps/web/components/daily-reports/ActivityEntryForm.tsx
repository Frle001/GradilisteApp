'use client'

import { useState, useMemo } from 'react'
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
  /** Pre-fills form fields when editing an existing local activity. */
  initialActivity?: ActivityInputUI
}

const ACTIVITY_TYPES: ActivityType[] = ['montaza', 'demontaza', 'other']

const inputCls = 'w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text placeholder:text-gi-text-subtle focus:outline-none focus:border-gi-yellow transition-colors disabled:opacity-50'
const labelCls = 'block text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1.5'

export default function ActivityEntryForm({ materials, materialsLoading = false, onAdd, disabled = false, projectId, onMaterialAdded, initialActivity }: Props) {
  const [isVtk, setIsVtk] = useState(initialActivity?.is_vtk ?? false)
  const [materialId, setMaterialId] = useState<string>(initialActivity?.project_material_id ?? '')
  const [customName, setCustomName] = useState(initialActivity?.custom_material_name ?? '')
  const [quantity, setQuantity] = useState(initialActivity ? String(initialActivity.quantity) : '')
  const [unit, setUnit] = useState(initialActivity?.unit ?? '')
  const [activityType, setActivityType] = useState<ActivityType>(initialActivity?.activity_type ?? 'montaza')
  const [notes, setNotes] = useState(initialActivity?.notes ?? '')
  const [errors, setErrors] = useState<string[]>([])
  const [createModalName, setCreateModalName] = useState<string | null>(null)
  const isEditing = !!initialActivity

  const selectedMat = !isVtk && materialId ? materials.find(m => m.id === materialId) ?? null : null
  const isWorkItem = selectedMat?.tracking_type === 'work'

  // Sort: work-type and positive-stock materials first, zero-stock below. Same order for all activity types.
  const sortedMaterials = useMemo(() => {
    return [...materials].sort((a, b) => {
      const aUsable = (a.tracking_type === 'work' || a.available_quantity > 0) ? 0 : 1
      const bUsable = (b.tracking_type === 'work' || b.available_quantity > 0) ? 0 : 1
      if (aUsable !== bUsable) return aUsable - bUsable
      return a.material_name.localeCompare(b.material_name, 'hr-HR')
    })
  }, [materials])

  // Disabled predicate for montaža: stock items with no available stock cannot be consumed.
  function isMontazaItemDisabled(m: FormDataMaterial): boolean {
    return m.tracking_type === 'stock' && m.available_quantity <= 0
  }

  function validate(): string[] {
    const errs: string[] = []

    if (isVtk && materialId) {
      errs.push('Nije moguće odabrati projekt-materijal i VTR način istovremeno.')
    }

    if (isVtk) {
      if (!customName.trim()) errs.push('Naziv materijala je obavezan za VTR aktivnost.')
    } else {
      if (!materialId) errs.push('Odaberite materijal.')
      if (materialId && activityType === 'montaza' && !isWorkItem) {
        if (selectedMat && selectedMat.available_quantity <= 0) {
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

    setMaterialId('')
    setCustomName('')
    setQuantity('')
    setUnit('')
    setActivityType('montaza')
    setNotes('')
    setIsVtk(false)
  }

  return (
    <div className="bg-gi-workspace border border-gi-border rounded-lg p-4 space-y-4">
      {/* Header: title + VTR toggle */}
      <div className="flex items-start gap-3">
        <div className="flex-1">
          <h4 className="text-sm font-medium text-gi-text">
            {isEditing ? 'Uredi aktivnost' : 'Dodaj aktivnost'}
          </h4>
          {isEditing && (
            <p className="text-xs text-gi-text-subtle mt-0.5">Izmijeni i klikni spremi za ažuriranje.</p>
          )}
        </div>
        <label className="flex items-center gap-2 cursor-pointer select-none shrink-0">
          <div className="text-right">
            <span className="text-xs text-gi-text-subtle block">VTR aktivnost</span>
            <span className="text-[10px] text-gi-text-subtle/60 block leading-tight">Materijal izvan liste projekta</span>
          </div>
          <div
            role="checkbox"
            aria-checked={isVtk}
            onClick={handleVtkToggle}
            className={`relative w-9 h-5 rounded-full transition-colors cursor-pointer ${isVtk ? 'bg-amber-500' : 'bg-gi-card-elevated border border-gi-border'}`}
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
            <label className={labelCls}>Naziv materijala (VTR) *</label>
            <input
              type="text"
              value={customName}
              disabled={disabled}
              onChange={e => setCustomName(e.target.value)}
              placeholder="Unesi naziv…"
              className={inputCls}
            />
          </div>
        ) : (
          <div className="sm:col-span-2">
            <label className={labelCls}>Materijal *</label>
            <MaterialPicker
              materials={sortedMaterials}
              loading={materialsLoading}
              value={materialId}
              onChange={handleMaterialChange}
              disabled={disabled}
              emptyMessage={
                activityType === 'montaza'
                  ? 'Nema dostupnog materijala na projektu.'
                  : 'Nema materijala na projektu.'
              }
              isItemDisabled={activityType === 'montaza' ? isMontazaItemDisabled : undefined}
              onCreateNew={!disabled && projectId ? handleCreateNew : undefined}
            />
            {isWorkItem && (
              <p className="mt-1.5 text-xs text-emerald-400">
                Upisana količina evidentira izvedeni rad i ne oduzima se sa zalihe.
              </p>
            )}
            {selectedMat && !isWorkItem && (
              <p className="mt-1.5 text-xs text-gi-text-subtle">
                Na stanju:{' '}
                <span className={selectedMat.available_quantity > 0 ? 'text-emerald-400 font-medium' : 'text-red-400 font-medium'}>
                  {selectedMat.available_quantity} {selectedMat.unit}
                </span>
              </p>
            )}
          </div>
        )}

        {/* Quantity */}
        <div>
          <label className={labelCls}>Količina *</label>
          <input
            type="number"
            inputMode="decimal"
            min={0}
            step="any"
            value={quantity}
            disabled={disabled}
            onChange={e => setQuantity(e.target.value)}
            placeholder="0"
            className={inputCls}
          />
        </div>

        {/* Unit */}
        <div>
          <label className={labelCls}>Jedinica mjere *</label>
          <input
            type="text"
            value={unit}
            disabled={disabled}
            onChange={e => setUnit(e.target.value)}
            placeholder="kom, m, m², kg…"
            className={inputCls}
          />
        </div>

        {/* Activity Type */}
        <div>
          <label className={labelCls}>Vrsta aktivnosti *</label>
          <select
            value={activityType}
            disabled={disabled}
            onChange={e => {
              const newType = e.target.value as ActivityType
              setActivityType(newType)
              // Switching to montaža: a zero-stock stock material is no longer valid — clear it.
              if (newType === 'montaza' && materialId) {
                const mat = materials.find(m => m.id === materialId)
                if (mat && mat.tracking_type !== 'work' && mat.available_quantity <= 0) {
                  setMaterialId('')
                  setUnit('')
                }
              }
            }}
            className={inputCls}
          >
            {ACTIVITY_TYPES.map(t => (
              <option key={t} value={t}>{ACTIVITY_TYPE_LABELS[t]}</option>
            ))}
          </select>
        </div>

        {/* Notes */}
        <div>
          <label className={labelCls}>Napomena</label>
          <input
            type="text"
            value={notes}
            disabled={disabled}
            onChange={e => setNotes(e.target.value)}
            placeholder="Opcionalno…"
            className={inputCls}
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
        className={[
          'w-full sm:w-auto px-4 py-2 text-sm font-medium rounded-lg transition-colors disabled:opacity-50',
          isEditing
            ? 'bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy'
            : 'bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted hover:text-gi-text',
        ].join(' ')}
      >
        {isEditing ? 'Spremi izmjene' : '+ Dodaj aktivnost'}
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
