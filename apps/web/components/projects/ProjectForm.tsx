'use client'

import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { type CreateProjectPayload } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

interface PoslovodaOption {
  id: string
  label: string
}

interface ProjectFormProps {
  defaultValues?: Partial<CreateProjectPayload>
  onSubmit: (data: CreateProjectPayload) => Promise<void>
  submitLabel: string
  isSubmitting: boolean
  serverError: string | null
  hideEndDate?: boolean
}

export default function ProjectForm({
  defaultValues,
  onSubmit,
  submitLabel,
  isSubmitting,
  serverError,
  hideEndDate = false,
}: ProjectFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<CreateProjectPayload>({ defaultValues: defaultValues ?? {} })

  const [poslovodaOptions, setPoslovodaOptions] = useState<PoslovodaOption[]>([])

  useEffect(() => {
    apiClient
      .get('/employees?role=poslovoda&active=true')
      .then((res) => {
        const emps = res.data.employees ?? []
        setPoslovodaOptions(
          emps.map((e: { id: string; first_name: string; last_name: string }) => ({
            id: e.id,
            label: `${e.first_name} ${e.last_name}`,
          }))
        )
      })
      .catch(() => {})
  }, [])

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
      {/* Name */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">
          Naziv projekta <span className="text-red-400">*</span>
        </label>
        <input
          type="text"
          className={inputCls(!!errors.name)}
          placeholder="Gradilište Osijek Centar"
          {...register('name', { required: 'Naziv projekta je obavezan' })}
        />
        {errors.name && <FieldError msg={errors.name.message} />}
      </div>

      {/* Address */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">Adresa</label>
        <input
          type="text"
          className={inputCls(false)}
          placeholder="Ulica 1, Grad"
          {...register('address')}
        />
      </div>

      {/* Description */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">Opis</label>
        <textarea
          rows={3}
          className={inputCls(false) + ' resize-none'}
          placeholder="Kratki opis projekta..."
          {...register('description')}
        />
      </div>

      {/* Dates */}
      {hideEndDate ? (
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">Datum početka</label>
          <input type="date" className={inputCls(false)} {...register('start_date')} />
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1.5">Datum početka</label>
            <input type="date" className={inputCls(false)} {...register('start_date')} />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1.5">Datum završetka</label>
            <input type="date" className={inputCls(false)} {...register('end_date')} />
          </div>
        </div>
      )}

      {/* Primary Poslovoda */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">Poslovođa</label>
        <select className={inputCls(false)} {...register('primary_poslovoda_id')}>
          <option value="">— Bez poslovođe —</option>
          {poslovodaOptions.map((o) => (
            <option key={o.id} value={o.id}>
              {o.label}
            </option>
          ))}
        </select>
        {poslovodaOptions.length === 0 && (
          <p className="text-slate-500 text-xs mt-1">Nema dostupnih poslovođa.</p>
        )}
      </div>

      {/* Server error */}
      {serverError && (
        <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{serverError}</p>
        </div>
      )}

      {/* Submit */}
      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full py-2.5 px-4 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed transition"
      >
        {isSubmitting ? 'Sprema...' : submitLabel}
      </button>
    </form>
  )
}

function inputCls(hasError: boolean) {
  return `w-full px-3.5 py-2.5 bg-slate-800 border ${
    hasError ? 'border-red-600' : 'border-slate-700'
  } rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition`
}

function FieldError({ msg }: { msg?: string }) {
  return <p className="text-red-400 text-xs mt-1">{msg}</p>
}
