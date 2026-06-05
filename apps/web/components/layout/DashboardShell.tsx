'use client'

import Link from 'next/link'
import { type MeUser, type MeEmployee } from '@/hooks/useAuth'
import { ROLE_LABELS } from '@/lib/types/employees'

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
      <header className="border-b border-slate-800 bg-slate-900">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3 min-w-0">
            {backHref ? (
              <Link
                href={backHref}
                className="text-slate-400 hover:text-white shrink-0 transition"
                aria-label="Natrag"
              >
                ←
              </Link>
            ) : null}
            <Link href="/dashboard" className="font-bold text-white text-lg shrink-0">
              Gradilište
            </Link>
            {title ? (
              <>
                <span className="text-slate-600">/</span>
                <span className="text-slate-300 text-sm truncate">{title}</span>
              </>
            ) : null}
          </div>

          <div className="flex items-center gap-4 shrink-0">
            {action}
            <div className="text-right hidden sm:block">
              <p className="text-sm font-medium text-white">{displayName}</p>
              <p className="text-xs text-slate-400">{ROLE_LABELS[user.role] ?? user.role}</p>
            </div>
            <button
              onClick={onLogout}
              className="text-sm text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-3 py-1.5 rounded-lg transition"
            >
              Odjava
            </button>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className={`${wide ? 'max-w-screen-xl' : 'max-w-5xl'} mx-auto px-6 py-10`}>{children}</main>
    </div>
  )
}
