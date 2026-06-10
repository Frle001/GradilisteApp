'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import { RoleBadge, StatusBadge } from '@/components/employees/EmployeeBadge'
import EmployeeAssetsList from '@/components/employees/EmployeeAssetsList'
import { type EmployeeDetail, type Asset, canManageEmployees, fullName } from '@/lib/types/employees'
import { canViewOtherInventory } from '@/lib/types/inventory'
import apiClient from '@/lib/api-client'

export default function EmployeeDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string

  const { user, employee: authEmployee, isLoading, logout } = useAuth()
  const [emp, setEmp] = useState<EmployeeDetail | null>(null)
  const [assets, setAssets] = useState<Asset[]>([])
  const [dataLoading, setDataLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const fetchEmployee = () => {
    setError(null)
    return apiClient
      .get(`/employees/${id}`)
      .then((res) => setEmp(res.data.employee))
      .catch((err) => {
        if (err?.response?.status === 404 || err?.response?.status === 403) {
          router.replace('/dashboard/employees')
        } else {
          setError('Greška pri učitavanju zaposlenika.')
        }
      })
  }

  const fetchAssets = () => {
    return apiClient
      .get(`/employees/${id}/assets`)
      .then((res) => setAssets(res.data.assets ?? []))
      .catch(() => setAssets([]))
  }

  useEffect(() => {
    if (!isLoading && user) {
      setDataLoading(true)
      Promise.all([fetchEmployee(), fetchAssets()]).finally(() => setDataLoading(false))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, user, id])

  async function handleToggleActive() {
    if (!emp) return
    setActionError(null)
    const endpoint = emp.active ? 'deactivate' : 'activate'
    try {
      await apiClient.patch(`/employees/${id}/${endpoint}`)
      await fetchEmployee()
    } catch {
      setActionError('Greška pri promjeni statusa.')
    }
  }

  if (isLoading || dataLoading) return <LoadingScreen />
  if (!user) return null
  if (error) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <p className="text-red-400 text-sm">{error}</p>
      </div>
    )
  }
  if (!emp) return null

  const canManage = canManageEmployees(user.role)
  const canViewInv = canViewOtherInventory(user.role)
  const name = fullName(emp)

  return (
    <DashboardShell
      user={user}
      employee={authEmployee}
      title={name}
      backHref="/dashboard/employees"
      onLogout={logout}
      action={
        canManage ? (
          <div className="flex items-center gap-2">
            {canViewInv && (
              <Link
                href={`/dashboard/inventory/employees/${id}`}
                className="text-sm text-slate-300 border border-slate-700 hover:border-slate-500 px-3 py-1.5 rounded-lg transition"
              >
                Stanje robe
              </Link>
            )}
            <Link
              href={`/dashboard/employees/${id}/edit`}
              className="text-sm text-white border border-slate-700 hover:border-slate-500 px-3 py-1.5 rounded-lg transition"
            >
              Uredi
            </Link>
          </div>
        ) : undefined
      }
    >
      <div className="space-y-8">
        {/* Identity card */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-4">
          <div className="flex items-start justify-between gap-4 flex-wrap">
            <div>
              <h1 className="text-2xl font-bold text-white">{name}</h1>
              <div className="flex items-center gap-2 mt-2 flex-wrap">
                <RoleBadge role={emp.role} />
                <StatusBadge active={emp.active} />
              </div>
            </div>
            {canManage && (
              <button
                onClick={handleToggleActive}
                className={`text-sm border px-3 py-1.5 rounded-lg transition ${
                  emp.active
                    ? 'border-red-800 text-red-400 hover:bg-red-950'
                    : 'border-green-800 text-green-400 hover:bg-green-950'
                }`}
              >
                {emp.active ? 'Deaktiviraj' : 'Aktiviraj'}
              </button>
            )}
          </div>

          {actionError && (
            <p className="text-red-400 text-sm">{actionError}</p>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
            <InfoRow label="Email" value={emp.email ?? '—'} />
            <InfoRow label="Telefon" value={emp.phone ?? '—'} />
            {emp.supervisor && (
              <InfoRow
                label="Nadzornik"
                value={`${fullName(emp.supervisor)} (${emp.supervisor.role})`}
              />
            )}
            <InfoRow
              label="Kreiran"
              value={new Date(emp.created_at).toLocaleDateString('hr-HR')}
            />
          </div>
        </div>

        {/* Assets */}
        <EmployeeAssetsList
          employeeID={id}
          assets={assets}
          canManage={canManage}
          onAssetChange={() => fetchAssets()}
        />
      </div>
    </DashboardShell>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-slate-500 uppercase tracking-wider">{label}</p>
      <p className="text-sm text-white mt-0.5">{value}</p>
    </div>
  )
}
