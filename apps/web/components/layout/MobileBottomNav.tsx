'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  Home,
  FolderOpen,
  FileText,
  Users,
  Clock,
  CalendarDays,
  Archive,
  ClipboardList,
  Receipt,
  BarChart2,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface NavItem {
  href: string
  label: string
  Icon: LucideIcon
  exact?: boolean
}

const navByRole: Record<string, NavItem[]> = {
  direktor: [
    { href: '/dashboard', label: 'Početna', Icon: Home, exact: true },
    { href: '/dashboard/projects', label: 'Projekti', Icon: FolderOpen },
    { href: '/dashboard/daily-reports', label: 'Izvještaji', Icon: FileText },
    { href: '/dashboard/analytics', label: 'Analitika', Icon: BarChart2 },
    { href: '/dashboard/employees', label: 'Zaposlenici', Icon: Users },
  ],
  inzenjer: [
    { href: '/dashboard', label: 'Početna', Icon: Home, exact: true },
    { href: '/dashboard/projects', label: 'Projekti', Icon: FolderOpen },
    { href: '/dashboard/daily-reports', label: 'Izvještaji', Icon: FileText },
    { href: '/dashboard/analytics', label: 'Analitika', Icon: BarChart2 },
    { href: '/dashboard/employees', label: 'Zaposlenici', Icon: Users },
  ],
  administracija: [
    { href: '/dashboard', label: 'Početna', Icon: Home, exact: true },
    { href: '/dashboard/employees', label: 'Zaposlenici', Icon: Users },
    { href: '/dashboard/zaduzenja', label: 'Zaduženja', Icon: Archive },
    { href: '/dashboard/dokumentacija', label: 'Dokumentacija', Icon: ClipboardList },
    { href: '/dashboard/racuni', label: 'Računi', Icon: Receipt },
  ],
  poslovoda: [
    { href: '/dashboard', label: 'Početna', Icon: Home, exact: true },
    { href: '/dashboard/schedule', label: 'Raspored', Icon: CalendarDays },
    { href: '/dashboard/my-hours', label: 'Moji sati', Icon: Clock },
    { href: '/dashboard/daily-reports', label: 'Izvještaji', Icon: FileText },
    { href: '/dashboard/projects', label: 'Projekti', Icon: FolderOpen },
  ],
  radnik: [
    { href: '/dashboard', label: 'Početna', Icon: Home, exact: true },
    { href: '/dashboard/schedule', label: 'Raspored', Icon: CalendarDays },
    { href: '/dashboard/my-hours', label: 'Moji sati', Icon: Clock },
    { href: '/dashboard/projects', label: 'Projekti', Icon: FolderOpen },
    { href: '/dashboard/r1', label: 'R1', Icon: Receipt },
  ],
}

interface Props {
  role: string
}

export default function MobileBottomNav({ role }: Props) {
  const pathname = usePathname()
  const items = navByRole[role] ?? []

  if (items.length === 0) return null

  function isActive(item: NavItem): boolean {
    if (item.exact) return pathname === item.href
    return pathname === item.href || pathname.startsWith(item.href + '/')
  }

  return (
    <nav
      className="fixed bottom-0 left-0 right-0 z-40 sm:hidden bg-slate-900/95 backdrop-blur border-t border-slate-800 pb-safe"
      aria-label="Navigacija"
    >
      <div className="flex items-stretch">
        {items.map((item) => {
          const active = isActive(item)
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex flex-1 flex-col items-center justify-center gap-0.5 py-2 min-h-[56px] text-[10px] font-medium transition-colors active:bg-slate-800 ${
                active
                  ? 'text-white'
                  : 'text-slate-500 hover:text-slate-300'
              }`}
              aria-current={active ? 'page' : undefined}
            >
              <item.Icon
                className={`w-5 h-5 ${active ? 'text-blue-400' : ''}`}
                strokeWidth={active ? 2.5 : 1.8}
              />
              <span>{item.label}</span>
            </Link>
          )
        })}
      </div>
    </nav>
  )
}
