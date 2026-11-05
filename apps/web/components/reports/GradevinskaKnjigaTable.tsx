import { type GradevinskaKnjigaRow } from '@/lib/types/reports'
import DailyReportStatusBadge from '@/components/daily-reports/DailyReportStatusBadge'
import { type ReportStatus, ACTIVITY_TYPE_LABELS, type ActivityType } from '@/lib/types/daily-reports'
import VTKBadge from '@/components/reports/VTKBadge'

function formatDate(s: string) {
  const [y, m, d] = s.split('-')
  return `${d}.${m}.${y}`
}

interface Props {
  rows: GradevinskaKnjigaRow[]
  loading?: boolean
}

const ACTIVITY_TYPE_BADGE: Record<string, string> = {
  montaza:   'bg-blue-900/40 text-blue-300 border border-blue-700/50',
  demontaza: 'bg-amber-900/40 text-amber-300 border border-amber-700/50',
  other:     'bg-gi-card-elevated text-gi-text-muted border border-gi-border',
}

function ActivityBadge({ type }: { type: string }) {
  const label = ACTIVITY_TYPE_LABELS[type as ActivityType] ?? type
  const cls = ACTIVITY_TYPE_BADGE[type] ?? ACTIVITY_TYPE_BADGE.other
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold ${cls}`}>
      {label}
    </span>
  )
}

function LoadingSkeleton() {
  const cols = 10
  return (
    <div className="rounded-xl border border-gi-border overflow-hidden">
      <div className="bg-gi-card px-4 py-3 border-b border-gi-border grid gap-3" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
        {Array.from({ length: cols }).map((_, i) => (
          <div key={i} className="h-3 rounded bg-gi-border animate-pulse" />
        ))}
      </div>
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="px-4 py-3 border-b border-gi-border/60 grid gap-3 last:border-0" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
          {Array.from({ length: cols }).map((_, j) => (
            <div key={j} className="h-3 rounded bg-gi-card-elevated animate-pulse" style={{ width: `${55 + (i * j * 7) % 45}%` }} />
          ))}
        </div>
      ))}
    </div>
  )
}

export default function GradevinskaKnjigaTable({ rows, loading }: Props) {
  if (loading) return <LoadingSkeleton />

  if (rows.length === 0) {
    return (
      <div className="rounded-xl border border-gi-border bg-gi-card flex flex-col items-center justify-center py-16 gap-3">
        <svg className="w-8 h-8 text-gi-text-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3v11.25A2.25 2.25 0 006 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0118 16.5h-2.25m-7.5 0h7.5m-7.5 0l-1 3m8.5-3l1 3m0 0l.5 1.5m-.5-1.5h-9.5m0 0l-.5 1.5M9 11.25v1.5M12 9v3.75m3-6v6" />
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
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider">Materijal</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Aktivnost</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider text-right whitespace-nowrap">Količina</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Jed.</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">VTR</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider whitespace-nowrap">Status</th>
              <th className="px-4 py-3 text-xs font-semibold text-gi-text-muted uppercase tracking-wider">Napomena</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gi-border/50">
            {rows.map((row, i) => {
              const matName = row.material_name || row.custom_material_name || '—'
              return (
                <tr
                  key={`${row.activity_id}-${i}`}
                  className="hover:bg-gi-navy-hover transition-colors"
                >
                  <td className="px-4 py-3 text-gi-text-muted whitespace-nowrap font-medium tabular-nums">
                    {formatDate(row.report_date)}
                  </td>
                  <td className="px-4 py-3 max-w-[180px]">
                    <p className="text-gi-text font-medium truncate" title={row.project_name}>{row.project_name}</p>
                    {row.project_address && (
                      <p className="text-xs text-gi-text-subtle truncate mt-0.5">{row.project_address}</p>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gi-text-muted whitespace-nowrap">{row.poslovoda_name}</td>
                  <td className="px-4 py-3 max-w-[160px]">
                    <p className="text-gi-text font-medium truncate" title={matName}>{matName}</p>
                    {row.material_code && (
                      <p className="text-xs text-gi-text-subtle mt-0.5">{row.material_code}</p>
                    )}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <ActivityBadge type={row.activity_type} />
                  </td>
                  <td className="px-4 py-3 text-right font-mono font-semibold text-gi-text whitespace-nowrap">
                    {row.quantity.toFixed(2)}
                  </td>
                  <td className="px-4 py-3 text-gi-text-muted whitespace-nowrap">{row.unit}</td>
                  <td className="px-4 py-3"><VTKBadge isVtk={row.is_vtk} /></td>
                  <td className="px-4 py-3">
                    <DailyReportStatusBadge status={row.status as ReportStatus} />
                  </td>
                  <td className="px-4 py-3 text-gi-text-muted max-w-[150px] truncate" title={row.notes ?? undefined}>
                    {row.notes || <span className="text-gi-text-subtle">—</span>}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* Mobile cards */}
      <div className="sm:hidden rounded-xl border border-gi-border overflow-hidden divide-y divide-gi-border/50">
        {rows.map((row, i) => {
          const matName = row.material_name || row.custom_material_name || '—'
          return (
            <div key={`m-${row.activity_id}-${i}`} className="bg-gi-card p-4 space-y-2">
              {/* Material name + quantity */}
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-gi-text truncate" title={matName}>{matName}</p>
                  {row.material_code && (
                    <p className="text-xs text-gi-text-subtle">{row.material_code}</p>
                  )}
                </div>
                <div className="shrink-0 text-right">
                  <span className="text-lg font-bold text-gi-text tabular-nums">{row.quantity.toFixed(2)}</span>
                  <span className="text-gi-text-muted text-xs ml-1">{row.unit}</span>
                </div>
              </div>

              {/* Date + project */}
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs text-gi-text-subtle">{formatDate(row.report_date)}</span>
                <span className="text-gi-border">·</span>
                <span className="text-xs text-gi-text-muted truncate">{row.project_name}</span>
              </div>

              {/* Badges */}
              <div className="flex items-center gap-2 flex-wrap">
                <ActivityBadge type={row.activity_type} />
                <VTKBadge isVtk={row.is_vtk} />
                <DailyReportStatusBadge status={row.status as ReportStatus} />
              </div>

              {/* Notes */}
              {row.notes && (
                <p className="text-xs text-gi-text-subtle line-clamp-2">{row.notes}</p>
              )}

              {/* Poslovoda */}
              <p className="text-xs text-gi-text-subtle">{row.poslovoda_name}</p>
            </div>
          )
        })}
      </div>
    </>
  )
}
