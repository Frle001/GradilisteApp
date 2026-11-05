'use client'

import { useEffect, useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { Plus, Upload, AlertTriangle } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
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
type Panel = 'list' | 'import'

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
  const [activePanel, setActivePanel] = useState<Panel>('list')

  const [formMode, setFormMode] = useState<FormMode>('none')
  const [editTarget, setEditTarget] = useState<MaterialListItem | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [deleteNotice, setDeleteNotice] = useState<{ kind: 'deleted' | 'deactivated'; name: string } | null>(null)

  const canManage = !!user && canManageMaterials(user.role) && projectStatus === 'active'
  const canImport = !!user && canImportMaterials(user.role) && projectStatus === 'active'

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

  useEffect(() => { fetchMaterials() }, [fetchMaterials])

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

  async function handleDelete(materialId: string) {
    const target = materials.find(m => m.id === materialId)
    const name = target?.material_name ?? 'materijal'
    if (!confirm(`Ukloniti "${name}"?\n\nAko ne postoje povezani zapisi, materijal će biti trajno obrisan. Ako postoje, bit će deaktiviran (sakriven s lista).`)) return
    setDeleteNotice(null)
    try {
      const res = await apiClient.delete(`/projects/${projectId}/materials/${materialId}`)
      if (res.status === 204) {
        setDeleteNotice({ kind: 'deleted', name })
      } else {
        setDeleteNotice({ kind: 'deactivated', name })
      }
      fetchMaterials()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      alert(err?.response?.data?.error ?? 'Greška pri uklanjanju materijala.')
    }
  }

  function openEdit(m: MaterialListItem) {
    setEditTarget(m)
    setFormMode('edit')
    setFormError(null)
    setActivePanel('list')
  }

  function openCreate() {
    setEditTarget(null)
    setFormMode('create')
    setFormError(null)
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  return (
    <AppShell
      user={user}
      employee={employee}
      title={`${projectName} — Materijali`}
      backHref={`/dashboard/projects/${projectId}`}
      onLogout={logout}
      action={
        canManage && formMode === 'none' ? (
          <div className="flex items-center gap-2">
            {canImport && (
              <button
                onClick={() => setActivePanel((prev: Panel) => prev === 'import' ? 'list' : 'import')}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs font-medium transition-colors ${
                  activePanel === 'import'
                    ? 'bg-gi-yellow/10 border-gi-yellow/40 text-gi-yellow'
                    : 'bg-gi-card-elevated border-gi-border text-gi-text-muted hover:text-gi-text'
                }`}
              >
                <Upload className="w-3.5 h-3.5" />Excel
              </button>
            )}
            <button
              onClick={openCreate}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-xs font-semibold rounded-lg transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />Dodaj
            </button>
          </div>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-gi-text">{projectName}</h1>
            <p className="text-gi-text-subtle text-sm mt-0.5">Plan i praćenje materijala</p>
          </div>
        </div>

        {projectStatus !== 'active' && (
          <div className="flex items-center gap-2.5 bg-amber-900/20 border border-amber-800/40 rounded-lg px-4 py-3">
            <AlertTriangle className="w-4 h-4 text-amber-400 flex-shrink-0" />
            <p className="text-amber-300 text-sm">
              Projekt je <strong>{projectStatus === 'closed' ? 'zatvoren' : 'arhiviran'}</strong> — dodavanje i uređivanje materijala nije dozvoljeno.
            </p>
          </div>
        )}

        {/* Inline form */}
        {formMode !== 'none' && (
          <div className="bg-gi-card border border-gi-border rounded-xl p-5">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-gi-text font-semibold text-sm">
                {formMode === 'edit' ? 'Uredi materijal' : 'Novi materijal'}
              </h2>
              <button
                onClick={() => { setFormMode('none'); setEditTarget(null) }}
                className="text-gi-text-subtle hover:text-gi-text text-xs transition-colors"
              >
                Odustani
              </button>
            </div>
            <ProjectMaterialForm
              initial={editTarget}
              onSubmit={handleFormSubmit}
              onCancel={() => { setFormMode('none'); setEditTarget(null) }}
              isSubmitting={isSubmitting}
              error={formError}
            />
          </div>
        )}

        {/* Excel import panel */}
        {canImport && formMode === 'none' && activePanel === 'import' && (
          <div className="bg-gi-card border border-gi-border rounded-xl p-5">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-gi-text font-semibold text-sm">Uvoz iz Excela</h2>
              <button
                onClick={() => setActivePanel('list')}
                className="text-gi-text-subtle hover:text-gi-text text-xs transition-colors"
              >
                Zatvori
              </button>
            </div>
            <ProjectMaterialImport projectId={projectId} onConfirmed={() => { fetchMaterials(); setActivePanel('list') }} />
          </div>
        )}

        {error && (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{error}</p>
          </div>
        )}

        <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
          {/* Table toolbar */}
          <div className="px-4 py-3 border-b border-gi-border flex flex-col sm:flex-row gap-2">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Pretraži po nazivu ili šifri..."
              className="flex-1 px-3 py-1.5 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text placeholder:text-gi-text-subtle text-sm focus:outline-none focus:border-gi-yellow transition-colors"
            />
            <label className="flex items-center gap-2 text-gi-text-muted text-sm cursor-pointer whitespace-nowrap px-3 py-1.5 bg-gi-card-elevated border border-gi-border rounded-lg">
              <input
                type="checkbox"
                checked={showInactive}
                onChange={(e) => setShowInactive(e.target.checked)}
                className="accent-gi-yellow w-3.5 h-3.5"
              />
              Prikaži neaktivne
            </label>
          </div>

          {deleteNotice && (
            <div className={`flex items-start justify-between gap-3 border-b px-4 py-3 ${
              deleteNotice.kind === 'deleted'
                ? 'bg-green-900/20 border-green-800/40'
                : 'bg-amber-900/20 border-amber-800/40'
            }`}>
              <p className={`text-sm ${deleteNotice.kind === 'deleted' ? 'text-green-300' : 'text-amber-300'}`}>
                {deleteNotice.kind === 'deleted'
                  ? `"${deleteNotice.name}" je trajno obrisan.`
                  : `"${deleteNotice.name}" je deaktiviran — postoje povezani zapisi (dnevna izvješća, kupnje ili zaduženja).`}
              </p>
              <button
                onClick={() => setDeleteNotice(null)}
                className="text-gi-text-subtle hover:text-gi-text shrink-0 text-xs"
              >
                ✕
              </button>
            </div>
          )}

          <div className="p-5">
            {isLoadingMaterials ? (
              <div className="flex items-center justify-center h-32">
                <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje materijala...</p>
              </div>
            ) : (
              <ProjectMaterialTable
                materials={materials}
                canManage={canManage}
                onEdit={openEdit}
                onDelete={handleDelete}
                showInactive={showInactive}
              />
            )}
          </div>
        </div>
      </div>
    </AppShell>
  )
}
