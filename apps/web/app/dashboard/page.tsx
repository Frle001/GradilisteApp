'use client'

import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import { ROLE_LABELS } from '@/lib/types/employees'
import LoadingScreen from '@/components/ui/LoadingScreen'
import MobileBottomNav from '@/components/layout/MobileBottomNav'

interface DashboardCard {
  title: string
  description: string
  icon: string
  href?: string // set when the module is implemented
}

const roleCards: Record<string, DashboardCard[]> = {
  direktor: [
    { title: 'Zaposlenici', description: 'Upravljanje osobljem', icon: '👥', href: '/dashboard/employees' },
    { title: 'Projekti / Gradilišta', description: 'Pregled i upravljanje projektima', icon: '🏗️', href: '/dashboard/projects' },
    { title: 'Novi projekt', description: 'Pokretanje novog gradilišta', icon: '➕', href: '/dashboard/projects/new' },
    { title: 'Arhiva projekata', description: 'Zatvoreni i arhivirani projekti', icon: '🗄️', href: '/dashboard/projects/archive' },
    { title: 'Dnevni izvještaji', description: 'Pregled svih dnevnih izvještaja', icon: '📋', href: '/dashboard/daily-reports' },
    { title: 'Upisi materijala', description: 'Evidencija kupljenog materijala', icon: '📦', href: '/dashboard/material-purchases' },
    { title: 'Stanje robe', description: 'Osobna zadužnica i prijenosi', icon: '🗂️', href: '/dashboard/inventory' },
    { title: 'Građevinski dnevnik', description: 'Evidencija radnih sati po gradilištu', icon: '📓', href: '/dashboard/reports/gradevinski-dnevnik' },
    { title: 'Građevinska knjiga', description: 'Aktivnosti i utrošak materijala', icon: '📚', href: '/dashboard/reports/gradevinska-knjiga' },
  ],
  inzenjer: [
    { title: 'Zaposlenici', description: 'Pregled osoblja', icon: '👥', href: '/dashboard/employees' },
    { title: 'Projekti / Gradilišta', description: 'Pregled i upravljanje projektima', icon: '🏗️', href: '/dashboard/projects' },
    { title: 'Novi projekt', description: 'Pokretanje novog gradilišta', icon: '➕', href: '/dashboard/projects/new' },
    { title: 'Arhiva projekata', description: 'Zatvoreni i arhivirani projekti', icon: '🗄️', href: '/dashboard/projects/archive' },
    { title: 'Dnevni izvještaji', description: 'Pregled svih dnevnih izvještaja', icon: '📋', href: '/dashboard/daily-reports' },
    { title: 'Upisi materijala', description: 'Evidencija kupljenog materijala', icon: '📦', href: '/dashboard/material-purchases' },
    { title: 'Stanje robe', description: 'Osobna zadužnica i prijenosi', icon: '🗂️', href: '/dashboard/inventory' },
    { title: 'Građevinski dnevnik', description: 'Evidencija radnih sati po gradilištu', icon: '📓', href: '/dashboard/reports/gradevinski-dnevnik' },
    { title: 'Građevinska knjiga', description: 'Aktivnosti i utrošak materijala', icon: '📚', href: '/dashboard/reports/gradevinska-knjiga' },
  ],
  administracija: [
    { title: 'Zaposlenici', description: 'Upravljanje zaposlenicima', icon: '👥', href: '/dashboard/employees' },
    { title: 'Projekti / Gradilišta', description: 'Pregled projekata', icon: '🏗️', href: '/dashboard/projects' },
    { title: 'Arhiva projekata', description: 'Zatvoreni i arhivirani projekti', icon: '🗄️', href: '/dashboard/projects/archive' },
    { title: 'Upisi materijala', description: 'Pregled upisa materijala', icon: '📦', href: '/dashboard/material-purchases' },
    { title: 'Stanje robe', description: 'Osobna zadužnica i prijenosi', icon: '🗂️', href: '/dashboard/inventory' },
  ],
  poslovoda: [
    { title: 'Moj tim', description: 'Pregled vaših zaposlenika', icon: '👥', href: '/dashboard/employees' },
    { title: 'Moja gradilišta', description: 'Aktivni projekti dodijeljeni vama', icon: '🏗️', href: '/dashboard/projects' },
    { title: 'Dnevni izvještaj', description: 'Unos dnevnog izvještaja', icon: '📋', href: '/dashboard/daily-reports' },
    { title: 'Upis materijala', description: 'Evidentiranje kupljenog materijala', icon: '📦', href: '/dashboard/material-purchases/new' },
    { title: 'Moji upisi', description: 'Pregled vaših upisa materijala', icon: '🛒', href: '/dashboard/material-purchases' },
    { title: 'Osobno stanje robe', description: 'Zadužnica, oprema i prijenosi', icon: '🗂️', href: '/dashboard/inventory' },
    { title: 'Građevinski dnevnik', description: 'Evidencija radnih sati', icon: '📓', href: '/dashboard/reports/gradevinski-dnevnik' },
    { title: 'Građevinska knjiga', description: 'Aktivnosti i utrošak materijala', icon: '📚', href: '/dashboard/reports/gradevinska-knjiga' },
  ],
  radnik: [
    { title: 'Moji sati', description: 'Unos vlastitih dnevnih radnih sati po projektu.', icon: '⏱️', href: '/dashboard/my-hours' },
    { title: 'Projekti / Gradilišta', description: 'Pregled projekata tvrtke', icon: '🏗️', href: '/dashboard/projects' },
  ],
}

export default function DashboardPage() {
  const { user, employee, isLoading, logout } = useAuth()

  if (isLoading) return <LoadingScreen />

  if (!user) return null

  const displayName = employee
    ? `${employee.first_name} ${employee.last_name}`
    : user.email

  const cards = roleCards[user.role] ?? []

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      {/* Header */}
      <header className="border-b border-slate-800 bg-slate-900 pwa-header">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 py-3 sm:py-4 flex items-center justify-between">
          <div>
            <span className="font-bold text-white text-base sm:text-lg">Gradilište</span>
          </div>
          <div className="flex items-center gap-3 sm:gap-4">
            <div className="text-right hidden sm:block">
              <p className="text-sm font-medium text-white">{displayName}</p>
              <p className="text-xs text-slate-400">{ROLE_LABELS[user.role] ?? user.role}</p>
            </div>
            <button
              onClick={logout}
              className="text-sm text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-3 py-2 sm:py-1.5 rounded-lg transition"
            >
              Odjava
            </button>
          </div>
        </div>
      </header>

      {/* Main */}
      <main className="max-w-5xl mx-auto px-4 sm:px-6 py-6 sm:py-10 pb-nav sm:pb-10">
        {/* Welcome */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-white">
            Dobrodošli, {displayName}
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            {ROLE_LABELS[user.role] ?? user.role} &middot; {user.email}
          </p>
        </div>

        {/* Role-based navigation cards */}
        {cards.length > 0 ? (
          <>
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-4">
              Dostupni moduli
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {cards.map((card) =>
                card.href ? (
                  <Link
                    key={card.title}
                    href={card.href}
                    className="bg-slate-900 border border-slate-800 rounded-xl p-5 flex flex-col gap-3 hover:border-slate-600 hover:bg-slate-800/50 transition cursor-pointer"
                  >
                    <span className="text-2xl">{card.icon}</span>
                    <div>
                      <h3 className="font-semibold text-white text-sm">{card.title}</h3>
                      <p className="text-slate-400 text-xs mt-0.5">{card.description}</p>
                    </div>
                    <span className="text-xs text-slate-500 font-medium">Otvori →</span>
                  </Link>
                ) : (
                  <div
                    key={card.title}
                    className="bg-slate-900 border border-slate-800 rounded-xl p-5 flex flex-col gap-3 opacity-60"
                  >
                    <span className="text-2xl">{card.icon}</span>
                    <div>
                      <h3 className="font-semibold text-white text-sm">{card.title}</h3>
                      <p className="text-slate-400 text-xs mt-0.5">{card.description}</p>
                    </div>
                    <span className="text-xs text-slate-600 font-medium">Uskoro dostupno</span>
                  </div>
                )
              )}
            </div>
          </>
        ) : (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 text-center">
            <p className="text-slate-400 text-sm">Nema dostupnih modula za ovu ulogu.</p>
          </div>
        )}

        {/* Debug info in development */}
        {process.env.NODE_ENV === 'development' && (
          <div className="mt-10 bg-slate-900 border border-slate-800 rounded-xl p-5">
            <p className="text-xs font-mono text-slate-500 mb-2">Dev info</p>
            <pre className="text-xs text-slate-400 overflow-auto">
              {JSON.stringify({ user, employee }, null, 2)}
            </pre>
          </div>
        )}
      </main>

      <MobileBottomNav role={user.role} />
    </div>
  )
}
