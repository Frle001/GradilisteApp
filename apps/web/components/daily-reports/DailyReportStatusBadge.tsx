import { type ReportStatus, REPORT_STATUS_LABELS } from '@/lib/types/daily-reports'

const STATUS_STYLES: Record<ReportStatus, string> = {
  draft:     'bg-slate-700/70 text-slate-300 border border-slate-600',
  submitted: 'bg-blue-500/15 text-blue-300 border border-blue-500/40',
  approved:  'bg-emerald-500/15 text-emerald-300 border border-emerald-500/40',
  rejected:  'bg-red-500/15 text-red-300 border border-red-500/40',
}

interface Props {
  status: ReportStatus
  className?: string
}

export default function DailyReportStatusBadge({ status, className = '' }: Props) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-semibold ${STATUS_STYLES[status]} ${className}`}>
      {REPORT_STATUS_LABELS[status]}
    </span>
  )
}
