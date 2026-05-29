'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import apiClient from '@/lib/api-client'
import { setToken, setUser, isAuthenticated, type AuthUser } from '@/lib/auth'

interface LoginFormData {
  email: string
  password: string
}

export default function LoginPage() {
  const router = useRouter()
  const [serverError, setServerError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>()

  useEffect(() => {
    if (isAuthenticated()) {
      router.replace('/dashboard')
    }
  }, [router])

  async function onSubmit(data: LoginFormData) {
    setServerError(null)
    setIsLoading(true)
    try {
      const res = await apiClient.post('/auth/login', {
        email: data.email,
        password: data.password,
      })
      const { access_token, user } = res.data
      setToken(access_token)
      setUser(user as AuthUser)
      router.push('/dashboard')
    } catch (err: unknown) {
      const error = err as { response?: { status?: number; data?: { error?: string } } }
      const status = error?.response?.status
      const message = error?.response?.data?.error

      if (status === 401) {
        setServerError(message || 'Neispravna email adresa ili lozinka')
      } else if (status === 0 || !error?.response) {
        setServerError('Ne mogu se spojiti na server. Provjerite vezu.')
      } else {
        setServerError(message || 'Greška na serveru. Pokušajte ponovo.')
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-white tracking-tight">Gradilište App</h1>
          <p className="text-slate-400 text-sm mt-1">Prijavite se u sustav</p>
        </div>

        {/* Card */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 shadow-xl">
          <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
            {/* Email */}
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-1.5">
                Email adresa
              </label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                className="w-full px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition"
                placeholder="ime@primjer.com"
                {...register('email', {
                  required: 'Email adresa je obavezna',
                  pattern: {
                    value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
                    message: 'Unesite ispravnu email adresu',
                  },
                })}
              />
              {errors.email && (
                <p className="text-red-400 text-xs mt-1">{errors.email.message}</p>
              )}
            </div>

            {/* Password */}
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-1.5">
                Lozinka
              </label>
              <input
                id="password"
                type="password"
                autoComplete="current-password"
                className="w-full px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition"
                placeholder="••••••••"
                {...register('password', {
                  required: 'Lozinka je obavezna',
                  minLength: { value: 6, message: 'Lozinka mora imati najmanje 6 znakova' },
                })}
              />
              {errors.password && (
                <p className="text-red-400 text-xs mt-1">{errors.password.message}</p>
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
              disabled={isLoading}
              className="w-full py-2.5 px-4 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-slate-400 disabled:opacity-50 disabled:cursor-not-allowed transition"
            >
              {isLoading ? 'Prijava...' : 'Prijava'}
            </button>
          </form>
        </div>

        <p className="text-center text-slate-600 text-xs mt-6">
          Sustav za upravljanje građevinskim radovima
        </p>
      </div>
    </div>
  )
}
