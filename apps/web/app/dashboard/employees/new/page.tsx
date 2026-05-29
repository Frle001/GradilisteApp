'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import DashboardShell from '@/components/layout/DashboardShell'
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

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <p className="text-slate-400 text-sm">Učitavanje...</p>
      </div>
    )
  }
  if (!user) return null

  if (!canManageEmployees(user.role)) {
    router.replace('/dashboard/employees')
    return null
  }

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Novi zaposlenik"
      backHref="/dashboard/employees"
      onLogout={logout}
    >
      <div className="max-w-lg">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-white">Novi zaposlenik</h1>
          <p className="text-slate-400 text-sm mt-1">Dodajte novog zaposlenika u sustav</p>
        </div>

        {createResult ? (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-green-900 flex items-center justify-center shrink-0">
                <span className="text-green-400 text-sm font-bold">✓</span>
              </div>
              <p className="text-white font-semibold">Zaposlenik je kreiran</p>
            </div>

            {createResult.loginCreated && createResult.temporaryPassword ? (
              <div className="bg-slate-800 border border-slate-700 rounded-lg px-4 py-3 space-y-1">
                <p className="text-slate-300 text-sm">Kreiran korisnički račun za prijavu u sustav.</p>
                <p className="text-slate-400 text-sm">
                  Privremena lozinka:{' '}
                  <span className="font-mono text-white bg-slate-700 px-1.5 py-0.5 rounded text-xs">
                    {createResult.temporaryPassword}
                  </span>
                </p>
                {createResult.mustChangePassword && (
                  <p className="text-amber-400 text-xs">Zaposlenik mora promijeniti lozinku pri prvoj prijavi.</p>
                )}
              </div>
            ) : (
              <div className="bg-slate-800 border border-slate-700 rounded-lg px-4 py-3">
                <p className="text-slate-300 text-sm">Nije kreiran korisnički račun za ulogu radnik.</p>
              </div>
            )}

            <div className="flex flex-col sm:flex-row gap-3 pt-1">
              <Link
                href={`/dashboard/employees/${createResult.id}`}
                className="flex-1 py-2.5 px-4 bg-white text-slate-900 font-semibold rounded-lg text-sm text-center hover:bg-slate-100 transition"
              >
                Prikaži zaposlenika →
              </Link>
              <button
                onClick={() => setCreateResult(null)}
                className="flex-1 py-2.5 px-4 bg-slate-800 border border-slate-700 text-slate-300 font-medium rounded-lg text-sm hover:bg-slate-700 transition"
              >
                Kreiraj još jednog
              </button>
            </div>
          </div>
        ) : (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
            <EmployeeForm
              onSubmit={handleSubmit}
              submitLabel="Kreiraj zaposlenika"
              isSubmitting={isSubmitting}
              serverError={serverError}
            />
          </div>
        )}
      </div>
    </DashboardShell>
  )
}
