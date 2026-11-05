'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import EmployeeTable from '@/components/employees/EmployeeTable'
import { type Employee, canManageEmployees, ROLE_LABELS, VALID_ROLES } from '@/lib/types/employees'
import apiClient from '@/lib/api-client'

export default function EmployeesPage() {
  const { user, employee, isLoading, logout } = useAuth()

  const [employees, setEmployees] = useState<Employee[]>([])
  const [dataLoading, setDataLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Filters
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState('')
  const [showInactive, setShowInactive] = useState(false)

  const fetchEmployees = () => {
    if (!user) return
    setDataLoading(true)
    setError(null)

    const params = new URLSearchParams()
    if (search) params.set('search', search)
    if (roleFilter) params.set('role', roleFilter)
    if (!showInactive) params.set('active', 'true')

    apiClient
      .get(`/employees?${params}`)
      .then((res) => setEmployees(res.data.employees ?? []))
      .catch(() => setError('Greška pri učitavanju zaposlenika.'))
      .finally(() => setDataLoading(false))
  }

  useEffect(() => {
    if (!isLoading && user) {
      fetchEmployees()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, user, showInactive, roleFilter])

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    fetchEmployees()
  }

  async function handleToggleActive(id: string, currentlyActive: boolean) {
    const endpoint = currentlyActive ? 'deactivate' : 'activate'
    try {
      await apiClient.patch(`/employees/${id}/${endpoint}`)
      fetchEmployees()
    } catch {
      setError('Greška pri promjeni statusa zaposlenika.')
    }
  }

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const canManage = canManageEmployees(user.role)
  const isPoslovoda = user.role === 'poslovoda'
  const title = isPoslovoda ? 'Moj tim' : 'Zaposlenici'

  const inputCls = 'px-3.5 py-2.5 bg-gi-card-elevated border border-gi-border rounded-lg text-gi-text placeholder:text-gi-text-subtle text-sm focus:outline-none focus:border-gi-yellow transition-colors'

  return (
    <AppShell
      user={user}
      employee={employee}
      title={title}
      backHref="/dashboard"
      onLogout={logout}
      action={
        canManage ? (
          <Link
            href="/dashboard/employees/new"
            className="inline-flex items-center gap-1.5 text-sm font-semibold bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy px-4 py-2 rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            Novi zaposlenik
          </Link>
        ) : undefined
      }
    >
      {/* Page heading + unified filter toolbar */}
      <div className="mb-5">
        <div className="flex items-baseline justify-between gap-4 mb-3">
          <div>
            <h1 className="text-xl font-bold text-gi-text">{title}</h1>
            <p className="text-gi-text-subtle text-xs mt-0.5">
              {isPoslovoda
                ? 'Pregled vašeg tima i dodjeljenih zaposlenika'
                : 'Upravljanje zaposlenicima tvrtke'}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <form onSubmit={handleSearch} className="flex gap-1.5 flex-1 min-w-[200px]">
            <div className="relative flex-1">
              <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gi-text-subtle pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
              </svg>
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Pretraži zaposlenike..."
                className={inputCls + ' w-full pl-8'}
              />
            </div>
            <button
              type="submit"
              className="px-3.5 py-2.5 bg-gi-card-elevated hover:bg-gi-border border border-gi-border text-gi-text-muted text-sm rounded-lg transition-colors whitespace-nowrap"
            >
              Traži
            </button>
          </form>

          <select
            value={roleFilter}
            onChange={(e) => setRoleFilter(e.target.value)}
            className={inputCls + ' min-w-[130px]'}
          >
            <option value="">Sve uloge</option>
            {VALID_ROLES.map((r) => (
              <option key={r} value={r}>{ROLE_LABELS[r]}</option>
            ))}
          </select>

          <label className="flex items-center gap-2 text-sm text-gi-text-muted cursor-pointer select-none whitespace-nowrap py-2.5 px-3.5 bg-gi-card-elevated border border-gi-border rounded-lg">
            <input
              type="checkbox"
              checked={showInactive}
              onChange={(e) => setShowInactive(e.target.checked)}
              className="rounded border-gi-border bg-transparent accent-gi-yellow"
            />
            Prikaži neaktivne
          </label>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4 bg-red-900/30 border border-red-800/60 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      )}

      {/* Table */}
      {dataLoading ? (
        <div className="bg-gi-card border border-gi-border rounded-xl p-10 text-center">
          <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje zaposlenika...</p>
        </div>
      ) : (
        <EmployeeTable
          employees={employees}
          canManage={canManage}
          onToggleActive={handleToggleActive}
        />
      )}
    </AppShell>
  )
}
