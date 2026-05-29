'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import apiClient from '@/lib/api-client'
import { getToken, setMustChangePassword } from '@/lib/auth'

interface FormData {
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

export default function ChangePasswordPage() {
  const router = useRouter()
  const [serverError, setServerError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<FormData>()

  const newPassword = watch('newPassword')

  useEffect(() => {
    if (!getToken()) {
      router.replace('/login')
    }
  }, [router])

  async function onSubmit(data: FormData) {
    setServerError(null)
    setIsSubmitting(true)
    try {
      await apiClient.patch('/auth/change-password', {
        currentPassword: data.currentPassword,
        newPassword: data.newPassword,
      })
      setMustChangePassword(false)
      router.push('/dashboard')
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setServerError(e?.response?.data?.error ?? 'Greška pri promjeni lozinke.')
    } finally {
      setIsSubmitting(false)
    }
  }

  const inputCls = (hasError: boolean) =>
    `w-full px-3.5 py-2.5 bg-slate-800 border ${
      hasError ? 'border-red-600' : 'border-slate-700'
    } rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition`

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-white tracking-tight">Promjena lozinke</h1>
          <p className="text-slate-400 text-sm mt-1">
            Vaš račun koristi privremenu lozinku. Molimo postavite novu lozinku.
          </p>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 shadow-xl">
          <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1.5">
                Trenutna lozinka
              </label>
              <input
                type="password"
                autoComplete="current-password"
                className={inputCls(!!errors.currentPassword)}
                placeholder="••••••••"
                {...register('currentPassword', { required: 'Trenutna lozinka je obavezna' })}
              />
              {errors.currentPassword && (
                <p className="text-red-400 text-xs mt-1">{errors.currentPassword.message}</p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1.5">
                Nova lozinka
              </label>
              <input
                type="password"
                autoComplete="new-password"
                className={inputCls(!!errors.newPassword)}
                placeholder="••••••••"
                {...register('newPassword', {
                  required: 'Nova lozinka je obavezna',
                  minLength: { value: 8, message: 'Nova lozinka mora imati najmanje 8 znakova' },
                })}
              />
              {errors.newPassword && (
                <p className="text-red-400 text-xs mt-1">{errors.newPassword.message}</p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1.5">
                Potvrda nove lozinke
              </label>
              <input
                type="password"
                autoComplete="new-password"
                className={inputCls(!!errors.confirmPassword)}
                placeholder="••••••••"
                {...register('confirmPassword', {
                  required: 'Potvrda lozinke je obavezna',
                  validate: (v) => v === newPassword || 'Lozinke se ne podudaraju',
                })}
              />
              {errors.confirmPassword && (
                <p className="text-red-400 text-xs mt-1">{errors.confirmPassword.message}</p>
              )}
            </div>

            {serverError && (
              <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
                <p className="text-red-300 text-sm">{serverError}</p>
              </div>
            )}

            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full py-2.5 px-4 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 disabled:opacity-50 disabled:cursor-not-allowed transition"
            >
              {isSubmitting ? 'Spremanje...' : 'Postavi novu lozinku'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
