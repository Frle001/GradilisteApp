'use client'

import { useCallback, useEffect, useState } from 'react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import TransferHistoryTable from '@/components/inventory/TransferHistoryTable'
import ReportPaginationBar from '@/components/reports/ReportPaginationBar'
import apiClient from '@/lib/api-client'
import type { TransferListResponse } from '@/lib/types/inventory'

export default function TransfersPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const [data, setData] = useState<TransferListResponse | null>(null)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async (p: number) => {
    setLoading(true)
    setError(null)
    try {
      const res = await apiClient.get(`/inventory/transfers?page=${p}&per_page=25`)
      setData(res.data)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? 'Greška pri učitavanju prijenosa.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchData(page) }, [page, fetchData])

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Povijest prijenosa"
      backHref="/dashboard/inventory"
      onLogout={logout}
      wide
    >
      <div className="space-y-5">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Povijest prijenosa</h1>
          <p className="text-sm text-slate-400 mt-1">Evidencija prijenosa opreme i materijala između zaposlenika</p>
        </div>

        {error && (
          <div className="rounded-xl bg-red-950 border border-red-800 text-red-300 text-sm px-4 py-3 flex items-center gap-2">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            {error}
          </div>
        )}

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-5">
          <TransferHistoryTable
            items={data?.data ?? []}
            loading={loading}
            currentEmployeeId={employee?.id}
          />
        </div>

        {data && data.pagination.total > 0 && (
          <ReportPaginationBar
            pagination={data.pagination}
            onPageChange={setPage}
          />
        )}
      </div>
    </DashboardShell>
  )
}
