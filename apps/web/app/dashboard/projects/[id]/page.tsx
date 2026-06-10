'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
import ProjectStatusBadge from '@/components/projects/ProjectStatusBadge'
import { type ProjectDetail, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { user, employee, isLoading, logout } = useAuth()

  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [isLoadingProject, setIsLoadingProject] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const canManage = !!user && canManageProjects(user.role)

  useEffect(() => {
    if (!user) return
    apiClient
      .get(`/projects/${id}`)
      .then((res) => {
        setProject(res.data.project)
      })
      .catch((e: unknown) => {
        const err = e as { response?: { status?: number } }
        if (err?.response?.status === 403 || err?.response?.status === 404) {
          setError('Projekt nije pronađen ili nemate pristup.')
        } else {
          setError('Greška pri učitavanju projekta.')
        }
      })
      .finally(() => setIsLoadingProject(false))
  }, [user, id])

  async function handleStatusAction(action: 'close' | 'archive' | 'reactivate') {
    const labels = { close: 'zatvoriti', archive: 'arhivirati', reactivate: 'reaktivirati' }
    if (!confirm(`Sigurno želite ${labels[action]} ovaj projekt?`)) return
    setActionError(null)
    try {
      await apiClient.patch(`/projects/${id}/${action}`)
      const res = await apiClient.get(`/projects/${id}`)
      setProject(res.data.project)
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      setActionError(err?.response?.data?.error ?? 'Greška pri promjeni statusa.')
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const isActive = project?.status === 'active'

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title={project?.name ?? 'Projekt'}
      backHref="/dashboard/projects"
      onLogout={logout}
      action={
        canManage && project && isActive ? (
          <Link
            href={`/dashboard/projects/${id}/edit`}
            className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
          >
            Uredi
          </Link>
        ) : undefined
      }
    >
      {isLoadingProject ? (
        <LoadingOverlay />
      ) : error ? (
        <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      ) : project ? (
        <div className="space-y-6 max-w-2xl">
          {/* Read-only warning for non-active projects */}
          {!isActive && (
            <div className="bg-amber-950 border border-amber-800 rounded-lg px-4 py-3 flex items-center gap-3">
              <svg className="w-4 h-4 text-amber-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
              </svg>
              <p className="text-amber-300 text-sm">
                Ovaj projekt je <strong>{project.status === 'closed' ? 'zatvoren' : 'arhiviran'}</strong> — izmjene nisu moguće. Projekt je samo za čitanje.
              </p>
            </div>
          )}

          {/* Header card */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
            <div className="flex items-start justify-between gap-4 flex-wrap mb-4">
              <div>
                <h1 className="text-2xl font-bold text-white">{project.name}</h1>
                {project.address && <p className="text-slate-400 text-sm mt-1">{project.address}</p>}
              </div>
              <ProjectStatusBadge status={project.status} />
            </div>

            {project.description && (
              <p className="text-slate-300 text-sm mb-4">{project.description}</p>
            )}

            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
              <div>
                <dt className="text-slate-500">Poslovođa</dt>
                <dd className="text-white mt-0.5">{project.primary_poslovoda_name ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Datum početka</dt>
                <dd className="text-white mt-0.5">{project.start_date ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Datum završetka</dt>
                <dd className="text-white mt-0.5">{project.end_date ?? '—'}</dd>
              </div>
              {project.closed_at && (
                <div>
                  <dt className="text-slate-500">Zatvoreno</dt>
                  <dd className="text-white mt-0.5">{new Date(project.closed_at).toLocaleDateString('hr-HR')}</dd>
                </div>
              )}
            </dl>

            {/* Status actions */}
            {canManage && (
              <div className="flex flex-wrap gap-3 mt-5 pt-4 border-t border-slate-800">
                {project.status === 'active' && (
                  <button
                    onClick={() => handleStatusAction('close')}
                    className="px-3 py-1.5 bg-amber-900 border border-amber-700 text-amber-300 text-xs font-medium rounded-lg hover:bg-amber-800 transition"
                  >
                    Zatvori projekt
                  </button>
                )}
                {project.status !== 'archived' && (
                  <button
                    onClick={() => handleStatusAction('archive')}
                    className="px-3 py-1.5 bg-slate-800 border border-slate-700 text-slate-400 text-xs font-medium rounded-lg hover:bg-slate-700 transition"
                  >
                    Arhiviraj projekt
                  </button>
                )}
                {project.status !== 'active' && (
                  <button
                    onClick={() => handleStatusAction('reactivate')}
                    className="px-3 py-1.5 bg-green-900 border border-green-700 text-green-300 text-xs font-medium rounded-lg hover:bg-green-800 transition"
                  >
                    Reaktiviraj projekt
                  </button>
                )}
              </div>
            )}

            {actionError && <p className="text-red-400 text-xs mt-3">{actionError}</p>}
          </div>

          {/* Assignments */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6">
            <h2 className="text-white font-semibold mb-4">Dodjele zaposlenika</h2>
            {project.assignments.length === 0 ? (
              <p className="text-slate-500 text-sm">Nema dodijeljenih zaposlenika.</p>
            ) : (
              <div className="space-y-2">
                {project.assignments.map((a) => (
                  <div key={a.id} className="flex items-center justify-between py-2 border-b border-slate-800 last:border-0">
                    <div>
                      <p className="text-white text-sm">{a.employee_name}</p>
                      <p className="text-slate-500 text-xs capitalize">{a.role_on_project}</p>
                    </div>
                    <span className={`text-xs px-2 py-0.5 rounded ${a.active ? 'text-green-400 bg-green-900/40' : 'text-slate-500 bg-slate-800'}`}>
                      {a.active ? 'Aktivan' : 'Neaktivan'}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Placeholder sections */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <PlaceholderSection
              title="Materijali projekta"
              count={project.materials_count}
              label="stavki"
              href={`/dashboard/projects/${id}/materials`}
            />
            <PlaceholderSection
              title="Dnevni izvještaji"
              count={project.daily_reports_count}
              label="izvještaja"
            />
            <PlaceholderSection title="Građevinski dnevnik" />
            <PlaceholderSection title="Građevinska knjiga" />
          </div>
        </div>
      ) : null}
    </DashboardShell>
  )
}

function PlaceholderSection({ title, count, label, href }: { title: string; count?: number; label?: string; href?: string }) {
  const content = (
    <>
      <h3 className={`font-medium text-sm mb-1 ${href ? 'text-white' : 'text-slate-300'}`}>{title}</h3>
      {count !== undefined ? (
        <p className="text-slate-500 text-xs">{count} {label}</p>
      ) : (
        <p className="text-slate-600 text-xs italic">Uskoro dostupno</p>
      )}
    </>
  )

  if (href) {
    return (
      <Link
        href={href}
        className="block bg-slate-900 border border-slate-700 rounded-xl p-5 hover:border-slate-600 hover:bg-slate-800/50 transition"
      >
        {content}
      </Link>
    )
  }

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 opacity-60">
      {content}
    </div>
  )
}
