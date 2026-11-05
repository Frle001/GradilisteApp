'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import EmployeeForm from '@/components/employees/EmployeeForm'
import { type CreateEmployeePayload, canManageEmployees } from '@/lib/types/employees'
import apiClient from '@/lib/api-client'

interface CreateResult {
  id: string
  loginCreated: boolean
  temporaryPassword: string | null
  mustChangePassword: boolean
}

export default function NewEmployeePage() {
  const router = useRouter()
  const { user, employee, isLoading, logout } = useAuth()
  const [serverError, setServerError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [createResult, setCreateResult] = useState<CreateResult | null>(null)

  async function handleSubmit(data: CreateEmployeePayload) {
    setServerError(null)
    setIsSubmitting(true)
    try {
      const res = await apiClient.post('/employees', {
        first_name: data.first_name,
        last_name: data.last_name,
        role: data.role,
        email: data.email || null,
        phone: data.phone || null,
        supervisor_id: data.supervisor_id || null,
      })
      setCreateResult({
        id: res.data.employee.id,
        loginCreated: res.data.loginCreated,
        temporaryPassword: res.data.temporaryPassword ?? null,
        mustChangePassword: res.data.mustChangePassword ?? false,
      })
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setServerError(e?.response?.data?.error ?? 'Greška pri kreiranju zaposlenika.')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  if (!canManageEmployees(user.role)) {
    router.replace('/dashboard/employees')
    return null
  }

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Novi zaposlenik"
      backHref="/dashboard/employees"
      onLogout={logout}
    >
      <div className="max-w-xl mx-auto">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-gi-text">Novi zaposlenik</h1>
          <p className="text-gi-text-muted text-sm mt-1">Dodajte novog zaposlenika u sustav</p>
        </div>

        {createResult ? (
          <div className="bg-gi-card border border-gi-border rounded-xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-green-900/40 border border-green-800 flex items-center justify-center shrink-0">
                <svg className="w-4 h-4 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                </svg>
              </div>
              <p className="text-gi-text font-semibold">Zaposlenik je kreiran</p>
            </div>

            {createResult.loginCreated && createResult.temporaryPassword ? (
              <div className="bg-gi-card-elevated border border-gi-border rounded-lg px-4 py-3 space-y-2">
                <p className="text-gi-text-muted text-sm">Kreiran korisnički račun za prijavu u sustav.</p>
                <p className="text-gi-text-muted text-sm">
                  Privremena lozinka:{' '}
                  <span className="font-mono text-gi-text bg-gi-card px-2 py-0.5 rounded text-xs border border-gi-border select-all">
                    {createResult.temporaryPassword}
                  </span>
                </p>
                {createResult.mustChangePassword && (
                  <p className="text-amber-400 text-xs font-medium">Zaposlenik mora promijeniti lozinku pri prvoj prijavi.</p>
                )}
                <p className="text-gi-text-subtle text-xs">Prikazano samo jednom — podijelite sigurnim kanalom.</p>
              </div>
            ) : (
              <div className="bg-gi-card-elevated border border-gi-border rounded-lg px-4 py-3">
                <p className="text-gi-text-muted text-sm">Nije kreiran korisnički račun za ulogu radnik.</p>
              </div>
            )}

            <div className="flex flex-col sm:flex-row gap-3 pt-1">
              <Link
                href={`/dashboard/employees/${createResult.id}`}
                className="flex-1 py-2.5 px-4 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy font-semibold rounded-lg text-sm text-center transition-colors"
              >
                Prikaži zaposlenika →
              </Link>
              <button
                onClick={() => setCreateResult(null)}
                className="flex-1 py-2.5 px-4 bg-gi-card-elevated border border-gi-border text-gi-text-muted font-medium rounded-lg text-sm hover:bg-gi-border transition-colors"
              >
                Kreiraj još jednog
              </button>
            </div>
          </div>
        ) : (
          <div className="bg-gi-card border border-gi-border rounded-xl p-6">
            <EmployeeForm
              onSubmit={handleSubmit}
              submitLabel="Kreiraj zaposlenika"
              isSubmitting={isSubmitting}
              serverError={serverError}
            />
          </div>
        )}
      </div>
    </AppShell>
  )
}
