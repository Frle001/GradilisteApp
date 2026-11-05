'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import ProjectForm from '@/components/projects/ProjectForm'
import { type CreateProjectPayload, type ProjectDetail, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

export default function EditProjectPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { user, employee, isLoading, logout } = useAuth()

  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [isLoadingProject, setIsLoadingProject] = useState(true)
  const [serverError, setServerError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (!user) return
    apiClient
      .get(`/projects/${id}`)
      .then((res) => setProject(res.data.project))
      .catch(() => setProject(null))
      .finally(() => setIsLoadingProject(false))
  }, [user, id])

  async function handleSubmit(data: CreateProjectPayload) {
    setServerError(null)
    setIsSubmitting(true)
    try {
      await apiClient.put(`/projects/${id}`, {
        name: data.name,
        address: data.address || null,
        description: data.description || null,
        start_date: data.start_date || null,
        end_date: data.end_date || null,
      })

      // Only call assign-poslovoda when the value has actually changed to avoid
      // a pointless deactivate+reactivate cycle on every project edit.
      const newId = data.primary_poslovoda_id || null
      const oldId = project?.primary_poslovoda_id ?? null
      if (newId && newId !== oldId) {
        await apiClient.patch(`/projects/${id}/assign-poslovoda`, {
          poslovoda_id: newId,
        })
      }

      router.push(`/dashboard/projects/${id}`)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setServerError(e?.response?.data?.error ?? 'Greška pri ažuriranju projekta.')
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
    <AppShell
      user={user}
      employee={employee}
      title="Uredi projekt"
      backHref={`/dashboard/projects/${id}`}
      onLogout={logout}
    >
      <div className="max-w-xl mx-auto">
        <div className="mb-6">
          <h1 className="text-xl font-bold text-gi-text">Uredi projekt</h1>
          {project && <p className="text-gi-text-subtle text-sm mt-1">{project.name}</p>}
        </div>

        {isLoadingProject ? (
          <div className="flex items-center justify-center h-32 border border-gi-border rounded-xl bg-gi-card">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje...</p>
          </div>
        ) : !project ? (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">Projekt nije pronađen.</p>
          </div>
        ) : (
          <div className="bg-gi-card border border-gi-border rounded-xl p-6">
            <ProjectForm
              defaultValues={{
                name: project.name,
                address: project.address,
                description: project.description,
                start_date: project.start_date,
                end_date: project.end_date,
                primary_poslovoda_id: project.primary_poslovoda_id,
              }}
              onSubmit={handleSubmit}
              submitLabel="Spremi izmjene"
              isSubmitting={isSubmitting}
              serverError={serverError}
            />
          </div>
        )}
      </div>
    </AppShell>
  )
}
