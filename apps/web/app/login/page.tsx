'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { Eye, EyeOff } from 'lucide-react'
import BrandMark from '@/components/brand/BrandMark'
import { BRAND } from '@/lib/platform-brand'
import apiClient from '@/lib/api-client'
import { setToken, setUser, setMustChangePassword, isAuthenticated, type AuthUser } from '@/lib/auth'

interface LoginFormData {
  email: string
  password: string
}

// ── Circuit illustration for right panel ──────────────────────────────────────

function CircuitIllustration() {
  return (
    <svg
      viewBox="0 0 320 320"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="w-64 h-64 sm:w-72 sm:h-72"
      aria-hidden
    >
      {/* Outer circuit traces */}
      <path d="M 20 160 H 90 V 100 H 160" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 160 100 H 230 V 160 H 300" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 20 160 H 90 V 220 H 160" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 160 220 H 230 V 160 H 300" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 160 20 V 90" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 160 230 V 300" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 60 60 L 90 90" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 260 60 L 230 90" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 60 260 L 90 230" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M 260 260 L 230 230" stroke="#32445E" strokeWidth="1.5" strokeLinecap="round" />

      {/* Trace nodes */}
      <circle cx="90" cy="100" r="4" fill="#32445E" />
      <circle cx="230" cy="100" r="4" fill="#32445E" />
      <circle cx="90" cy="220" r="4" fill="#32445E" />
      <circle cx="230" cy="220" r="4" fill="#32445E" />

      {/* Yellow accent nodes */}
      <circle cx="160" cy="90" r="5" fill="#F4B82F" fillOpacity="0.9" />
      <circle cx="160" cy="230" r="5" fill="#F4B82F" fillOpacity="0.9" />
      <circle cx="90" cy="160" r="5" fill="#F4B82F" fillOpacity="0.7" />
      <circle cx="230" cy="160" r="5" fill="#F4B82F" fillOpacity="0.7" />
      <circle cx="60" cy="60" r="4" fill="#F4B82F" fillOpacity="0.5" />
      <circle cx="260" cy="60" r="4" fill="#F4B82F" fillOpacity="0.5" />
      <circle cx="60" cy="260" r="4" fill="#F4B82F" fillOpacity="0.4" />
      <circle cx="260" cy="260" r="4" fill="#F4B82F" fillOpacity="0.4" />

      {/* Outer ring */}
      <circle cx="160" cy="160" r="58" stroke="#32445E" strokeWidth="1.5" />

      {/* Inner glow ring */}
      <circle cx="160" cy="160" r="46" stroke="#F4B82F" strokeOpacity="0.25" strokeWidth="1" />

      {/* Tick marks around ring */}
      <circle cx="160" cy="101" r="2.5" fill="#F4B82F" fillOpacity="0.7" />
      <circle cx="202" cy="128" r="2.5" fill="#F4B82F" fillOpacity="0.6" />
      <circle cx="202" cy="192" r="2.5" fill="#F4B82F" fillOpacity="0.6" />
      <circle cx="160" cy="219" r="2.5" fill="#F4B82F" fillOpacity="0.7" />
      <circle cx="118" cy="192" r="2.5" fill="#F4B82F" fillOpacity="0.6" />
      <circle cx="118" cy="128" r="2.5" fill="#F4B82F" fillOpacity="0.6" />

      {/* Central lightning bolt */}
      <path
        d="M 170 128 L 150 165 H 163 L 150 192 L 177 157 H 162 L 174 128 Z"
        fill="#F4B82F"
        stroke="#F4B82F"
        strokeWidth="0.5"
        strokeLinejoin="round"
      />
    </svg>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

const inputCls = [
  'w-full px-3.5 py-2.5 bg-gi-card-elevated border border-gi-border rounded-lg',
  'text-gi-text placeholder:text-gi-text-subtle text-sm',
  'focus:outline-none focus:border-gi-yellow focus:ring-1 focus:ring-gi-yellow/40',
  'transition-colors',
].join(' ')

export default function LoginPage() {
  const router = useRouter()
  const [serverError, setServerError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

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
      const { access_token, user, mustChangePassword } = res.data
      setToken(access_token)
      setUser(user as AuthUser)
      setMustChangePassword(!!mustChangePassword)
      if (mustChangePassword) {
        router.push('/change-password')
      } else {
        router.push('/dashboard')
      }
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
    <div className="min-h-screen bg-gi-navy flex">

      {/* ── Left panel: form ── */}
      <div className="flex-1 lg:flex-none lg:w-[520px] flex flex-col justify-center px-8 sm:px-12 py-12">

        <BrandMark variant="login" />

        {/* Heading */}
        <div className="mb-7">
          <h1 className="text-2xl font-bold text-gi-text tracking-tight">Dobrodošli natrag</h1>
          <p className="text-gi-text-subtle text-sm mt-1">{BRAND.loginSubtitle}</p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">

          {/* Email */}
          <div>
            <label htmlFor="email" className="block text-xs font-medium text-gi-text-muted mb-1.5">
              Email adresa
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="ime@primjer.com"
              className={inputCls}
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
            <label htmlFor="password" className="block text-xs font-medium text-gi-text-muted mb-1.5">
              Lozinka
            </label>
            <div className="relative">
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                placeholder="••••••••"
                className={inputCls + ' pr-10'}
                {...register('password', {
                  required: 'Lozinka je obavezna',
                  minLength: { value: 6, message: 'Lozinka mora imati najmanje 6 znakova' },
                })}
              />
              <button
                type="button"
                onClick={() => setShowPassword(v => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gi-text-subtle hover:text-gi-text transition-colors p-0.5"
                aria-label={showPassword ? 'Sakrij lozinku' : 'Prikaži lozinku'}
              >
                {showPassword
                  ? <EyeOff className="w-4 h-4" />
                  : <Eye className="w-4 h-4" />
                }
              </button>
            </div>
            {errors.password && (
              <p className="text-red-400 text-xs mt-1">{errors.password.message}</p>
            )}
          </div>

          {/* Server error */}
          {serverError && (
            <div className="bg-red-600/10 border border-red-600/30 rounded-lg px-4 py-3">
              <p className="text-red-400 text-sm">{serverError}</p>
            </div>
          )}

          {/* Submit */}
          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-3 px-4 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy font-bold rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-gi-yellow/50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-lg shadow-gi-yellow/20 mt-2"
          >
            {isLoading ? 'Prijava…' : 'Prijava'}
          </button>
        </form>

        <p className="mt-8 text-gi-text-subtle text-xs">
          Sustav za upravljanje gradilištem
        </p>
      </div>

      {/* ── Right panel: illustration (desktop only) ── */}
      <div className="hidden lg:flex flex-1 bg-gi-workspace border-l border-gi-border flex-col items-center justify-center px-12 py-12 relative overflow-hidden">

        {/* Background glow */}
        <div
          className="absolute inset-0 pointer-events-none"
          style={{ background: 'radial-gradient(ellipse at 50% 40%, rgba(244,184,47,0.06) 0%, transparent 65%)' }}
          aria-hidden
        />

        {/* Circuit illustration */}
        <div className="relative z-10 flex flex-col items-center gap-8">
          <CircuitIllustration />

          <div className="max-w-xs text-center">
            <p className="text-gi-text font-semibold text-lg leading-snug">
              Upravljanje gradilištem<br />na jednom mjestu
            </p>
            <ul className="mt-5 space-y-3 text-left">
              {[
                'Projekti i gradilišta',
                'Zaposlenici i plaće',
                'Materijali i nabava',
                'Analitika poslovanja',
              ].map(item => (
                <li key={item} className="flex items-center gap-2.5 text-sm text-gi-text-muted">
                  <span className="w-1.5 h-1.5 rounded-full bg-gi-yellow flex-shrink-0" />
                  {item}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>

    </div>
  )
}
