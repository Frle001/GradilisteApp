'use client'

import { useForm, Controller } from 'react-hook-form'
import { type MaterialListItem, type CreateMaterialPayload, type TrackingType } from '@/lib/types/project-materials'

interface Props {
  initial?: MaterialListItem | null
  onSubmit: (data: CreateMaterialPayload) => Promise<void>
  onCancel: () => void
  isSubmitting: boolean
  error: string | null
}

export default function ProjectMaterialForm({ initial, onSubmit, onCancel, isSubmitting, error }: Props) {
  const { register, handleSubmit, control, watch, formState: { errors } } = useForm<CreateMaterialPayload>({
    defaultValues: {
      material_name:    initial?.material_name    ?? '',
      material_code:    initial?.material_code    ?? '',
      planned_quantity: initial?.planned_quantity  ?? 0,
      unit:             initial?.unit              ?? '',
      tracking_type:    initial?.tracking_type     ?? 'stock',
    },
  })

  const isEdit = !!initial
  const trackingType = watch('tracking_type')

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      {/* Tracking type */}
      <div>
        <label className="block text-slate-400 text-xs mb-2">Vrsta stavke *</label>
        <Controller
          name="tracking_type"
          control={control}
          render={({ field }) => (
            <div className="flex gap-2">
              {([['stock', 'Materijal', 'Fizički materijal s praćenjem zaliha'], ['work', 'Radna aktivnost', 'Mjerljivi rad bez zaliha (npr. Štemanje, kopanje)']] as [TrackingType, string, string][]).map(([val, label, hint]) => (
                <button
                  key={val}
                  type="button"
                  onClick={() => field.onChange(val)}
                  className={[
                    'flex-1 px-3 py-2.5 rounded-lg border text-sm text-left transition',
                    field.value === val
                      ? val === 'work'
                        ? 'bg-emerald-900/40 border-emerald-600 text-emerald-200'
                        : 'bg-blue-900/40 border-blue-600 text-blue-200'
                      : 'bg-slate-800 border-slate-700 text-slate-400 hover:border-slate-500',
                  ].join(' ')}
                >
                  <p className="font-medium">{label}</p>
                  <p className="text-xs opacity-70 mt-0.5">{hint}</p>
                </button>
              ))}
            </div>
          )}
        />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="sm:col-span-2">
          <label className="block text-slate-400 text-xs mb-1">
            {trackingType === 'work' ? 'Naziv aktivnosti *' : 'Naziv materijala *'}
          </label>
          <input
            {...register('material_name', { required: 'Naziv je obavezan' })}
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
            placeholder={trackingType === 'work' ? 'npr. Štemanje, Iskop temelja' : 'npr. Beton C25/30'}
          />
          {errors.material_name && <p className="text-red-400 text-xs mt-1">{errors.material_name.message}</p>}
        </div>

        {trackingType === 'stock' && (
          <div>
            <label className="block text-slate-400 text-xs mb-1">Šifra materijala</label>
            <input
              {...register('material_code')}
              className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
              placeholder="npr. B-25-30"
            />
          </div>
        )}

        <div>
          <label className="block text-slate-400 text-xs mb-1">Jedinica mjere *</label>
          <input
            {...register('unit', { required: 'Jedinica mjere je obavezna' })}
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
            placeholder={trackingType === 'work' ? 'npr. m, m², m³, h, kom' : 'npr. m³, kom, kg'}
          />
          {errors.unit && <p className="text-red-400 text-xs mt-1">{errors.unit.message}</p>}
        </div>

        <div>
          <label className="block text-slate-400 text-xs mb-1">
            {trackingType === 'work' ? 'Planirana količina radova *' : 'Planirana količina *'}
          </label>
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
          {trackingType === 'work' && (
            <p className="text-xs text-slate-500 mt-1">Prati se samo napredak — zalihe se ne oduzimaju.</p>
          )}
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
          {isSubmitting ? 'Spremanje...' : isEdit ? 'Spremi promjene' : 'Dodaj stavku'}
        </button>
      </div>
    </form>
  )
}
