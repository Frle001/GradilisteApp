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

const isSelfEntered = (source: string) =>
  source === 'Radnik unos' || source === 'Poslovoda unos'

function LoadingSkeleton() {
  return (
    <div className="rounded-xl border border-gi-border overflow-hidden">
      <div className="bg-gi-card px-4 py-3 border-b border-gi-border grid grid-cols-8 gap-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="h-3 rounded bg-gi-border animate-pulse" />
        ))}
      </div>
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="px-4 py-3 border-b border-gi-border/60 grid grid-cols-8 gap-4 last:border-0">
          {Array.from({ length: 8 }).map((_, j) => (
            <div key={j} className="h-3 rounded bg-gi-card-elevated animate-pulse" style={{ width: `${60 + (i * j * 7) % 40}%` }} />
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
      <div className="rounded-xl border border-gi-border bg-gi-card flex flex-col items-center justify-center py-16 gap-3">
        <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
        </svg>
        <p className="text-gi-text-subtle text-sm">Nema pronađenih zapisa</p>
      </div>
    )
  }

  return (
    <>
      {/* Desktop table */}
      <div className="hidden sm:block overflow-x-auto rounded-xl border border-gi-border">
        <table className="w-full text-sm text-left">
          <thead>
            <tr className="bg-gi-card border-b border-gi-border">
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Datum</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider">Gradilište</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Poslovođa</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Radnik</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider text-right whitespace-nowrap">Sati</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Status</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider">Napomena</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Izvor</th>
              {manager && <th className="px-4 py-3" />}
            </tr>
          </thead>
          <tbody className="divide-y divide-gi-border/50">
            {rows.map((row, i) => (
              <tr
                key={`${row.report_id}-${row.worker_id}-${i}`}
                className="hover:bg-gi-navy-hover transition-colors"
              >
                <td className="px-4 py-3 text-gi-text-muted whitespace-nowrap font-medium tabular-nums">
                  {formatDate(row.report_date)}
                </td>
                <td className="px-4 py-3 max-w-[220px]">
                  <p className="text-gi-text font-medium truncate" title={row.project_name}>{row.project_name}</p>
                  {row.project_address && (
                    <p className="text-xs text-gi-text-subtle truncate mt-0.5">{row.project_address}</p>
                  )}
                </td>
                <td className="px-4 py-3 text-gi-text-muted whitespace-nowrap">{row.poslovoda_name}</td>
                <td className="px-4 py-3 text-gi-text whitespace-nowrap font-medium">
                  {row.worker_name}
                </td>
                <td className="px-4 py-3 text-right font-mono font-semibold text-gi-text whitespace-nowrap">
                  {row.hours_worked.toFixed(2)}
                  <span className="text-gi-text-subtle font-normal text-xs ml-0.5">h</span>
                </td>
                <td className="px-4 py-3">
                  {isSelfEntered(row.source)
                    ? <span className="text-gi-text-subtle">—</span>
                    : <DailyReportStatusBadge status={row.status as ReportStatus} />
                  }
                </td>
                <td className="px-4 py-3 max-w-[200px]">
                  <div className="flex flex-col gap-1">
                    {row.notes
                      ? <span className="text-gi-text-muted truncate text-xs" title={row.notes}>{row.notes}</span>
                      : <span className="text-gi-text-subtle">—</span>
                    }
                    {row.work_description && (
                      <span className="text-xs text-emerald-400 italic truncate" title={row.work_description}>
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
                    isSelfEntered(row.source)
                      ? 'bg-violet-500/15 text-violet-300 border border-violet-500/40'
                      : 'bg-gi-card-elevated text-gi-text-muted border border-gi-border'
                  }`}>
                    {row.source}
                  </span>
                </td>
                {manager && (
                  <td className="px-4 py-3 whitespace-nowrap">
                    {row.worker_hours_id ? (
                      <button
                        onClick={() => setSelectedId(row.worker_hours_id!)}
                        className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-2.5 py-1 rounded-lg transition-colors"
                      >
                        Detalji
                      </button>
                    ) : (
                      <span className="text-gi-text-subtle text-xs">—</span>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile cards */}
      <div className="sm:hidden rounded-xl border border-gi-border overflow-hidden divide-y divide-gi-border/50">
        {rows.map((row, i) => (
          <div
            key={`m-${row.report_id}-${row.worker_id}-${i}`}
            className="bg-gi-card p-4 space-y-2"
          >
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <p className="text-sm font-semibold text-gi-text">{row.worker_name}</p>
                <p className="text-xs text-gi-text-muted truncate">{row.project_name}</p>
              </div>
              <div className="text-right shrink-0">
                <span className="text-lg font-bold text-gi-text tabular-nums">{row.hours_worked.toFixed(2)}</span>
                <span className="text-gi-text-subtle text-xs ml-0.5">h</span>
              </div>
            </div>

            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-xs text-gi-text-subtle">{formatDate(row.report_date)}</span>
              <span className="text-gi-border">·</span>
              <span className="text-xs text-gi-text-subtle">{row.poslovoda_name}</span>
              <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                isSelfEntered(row.source)
                  ? 'bg-violet-500/15 text-violet-300 border border-violet-500/40'
                  : 'bg-gi-card-elevated text-gi-text-muted border border-gi-border'
              }`}>
                {row.source}
              </span>
              {!isSelfEntered(row.source) && (
                <DailyReportStatusBadge status={row.status as ReportStatus} />
              )}
            </div>

            {(row.notes || row.work_description) && (
              <div className="space-y-0.5">
                {row.notes && <p className="text-xs text-gi-text-muted line-clamp-2">{row.notes}</p>}
                {row.work_description && (
                  <p className="text-xs text-emerald-400 italic line-clamp-1">Opis: {row.work_description}</p>
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
                    className="text-xs text-gi-text-muted hover:text-gi-text border border-gi-border hover:border-gi-text-subtle px-2.5 py-1 rounded-lg transition-colors"
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
