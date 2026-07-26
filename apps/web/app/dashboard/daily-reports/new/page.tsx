'use client'

import { useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import DailyReportForm from '@/components/daily-reports/DailyReportForm'
import { type DailyReportFormData, type CreateDailyReportPayload } from '@/lib/types/daily-reports'
import apiClient from '@/lib/api-client'

export default function NewDailyReportPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()
  const searchParams = useSearchParams()
  const initialProjectId = searchParams.get('project_id') ?? ''

  const [formData, setFormData] = useState<DailyReportFormData | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)

  // Lifted photo state — the form reads and updates this via props.
  // Files are never serialised to JSON or localStorage.
  const [pendingPhotos, setPendingPhotos] = useState<File[]>([])

  useEffect(() => {
    if (!user) return
    apiClient
      .get('/daily-reports/form-data')
      .then(res => setFormData(res.data))
      .catch(() => setLoadError('Greška pri učitavanju podataka forme.'))
  }, [user])

  async function handleSubmit(payload: CreateDailyReportPayload): Promise<{ id: string }> {
    const res = await apiClient.post('/daily-reports', payload)
    return { id: res.data.id }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) { router.replace('/login'); return null }

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Novi dnevni izvještaj"
      backHref="/dashboard/daily-reports"
      onLogout={logout}
    >
      <div className="max-w-3xl space-y-6">
        <h1 className="text-2xl font-bold text-white">Novi dnevni izvještaj</h1>

        {loadError && (
          <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{loadError}</p>
          </div>
        )}

        {!formData && !loadError && (
          <p className="text-slate-400 text-sm">Učitavanje podataka…</p>
        )}

        {formData && (
          <DailyReportForm
            formData={formData}
            initialProjectId={initialProjectId}
            onSubmit={handleSubmit}
            submitLabel="Kreiraj izvještaj"
            pendingPhotos={pendingPhotos}
            onPendingPhotosChange={setPendingPhotos}
          />
        )}
      </div>
    </DashboardShell>
  )
}
