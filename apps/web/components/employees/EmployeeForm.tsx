'use client'

import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { type Employee, type CreateEmployeePayload, VALID_ROLES, ROLE_LABELS, fullName } from '@/lib/types/employees'
import apiClient from '@/lib/api-client'

interface EmployeeFormProps {
  defaultValues?: Partial<CreateEmployeePayload>
  onSubmit: (data: CreateEmployeePayload) => Promise<void>
  submitLabel: string
  isSubmitting: boolean
  serverError: string | null
}

interface SupervisorOption {
  id: string
  label: string
}

// Roles that require email (a user account will be created for them)
const LOGIN_ROLES = new Set(['direktor', 'inzenjer', 'administracija', 'poslovoda', 'radnik'])

export default function EmployeeForm({
  defaultValues,
  onSubmit,
  submitLabel,
  isSubmitting,
  serverError,
}: EmployeeFormProps) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<CreateEmployeePayload>({ defaultValues: defaultValues ?? {} })

  const [supervisors, setSupervisors] = useState<SupervisorOption[]>([])
  const role = watch('role')
  const isLoginRole = !!role && LOGIN_ROLES.has(role)
  const isRadnik = role === 'radnik'

  // Load poslovoda employees for the supervisor select (shown only for radnik)
  useEffect(() => {
    if (!isRadnik) return
    apiClient
      .get('/employees?role=poslovoda&active=true')
      .then((res) => {
        const emps: Employee[] = res.data.employees ?? []
        setSupervisors(emps.map((e) => ({ id: e.id, label: fullName(e) })))
      })
      .catch(() => {})
  }, [isRadnik])

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
      {/* First name + Last name */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">
            Ime <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            className={inputCls(!!errors.first_name)}
            placeholder="Ime"
            {...register('first_name', { required: 'Ime je obavezno' })}
          />
          {errors.first_name && <FieldError msg={errors.first_name.message} />}
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">
            Prezime <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            className={inputCls(!!errors.last_name)}
            placeholder="Prezime"
            {...register('last_name', { required: 'Prezime je obavezno' })}
          />
          {errors.last_name && <FieldError msg={errors.last_name.message} />}
        </div>
      </div>

      {/* Role */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">
          Uloga <span className="text-red-400">*</span>
        </label>
        <select
          className={inputCls(!!errors.role)}
          {...register('role', { required: 'Uloga je obavezna' })}
        >
          <option value="">— Odaberi ulogu —</option>
          {VALID_ROLES.map((r) => (
            <option key={r} value={r}>
              {ROLE_LABELS[r]}
            </option>
          ))}
        </select>
        {errors.role && <FieldError msg={errors.role.message} />}
        {isLoginRole && (
          <p className="text-slate-500 text-xs mt-1">
            Ova uloga dobiva korisnički račun za prijavu u sustav.
          </p>
        )}
      </div>

      {/* Supervisor — required for radnik */}
      {isRadnik && (
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1.5">
            Poslovođa (nadzornik) <span className="text-red-400">*</span>
          </label>
          <select
            className={inputCls(!!errors.supervisor_id)}
            {...register('supervisor_id', {
              required: 'Nadzornik je obavezan za radnika',
            })}
          >
            <option value="">— Odaberi poslovođu —</option>
            {supervisors.map((s) => (
              <option key={s.id} value={s.id}>
                {s.label}
              </option>
            ))}
          </select>
          {errors.supervisor_id && <FieldError msg={errors.supervisor_id.message} />}
          {supervisors.length === 0 && (
            <p className="text-slate-500 text-xs mt-1">
              Nema dostupnih poslovođa. Najprije kreirajte poslovođu.
            </p>
          )}
        </div>
      )}

      {/* Email — required for login roles, optional for radnik */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">
          Email adresa
          {isLoginRole && <span className="text-red-400"> *</span>}
        </label>
        <input
          type="email"
          className={inputCls(!!errors.email)}
          placeholder="ime@primjer.com"
          {...register('email', {
            required: isLoginRole ? 'Email je obavezan za ovu ulogu' : false,
            pattern: {
              value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
              message: 'Unesite ispravnu email adresu',
            },
          })}
        />
        {errors.email && <FieldError msg={errors.email.message} />}
        {isLoginRole && !errors.email && (
          <p className="text-slate-500 text-xs mt-1">
            Privremena lozinka bit će automatski generirana nakon spremanja zaposlenika.
          </p>
        )}
      </div>

      {/* Phone */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1.5">
          Broj telefona
        </label>
        <input
          type="tel"
          className={inputCls(false)}
          placeholder="+385 91 234 5678"
          {...register('phone')}
        />
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
