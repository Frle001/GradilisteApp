'use client'

import { useForm } from 'react-hook-form'
import { type MaterialListItem, type CreateMaterialPayload } from '@/lib/types/project-materials'

interface Props {
  initial?: MaterialListItem | null
  onSubmit: (data: CreateMaterialPayload) => Promise<void>
  onCancel: () => void
  isSubmitting: boolean
  error: string | null
}

export default function ProjectMaterialForm({ initial, onSubmit, onCancel, isSubmitting, error }: Props) {
  const { register, handleSubmit, formState: { errors } } = useForm<CreateMaterialPayload>({
    defaultValues: {
      material_name: initial?.material_name ?? '',
      material_code: initial?.material_code ?? '',
      planned_quantity: initial?.planned_quantity ?? 0,
      unit: initial?.unit ?? '',
    },
  })

  const isEdit = !!initial

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="sm:col-span-2">
          <label className="block text-slate-400 text-xs mb-1">Naziv materijala *</label>
          <input
            {...register('material_name', { required: 'Naziv je obavezan' })}
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
            placeholder="npr. Beton C25/30"
          />
          {errors.material_name && <p className="text-red-400 text-xs mt-1">{errors.material_name.message}</p>}
        </div>

        <div>
          <label className="block text-slate-400 text-xs mb-1">Šifra materijala</label>
          <input
            {...register('material_code')}
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
            placeholder="npr. B-25-30"
          />
        </div>

        <div>
          <label className="block text-slate-400 text-xs mb-1">Jedinica mjere *</label>
          <input
            {...register('unit', { required: 'Jedinica mjere je obavezna' })}
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
            placeholder="npr. m³, kom, kg"
          />
          {errors.unit && <p className="text-red-400 text-xs mt-1">{errors.unit.message}</p>}
        </div>

        <div>
          <label className="block text-slate-400 text-xs mb-1">Planirana količina *</label>
          <input
            {...register('planned_quantity', {
              required: 'Količina je obavezna',
              valueAsNumber: true,
              min: { value: 0, message: 'Mora biti >= 0' },
            })}
            type="number"
            step="0.01"
            min="0"
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
          />
          {errors.planned_quantity && <p className="text-red-400 text-xs mt-1">{errors.planned_quantity.message}</p>}
        </div>
      </div>

      {error && (
        <div className="bg-red-950 border border-red-800 rounded-lg px-3 py-2">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      )}

      <div className="flex gap-3 justify-end pt-2">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-slate-400 hover:text-white text-sm transition"
        >
          Odustani
        </button>
        <button
          type="submit"
          disabled={isSubmitting}
          className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition disabled:opacity-50"
        >
          {isSubmitting ? 'Spremanje...' : isEdit ? 'Spremi promjene' : 'Dodaj materijal'}
        </button>
      </div>
    </form>
  )
}
