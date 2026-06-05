import { type GradevinskiDnevnikRow } from '@/lib/types/reports'
import DailyReportStatusBadge from '@/components/daily-reports/DailyReportStatusBadge'
import { type ReportStatus } from '@/lib/types/daily-reports'

function formatDate(s: string) {
  const [y, m, d] = s.split('-')
  return `${d}.${m}.${y}`
}

interface Props {
  rows: GradevinskiDnevnikRow[]
  loading?: boolean
}

function LoadingSkeleton() {
  return (
    <div className="rounded-xl border border-slate-700 overflow-hidden">
      <div className="bg-slate-800/80 px-4 py-3 border-b border-slate-700 grid grid-cols-7 gap-4">
        {Array.from({ length: 7 }).map((_, i) => (
          <div key={i} className="h-3 rounded bg-slate-700 animate-pulse" />
        ))}
      </div>
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="px-4 py-3 border-b border-slate-800 grid grid-cols-7 gap-4 last:border-0">
          {Array.from({ length: 7 }).map((_, j) => (
            <div key={j} className="h-3 rounded bg-slate-800 animate-pulse" style={{ width: `${60 + (i * j * 7) % 40}%` }} />
          ))}
        </div>
      ))}
    </div>
  )
}

export default function GradevinskiDnevnikTable({ rows, loading }: Props) {
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
    <div className="overflow-x-auto rounded-xl border border-slate-700">
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
              <td className="px-4 py-3 text-slate-200 whitespace-nowrap font-medium">{row.worker_name}</td>
              <td className="px-4 py-3 text-right font-mono font-semibold text-slate-100 whitespace-nowrap">
                {row.hours_worked.toFixed(2)}
                <span className="text-slate-500 font-normal text-xs ml-0.5">h</span>
              </td>
              <td className="px-4 py-3">
                <DailyReportStatusBadge status={row.status as ReportStatus} />
              </td>
              <td className="px-4 py-3 text-slate-400 max-w-[180px] truncate" title={row.notes ?? undefined}>
                {row.notes || <span className="text-slate-700">—</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
