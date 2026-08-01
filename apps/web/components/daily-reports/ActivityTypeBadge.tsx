import { type ActivityType, ACTIVITY_TYPE_LABELS } from '@/lib/types/daily-reports'

const TYPE_STYLES: Record<ActivityType, string> = {
  montaza:   'bg-indigo-900/40 text-indigo-300 border border-indigo-800',
  demontaza: 'bg-orange-900/40 text-orange-300 border border-orange-800',
  other:     'bg-slate-800 text-slate-400 border border-slate-700',
}

interface Props {
  type: ActivityType
  isVtk?: boolean
  trackingType?: 'stock' | 'work'
  className?: string
}

export default function ActivityTypeBadge({ type, isVtk = false, trackingType, className = '' }: Props) {
  if (trackingType === 'work') {
    return (
      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-900/40 text-emerald-300 border border-emerald-800 ${className}`}>
        Radna aktivnost
      </span>
    )
  }
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${TYPE_STYLES[type]} ${className}`}>
      {ACTIVITY_TYPE_LABELS[type]}
      {isVtk && <span className="text-amber-400 font-semibold">VTR</span>}
    </span>
  )
}
