'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
import ProjectMaterialTable from '@/components/project-materials/ProjectMaterialTable'
import ProjectMaterialForm from '@/components/project-materials/ProjectMaterialForm'
import ProjectMaterialImport from '@/components/project-materials/ProjectMaterialImport'
import {
  type MaterialListItem,
  type CreateMaterialPayload,
  canManageMaterials,
  canImportMaterials,
} from '@/lib/types/project-materials'
import apiClient from '@/lib/api-client'

type FormMode = 'none' | 'create' | 'edit'

export default function ProjectMaterialsPage() {
  const { id: projectId } = useParams<{ id: string }>()
  const { user, employee, isLoading, logout } = useAuth()

  const [projectName, setProjectName] = useState<string>('Materijali projekta')
  const [projectStatus, setProjectStatus] = useState<string>('active')
  const [materials, setMaterials] = useState<MaterialListItem[]>([])
  const [isLoadingMaterials, setIsLoadingMaterials] = useState(true)
  const [search, setSearch] = useState('')
  const [showInactive, setShowInactive] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [formMode, setFormMode] = useState<FormMode>('none')
  const [editTarget, setEditTarget] = useState<MaterialListItem | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const canManage = !!user && canManageMaterials(user.role) && projectStatus === 'active'
  const canImport = !!user && canImportMaterials(user.role) && projectStatus === 'active'

  // Load project name + status
  useEffect(() => {
    if (!user) return
    apiClient.get(`/projects/${projectId}`).then((res) => {
      setProjectName(res.data.project?.name ?? 'Materijali projekta')
      setProjectStatus(res.data.project?.status ?? 'active')
    }).catch(() => {})
  }, [user, projectId])

  const fetchMaterials = useCallback(() => {
    if (!user) return
    setIsLoadingMaterials(true)
    const params = new URLSearchParams()
    if (search) params.set('search', search)
    if (showInactive) params.set('active', 'false')

    apiClient
      .get(`/projects/${projectId}/materials${params.toString() ? '?' + params.toString() : ''}`)
      .then((res) => {
        setMaterials(res.data.materials ?? [])
        setError(null)
      })
      .catch(() => setError('Greška pri učitavanju materijala.'))
      .finally(() => setIsLoadingMaterials(false))
  }, [user, projectId, search, showInactive])

  useEffect(() => {
    fetchMaterials()
  }, [fetchMaterials])

  async function handleFormSubmit(data: CreateMaterialPayload) {
    setIsSubmitting(true)
    setFormError(null)
    try {
      if (formMode === 'edit' && editTarget) {
        await apiClient.put(`/projects/${projectId}/materials/${editTarget.id}`, data)
      } else {
        await apiClient.post(`/projects/${projectId}/materials`, data)
      }
      setFormMode('none')
      setEditTarget(null)
      fetchMaterials()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setFormError(err?.response?.data?.error ?? 'Greška pri spremanju materijala.')
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleDeactivate(materialId: string) {
    if (!confirm('Deaktivirati ovaj materijal?')) return
    try {
      await apiClient.patch(`/projects/${projectId}/materials/${materialId}/deactivate`)
      fetchMaterials()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      alert(err?.response?.data?.error ?? 'Greška pri deaktivaciji.')
    }
  }

  function openEdit(m: MaterialListItem) {
    setEditTarget(m)
    setFormMode('edit')
    setFormError(null)
  }

  function openCreate() {
    setEditTarget(null)
    setFormMode('create')
    setFormError(null)
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Materijali projekta"
      backHref={`/dashboard/projects/${projectId}`}
      onLogout={logout}
      action={
        canManage && formMode === 'none' ? (
          <button
            onClick={openCreate}
            className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
          >
            + Dodaj materijal
          </button>
        ) : undefined
      }
    >
      <div className="space-y-5 max-w-5xl">
        <h1 className="text-xl font-bold text-white">{projectName} — Materijali</h1>

        {projectStatus !== 'active' && (
          <div className="bg-amber-950 border border-amber-800 rounded-lg px-4 py-3 flex items-center gap-3">
            <svg className="w-4 h-4 text-amber-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <p className="text-amber-300 text-sm">
              Projekt je <strong>{projectStatus === 'closed' ? 'zatvoren' : 'arhiviran'}</strong> — dodavanje i uređivanje materijala nije dozvoljeno.
            </p>
          </div>
        )}

        {/* Inline form */}
        {formMode !== 'none' && (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
            <h2 className="text-white font-medium text-sm mb-4">
              {formMode === 'edit' ? 'Uredi materijal' : 'Novi materijal'}
            </h2>
            <ProjectMaterialForm
              initial={editTarget}
              onSubmit={handleFormSubmit}
              onCancel={() => { setFormMode('none'); setEditTarget(null) }}
              isSubmitting={isSubmitting}
              error={formError}
            />
          </div>
        )}

        {/* Excel import */}
        {canImport && formMode === 'none' && (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
            <h2 className="text-white font-medium text-sm mb-3">Uvoz iz Excela</h2>
            <ProjectMaterialImport projectId={projectId} onConfirmed={fetchMaterials} />
          </div>
        )}

        {/* Filters */}
        <div className="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Pretraži po nazivu ili šifri..."
            className="flex-1 px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500"
          />
          <label className="flex items-center gap-2 text-slate-400 text-sm cursor-pointer whitespace-nowrap">
            <input
              type="checkbox"
              checked={showInactive}
              onChange={(e) => setShowInactive(e.target.checked)}
              className="accent-slate-500"
            />
            Prikaži neaktivne
          </label>
        </div>

        {error && (
          <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{error}</p>
          </div>
        )}

        {/* Materials list */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
          {isLoadingMaterials ? (
            <LoadingOverlay />
          ) : (
            <ProjectMaterialTable
              materials={materials}
              canManage={canManage}
              onEdit={openEdit}
              onDeactivate={handleDeactivate}
              showInactive={showInactive}
            />
          )}
        </div>
      </div>
    </DashboardShell>
  )
}
