'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
import ProjectTable from '@/components/projects/ProjectTable'
import { type ProjectListItem, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

export default function ProjectsPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [projects, setProjects] = useState<ProjectListItem[]>([])
  const [isLoadingProjects, setIsLoadingProjects] = useState(true)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [error, setError] = useState<string | null>(null)

  const canManage = !!user && canManageProjects(user.role)

  const fetchProjects = useCallback(() => {
    if (!user) return
    setIsLoadingProjects(true)
    const params = new URLSearchParams()
    if (search) params.set('search', search)
    if (statusFilter) params.set('status', statusFilter)

    apiClient
      .get(`/projects${params.toString() ? '?' + params.toString() : ''}`)
      .then((res) => {
        setProjects(res.data.projects ?? [])
        setError(null)
      })
      .catch(() => setError('Greška pri učitavanju projekata.'))
      .finally(() => setIsLoadingProjects(false))
  }, [user, search, statusFilter])

  useEffect(() => {
    fetchProjects()
  }, [fetchProjects])

  useEffect(() => {
    if (!isLoading && user?.role === 'administracija') router.replace('/dashboard')
  }, [isLoading, user, router])

  async function handleClose(id: string) {
    if (!confirm('Zatvoriti ovaj projekt?')) return
    try {
      await apiClient.patch(`/projects/${id}/close`)
      fetchProjects()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      alert(err?.response?.data?.error ?? 'Greška pri zatvaranju projekta.')
    }
  }

  async function handleArchive(id: string) {
    if (!confirm('Arhivirati ovaj projekt?')) return
    try {
      await apiClient.patch(`/projects/${id}/archive`)
      fetchProjects()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      alert(err?.response?.data?.error ?? 'Greška pri arhiviranju projekta.')
    }
  }

  async function handleReactivate(id: string) {
    if (!confirm('Reaktivirati ovaj projekt?')) return
    try {
      await apiClient.patch(`/projects/${id}/reactivate`)
      fetchProjects()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      alert(err?.response?.data?.error ?? 'Greška pri reaktiviranju projekta.')
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user || user.role === 'administracija') return null

  const title = 'Projekti / Gradilišta'

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title={title}
      backHref="/dashboard"
      onLogout={logout}
      action={
        canManage ? (
          <Link
            href="/dashboard/projects/new"
            className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
          >
            + Novi projekt
          </Link>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div className="flex flex-col sm:flex-row gap-3">
          <h1 className="text-2xl font-bold text-white">{title}</h1>
        </div>

        {/* Filters */}
        <div className="flex flex-col sm:flex-row gap-3">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Pretraži po nazivu ili adresi..."
            className="flex-1 px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition"
          />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3.5 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition"
          >
            <option value="">Svi statusi</option>
            <option value="active">Aktivni</option>
            <option value="closed">Zatvoreni</option>
            <option value="archived">Arhivirani</option>
          </select>
        </div>

        {/* Error */}
        {error && (
          <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{error}</p>
          </div>
        )}

        {/* Table */}
        {isLoadingProjects ? (
          <LoadingOverlay />
        ) : (
          <ProjectTable
            projects={projects}
            canManage={canManage}
            onClose={handleClose}
            onArchive={handleArchive}
            onReactivate={handleReactivate}
          />
        )}
      </div>
    </DashboardShell>
  )
}
