'use client'

import { useState } from 'react'
import apiClient from '@/lib/api-client'
import { type ReportFilter } from '@/lib/types/reports'

interface Props {
  endpoint: string
  filename: string
  filter: ReportFilter
  disabled?: boolean
}

function filterToParams(f: ReportFilter): URLSearchParams {
  const p = new URLSearchParams()
  if (f.project_id)    p.set('project_id',    f.project_id)
  if (f.poslovoda_id)  p.set('poslovoda_id',   f.poslovoda_id)
  if (f.worker_id)     p.set('worker_id',      f.worker_id)
  if (f.material_id)   p.set('material_id',    f.material_id)
  if (f.activity_type) p.set('activity_type',  f.activity_type)
  if (f.is_vtk !== undefined) p.set('is_vtk', String(f.is_vtk))
  if (f.date_from)     p.set('date_from',      f.date_from)
  if (f.date_to)       p.set('date_to',        f.date_to)
  if (f.status)        p.set('status',         f.status)
  if (f.search)        p.set('search',         f.search)
  return p
}

export default function ExportExcelButton({ endpoint, filename, filter, disabled }: Props) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleExport() {
    setLoading(true)
    setError(null)
    try {
      const params = filterToParams(filter)
      const url = `${endpoint}?${params.toString()}`
      const res = await apiClient.get(url, { responseType: 'blob' })
      const blob = new Blob([res.data], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      })
      const href = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = href
      a.download = filename
      a.click()
      URL.revokeObjectURL(href)
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosErr = err as { response?: { data?: { error?: string } } }
        setError(axiosErr.response?.data?.error ?? 'Greška pri izvozu.')
      } else {
        setError('Greška pri izvozu.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col items-end gap-1">
      <button
        onClick={handleExport}
        disabled={disabled || loading}
        className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-green-700 hover:bg-green-600 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium transition-colors"
      >
        {loading ? (
          <>
            <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
            </svg>
            Izvoz…
          </>
        ) : (
          <>
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M7 10l5 5 5-5M12 15V3" />
            </svg>
            Izvoz Excel
          </>
        )}
      </button>
      {error && <span className="text-xs text-red-400">{error}</span>}
    </div>
  )
}
