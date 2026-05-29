'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import DashboardShell from '@/components/layout/DashboardShell'
import EmployeeForm from '@/components/employees/EmployeeForm'
import { type EmployeeDetail, type CreateEmployeePayload, canManageEmployees, fullName } from '@/lib/types/employees'
import apiClient from '@/lib/api-client'

export default function EditEmployeePage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string

  const { user, employee: authEmployee, isLoading, logout } = useAuth()
  const [emp, setEmp] = useState<EmployeeDetail | null>(null)
  const [dataLoading, setDataLoading] = useState(true)
  const [serverError, setServerError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (!isLoading && user) {
      if (!canManageEmployees(user.role)) {
        router.replace(`/dashboard/employees/${id}`)
        return
      }
      apiClient
        .get(`/employees/${id}`)
        .then((res) => setEmp(res.data.employee))
        .catch(() => router.replace('/dashboard/employees'))
        .finally(() => setDataLoading(false))
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, user, id])

  async function handleSubmit(data: CreateEmployeePayload) {
    setServerError(null)
    setIsSubmitting(true)
    try {
      await apiClient.put(`/employees/${id}`, {
        first_name: data.first_name,
        last_name: data.last_name,
        role: data.role,
        email: data.email || null,
        phone: data.phone || null,
        supervisor_id: data.supervisor_id || null,
      })
      router.push(`/dashboard/employees/${id}`)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setServerError(e?.response?.data?.error ?? 'Greška pri ažuriranju zaposlenika.')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (isLoading || dataLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <p className="text-slate-400 text-sm">Učitavanje...</p>
      </div>
    )
  }
  if (!user || !emp) return null

  const name = fullName(emp)

  return (
    <DashboardShell
      user={user}
      employee={authEmployee}
      title={`Uredi — ${name}`}
      backHref={`/dashboard/employees/${id}`}
      onLogout={logout}
    >
      <div className="max-w-lg">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-white">Uredi zaposlenika</h1>
          <p className="text-slate-400 text-sm mt-1">{name}</p>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
          <EmployeeForm
            defaultValues={{
              first_name: emp.first_name,
              last_name: emp.last_name,
              role: emp.role,
              email: emp.email ?? undefined,
              phone: emp.phone ?? undefined,
              supervisor_id: emp.supervisor_id ?? undefined,
            }}
            onSubmit={handleSubmit}
            submitLabel="Spremi izmjene"
            isSubmitting={isSubmitting}
            serverError={serverError}
          />
        </div>
      </div>
    </DashboardShell>
  )
}
