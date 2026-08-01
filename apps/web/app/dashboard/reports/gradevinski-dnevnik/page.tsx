'use client'

import { useEffect, useState, useCallback } from 'react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import apiClient from '@/lib/api-client'
import {
  type GradevinskiDnevnikResponse,
  type ReportFilter,
  type ReportFilterOptions,
} from '@/lib/types/reports'
import DashboardShell from '@/components/layout/DashboardShell'
import ReportFilters from '@/components/reports/ReportFilters'
import GradevinskiDnevnikTable from '@/components/reports/GradevinskiDnevnikTable'
import { DnevnikSummaryCards } from '@/components/reports/ReportSummaryCards'
import ReportPaginationBar from '@/components/reports/ReportPaginationBar'
import ExportExcelButton from '@/components/reports/ExportExcelButton'

export default function GradevinskiDnevnikPage() {
  const { user, employee, isLoading, logout } = useAuth()

  const [filter, setFilter] = useState<ReportFilter>({ page: 1, per_page: 25 })
  const [options, setOptions] = useState<ReportFilterOptions | null>(null)
  const [optionsLoading, setOptionsLoading] = useState(true)
  const [optionsError, setOptionsError] = useState<string | null>(null)
  const [data, setData] = useState<GradevinskiDnevnikResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    apiClient.get('/reports/filter-options')
      .then(res => { setOptions(res.data); setOptionsError(null) })
      .catch(() => {
        // Unblock the dropdowns even on failure — show empty lists with an error notice.
        setOptions({ projects: [], poslovode: [], workers: [], materials: [], activity_types: [], statuses: [] })
        setOptionsError('Greška pri učitavanju opcija filtera. Filteri gradilišta, radnika i poslovođe nisu dostupni.')
      })
      .finally(() => setOptionsLoading(false))
  }, [])

  const fetchData = useCallback(async (f: ReportFilter) => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams()
      if (f.project_id)    params.set('project_id',    f.project_id)
      if (f.poslovoda_id)  params.set('poslovoda_id',   f.poslovoda_id)
      if (f.worker_id)     params.set('worker_id',      f.worker_id)
      if (f.date_from)     params.set('date_from',      f.date_from)
      if (f.date_to)       params.set('date_to',        f.date_to)
      if (f.status)        params.set('status',         f.status)
      if (f.search)        params.set('search',         f.search)
      params.set('page',     String(f.page ?? 1))
      params.set('per_page', String(f.per_page ?? 25))
      const res = await apiClient.get(`/reports/gradevinski-dnevnik?${params}`)
      setData(res.data)
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { error?: string } } }
      const msg = e?.response?.data?.error
      const status = e?.response?.status
      setError(msg ? `${status ?? ''} ${msg}` : 'Greška pri učitavanju podataka.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchData(filter) }, [filter, fetchData])

  function handleFilterChange(f: ReportFilter) {
    setFilter({ ...f, page: 1, per_page: filter.per_page ?? 25 })
  }

  function handlePageChange(page: number) {
    setFilter(prev => ({ ...prev, page }))
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Građevinski dnevnik"
      backHref="/dashboard"
      onLogout={logout}
      wide
    >
      <div className="space-y-5">
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight">Građevinski dnevnik</h1>
            <p className="text-sm text-slate-400 mt-1">Pregled radnih sati po radnicima i gradilištu</p>
          </div>
          <ExportExcelButton
            endpoint="/reports/gradevinski-dnevnik/export"
            filename={`gradjevinski-dnevnik-${new Date().toISOString().slice(0, 10)}.xlsx`}
            filter={filter}
          />
        </div>

        <ReportFilters
          filter={filter}
          options={options}
          onChange={handleFilterChange}
          showWorker
          loading={optionsLoading}
          hideSelectFilters
        />

        {optionsError && (
          <div className="rounded-xl bg-amber-950 border border-amber-800 text-amber-300 text-sm px-4 py-3">
            {optionsError}
          </div>
        )}

        {data && <DnevnikSummaryCards summary={data.summary} />}

        {error && (
          <div className="rounded-xl bg-red-950 border border-red-800 text-red-300 text-sm px-4 py-3 flex items-center gap-2">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            {error}
          </div>
        )}

        <GradevinskiDnevnikTable rows={data?.data ?? []} loading={loading} userRole={user.role} />

        {data && data.pagination.total > 0 && (
          <ReportPaginationBar pagination={data.pagination} onPageChange={handlePageChange} />
        )}
      </div>
    </DashboardShell>
  )
}
