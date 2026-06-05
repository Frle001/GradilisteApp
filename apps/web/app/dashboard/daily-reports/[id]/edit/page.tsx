'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import DashboardShell from '@/components/layout/DashboardShell'
import DailyReportForm from '@/components/daily-reports/DailyReportForm'
import {
  type DailyReportDetail,
  type DailyReportFormData,
  type CreateDailyReportPayload,
  canEditDailyReport,
} from '@/lib/types/daily-reports'
import apiClient from '@/lib/api-client'

export default function EditDailyReportPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams<{ id: string }>()
  const router = useRouter()

  const [report, setReport] = useState<DailyReportDetail | null>(null)
  const [formData, setFormData] = useState<DailyReportFormData | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)

  useEffect(() => {
    if (!user || !params.id) return
    Promise.all([
      apiClient.get(`/daily-reports/${params.id}`),
      apiClient.get('/daily-reports/form-data'),
    ])
      .then(([reportRes, formRes]) => {
        setReport(reportRes.data.daily_report)
        setFormData(formRes.data)
      })
      .catch(() => setLoadError('Greška pri učitavanju podataka za uređivanje.'))
  }, [user, params.id])

  const callerEmpId = employee?.id ?? null
  const canEdit =
    !!report && !!user && canEditDailyReport(user.role, callerEmpId, report.poslovoda.id, report.status)

  useEffect(() => {
    // redirect if the loaded report cannot be edited
    if (report && user && !canEdit) {
      router.replace(`/dashboard/daily-reports/${params.id}`)
    }
  }, [report, user, canEdit, router, params.id])

  async function handleSubmit(payload: CreateDailyReportPayload): Promise<void> {
    await apiClient.put(`/daily-reports/${params.id}`, payload)
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <p className="text-slate-400 text-sm">Učitavanje...</p>
      </div>
    )
  }
  if (!user) { router.replace('/login'); return null }

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Uredi dnevni izvještaj"
      backHref={params.id ? `/dashboard/daily-reports/${params.id}` : '/dashboard/daily-reports'}
      onLogout={logout}
    >
      <div className="max-w-3xl space-y-6">
        <h1 className="text-2xl font-bold text-white">Uredi dnevni izvještaj</h1>

        {loadError && (
          <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{loadError}</p>
          </div>
        )}

        {!report && !loadError && (
          <p className="text-slate-400 text-sm">Učitavanje…</p>
        )}

        {report && formData && canEdit && (
          <DailyReportForm
            formData={formData}
            existing={report}
            onSubmit={handleSubmit}
            submitLabel="Spremi izmjene"
          />
        )}
      </div>
    </DashboardShell>
  )
}
