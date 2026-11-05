'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import ProjectStatusBadge from '@/components/projects/ProjectStatusBadge'
import { type ArchiveListItem, type ArchiveListResponse, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

export default function ProjectArchivePage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [items, setItems] = useState<ArchiveListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'' | 'closed' | 'archived'>('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

  const [reactivating, setReactivating] = useState<string | null>(null)

  const canManage = !!user && canManageProjects(user.role)

  const fetchArchive = useCallback(() => {
    if (!user) return
    setLoading(true)
    const params = new URLSearchParams()
    params.set('page', String(page))
    params.set('per_page', '25')
    if (search) params.set('search', search)
    if (statusFilter) params.set('status', statusFilter)
    if (dateFrom) params.set('date_from', dateFrom)
    if (dateTo) params.set('date_to', dateTo)

    apiClient
      .get<ArchiveListResponse>(`/projects/archive?${params.toString()}`)
      .then((res) => {
        setItems(res.data.projects ?? [])
        setTotal(res.data.total)
        setTotalPages(res.data.total_pages)
        setError(null)
      })
      .catch(() => setError('Greška pri učitavanju arhive.'))
      .finally(() => setLoading(false))
  }, [user, page, search, statusFilter, dateFrom, dateTo])

  useEffect(() => { fetchArchive() }, [fetchArchive])

  useEffect(() => { setPage(1) }, [search, statusFilter, dateFrom, dateTo])

  useEffect(() => {
    if (!isLoading && user?.role === 'administracija') router.replace('/dashboard')
  }, [isLoading, user, router])

  async function handleReactivate(id: string, name: string) {
    if (!confirm(`Reaktivirati projekt "${name}"?`)) return
    setReactivating(id)
    try {
      await apiClient.patch(`/projects/${id}/reactivate`)
      fetchArchive()
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } }
      alert(err?.response?.data?.error ?? 'Greška pri reaktiviranju.')
    } finally {
      setReactivating(null)
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user || user.role === 'administracija') return null

  const inputCls = 'w-full px-3.5 py-2 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text placeholder:text-gi-text-subtle text-sm focus:outline-none focus:border-gi-yellow transition-colors'

  return (
    <AppShell
      user={user}
      employee={employee}
      title="Arhiva projekata"
      backHref="/dashboard/projects"
      onLogout={logout}
    >
      <div className="space-y-5">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-gi-text">Arhiva projekata</h1>
            <p className="text-gi-text-subtle text-sm mt-0.5">Zatvoreni i arhivirani projekti</p>
          </div>
          <span className="text-gi-text-subtle text-sm">{total} projekt{total === 1 ? '' : 'a'}</span>
        </div>

        {/* Filters */}
        <div className="bg-gi-card border border-gi-border rounded-xl p-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Pretraži po nazivu..."
              className={inputCls}
            />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as '' | 'closed' | 'archived')}
              className={inputCls}
            >
              <option value="">Sve (zatvoreni + arhivirani)</option>
              <option value="closed">Samo zatvoreni</option>
              <option value="archived">Samo arhivirani</option>
            </select>
            <div className="flex flex-col gap-1">
              <label className="text-[10px] text-gi-text-subtle uppercase tracking-wider">Zatvoreno od</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className={inputCls}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-[10px] text-gi-text-subtle uppercase tracking-wider">Zatvoreno do</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className={inputCls}
              />
            </div>
          </div>
        </div>

        {error && (
          <div className="bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{error}</p>
          </div>
        )}

        {/* Table */}
        <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje arhive...</p>
            </div>
          ) : items.length === 0 ? (
            <div className="text-center py-12 text-gi-text-subtle text-sm">
              Nema arhiviranih ili zatvorenih projekata.
            </div>
          ) : (
            <>
              {/* Desktop */}
              <div className="hidden md:block overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gi-border">
                      <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Projekt</th>
                      <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Status</th>
                      <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Poslovođa</th>
                      <th className="text-left px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Zatvoreno</th>
                      <th className="text-right px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Izvještaji</th>
                      <th className="text-right px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Nabave</th>
                      <th className="text-right px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Materijali</th>
                      <th className="text-right px-4 py-3 text-gi-text-subtle font-medium text-xs uppercase tracking-wider">Akcije</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gi-border/60">
                    {items.map((item) => (
                      <tr key={item.id} className="hover:bg-gi-card-elevated transition-colors">
                        <td className="px-4 py-3">
                          <Link
                            href={`/dashboard/projects/archive/${item.id}`}
                            className="text-gi-text font-medium hover:text-gi-yellow transition-colors"
                          >
                            {item.name}
                          </Link>
                          {item.address && (
                            <p className="text-gi-text-subtle text-xs mt-0.5">{item.address}</p>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <ProjectStatusBadge status={item.status} />
                        </td>
                        <td className="px-4 py-3 text-gi-text-muted text-sm">
                          {item.primary_poslovoda_name ?? <span className="text-gi-text-subtle">—</span>}
                        </td>
                        <td className="px-4 py-3 text-gi-text-subtle text-xs">
                          {item.closed_at ? new Date(item.closed_at).toLocaleDateString('hr-HR') : '—'}
                          {item.closed_by_name && (
                            <p className="text-gi-text-subtle/60 mt-0.5">{item.closed_by_name}</p>
                          )}
                        </td>
                        <td className="px-4 py-3 text-right text-gi-text-muted">{item.daily_reports_count}</td>
                        <td className="px-4 py-3 text-right text-gi-text-muted">{item.material_purchases_count}</td>
                        <td className="px-4 py-3 text-right text-gi-text-muted">{item.project_materials_count}</td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-3">
                            <Link
                              href={`/dashboard/projects/archive/${item.id}`}
                              className="text-xs text-gi-text-muted hover:text-gi-yellow transition-colors"
                            >
                              Pregled
                            </Link>
                            {canManage && (
                              <button
                                onClick={() => handleReactivate(item.id, item.name)}
                                disabled={reactivating === item.id}
                                className="text-xs text-green-400 hover:text-green-300 disabled:opacity-40 transition-colors"
                              >
                                {reactivating === item.id ? '...' : 'Reaktiviraj'}
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Mobile cards */}
              <div className="md:hidden divide-y divide-gi-border/60">
                {items.map((item) => (
                  <div key={item.id} className="p-4 space-y-2">
                    <div className="flex items-start justify-between gap-2">
                      <Link
                        href={`/dashboard/projects/archive/${item.id}`}
                        className="text-gi-text font-medium hover:text-gi-yellow transition-colors flex-1 min-w-0 truncate"
                      >
                        {item.name}
                      </Link>
                      <ProjectStatusBadge status={item.status} />
                    </div>
                    {item.address && <p className="text-gi-text-subtle text-xs">{item.address}</p>}
                    <div className="flex items-center gap-4 text-xs text-gi-text-subtle">
                      <span>{item.primary_poslovoda_name ?? '—'}</span>
                      {item.closed_at && <span>{new Date(item.closed_at).toLocaleDateString('hr-HR')}</span>}
                    </div>
                    <div className="flex items-center gap-4 text-xs text-gi-text-subtle">
                      <span>{item.daily_reports_count} izvještaja</span>
                      <span>{item.project_materials_count} materijala</span>
                    </div>
                    <div className="flex gap-3 pt-1">
                      <Link href={`/dashboard/projects/archive/${item.id}`} className="text-xs text-gi-text-muted hover:text-gi-yellow transition-colors">
                        Pregled
                      </Link>
                      {canManage && (
                        <button
                          onClick={() => handleReactivate(item.id, item.name)}
                          disabled={reactivating === item.id}
                          className="text-xs text-green-400 hover:text-green-300 disabled:opacity-40 transition-colors"
                        >
                          {reactivating === item.id ? '...' : 'Reaktiviraj'}
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between text-sm">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-4 py-2 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text-muted disabled:opacity-40 hover:bg-gi-card-elevated/80 hover:text-gi-text transition-colors text-xs"
            >
              Prethodna
            </button>
            <span className="text-gi-text-subtle text-xs">Stranica {page} / {totalPages}</span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-4 py-2 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text-muted disabled:opacity-40 hover:bg-gi-card-elevated/80 hover:text-gi-text transition-colors text-xs"
            >
              Sljedeća
            </button>
          </div>
        )}
      </div>
    </AppShell>
  )
}
