import { type ProjectStatus, STATUS_LABELS } from '@/lib/types/projects'

interface Props {
  status: ProjectStatus
}

const statusStyles: Record<ProjectStatus, string> = {
  active: 'bg-green-900 text-green-300 border-green-800',
  closed: 'bg-slate-800 text-slate-400 border-slate-700',
  archived: 'bg-amber-900 text-amber-300 border-amber-800',
}

export default function ProjectStatusBadge({ status }: Props) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border ${statusStyles[status] ?? statusStyles.closed}`}>
      {STATUS_LABELS[status] ?? status}
    </span>
  )
}
