import { ROLE_LABELS } from '@/lib/types/employees'

const roleColors: Record<string, string> = {
  direktor: 'bg-blue-950 text-blue-300 border-blue-800',
  inzenjer: 'bg-purple-950 text-purple-300 border-purple-800',
  administracija: 'bg-emerald-950 text-emerald-300 border-emerald-800',
  poslovoda: 'bg-orange-950 text-orange-300 border-orange-800',
  radnik: 'bg-slate-800 text-slate-300 border-slate-700',
}

interface RoleBadgeProps {
  role: string
  className?: string
}

export function RoleBadge({ role, className = '' }: RoleBadgeProps) {
  const color = roleColors[role] ?? 'bg-slate-800 text-slate-300 border-slate-700'
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded border text-xs font-medium ${color} ${className}`}
    >
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
          ? 'bg-green-950 text-green-300 border-green-800'
          : 'bg-red-950 text-red-400 border-red-900'
      } ${className}`}
    >
      {active ? 'Aktivan' : 'Neaktivan'}
    </span>
  )
}
