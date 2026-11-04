import type { ReactNode } from 'react'
import { type GradevinskiDnevnikSummary, type GradevinskaKnjigaSummary } from '@/lib/types/reports'

interface CardProps {
  label: string
  value: string | number
  accentClass: string
  icon: ReactNode
}

function Card({ label, value, accentClass, icon }: CardProps) {
  return (
    <div className={`bg-gi-card border border-gi-border rounded-xl px-5 py-4 flex items-center gap-4 border-l-4 ${accentClass}`}>
      <div className="flex-shrink-0 text-gi-text-subtle">{icon}</div>
      <div className="min-w-0">
        <p className="text-xs font-medium text-gi-text-muted uppercase tracking-wider mb-1">{label}</p>
        <p className="text-2xl font-bold text-gi-text tabular-nums leading-none">{value}</p>
      </div>
    </div>
  )
}

export function DnevnikSummaryCards({ summary }: { summary: GradevinskiDnevnikSummary }) {
  const fmtHours = (n: number) =>
    n.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
      <Card
        label="Ukupno sati"
        value={fmtHours(summary.total_hours)}
        accentClass="border-l-gi-yellow"
        icon={
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        }
      />
      <Card
        label="Broj radnika"
        value={summary.workers_count.toLocaleString('hr-HR')}
        accentClass="border-l-violet-500"
        icon={
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
          </svg>
        }
      />
      <Card
        label="Gradilišta"
        value={summary.projects_count.toLocaleString('hr-HR')}
        accentClass="border-l-emerald-500"
        icon={
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 21h19.5m-18-18v18m10.5-18v18m6-13.5V21M6.75 6.75h.75m-.75 3h.75m-.75 3h.75m3-6h.75m-.75 3h.75m-.75 3h.75M6.75 21v-3.375c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21M3 3h12m-.75 4.5H21m-3.75 3.75h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008z" />
          </svg>
        }
      />
    </div>
  )
}

export function KnjigaSummaryCards({ summary }: { summary: GradevinskaKnjigaSummary }) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
      <Card
        label="Broj aktivnosti"
        value={summary.activities_count.toLocaleString('hr-HR')}
        accentClass="border-l-gi-yellow"
        icon={
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 6.75h12M8.25 12h12m-12 5.25h12M3.75 6.75h.007v.008H3.75V6.75zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zM3.75 12h.007v.008H3.75V12zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm-.375 5.25h.007v.008H3.75v-.008zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z" />
          </svg>
        }
      />
      <Card
        label="VTR stavki"
        value={summary.vtk_count.toLocaleString('hr-HR')}
        accentClass="border-l-amber-500"
        icon={
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L9.568 3z" />
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 6h.008v.008H6V6z" />
          </svg>
        }
      />
      <Card
        label="Gradilišta"
        value={summary.projects_count.toLocaleString('hr-HR')}
        accentClass="border-l-emerald-500"
        icon={
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 21h19.5m-18-18v18m10.5-18v18m6-13.5V21M6.75 6.75h.75m-.75 3h.75m-.75 3h.75m3-6h.75m-.75 3h.75m-.75 3h.75M6.75 21v-3.375c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21M3 3h12m-.75 4.5H21m-3.75 3.75h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008zm0 3h.008v.008h-.008v-.008z" />
          </svg>
        }
      />
    </div>
  )
}
