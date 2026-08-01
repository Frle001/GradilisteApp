'use client'

import { useState } from 'react'
import { type GradevinskiDnevnikRow } from '@/lib/types/reports'
import DailyReportStatusBadge from '@/components/daily-reports/DailyReportStatusBadge'
import { type ReportStatus } from '@/lib/types/daily-reports'
import WorkerHoursDetailPanel from '@/components/reports/WorkerHoursDetailPanel'

function formatDate(s: string) {
  const [y, m, d] = s.split('-')
  return `${d}.${m}.${y}`
}

interface Props {
  rows: GradevinskiDnevnikRow[]
  loading?: boolean
  userRole?: string
}

const isManager = (role?: string) => role === 'direktor' || role === 'inzenjer'

function LoadingSkeleton() {
  return (
    <div className="rounded-xl border border-slate-700 overflow-hidden">
      <div className="bg-slate-800/80 px-4 py-3 border-b border-slate-700 grid grid-cols-8 gap-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="h-3 rounded bg-slate-700 animate-pulse" />
        ))}
      </div>
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="px-4 py-3 border-b border-slate-800 grid grid-cols-8 gap-4 last:border-0">
          {Array.from({ length: 8 }).map((_, j) => (
            <div key={j} className="h-3 rounded bg-slate-800 animate-pulse" style={{ width: `${60 + (i * j * 7) % 40}%` }} />
          ))}
        </div>
      ))}
    </div>
  )
}

export default function GradevinskiDnevnikTable({ rows, loading, userRole }: Props) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const manager = isManager(userRole)

  if (loading) return <LoadingSkeleton />

  if (rows.length === 0) {
    return (
      <div className="rounded-xl border border-slate-700 bg-slate-800/30 flex flex-col items-center justify-center py-16 gap-3">
        <svg className="w-8 h-8 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
        </svg>
        <p className="text-slate-500 text-sm">Nema pronađenih zapisa</p>
      </div>
    )
  }

  return (
    <>
      {/* Desktop table */}
      <div className="hidden sm:block overflow-x-auto rounded-xl border border-slate-700">
        <table className="w-full text-sm text-left">
          <thead>
            <tr className="bg-slate-800/80 border-b border-slate-700">
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Datum</th>
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Gradilište</th>
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Poslovođa</th>
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Radnik</th>
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider text-right whitespace-nowrap">Sati</th>
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Status</th>
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Napomena</th>
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Izvor</th>
              {manager && (
                <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap"></th>
              )}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800">
            {rows.map((row, i) => (
              <tr
                key={`${row.report_id}-${row.worker_id}-${i}`}
                className="hover:bg-slate-800/40 transition-colors"
              >
                <td className="px-4 py-3 text-slate-300 whitespace-nowrap font-medium tabular-nums">
                  {formatDate(row.report_date)}
                </td>
                <td className="px-4 py-3 max-w-[220px]">
                  <p className="text-slate-100 font-medium truncate" title={row.project_name}>{row.project_name}</p>
                  {row.project_address && (
                    <p className="text-xs text-slate-500 truncate mt-0.5">{row.project_address}</p>
                  )}
                </td>
                <td className="px-4 py-3 text-slate-300 whitespace-nowrap">{row.poslovoda_name}</td>
                <td className="px-4 py-3 text-slate-200 whitespace-nowrap font-medium">
                  {row.worker_name}
                </td>
                <td className="px-4 py-3 text-right font-mono font-semibold text-slate-100 whitespace-nowrap">
                  {row.hours_worked.toFixed(2)}
                  <span className="text-slate-500 font-normal text-xs ml-0.5">h</span>
                </td>
                <td className="px-4 py-3">
                  {row.source === 'Radnik unos' || row.source === 'Poslovoda unos'
                    ? <span className="text-slate-600">—</span>
                    : <DailyReportStatusBadge status={row.status as ReportStatus} />
                  }
                </td>
                <td className="px-4 py-3 max-w-[200px]">
                  <div className="flex flex-col gap-1">
                    {row.notes
                      ? <span className="text-slate-400 truncate text-xs" title={row.notes}>{row.notes}</span>
                      : <span className="text-slate-700">—</span>
                    }
                    {row.work_description && (
                      <span className="text-xs text-emerald-500 italic truncate" title={row.work_description}>
                        Opis: {row.work_description}
                      </span>
                    )}
                    {manager && (
                      <div className="flex gap-1 mt-0.5 flex-wrap">
                        {row.has_revisions && (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-500/15 text-amber-300 border border-amber-600/40">
                            Ispravljeno
                          </span>
                        )}
                        {row.has_comments && (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-blue-500/15 text-blue-300 border border-blue-600/40">
                            Komentar
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                </td>
                <td className="px-4 py-3 whitespace-nowrap">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold ${
                    row.source === 'Radnik unos' || row.source === 'Poslovoda unos'
                      ? 'bg-violet-500/15 text-violet-300 border border-violet-500/40'
                      : 'bg-slate-700/70 text-slate-300 border border-slate-600'
                  }`}>
                    {row.source}
                  </span>
                </td>
                {manager && (
                  <td className="px-4 py-3 whitespace-nowrap">
                    {row.worker_hours_id ? (
                      <button
                        onClick={() => setSelectedId(row.worker_hours_id!)}
                        className="text-xs text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-2.5 py-1 rounded-lg transition"
                      >
                        Detalji
                      </button>
                    ) : (
                      <span className="text-slate-700 text-xs">—</span>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile cards */}
      <div className="sm:hidden space-y-3">
        {rows.map((row, i) => (
          <div
            key={`m-${row.report_id}-${row.worker_id}-${i}`}
            className="rounded-xl border border-slate-700 bg-slate-900 p-4 space-y-2"
          >
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-sm font-semibold text-white">{row.worker_name}</p>
                <p className="text-xs text-slate-400">{row.project_name}</p>
              </div>
              <div className="text-right shrink-0">
                <span className="text-lg font-bold text-white">{row.hours_worked.toFixed(2)}</span>
                <span className="text-slate-500 text-xs ml-0.5">h</span>
              </div>
            </div>

            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-xs text-slate-500">{formatDate(row.report_date)}</span>
              <span className="text-slate-700">·</span>
              <span className="text-xs text-slate-500">{row.poslovoda_name}</span>
              <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                row.source === 'Radnik unos' || row.source === 'Poslovoda unos'
                  ? 'bg-violet-500/15 text-violet-300 border border-violet-500/40'
                  : 'bg-slate-700/70 text-slate-300 border border-slate-600'
              }`}>
                {row.source}
              </span>
              {!(row.source === 'Radnik unos' || row.source === 'Poslovoda unos') && (
                <DailyReportStatusBadge status={row.status as ReportStatus} />
              )}
            </div>

            {(row.notes || row.work_description) && (
              <div className="space-y-0.5">
                {row.notes && <p className="text-xs text-slate-400 line-clamp-2">{row.notes}</p>}
                {row.work_description && (
                  <p className="text-xs text-emerald-500 italic line-clamp-1">Opis: {row.work_description}</p>
                )}
              </div>
            )}

            {manager && (
              <div className="flex items-center justify-between gap-2 pt-1">
                <div className="flex gap-1 flex-wrap">
                  {row.has_revisions && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-500/15 text-amber-300 border border-amber-600/40">
                      Ispravljeno
                    </span>
                  )}
                  {row.has_comments && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-blue-500/15 text-blue-300 border border-blue-600/40">
                      Komentar
                    </span>
                  )}
                </div>
                {row.worker_hours_id && (
                  <button
                    onClick={() => setSelectedId(row.worker_hours_id!)}
                    className="text-xs text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-2.5 py-1 rounded-lg transition"
                  >
                    Detalji
                  </button>
                )}
              </div>
            )}
          </div>
        ))}
      </div>

      {selectedId && (
        <WorkerHoursDetailPanel
          entryId={selectedId}
          onClose={() => setSelectedId(null)}
        />
      )}
    </>
  )
}
