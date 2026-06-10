'use client'

import Link from 'next/link'
import { type MeUser, type MeEmployee } from '@/hooks/useAuth'
import { ROLE_LABELS } from '@/lib/types/employees'
import MobileBottomNav from '@/components/layout/MobileBottomNav'

interface DashboardShellProps {
  user: MeUser
  employee: MeEmployee | null
  title: string
  backHref?: string
  action?: React.ReactNode
  children: React.ReactNode
  onLogout: () => void
  wide?: boolean
}

export default function DashboardShell({
  user,
  employee,
  title,
  backHref,
  action,
  children,
  onLogout,
  wide = false,
}: DashboardShellProps) {
  const displayName = employee
    ? `${employee.first_name} ${employee.last_name}`
    : user.email

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      {/* Header */}
      <header className="border-b border-slate-800 bg-slate-900 pwa-header">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 py-3 sm:py-4 flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 sm:gap-3 min-w-0">
            {backHref ? (
              <Link
                href={backHref}
                className="text-slate-400 hover:text-white shrink-0 transition p-1 -ml-1 rounded-lg"
                aria-label="Natrag"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
                </svg>
              </Link>
            ) : null}
            <Link href="/dashboard" className="font-bold text-white text-base sm:text-lg shrink-0">
              Gradilište
            </Link>
            {title ? (
              <>
                <span className="text-slate-600">/</span>
                <span className="text-slate-300 text-sm truncate">{title}</span>
              </>
            ) : null}
          </div>

          <div className="flex items-center gap-3 sm:gap-4 shrink-0">
            {action}
            <div className="text-right hidden sm:block">
              <p className="text-sm font-medium text-white">{displayName}</p>
              <p className="text-xs text-slate-400">{ROLE_LABELS[user.role] ?? user.role}</p>
            </div>
            <button
              onClick={onLogout}
              className="text-sm text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-3 py-2 sm:py-1.5 rounded-lg transition"
            >
              Odjava
            </button>
          </div>
        </div>
      </header>

      {/* Content — pb-nav adds bottom padding on mobile to avoid nav overlap */}
      <main
        className={`${wide ? 'max-w-screen-xl' : 'max-w-5xl'} mx-auto px-4 sm:px-6 py-6 sm:py-10 pb-nav sm:pb-10`}
      >
        {children}
      </main>

      {/* Mobile bottom navigation */}
      <MobileBottomNav role={user.role} />
    </div>
  )
}
