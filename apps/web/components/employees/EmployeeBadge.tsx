import { ROLE_LABELS } from '@/lib/types/employees'

const roleColors: Record<string, string> = {
  direktor:      'bg-blue-900/50 text-blue-300 border-blue-700',
  inzenjer:      'bg-purple-900/50 text-purple-300 border-purple-700',
  administracija:'bg-emerald-900/50 text-emerald-300 border-emerald-700',
  poslovoda:     'bg-amber-900/50 text-amber-300 border-amber-700',
  radnik:        'bg-gi-card-elevated text-gi-text-muted border-gi-border',
}

interface RoleBadgeProps {
  role: string
  className?: string
}

export function RoleBadge({ role, className = '' }: RoleBadgeProps) {
  const color = roleColors[role] ?? 'bg-gi-card-elevated text-gi-text-muted border-gi-border'
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded border text-xs font-medium ${color} ${className}`}>
      {ROLE_LABELS[role] ?? role}
    </span>
  )
}

interface StatusBadgeProps {
  active: boolean
  className?: string
}

export function StatusBadge({ active, className = '' }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded border text-xs font-medium ${
        active
          ? 'bg-green-900/40 text-green-300 border-green-800'
          : 'bg-red-900/40 text-red-400 border-red-800'
      } ${className}`}
    >
      {active ? 'Aktivan' : 'Neaktivan'}
    </span>
  )
}
