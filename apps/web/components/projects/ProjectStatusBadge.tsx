import { type ProjectStatus, STATUS_LABELS } from '@/lib/types/projects'

interface Props {
  status: ProjectStatus
}

const statusStyles: Record<ProjectStatus, string> = {
  active: 'bg-green-900/50 text-green-300 border-green-800/60',
  closed: 'bg-gi-card-elevated text-gi-text-muted border-gi-border',
  archived: 'bg-amber-900/40 text-amber-300 border-amber-800/50',
}

export default function ProjectStatusBadge({ status }: Props) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border ${statusStyles[status] ?? statusStyles.closed}`}>
      {STATUS_LABELS[status] ?? status}
    </span>
  )
}
