'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
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

  if (isLoading) return <LoadingScreen />
  if (!user) { router.replace('/login'); return null }

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Uredi dnevni izvještaj"
      backHref={params.id ? `/dashboard/daily-reports/${params.id}` : '/dashboard/daily-reports'}
      onLogout={logout}
    >
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <h1 className="text-xl font-bold text-gi-text">Uredi dnevni izvještaj</h1>
          {report && <p className="text-gi-text-subtle text-xs mt-0.5">{report.project.name}</p>}
        </div>

        {loadError && (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3 mb-5">
            <p className="text-red-300 text-sm">{loadError}</p>
          </div>
        )}

        {!report && !loadError && (
          <div className="bg-gi-card border border-gi-border rounded-xl p-10 text-center">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje…</p>
          </div>
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
    </AppShell>
  )
}
