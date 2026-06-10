'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import ProjectForm from '@/components/projects/ProjectForm'
import { type CreateProjectPayload, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

export default function NewProjectPage() {
  const router = useRouter()
  const { user, employee, isLoading, logout } = useAuth()
  const [serverError, setServerError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(data: CreateProjectPayload) {
    setServerError(null)
    setIsSubmitting(true)
    try {
      const res = await apiClient.post('/projects', {
        name: data.name,
        address: data.address || null,
        description: data.description || null,
        start_date: data.start_date || null,
        end_date: data.end_date || null,
        primary_poslovoda_id: data.primary_poslovoda_id || null,
      })
      router.push(`/dashboard/projects/${res.data.project.id}`)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setServerError(e?.response?.data?.error ?? 'Greška pri kreiranju projekta.')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  if (!canManageProjects(user.role)) {
    router.replace('/dashboard/projects')
    return null
  }

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Novi projekt"
      backHref="/dashboard/projects"
      onLogout={logout}
    >
      <div className="max-w-lg">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-white">Novi projekt</h1>
          <p className="text-slate-400 text-sm mt-1">Kreirajte novo gradilište</p>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
          <ProjectForm
            onSubmit={handleSubmit}
            submitLabel="Kreiraj projekt"
            isSubmitting={isSubmitting}
            serverError={serverError}
          />
        </div>
      </div>
    </DashboardShell>
  )
}
