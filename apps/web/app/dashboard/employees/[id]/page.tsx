'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import DashboardShell from '@/components/layout/DashboardShell'
import { RoleBadge, StatusBadge } from '@/components/employees/EmployeeBadge'
import EmployeeAssetsList from '@/components/employees/EmployeeAssetsList'
import { type EmployeeDetail, type Asset, canManageEmployees, canResetPassword, fullName } from '@/lib/types/employees'
import { canViewOtherInventory } from '@/lib/types/inventory'
import apiClient from '@/lib/api-client'

type ResetState = 'idle' | 'confirming' | 'loading' | 'done'
type HardDeleteState = 'idle' | 'confirming' | 'loading'

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

  const [resetState, setResetState] = useState<ResetState>('idle')
  const [resetPassword, setResetPassword] = useState<string | null>(null)
  const [resetError, setResetError] = useState<string | null>(null)

  const [hardDeleteState, setHardDeleteState] = useState<HardDeleteState>('idle')
  const [hardDeleteInput, setHardDeleteInput] = useState('')
  const [hardDeleteError, setHardDeleteError] = useState<string | null>(null)

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

  async function handleResetPassword() {
    setResetState('loading')
    setResetError(null)
    try {
      const res = await apiClient.patch(`/employees/${id}/reset-password`)
      setResetPassword(res.data.temporaryPassword)
      setResetState('done')
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setResetError(e?.response?.data?.error ?? 'Greška pri resetiranju lozinke.')
      setResetState('idle')
    }
  }

  async function handleHardDelete() {
    if (!emp) return
    setHardDeleteState('loading')
    setHardDeleteError(null)
    try {
      await apiClient.delete(`/employees/${id}`)
      router.replace('/dashboard/employees')
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setHardDeleteError(e?.response?.data?.error ?? 'Greška pri brisanju zaposlenika.')
      setHardDeleteState('confirming')
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
  const canHardDelete = user.role === 'direktor'
  const canReset = canResetPassword(user.role) && emp.role !== 'radnik'
  const canViewInv = canViewOtherInventory(user.role)
  const name = fullName(emp)
  const hardDeleteConfirmPhrase = name

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
            <div className="flex items-center gap-2 flex-wrap">
              {canReset && (
                <button
                  onClick={() => { setResetState('confirming'); setResetError(null) }}
                  className="text-sm border border-amber-800 text-amber-400 hover:bg-amber-950 px-3 py-1.5 rounded-lg transition"
                >
                  Resetiraj lozinku
                </button>
              )}
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
              {canHardDelete && (
                <button
                  onClick={() => { setHardDeleteState('confirming'); setHardDeleteInput(''); setHardDeleteError(null) }}
                  className="text-sm border border-red-900 text-red-500 hover:bg-red-950/50 px-3 py-1.5 rounded-lg transition"
                >
                  Trajno izbriši
                </button>
              )}
            </div>
          </div>

          {actionError && (
            <p className="text-red-400 text-sm">{actionError}</p>
          )}

          {resetError && (
            <p className="text-red-400 text-sm">{resetError}</p>
          )}

          {/* Reset password confirmation */}
          {resetState === 'confirming' && (
            <div className="bg-amber-950/40 border border-amber-800/60 rounded-lg px-4 py-3 space-y-3">
              <p className="text-amber-200 text-sm font-medium">Resetirati lozinku za {name}?</p>
              <p className="text-amber-300/70 text-xs">
                Generirat će se nova privremena lozinka. Zaposlenik mora promijeniti lozinku pri sljedećoj prijavi.
                Prikazat će se samo jednom.
              </p>
              <div className="flex gap-2">
                <button
                  onClick={handleResetPassword}
                  className="text-sm bg-amber-600 hover:bg-amber-500 text-white font-medium px-3 py-1.5 rounded-lg transition"
                >
                  Da, resetiraj
                </button>
                <button
                  onClick={() => setResetState('idle')}
                  className="text-sm border border-slate-700 text-slate-400 hover:bg-slate-800 px-3 py-1.5 rounded-lg transition"
                >
                  Odustani
                </button>
              </div>
            </div>
          )}

          {resetState === 'loading' && (
            <p className="text-slate-400 text-sm">Resetiranje lozinke...</p>
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

      {/* Hard delete confirmation modal */}
      {hardDeleteState !== 'idle' && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
          <div className="bg-slate-900 border border-red-900/60 rounded-xl p-6 w-full max-w-md space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-red-950 flex items-center justify-center shrink-0">
                <svg className="w-4 h-4 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
              </div>
              <p className="text-white font-semibold">Trajno brisanje zaposlenika</p>
            </div>
            <div className="bg-red-950/30 border border-red-900/40 rounded-lg px-4 py-3 space-y-1">
              <p className="text-red-300 text-sm font-medium">Ova radnja je nepovratna.</p>
              <p className="text-slate-400 text-xs">
                Koristite samo za testne ili slučajno stvorene zapise. Za stvarne zaposlenike koristite <strong className="text-slate-300">deaktivaciju</strong>.
              </p>
            </div>
            <div>
              <p className="text-slate-300 text-sm mb-2">
                Upišite <span className="font-mono text-white bg-slate-800 px-1.5 py-0.5 rounded">{hardDeleteConfirmPhrase}</span> ili <span className="font-mono text-white bg-slate-800 px-1.5 py-0.5 rounded">IZBRIŠI</span> za potvrdu:
              </p>
              <input
                type="text"
                value={hardDeleteInput}
                onChange={e => setHardDeleteInput(e.target.value)}
                autoFocus
                className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-1 focus:ring-red-700"
                placeholder="Upišite potvrdu..."
                disabled={hardDeleteState === 'loading'}
              />
            </div>
            {hardDeleteError && (
              <p className="text-red-400 text-sm">{hardDeleteError}</p>
            )}
            <div className="flex gap-2">
              <button
                onClick={handleHardDelete}
                disabled={
                  hardDeleteState === 'loading' ||
                  (hardDeleteInput !== hardDeleteConfirmPhrase && hardDeleteInput !== 'IZBRIŠI')
                }
                className="flex-1 py-2 bg-red-700 hover:bg-red-600 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition"
              >
                {hardDeleteState === 'loading' ? 'Brisanje...' : 'Trajno izbriši'}
              </button>
              <button
                onClick={() => setHardDeleteState('idle')}
                disabled={hardDeleteState === 'loading'}
                className="flex-1 py-2 border border-slate-700 text-slate-400 hover:bg-slate-800 text-sm rounded-lg transition disabled:opacity-40"
              >
                Odustani
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Reset password result modal */}
      {resetState === 'done' && resetPassword && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 w-full max-w-md space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-amber-900 flex items-center justify-center shrink-0">
                <span className="text-amber-400 text-sm font-bold">!</span>
              </div>
              <p className="text-white font-semibold">Lozinka je resetirana</p>
            </div>
            <div className="bg-slate-800 border border-slate-700 rounded-lg px-4 py-3 space-y-2">
              <p className="text-slate-300 text-sm">Privremena lozinka za <span className="text-white font-medium">{name}</span>:</p>
              <p className="font-mono text-lg text-white bg-slate-700 px-3 py-2 rounded select-all break-all">
                {resetPassword}
              </p>
              <p className="text-amber-400 text-xs font-medium">
                Prikazano samo jednom — zabilježite i podijelite sigurnim kanalom.
              </p>
              <p className="text-slate-400 text-xs">
                Zaposlenik mora promijeniti lozinku pri sljedećoj prijavi.
              </p>
            </div>
            <button
              onClick={() => { setResetState('idle'); setResetPassword(null) }}
              className="w-full py-2.5 px-4 bg-white text-slate-900 font-semibold rounded-lg text-sm hover:bg-slate-100 transition"
            >
              Zatvori
            </button>
          </div>
        </div>
      )}
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
