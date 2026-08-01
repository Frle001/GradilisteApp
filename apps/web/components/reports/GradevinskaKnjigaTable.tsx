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

function LoadingSkeleton() {
  const cols = 10
  return (
    <div className="rounded-xl border border-slate-700 overflow-hidden">
      <div className="bg-slate-800/80 px-4 py-3 border-b border-slate-700 grid gap-3" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
        {Array.from({ length: cols }).map((_, i) => (
          <div key={i} className="h-3 rounded bg-slate-700 animate-pulse" />
        ))}
      </div>
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="px-4 py-3 border-b border-slate-800 grid gap-3 last:border-0" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
          {Array.from({ length: cols }).map((_, j) => (
            <div key={j} className="h-3 rounded bg-slate-800 animate-pulse" style={{ width: `${55 + (i * j * 7) % 45}%` }} />
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
      <div className="rounded-xl border border-slate-700 bg-slate-800/30 flex flex-col items-center justify-center py-16 gap-3">
        <svg className="w-8 h-8 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3v11.25A2.25 2.25 0 006 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0118 16.5h-2.25m-7.5 0h7.5m-7.5 0l-1 3m8.5-3l1 3m0 0l.5 1.5m-.5-1.5h-9.5m0 0l-.5 1.5M9 11.25v1.5M12 9v3.75m3-6v6" />
        </svg>
        <p className="text-slate-500 text-sm">Nema pronađenih zapisa</p>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-slate-700">
      <table className="w-full text-sm text-left">
        <thead>
          <tr className="bg-slate-800/80 border-b border-slate-700">
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Datum</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Gradilište</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Poslovođa</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Materijal</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Aktivnost</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider text-right whitespace-nowrap">Količina</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Jed.</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">VTR</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Status</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Napomena</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {rows.map((row, i) => {
            const matName = row.material_name || row.custom_material_name || '—'
            const actLabel = ACTIVITY_TYPE_LABELS[row.activity_type as ActivityType] ?? row.activity_type
            return (
              <tr
                key={`${row.activity_id}-${i}`}
                className="hover:bg-slate-800/40 transition-colors"
              >
                <td className="px-4 py-3 text-slate-300 whitespace-nowrap font-medium tabular-nums">
                  {formatDate(row.report_date)}
                </td>
                <td className="px-4 py-3 max-w-[180px]">
                  <p className="text-slate-100 font-medium truncate" title={row.project_name}>{row.project_name}</p>
                  {row.project_address && (
                    <p className="text-xs text-slate-500 truncate mt-0.5">{row.project_address}</p>
                  )}
                </td>
                <td className="px-4 py-3 text-slate-300 whitespace-nowrap">{row.poslovoda_name}</td>
                <td className="px-4 py-3 max-w-[160px]">
                  <p className="text-slate-200 font-medium truncate" title={matName}>{matName}</p>
                  {row.material_code && (
                    <p className="text-xs text-slate-500 mt-0.5">{row.material_code}</p>
                  )}
                </td>
                <td className="px-4 py-3 text-slate-300 whitespace-nowrap">{actLabel}</td>
                <td className="px-4 py-3 text-right font-mono font-semibold text-slate-100 whitespace-nowrap">
                  {row.quantity.toFixed(2)}
                </td>
                <td className="px-4 py-3 text-slate-400 whitespace-nowrap">{row.unit}</td>
                <td className="px-4 py-3"><VTKBadge isVtk={row.is_vtk} /></td>
                <td className="px-4 py-3">
                  <DailyReportStatusBadge status={row.status as ReportStatus} />
                </td>
                <td className="px-4 py-3 text-slate-400 max-w-[150px] truncate" title={row.notes ?? undefined}>
                  {row.notes || <span className="text-slate-700">—</span>}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
