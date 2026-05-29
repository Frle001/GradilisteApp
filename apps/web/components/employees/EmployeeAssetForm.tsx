'use client'

import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { type Asset, type CreateAssetPayload, VALID_ASSET_TYPES, ASSET_TYPE_LABELS } from '@/lib/types/employees'
import apiClient from '@/lib/api-client'

interface EmployeeAssetFormProps {
  employeeID: string
  asset?: Asset // if provided, this is an edit form
  onSuccess: () => void
  onCancel: () => void
}

export default function EmployeeAssetForm({ employeeID, asset, onSuccess, onCancel }: EmployeeAssetFormProps) {
  const [serverError, setServerError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<CreateAssetPayload>({
    defaultValues: asset
      ? {
          asset_type: asset.asset_type,
          name: asset.name,
          quantity: asset.quantity,
          unit: asset.unit ?? undefined,
          serial_number: asset.serial_number ?? undefined,
          notes: asset.notes ?? undefined,
        }
      : { quantity: 1 },
  })

  async function onSubmit(data: CreateAssetPayload) {
    setServerError(null)
    setIsSubmitting(true)
    try {
      if (asset) {
        await apiClient.put(`/employees/${employeeID}/assets/${asset.id}`, {
          ...data,
          unit: data.unit || null,
          serial_number: data.serial_number || null,
          notes: data.notes || null,
        })
      } else {
        await apiClient.post(`/employees/${employeeID}/assets`, {
          ...data,
          unit: data.unit || null,
          serial_number: data.serial_number || null,
          notes: data.notes || null,
        })
      }
      onSuccess()
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setServerError(e?.response?.data?.error ?? 'Greška pri spremanju. Pokušajte ponovo.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
      {/* Asset type + Name */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">
            Vrsta <span className="text-red-400">*</span>
          </label>
          <select
            className={inputCls(!!errors.asset_type)}
            {...register('asset_type', { required: 'Vrsta je obavezna' })}
          >
            <option value="">— Odaberi —</option>
            {VALID_ASSET_TYPES.map((t) => (
              <option key={t} value={t}>{ASSET_TYPE_LABELS[t]}</option>
            ))}
          </select>
          {errors.asset_type && <p className="text-red-400 text-xs mt-1">{errors.asset_type.message}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">
            Naziv <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            className={inputCls(!!errors.name)}
            placeholder="npr. Bosch GBH 2-28"
            {...register('name', { required: 'Naziv je obavezan' })}
          />
          {errors.name && <p className="text-red-400 text-xs mt-1">{errors.name.message}</p>}
        </div>
      </div>

      {/* Quantity + Unit */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">
            Količina <span className="text-red-400">*</span>
          </label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            className={inputCls(!!errors.quantity)}
            {...register('quantity', {
              required: 'Količina je obavezna',
              min: { value: 0.01, message: 'Mora biti veće od 0' },
              valueAsNumber: true,
            })}
          />
          {errors.quantity && <p className="text-red-400 text-xs mt-1">{errors.quantity.message}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">Jedinica</label>
          <input
            type="text"
            className={inputCls(false)}
            placeholder="kom, kg, m..."
            {...register('unit')}
          />
        </div>
      </div>

      {/* Serial number */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">Serijski broj</label>
        <input
          type="text"
          className={inputCls(false)}
          placeholder="SN-123456"
          {...register('serial_number')}
        />
      </div>

      {/* Notes */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">Napomena</label>
        <textarea
          rows={2}
          className={inputCls(false) + ' resize-none'}
          placeholder="Opcijska napomena..."
          {...register('notes')}
        />
      </div>

      {serverError && (
        <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{serverError}</p>
        </div>
      )}

      <div className="flex gap-3">
        <button
          type="submit"
          disabled={isSubmitting}
          className="flex-1 py-2 px-4 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 disabled:opacity-50 transition"
        >
          {isSubmitting ? 'Sprema...' : asset ? 'Spremi izmjene' : 'Dodaj zaduženje'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="py-2 px-4 text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 rounded-lg text-sm transition"
        >
          Odustani
        </button>
      </div>
    </form>
  )
}

function inputCls(hasError: boolean) {
  return `w-full px-3.5 py-2.5 bg-slate-800 border ${
    hasError ? 'border-red-600' : 'border-slate-700'
  } rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition`
}
