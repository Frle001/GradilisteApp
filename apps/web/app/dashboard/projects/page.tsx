'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Plus } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
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

  useEffect(() => { fetchProjects() }, [fetchProjects])

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

  return (
    <AppShell user={user} employee={employee} title="Projekti / Gradilišta" backHref="/dashboard" onLogout={logout}>
      {/* ── Header ── */}
      <div className="flex flex-col sm:flex-row sm:items-center gap-3 mb-5">
        <div className="flex-1 min-w-0">
          <h1 className="text-xl font-bold text-gi-text">Projekti / Gradilišta</h1>
          <p className="text-xs text-gi-text-subtle mt-0.5">Pregled aktivnih i zatvorenih gradilišta</p>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          {canManage && (
            <Link
              href="/dashboard/projects/archive"
              className="px-3 py-2 rounded-lg border border-gi-border text-gi-text-muted hover:text-gi-text hover:bg-gi-card text-xs font-medium transition-colors"
            >
              Arhiva
            </Link>
          )}
          {canManage && (
            <Link
              href="/dashboard/projects/new"
              className="flex items-center gap-1.5 px-4 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors"
            >
              <Plus className="w-4 h-4" />Novi projekt
            </Link>
          )}
        </div>
      </div>

      {/* ── Filters ── */}
      <div className="flex flex-col sm:flex-row gap-2.5 mb-5">
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Pretraži po nazivu ili adresi..."
          className="flex-1 px-3.5 py-2 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text placeholder:text-gi-text-subtle text-sm focus:outline-none focus:border-gi-yellow transition-colors"
        />
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3.5 py-2 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text text-sm focus:outline-none focus:border-gi-yellow transition-colors"
        >
          <option value="">Svi statusi</option>
          <option value="active">Aktivni</option>
          <option value="closed">Zatvoreni</option>
          <option value="archived">Arhivirani</option>
        </select>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3 mb-4">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      )}

      {isLoadingProjects ? (
        <div className="flex items-center justify-center h-48 border border-gi-border rounded-xl bg-gi-card">
          <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje projekata...</p>
        </div>
      ) : (
        <ProjectTable
          projects={projects}
          canManage={canManage}
          onClose={handleClose}
          onArchive={handleArchive}
          onReactivate={handleReactivate}
        />
      )}
    </AppShell>
  )
}
