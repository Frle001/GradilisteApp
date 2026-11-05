'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import AppShell from '@/components/layout/AppShell'
import ProjectStatusBadge from '@/components/projects/ProjectStatusBadge'
import { type ArchiveSummary, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

function StatCard({ label, value, sub }: { label: string; value: number | string; sub?: string }) {
  return (
    <div className="bg-gi-card rounded-xl p-4">
      <p className="text-gi-text-muted text-xs uppercase tracking-wider mb-1">{label}</p>
      <p className="text-gi-text text-2xl font-bold">{value}</p>
      {sub && <p className="text-gi-text-subtle text-xs mt-1">{sub}</p>}
    </div>
  )
}

export default function ArchiveSummaryPage() {
  const { id } = useParams<{ id: string }>()
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [summary, setSummary] = useState<ArchiveSummary | null>(null)
  const [loadingData, setLoadingData] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reactivating, setReactivating] = useState(false)

  const canManage = !!user && canManageProjects(user.role)

  useEffect(() => {
    if (!isLoading && user?.role === 'administracija') router.replace('/dashboard')
  }, [isLoading, user, router])

  useEffect(() => {
    if (!user) return
    apiClient
      .get<{ summary: ArchiveSummary }>(`/projects/${id}/archive-summary`)
      .then((res) => {
        setSummary(res.data.summary)
        setError(null)
      })
      .catch((e: unknown) => {
        const err = e as { response?: { status?: number } }
        if (err?.response?.status === 404 || err?.response?.status === 403) {
          setError('Projekt nije pronađen ili nemate pristup.')
        } else {
          setError('Greška pri učitavanju sažetka.')
        }
      })
      .finally(() => setLoadingData(false))
  }, [user, id])

  async function handleReactivate() {
    if (!summary) return
    if (!confirm(`Reaktivirati projekt "${summary.name}"?`)) return
    setReactivating(true)
    try {
      await apiClient.patch(`/projects/${id}/reactivate`)
      const res = await apiClient.get<{ summary: ArchiveSummary }>(`/projects/${id}/archive-summary`)
      setSummary(res.data.summary)
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      alert(err?.response?.data?.error ?? 'Greška pri reaktiviranju.')
    } finally {
      setReactivating(false)
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user || user.role === 'administracija') return null

  return (
    <AppShell
      user={user}
      employee={employee}
      title={summary?.name ?? 'Sažetak projekta'}
      backHref="/dashboard/projects/archive"
      onLogout={logout}
    >
      <div className="space-y-6 max-w-4xl">
        {loadingData ? (
          <LoadingOverlay />
        ) : error ? (
          <div className="bg-red-950/40 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-400 text-sm">{error}</p>
          </div>
        ) : summary ? (
          <>
            {/* Header */}
            <div className="bg-gi-card border border-gi-border rounded-xl p-6">
              <div className="flex items-start justify-between gap-4 flex-wrap mb-4">
                <div>
                  <h1 className="text-2xl font-bold text-gi-text">{summary.name}</h1>
                  {summary.address && (
                    <p className="text-gi-text-muted text-sm mt-1">{summary.address}</p>
                  )}
                </div>
                <ProjectStatusBadge status={summary.status} />
              </div>

              <dl className="grid grid-cols-2 sm:grid-cols-3 gap-x-6 gap-y-3 text-sm">
                <div>
                  <dt className="text-gi-text-subtle">Datum početka</dt>
                  <dd className="text-gi-text mt-0.5">{summary.start_date ?? '—'}</dd>
                </div>
                <div>
                  <dt className="text-gi-text-subtle">Datum završetka</dt>
                  <dd className="text-gi-text mt-0.5">{summary.end_date ?? '—'}</dd>
                </div>
                <div>
                  <dt className="text-gi-text-subtle">Datum zatvaranja</dt>
                  <dd className="text-gi-text mt-0.5">
                    {summary.closed_at
                      ? new Date(summary.closed_at).toLocaleDateString('hr-HR')
                      : '—'}
                  </dd>
                </div>
                {summary.closed_by_name && (
                  <div>
                    <dt className="text-gi-text-subtle">Zatvorio</dt>
                    <dd className="text-gi-text mt-0.5">{summary.closed_by_name}</dd>
                  </div>
                )}
              </dl>

              {/* Assigned poslovode */}
              {summary.assigned_poslovode.length > 0 && (
                <div className="mt-4 pt-4 border-t border-gi-border">
                  <p className="text-gi-text-subtle text-xs uppercase tracking-wider mb-2">Poslovođe</p>
                  <div className="flex flex-wrap gap-2">
                    {summary.assigned_poslovode.map((a) => (
                      <span
                        key={a.id}
                        className={`text-xs px-2 py-1 rounded-full ${
                          a.active
                            ? 'bg-green-900/40 text-green-400 border border-green-800'
                            : 'bg-gi-card-elevated text-gi-text-subtle border border-gi-border'
                        }`}
                      >
                        {a.name}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Reactivate */}
              {canManage && summary.status !== 'active' && (
                <div className="mt-4 pt-4 border-t border-gi-border">
                  <button
                    onClick={handleReactivate}
                    disabled={reactivating}
                    className="px-4 py-2 bg-green-900 border border-green-700 text-green-300 text-sm font-medium rounded-lg hover:bg-green-800 transition disabled:opacity-40"
                  >
                    {reactivating ? 'Reaktiviranje...' : 'Reaktiviraj projekt'}
                  </button>
                </div>
              )}
            </div>

            {/* Dnevni izvještaji */}
            <div>
              <h2 className="text-gi-text font-semibold mb-3">Dnevni izvještaji</h2>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <StatCard label="Izvještaja" value={summary.daily_reports_count} />
                <StatCard
                  label="Radnih sati"
                  value={summary.worker_hours_total.toFixed(1)}
                  sub="ukupno"
                />
                <StatCard label="Aktivnosti" value={summary.activities_count} />
                <StatCard label="VTR stavki" value={summary.vtk_count} />
              </div>
              {summary.daily_reports_count > 0 && (
                <div className="mt-2">
                  <Link
                    href={`/dashboard/daily-reports?project_id=${id}`}
                    className="text-sm text-gi-yellow hover:text-gi-yellow-hover transition-colors"
                  >
                    Pregledaj dnevne izvještaje →
                  </Link>
                </div>
              )}
            </div>

            {/* Materijali */}
            <div>
              <h2 className="text-gi-text font-semibold mb-3">Materijali</h2>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                <StatCard label="Vrsta materijala" value={summary.project_materials_count} />
                <StatCard label="Nabavnih sesija" value={summary.purchase_sessions_count} />
                <StatCard label="Nabavnih stavki" value={summary.purchase_items_count} />
                <StatCard
                  label="Evidencija odgovornosti"
                  value={summary.material_responsibility_count}
                />
                <StatCard label="Prijenosa materijala" value={summary.transfers_count} />
              </div>
              {summary.project_materials_count > 0 && (
                <div className="mt-2">
                  <Link
                    href={`/dashboard/projects/${id}/materials`}
                    className="text-sm text-gi-yellow hover:text-gi-yellow-hover transition-colors"
                  >
                    Pregledaj materijale projekta →
                  </Link>
                </div>
              )}
            </div>
          </>
        ) : null}
      </div>
    </AppShell>
  )
}
