'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { Pencil, Package, Users, AlertTriangle } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import ProjectStatusBadge from '@/components/projects/ProjectStatusBadge'
import { type ProjectDetail, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'
import ProjectDocuments from '@/components/projects/ProjectDocuments'

type Tab = 'pregled' | 'zaposlenici' | 'materijali' | 'dokumenti'

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { user, employee, isLoading, logout } = useAuth()

  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [isLoadingProject, setIsLoadingProject] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<Tab>('pregled')

  const [hardDeleteState, setHardDeleteState] = useState<'idle' | 'confirming' | 'loading'>('idle')
  const [hardDeleteInput, setHardDeleteInput] = useState('')
  const [hardDeleteError, setHardDeleteError] = useState<string | null>(null)

  const canManage = !!user && canManageProjects(user.role)
  const canHardDelete = !!user && user.role === 'direktor'

  useEffect(() => {
    if (!isLoading && user?.role === 'administracija') router.replace('/dashboard')
  }, [isLoading, user, router])

  useEffect(() => {
    if (!user) return
    apiClient
      .get(`/projects/${id}`)
      .then((res) => setProject(res.data.project))
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

  async function handleHardDelete() {
    setHardDeleteState('loading')
    setHardDeleteError(null)
    try {
      await apiClient.delete(`/projects/${id}`)
      router.replace('/dashboard/projects')
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setHardDeleteError(e?.response?.data?.error ?? 'Greška pri brisanju projekta.')
      setHardDeleteState('confirming')
    }
  }

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
  if (!user || user.role === 'administracija') return null

  const isActive = project?.status === 'active'

  const tabs: { key: Tab; label: string }[] = [
    { key: 'pregled', label: 'Pregled' },
    { key: 'zaposlenici', label: 'Zaposlenici' },
    { key: 'materijali', label: 'Materijali' },
    { key: 'dokumenti', label: 'Dokumenti' },
  ]

  return (
    <AppShell
      user={user}
      employee={employee}
      title={project?.name ?? 'Projekt'}
      backHref="/dashboard/projects"
      onLogout={logout}
      action={
        canManage && project && isActive ? (
          <Link
            href={`/dashboard/projects/${id}/edit`}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated border border-gi-border hover:border-gi-border/80 text-gi-text-muted hover:text-gi-text text-xs font-medium transition-colors"
          >
            <Pencil className="w-3.5 h-3.5" />Uredi
          </Link>
        ) : undefined
      }
    >
      {isLoadingProject ? (
        <div className="flex items-center justify-center h-48 border border-gi-border rounded-xl bg-gi-card">
          <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje projekta...</p>
        </div>
      ) : error ? (
        <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      ) : project ? (
        <div className="space-y-5">
          {/* ── Project header card ── */}
          <div className="bg-gi-card border border-gi-border rounded-xl p-5">
            <div className="flex items-start justify-between gap-4 flex-wrap mb-3">
              <div className="min-w-0 flex-1">
                <h1 className="text-xl font-bold text-gi-text">{project.name}</h1>
                {project.address && <p className="text-gi-text-subtle text-sm mt-1">{project.address}</p>}
              </div>
              <ProjectStatusBadge status={project.status} />
            </div>

            {project.description && (
              <p className="text-gi-text-muted text-sm mb-4 leading-relaxed">{project.description}</p>
            )}

            <dl className="grid grid-cols-2 sm:grid-cols-4 gap-x-6 gap-y-3 text-sm">
              <div>
                <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Poslovođa</dt>
                <dd className="text-gi-text">{project.primary_poslovoda_name ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Početak</dt>
                <dd className="text-gi-text">{project.start_date ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Završetak</dt>
                <dd className="text-gi-text">{project.end_date ?? '—'}</dd>
              </div>
              {project.closed_at && (
                <div>
                  <dt className="text-[10px] text-gi-text-subtle uppercase tracking-wider mb-0.5">Zatvoreno</dt>
                  <dd className="text-gi-text">{new Date(project.closed_at).toLocaleDateString('hr-HR')}</dd>
                </div>
              )}
            </dl>

            {!isActive && (
              <div className="mt-4 flex items-center gap-2.5 bg-amber-900/20 border border-amber-800/40 rounded-lg px-3.5 py-2.5">
                <AlertTriangle className="w-4 h-4 text-amber-400 flex-shrink-0" />
                <p className="text-amber-300 text-sm">
                  Projekt je <strong>{project.status === 'closed' ? 'zatvoren' : 'arhiviran'}</strong> — izmjene nisu dozvoljene.
                </p>
              </div>
            )}

            {canManage && (
              <div className="flex flex-wrap gap-2 mt-4 pt-4 border-t border-gi-border">
                {project.status === 'active' && (
                  <button
                    onClick={() => handleStatusAction('close')}
                    className="px-3 py-1.5 bg-amber-900/40 border border-amber-700/60 text-amber-300 text-xs font-medium rounded-lg hover:bg-amber-800/50 transition-colors"
                  >
                    Zatvori projekt
                  </button>
                )}
                {project.status !== 'archived' && (
                  <button
                    onClick={() => handleStatusAction('archive')}
                    className="px-3 py-1.5 bg-gi-card-elevated border border-gi-border text-gi-text-muted text-xs font-medium rounded-lg hover:bg-gi-border hover:text-gi-text transition-colors"
                  >
                    Arhiviraj projekt
                  </button>
                )}
                {project.status !== 'active' && (
                  <button
                    onClick={() => handleStatusAction('reactivate')}
                    className="px-3 py-1.5 bg-green-900/40 border border-green-700/60 text-green-300 text-xs font-medium rounded-lg hover:bg-green-800/50 transition-colors"
                  >
                    Reaktiviraj projekt
                  </button>
                )}
              </div>
            )}

            {actionError && <p className="text-red-400 text-xs mt-3">{actionError}</p>}

            {canHardDelete && (
              <div className="pt-3 mt-2 border-t border-gi-border">
                <button
                  onClick={() => { setHardDeleteState('confirming'); setHardDeleteInput(''); setHardDeleteError(null) }}
                  className="text-xs border border-red-900/50 text-red-500 hover:bg-red-950/40 px-3 py-1.5 rounded-lg transition-colors"
                >
                  Trajno izbriši projekt
                </button>
                <p className="text-gi-text-subtle text-xs mt-1">Samo za testne ili slučajno stvorene projekte bez povijesti.</p>
              </div>
            )}
          </div>

          {/* ── Tabs ── */}
          <div className="border-b border-gi-border">
            <div className="flex gap-0 overflow-x-auto">
              {tabs.map((t) => (
                <button
                  key={t.key}
                  type="button"
                  onClick={() => setActiveTab(t.key)}
                  className={`px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 transition-colors ${
                    activeTab === t.key
                      ? 'border-gi-yellow text-gi-text'
                      : 'border-transparent text-gi-text-subtle hover:text-gi-text-muted'
                  }`}
                >
                  {t.label}
                </button>
              ))}
            </div>
          </div>

          {/* ── Tab content ── */}
          {activeTab === 'pregled' && (
            <div className="space-y-3">
              {/* Quick stats */}
              <div className="grid grid-cols-3 gap-3">
                <div className="bg-gi-card border border-gi-border rounded-xl px-4 py-3 text-center">
                  <p className="text-2xl font-bold text-gi-text">{project.materials_count}</p>
                  <p className="text-gi-text-subtle text-xs mt-1">stavki materijala</p>
                </div>
                <div className="bg-gi-card border border-gi-border rounded-xl px-4 py-3 text-center">
                  <p className="text-2xl font-bold text-gi-text">{project.assignments.filter(a => a.active).length}</p>
                  <p className="text-gi-text-subtle text-xs mt-1">aktivnih radnika</p>
                </div>
                <div className="bg-gi-card border border-gi-border rounded-xl px-4 py-3 text-center">
                  <p className="text-2xl font-bold text-gi-text">{project.daily_reports_count}</p>
                  <p className="text-gi-text-subtle text-xs mt-1">izvještaja</p>
                </div>
              </div>

              {/* Materials shortcut */}
              <Link
                href={`/dashboard/projects/${id}/materials`}
                className="flex items-center gap-3.5 p-4 bg-gi-card border border-gi-border rounded-xl hover:border-gi-border/80 hover:bg-gi-card-elevated transition-colors group"
              >
                <div className="w-9 h-9 rounded-lg bg-gi-yellow/10 text-gi-yellow flex items-center justify-center flex-shrink-0 group-hover:bg-gi-yellow/20 transition-colors">
                  <Package className="w-4.5 h-4.5" strokeWidth={1.8} />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-gi-text font-medium text-sm">Materijali projekta</p>
                  <p className="text-gi-text-subtle text-xs mt-0.5">Plan i praćenje materijala</p>
                </div>
                <svg className="w-4 h-4 text-gi-text-subtle group-hover:text-gi-text-muted transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
                </svg>
              </Link>
            </div>
          )}

          {activeTab === 'zaposlenici' && (
            <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
              <div className="px-5 py-3.5 border-b border-gi-border flex items-center justify-between">
                <h2 className="text-gi-text font-semibold text-sm">Dodjele zaposlenika</h2>
                <span className="text-gi-text-subtle text-xs">{project.assignments.filter(a => a.active).length} aktivnih</span>
              </div>
              {project.assignments.length === 0 ? (
                <div className="flex flex-col items-center gap-2 py-12 text-gi-text-subtle">
                  <Users className="w-8 h-8 text-gi-border" />
                  <p className="text-sm">Nema dodijeljenih zaposlenika.</p>
                </div>
              ) : (
                <div className="divide-y divide-gi-border/60">
                  {project.assignments.map((a) => {
                    const initials = a.employee_name.split(' ').map(p => p[0]).slice(0, 2).join('').toUpperCase()
                    return (
                      <div key={a.id} className="flex items-center justify-between px-5 py-3">
                        <div className="flex items-center gap-3">
                          <div className="w-7 h-7 rounded-full bg-gi-card-elevated border border-gi-border text-gi-text-muted text-[11px] font-bold flex items-center justify-center flex-shrink-0">
                            {initials}
                          </div>
                          <div>
                            <p className="text-gi-text text-sm font-medium">{a.employee_name}</p>
                            <p className="text-gi-text-subtle text-xs capitalize">{a.role_on_project}</p>
                          </div>
                        </div>
                        <span className={`text-xs px-2 py-0.5 rounded-full border ${
                          a.active
                            ? 'text-green-300 bg-green-900/30 border-green-800/50'
                            : 'text-gi-text-subtle bg-gi-card-elevated border-gi-border'
                        }`}>
                          {a.active ? 'Aktivan' : 'Neaktivan'}
                        </span>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )}

          {activeTab === 'materijali' && (
            <div className="bg-gi-card border border-gi-border rounded-xl p-5 text-center space-y-3">
              <Package className="w-10 h-10 text-gi-border mx-auto" />
              <div>
                <p className="text-gi-text font-medium">{project.materials_count} stavki materijala</p>
                <p className="text-gi-text-subtle text-sm mt-1">Upravljajte planom materijala za ovaj projekt</p>
              </div>
              <Link
                href={`/dashboard/projects/${id}/materials`}
                className="inline-flex items-center gap-1.5 px-4 py-2 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy text-sm font-semibold rounded-lg transition-colors"
              >
                <Package className="w-4 h-4" />
                Otvori materijale
              </Link>
            </div>
          )}

          {activeTab === 'dokumenti' && (
            <ProjectDocuments projectId={id} canManage={canManage} />
          )}
        </div>
      ) : null}

      {/* ── Hard delete modal ── */}
      {hardDeleteState !== 'idle' && project && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
          <div className="bg-gi-card border border-red-900/50 rounded-xl p-6 w-full max-w-md space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-red-950/60 flex items-center justify-center shrink-0">
                <AlertTriangle className="w-4 h-4 text-red-400" />
              </div>
              <p className="text-gi-text font-semibold">Trajno brisanje projekta</p>
            </div>
            <div className="bg-red-950/20 border border-red-900/30 rounded-lg px-4 py-3 space-y-1">
              <p className="text-red-300 text-sm font-medium">Ova radnja je nepovratna.</p>
              <p className="text-gi-text-muted text-xs">
                Koristite samo za testne ili slučajno stvorene projekte. Za stvarne projekte koristite <strong className="text-gi-text">arhiviranje</strong>.
              </p>
            </div>
            <div>
              <p className="text-gi-text-muted text-sm mb-2">
                Upišite <span className="font-mono text-gi-text bg-gi-card-elevated px-1.5 py-0.5 rounded">{project.name}</span> ili <span className="font-mono text-gi-text bg-gi-card-elevated px-1.5 py-0.5 rounded">IZBRIŠI</span> za potvrdu:
              </p>
              <input
                type="text"
                value={hardDeleteInput}
                onChange={e => setHardDeleteInput(e.target.value)}
                autoFocus
                className="w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text focus:outline-none focus:border-red-700"
                placeholder="Upišite potvrdu..."
                disabled={hardDeleteState === 'loading'}
              />
            </div>
            {hardDeleteError && <p className="text-red-400 text-sm">{hardDeleteError}</p>}
            <div className="flex gap-2">
              <button
                onClick={handleHardDelete}
                disabled={hardDeleteState === 'loading' || (hardDeleteInput !== project.name && hardDeleteInput !== 'IZBRIŠI')}
                className="flex-1 py-2 bg-red-700 hover:bg-red-600 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors"
              >
                {hardDeleteState === 'loading' ? 'Brisanje...' : 'Trajno izbriši'}
              </button>
              <button
                onClick={() => setHardDeleteState('idle')}
                disabled={hardDeleteState === 'loading'}
                className="flex-1 py-2 border border-gi-border text-gi-text-muted hover:bg-gi-card-elevated text-sm rounded-lg transition-colors disabled:opacity-40"
              >
                Odustani
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  )
}
