'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
import ProjectStatusBadge from '@/components/projects/ProjectStatusBadge'
import { type ArchiveListItem, type ArchiveListResponse, canManageProjects } from '@/lib/types/projects'
import apiClient from '@/lib/api-client'

const inputCls = `
  bg-slate-800 border border-slate-700 rounded-lg text-white text-sm
  placeholder-slate-500 px-3 py-2
  focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500
  transition-colors
`.replace(/\s+/g, ' ').trim()

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

  useEffect(() => {
    fetchArchive()
  }, [fetchArchive])

  // Reset to page 1 when filters change
  useEffect(() => {
    setPage(1)
  }, [search, statusFilter, dateFrom, dateTo])

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

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Arhiva projekata"
      backHref="/dashboard/projects"
      onLogout={logout}
    >
      <div className="space-y-5 max-w-6xl">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold text-white">Arhiva projekata</h1>
          <span className="text-slate-500 text-sm">{total} projekt{total === 1 ? '' : 'a'}</span>
        </div>

        {/* Filters */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Pretraži po nazivu..."
              className={inputCls + ' w-full'}
            />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as '' | 'closed' | 'archived')}
              className={inputCls + ' w-full'}
            >
              <option value="">Sve (zatvoreni + arhivirani)</option>
              <option value="closed">Samo zatvoreni</option>
              <option value="archived">Samo arhivirani</option>
            </select>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-slate-500">Zatvoreno od</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className={inputCls + ' w-full'}
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-slate-500">Zatvoreno do</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className={inputCls + ' w-full'}
              />
            </div>
          </div>
        </div>

        {error && (
          <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
            <p className="text-red-300 text-sm">{error}</p>
          </div>
        )}

        {/* Table */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          {loading ? (
            <LoadingOverlay />
          ) : items.length === 0 ? (
            <div className="text-center py-12 text-slate-500 text-sm">
              Nema arhiviranih ili zatvorenih projekata.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-800 text-left text-slate-500 text-xs uppercase tracking-wider">
                    <th className="px-4 py-3 font-medium">Projekt</th>
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium">Poslovođa</th>
                    <th className="px-4 py-3 font-medium">Zatvoreno</th>
                    <th className="px-4 py-3 font-medium text-right">Izvještaji</th>
                    <th className="px-4 py-3 font-medium text-right">Nabave</th>
                    <th className="px-4 py-3 font-medium text-right">Materijali</th>
                    <th className="px-4 py-3 font-medium text-right">Akcije</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800">
                  {items.map((item) => (
                    <tr key={item.id} className="hover:bg-slate-800/50 transition-colors">
                      <td className="px-4 py-3">
                        <Link
                          href={`/dashboard/projects/archive/${item.id}`}
                          className="text-white font-medium hover:text-blue-400 transition-colors"
                        >
                          {item.name}
                        </Link>
                        {item.address && (
                          <p className="text-slate-500 text-xs mt-0.5">{item.address}</p>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <ProjectStatusBadge status={item.status} />
                      </td>
                      <td className="px-4 py-3 text-slate-300">
                        {item.primary_poslovoda_name ?? <span className="text-slate-600">—</span>}
                      </td>
                      <td className="px-4 py-3 text-slate-400 text-xs">
                        {item.closed_at
                          ? new Date(item.closed_at).toLocaleDateString('hr-HR')
                          : '—'}
                        {item.closed_by_name && (
                          <p className="text-slate-600 mt-0.5">{item.closed_by_name}</p>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right text-slate-300">{item.daily_reports_count}</td>
                      <td className="px-4 py-3 text-right text-slate-300">{item.material_purchases_count}</td>
                      <td className="px-4 py-3 text-right text-slate-300">{item.project_materials_count}</td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Link
                            href={`/dashboard/projects/archive/${item.id}`}
                            className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
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
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between text-sm text-slate-400">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-3 py-1.5 bg-slate-800 rounded-lg disabled:opacity-40 hover:bg-slate-700 transition-colors"
            >
              Prethodna
            </button>
            <span>Stranica {page} / {totalPages}</span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-3 py-1.5 bg-slate-800 rounded-lg disabled:opacity-40 hover:bg-slate-700 transition-colors"
            >
              Sljedeća
            </button>
          </div>
        )}
      </div>
    </DashboardShell>
  )
}
