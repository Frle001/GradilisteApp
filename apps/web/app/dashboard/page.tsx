'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import apiClient from '@/lib/api-client'
import { clearAuth, getToken } from '@/lib/auth'

interface MeUser {
  id: string
  company_id: string
  employee_id: string | null
  email: string
  role: string
  active: boolean
}

interface MeEmployee {
  id: string
  first_name: string
  last_name: string
  role: string
}

interface DashboardCard {
  title: string
  description: string
  icon: string
}

// Role-based navigation sections (placeholders only — no modules implemented yet)
const roleCards: Record<string, DashboardCard[]> = {
  direktor: [
    { title: 'Dnevni izvještaji', description: 'Pregled svih dnevnih izvještaja', icon: '📋' },
    { title: 'Građevinski dnevnik', description: 'Evidencija radova na gradilištu', icon: '📓' },
    { title: 'Građevinska knjiga', description: 'Knjiga gradnje i dokumentacija', icon: '📚' },
    { title: 'Lista zaposlenika', description: 'Upravljanje osobljem', icon: '👥' },
    { title: 'Novi projekt', description: 'Pokretanje novog gradilišta', icon: '🏗️' },
  ],
  inzenjer: [
    { title: 'Dnevni izvještaji', description: 'Pregled svih dnevnih izvještaja', icon: '📋' },
    { title: 'Građevinski dnevnik', description: 'Evidencija radova na gradilištu', icon: '📓' },
    { title: 'Građevinska knjiga', description: 'Knjiga gradnje i dokumentacija', icon: '📚' },
    { title: 'Lista zaposlenika', description: 'Pregled osoblja', icon: '👥' },
    { title: 'Novi projekt', description: 'Pokretanje novog gradilišta', icon: '🏗️' },
  ],
  administracija: [
    { title: 'Lista zaposlenika', description: 'Upravljanje zaposlenicima', icon: '👥' },
    { title: 'Oprema / alati / vozila', description: 'Evidencija i dodjela imovine', icon: '🔧' },
  ],
  poslovoda: [
    { title: 'Dnevni izvještaj', description: 'Unos dnevnog izvještaja', icon: '📋' },
    { title: 'Upis materijala', description: 'Evidencija nabave materijala', icon: '📦' },
    { title: 'Osobno stanje robe', description: 'Pregled osobne zadužnice', icon: '🗂️' },
  ],
}

const roleLabels: Record<string, string> = {
  direktor: 'Direktor',
  inzenjer: 'Inženjer',
  administracija: 'Administracija',
  poslovoda: 'Poslovođa',
}

export default function DashboardPage() {
  const router = useRouter()
  const [user, setUser] = useState<MeUser | null>(null)
  const [employee, setEmployee] = useState<MeEmployee | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!getToken()) {
      router.replace('/login')
      return
    }

    apiClient
      .get('/auth/me')
      .then((res) => {
        setUser(res.data.user)
        setEmployee(res.data.employee ?? null)
      })
      .catch(() => {
        // Token invalid or expired
        clearAuth()
        router.replace('/login')
      })
      .finally(() => setIsLoading(false))
  }, [router])

  function handleLogout() {
    apiClient.post('/auth/logout').finally(() => {
      clearAuth()
      router.push('/login')
    })
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <p className="text-slate-400 text-sm">Učitavanje...</p>
      </div>
    )
  }

  if (!user) return null

  const displayName = employee
    ? `${employee.first_name} ${employee.last_name}`
    : user.email

  const cards = roleCards[user.role] ?? []

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      {/* Header */}
      <header className="border-b border-slate-800 bg-slate-900">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div>
            <span className="font-bold text-white text-lg">Gradilište App</span>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-right hidden sm:block">
              <p className="text-sm font-medium text-white">{displayName}</p>
              <p className="text-xs text-slate-400">{roleLabels[user.role] ?? user.role}</p>
            </div>
            <button
              onClick={handleLogout}
              className="text-sm text-slate-400 hover:text-white border border-slate-700 hover:border-slate-500 px-3 py-1.5 rounded-lg transition"
            >
              Odjava
            </button>
          </div>
        </div>
      </header>

      {/* Main */}
      <main className="max-w-5xl mx-auto px-6 py-10">
        {/* Welcome */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-white">
            Dobrodošli, {displayName}
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            {roleLabels[user.role] ?? user.role} &middot; {user.email}
          </p>
        </div>

        {/* Role-based navigation cards */}
        {cards.length > 0 ? (
          <>
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-4">
              Dostupni moduli
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {cards.map((card) => (
                <div
                  key={card.title}
                  className="bg-slate-900 border border-slate-800 rounded-xl p-5 flex flex-col gap-3 hover:border-slate-600 transition cursor-default"
                >
                  <span className="text-2xl">{card.icon}</span>
                  <div>
                    <h3 className="font-semibold text-white text-sm">{card.title}</h3>
                    <p className="text-slate-400 text-xs mt-0.5">{card.description}</p>
                  </div>
                  <span className="text-xs text-slate-600 font-medium">Uskoro dostupno</span>
                </div>
              ))}
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
    </div>
  )
}
