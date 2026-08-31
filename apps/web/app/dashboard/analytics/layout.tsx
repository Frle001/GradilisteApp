'use client'

import { useEffect } from 'react'
import { useRouter, usePathname } from 'next/navigation'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'

const FORBIDDEN_ROLES = ['administracija', 'poslovoda', 'radnik']

const SUBMODULES = [
  { href: '/dashboard/analytics/place', label: 'Plaće', emoji: '💰' },
]

export default function AnalytikaLayout({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    if (!isLoading && user && FORBIDDEN_ROLES.includes(user.role)) {
      router.replace('/dashboard')
    }
  }, [isLoading, user, router])

  if (isLoading || !user) return <LoadingScreen />
  if (FORBIDDEN_ROLES.includes(user.role)) return null

  return (
    <div className="min-h-screen bg-slate-950 text-white flex">
      {/* Sidebar — desktop */}
      <aside className="hidden md:flex flex-col w-52 shrink-0 border-r border-slate-800 bg-slate-900/60">
        <div className="px-4 pt-6 pb-4 border-b border-slate-800">
          <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Analitika</p>
        </div>
        <nav className="flex-1 px-2 py-3 space-y-0.5">
          {SUBMODULES.map((m) => {
            const active = pathname.startsWith(m.href)
            return (
              <Link
                key={m.href}
                href={m.href}
                className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  active
                    ? 'bg-blue-600/20 text-blue-300'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800'
                }`}
              >
                <span>{m.emoji}</span>
                <span>{m.label}</span>
              </Link>
            )
          })}
        </nav>
        <div className="px-4 py-4 border-t border-slate-800">
          <Link href="/dashboard" className="text-xs text-slate-500 hover:text-slate-300 transition-colors">
            ← Natrag
          </Link>
        </div>
      </aside>

      {/* Main */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Mobile top bar */}
        <div className="md:hidden flex items-center gap-2 px-4 py-3 border-b border-slate-800 bg-slate-900/60">
          <Link href="/dashboard" className="text-slate-500 hover:text-white text-sm">←</Link>
          <span className="text-slate-500 text-sm">Analitika</span>
          {SUBMODULES.filter((m) => pathname.startsWith(m.href)).map((m) => (
            <span key={m.href} className="text-white text-sm font-medium">/ {m.label}</span>
          ))}
          <div className="ml-auto flex gap-1">
            {SUBMODULES.map((m) => (
              <Link
                key={m.href}
                href={m.href}
                className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                  pathname.startsWith(m.href)
                    ? 'bg-blue-600/20 text-blue-300'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                {m.label}
              </Link>
            ))}
          </div>
        </div>

        <main className="flex-1 px-4 md:px-6 py-6 max-w-5xl w-full mx-auto">
          {children}
        </main>
      </div>
    </div>
  )
}
