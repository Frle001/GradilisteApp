'use client'

import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import { ClipboardList, Archive, ArrowRight } from 'lucide-react'

export default function GradevinskaevidencijaPage() {
  const { user, employee, isLoading, logout } = useAuth()

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const cards = [
    {
      title: 'Građevinski dnevnik',
      description: 'Evidencija radnih sati po gradilištu',
      href: '/dashboard/reports/gradevinski-dnevnik',
      Icon: ClipboardList,
    },
    {
      title: 'Građevinska knjiga',
      description: 'Aktivnosti i utrošak materijala',
      href: '/dashboard/reports/gradevinska-knjiga',
      Icon: Archive,
    },
  ]

  return (
    <AppShell user={user} employee={employee} title="Građevinska evidencija" backHref="/dashboard" onLogout={logout}>
      <div className="mb-5 sm:mb-7">
        <h1 className="text-gi-text text-[22px] sm:text-[26px] font-bold leading-tight">Građevinska evidencija</h1>
        <p className="text-gi-text-subtle text-sm mt-1">Dnevnik i knjiga radova po gradilištu</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-2 sm:gap-2.5 max-w-2xl">
        {cards.map((card) => (
          <Link
            key={card.href}
            href={card.href}
            className="group flex items-start gap-3.5 p-4 rounded-xl border bg-gi-card border-gi-border hover:border-gi-border/80 hover:bg-gi-card-elevated transition-all duration-150"
          >
            <div className="w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5 bg-white/[0.06] text-gi-text-muted group-hover:bg-gi-yellow/10 group-hover:text-gi-yellow transition-colors duration-150">
              <card.Icon className="w-[17px] h-[17px]" strokeWidth={1.8} />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-gi-text text-[13px] font-semibold leading-snug">{card.title}</h3>
              <p className="text-gi-text-subtle text-[11px] mt-0.5 leading-snug">{card.description}</p>
            </div>
            <ArrowRight className="w-3.5 h-3.5 flex-shrink-0 mt-1 text-gi-text-subtle/50 group-hover:text-gi-yellow transition-colors duration-150" strokeWidth={2} />
          </Link>
        ))}
      </div>
    </AppShell>
  )
}
