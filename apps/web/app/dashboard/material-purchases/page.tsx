'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import MaterialPurchaseTable from '@/components/material-purchases/MaterialPurchaseTable'
import ReportPaginationBar from '@/components/reports/ReportPaginationBar'
import apiClient from '@/lib/api-client'
import {
  type PurchaseListResponse,
  type PurchaseFilter,
  type PurchaseFormProject,
} from '@/lib/types/material-purchases'

const inputCls = `
  bg-slate-800 border border-slate-700 rounded-lg text-white text-sm
  placeholder-slate-500 px-3 py-2
  focus:outline-none focus:ring-2 focus:ring-slate-500 transition
`.replace(/\s+/g, ' ').trim()

export default function MaterialPurchasesPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const router = useRouter()

  const [filter, setFilter] = useState<PurchaseFilter>({ page: 1, per_page: 25 })
  const [data, setData] = useState<PurchaseListResponse | null>(null)
  const [projects, setProjects] = useState<PurchaseFormProject[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Load project options for filter dropdown
  useEffect(() => {
    apiClient.get('/material-purchases/form-data').then(res => {
      setProjects(res.data.projects ?? [])
    }).catch(() => {})
  }, [])

  const fetchData = useCallback(async (f: PurchaseFilter) => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams()
      if (f.project_id) params.set('project_id', f.project_id)
      if (f.buyer_id)   params.set('buyer_id', f.buyer_id)
      if (f.date_from)  params.set('date_from', f.date_from)
      if (f.date_to)    params.set('date_to', f.date_to)
      if (f.search)     params.set('search', f.search)
      params.set('page',     String(f.page ?? 1))
      params.set('per_page', String(f.per_page ?? 25))
      const res = await apiClient.get(`/material-purchases?${params}`)
      setData(res.data)
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: unknown } }
      const status = e?.response?.status
      const data = e?.response?.data
      const detail = (data && typeof data === 'object' && 'error' in data)
        ? (data as { error?: string }).error
        : typeof data === 'string' ? data : undefined
      let msg = 'Greška pri učitavanju podataka.'
      if (status !== undefined) msg += ` (HTTP ${status})`
      if (detail) msg += ` — ${detail}`
      setError(msg)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchData(filter) }, [filter, fetchData])

  useEffect(() => {
    if (!isLoading && user?.role === 'administracija') router.replace('/dashboard')
  }, [isLoading, user, router])

  function handleFilterChange(patch: Partial<PurchaseFilter>) {
    setFilter(prev => ({ ...prev, ...patch, page: 1 }))
  }

  if (isLoading) return <LoadingScreen />
  if (!user || user.role === 'administracija') return null

  const canCreate = ['direktor', 'inzenjer', 'poslovoda'].includes(user.role)
  const showBuyer = ['direktor', 'inzenjer', 'administracija'].includes(user.role)

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Upis materijala"
      backHref="/dashboard"
      onLogout={logout}
      wide
      action={
        canCreate ? (
          <Link
            href="/dashboard/material-purchases/new"
            className="px-4 py-2 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
          >
            + Novi upis
          </Link>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Upis materijala</h1>
          <p className="text-sm text-slate-400 mt-1">Evidencija kupljenog materijala po gradilištu</p>
        </div>

        {/* Filters */}
        <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-medium text-slate-400 uppercase tracking-wider">Gradilište</label>
              <select
                className={inputCls}
                value={filter.project_id ?? ''}
                onChange={e => handleFilterChange({ project_id: e.target.value || undefined })}
              >
                <option value="">Sva gradilišta</option>
                {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
              </select>
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-medium text-slate-400 uppercase tracking-wider">Datum od</label>
              <input
                type="date"
                className={inputCls}
                value={filter.date_from ?? ''}
                onChange={e => handleFilterChange({ date_from: e.target.value || undefined })}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-medium text-slate-400 uppercase tracking-wider">Datum do</label>
              <input
                type="date"
                className={inputCls}
                value={filter.date_to ?? ''}
                onChange={e => handleFilterChange({ date_to: e.target.value || undefined })}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-medium text-slate-400 uppercase tracking-wider">Pretraga</label>
              <input
                type="text"
                placeholder="Gradilište, kupac…"
                className={inputCls}
                value={filter.search ?? ''}
                onChange={e => handleFilterChange({ search: e.target.value || undefined })}
              />
            </div>
          </div>
        </div>

        {error && (
          <div className="rounded-xl bg-red-950 border border-red-800 text-red-300 text-sm px-4 py-3 flex items-center gap-2">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            {error}
          </div>
        )}

        <MaterialPurchaseTable rows={data?.data ?? []} loading={loading} showBuyer={showBuyer} />

        {data && data.pagination.total > 0 && (
          <ReportPaginationBar
            pagination={data.pagination}
            onPageChange={page => setFilter(prev => ({ ...prev, page }))}
          />
        )}
      </div>
    </DashboardShell>
  )
}
