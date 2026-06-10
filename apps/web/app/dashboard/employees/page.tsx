'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
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

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title={title}
      backHref="/dashboard"
      onLogout={logout}
      action={
        canManage ? (
          <Link
            href="/dashboard/employees/new"
            className="text-sm text-white bg-slate-700 hover:bg-slate-600 px-4 py-1.5 rounded-lg transition"
          >
            + Novi zaposlenik
          </Link>
        ) : undefined
      }
    >
      {/* Page heading */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">{title}</h1>
        <p className="text-slate-400 text-sm mt-1">
          {isPoslovoda
            ? 'Pregled vašeg tima i dodjeljenih zaposlenika'
            : 'Upravljanje zaposlenicima tvrtke'}
        </p>
      </div>

      {/* Filters */}
      <div className="mb-6 space-y-3">
        <form onSubmit={handleSearch} className="flex gap-2">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Pretraži po imenu, prezimenu, emailu..."
            className="flex-1 px-3.5 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 transition"
          />
          <button
            type="submit"
            className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white text-sm rounded-lg transition"
          >
            Traži
          </button>
        </form>

        <div className="flex items-center gap-4 flex-wrap">
          <select
            value={roleFilter}
            onChange={(e) => setRoleFilter(e.target.value)}
            className="px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-slate-500 transition"
          >
            <option value="">Sve uloge</option>
            {VALID_ROLES.map((r) => (
              <option key={r} value={r}>{ROLE_LABELS[r]}</option>
            ))}
          </select>

          <label className="flex items-center gap-2 text-sm text-slate-400 cursor-pointer">
            <input
              type="checkbox"
              checked={showInactive}
              onChange={(e) => setShowInactive(e.target.checked)}
              className="rounded"
            />
            Prikaži neaktivne
          </label>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4 bg-red-950 border border-red-800 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      )}

      {/* Table */}
      {dataLoading ? (
        <LoadingOverlay />
      ) : (
        <EmployeeTable
          employees={employees}
          canManage={canManage}
          onToggleActive={handleToggleActive}
        />
      )}
    </DashboardShell>
  )
}
