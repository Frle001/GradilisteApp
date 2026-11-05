'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import { RoleBadge, StatusBadge } from '@/components/employees/EmployeeBadge'
import EmployeeAssetsList from '@/components/employees/EmployeeAssetsList'
import { type EmployeeDetail, type Asset, canManageEmployees, canResetPassword, fullName, ROLE_LABELS } from '@/lib/types/employees'
import { canViewOtherInventory } from '@/lib/types/inventory'
import apiClient from '@/lib/api-client'

type ResetState = 'idle' | 'confirming' | 'loading' | 'done'
type HardDeleteState = 'idle' | 'confirming' | 'loading'
type DeactivateState = 'idle' | 'confirming' | 'loading'

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

  const [deactivateState, setDeactivateState] = useState<DeactivateState>('idle')
  const [deactivateError, setDeactivateError] = useState<string | null>(null)

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

  async function handleActivate() {
    if (!emp) return
    setActionError(null)
    try {
      await apiClient.patch(`/employees/${id}/activate`)
      await fetchEmployee()
    } catch {
      setActionError('Greška pri aktiviranju zaposlenika.')
    }
  }

  async function handleDeactivateConfirmed() {
    if (!emp) return
    setDeactivateState('loading')
    setDeactivateError(null)
    try {
      await apiClient.patch(`/employees/${id}/deactivate`)
      setDeactivateState('idle')
      await fetchEmployee()
    } catch {
      setDeactivateError('Greška pri deaktivaciji zaposlenika.')
      setDeactivateState('confirming')
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
      <div className="min-h-screen bg-gi-navy flex items-center justify-center">
        <p className="text-red-400 text-sm">{error}</p>
      </div>
    )
  }
  if (!emp) return null

  const canManage = canManageEmployees(user.role)
  const canHardDelete = user.role === 'direktor'
  const canReset = canResetPassword(user.role)
  const canViewInv = canViewOtherInventory(user.role)
  const name = fullName(emp)
  const initials = `${emp.first_name[0] ?? ''}${emp.last_name[0] ?? ''}`.toUpperCase()
  const hardDeleteConfirmPhrase = name
  const showManagement = canManage || canViewInv
  const showSecurity = canReset || canManage || canHardDelete

  return (
    <AppShell
      user={user}
      employee={authEmployee}
      title={name}
      backHref="/dashboard/employees"
      onLogout={logout}
    >
      <div className="space-y-4 max-w-2xl">
        {/* ── Section 1: Identity card ── */}
        <div className="bg-gi-card border border-gi-border rounded-xl overflow-hidden">
          {/* Header strip */}
          <div className="px-5 pt-5 pb-4 flex items-start gap-4">
            <div className="w-12 h-12 rounded-full bg-gi-card-elevated border border-gi-border flex items-center justify-center shrink-0 select-none">
              <span className="text-base font-bold text-gi-text-muted tracking-tight">{initials}</span>
            </div>
            <div className="flex-1 min-w-0 pt-0.5">
              <h1 className="text-xl font-bold text-gi-text leading-tight truncate">{name}</h1>
              <div className="flex items-center gap-2 mt-1.5 flex-wrap">
                <RoleBadge role={emp.role} />
                <StatusBadge active={emp.active} />
              </div>
            </div>
          </div>

          {/* Info grid */}
          <div className="px-5 pb-5 grid grid-cols-2 sm:grid-cols-4 gap-y-3 gap-x-4 border-t border-gi-border pt-4">
            <InfoRow label="Email" value={emp.email ?? '—'} />
            <InfoRow label="Telefon" value={emp.phone ?? '—'} />
            {emp.supervisor ? (
              <InfoRow
                label="Nadzornik"
                value={`${fullName(emp.supervisor)} · ${ROLE_LABELS[emp.supervisor.role] ?? emp.supervisor.role}`}
              />
            ) : (
              <InfoRow label="Nadzornik" value="—" />
            )}
            <InfoRow
              label="Kreiran"
              value={new Date(emp.created_at).toLocaleDateString('hr-HR')}
            />
          </div>
        </div>

        {/* ── Section 2: Management actions ── */}
        {showManagement && (
          <div className="bg-gi-card border border-gi-border rounded-xl px-5 py-4">
            <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider mb-3">Upravljanje</h2>
            <div className="flex flex-wrap gap-2">
              {canManage && (
                <Link
                  href={`/dashboard/employees/${id}/edit`}
                  className="inline-flex items-center gap-1.5 text-sm text-gi-text-muted border border-gi-border hover:bg-gi-card-elevated px-3.5 py-2 rounded-lg transition-colors"
                >
                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z" />
                  </svg>
                  Uredi profil
                </Link>
              )}
              {canViewInv && (
                <Link
                  href={`/dashboard/inventory/employees/${id}`}
                  className="inline-flex items-center gap-1.5 text-sm text-gi-text-muted border border-gi-border hover:bg-gi-card-elevated px-3.5 py-2 rounded-lg transition-colors"
                >
                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
                  </svg>
                  Stanje robe
                </Link>
              )}
            </div>
            {actionError && <p className="text-red-400 text-sm mt-3">{actionError}</p>}
          </div>
        )}

        {/* ── Section 3: Security & Access ── */}
        {showSecurity && (
          <div className="bg-gi-card border border-gi-border rounded-xl px-5 py-4 space-y-4">
            <h2 className="text-xs font-medium text-gi-text-subtle uppercase tracking-wider">Pristup i sigurnost</h2>

            {/* Password reset */}
            {canReset && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gi-text-muted font-medium">Lozinka</p>
                    <p className="text-xs text-gi-text-subtle">Generiranje privremene lozinke za prijavu</p>
                  </div>
                  {resetState === 'idle' && (
                    <button
                      onClick={() => { setResetState('confirming'); setResetError(null) }}
                      className="text-sm border border-amber-800/60 text-amber-400 hover:bg-amber-950/40 px-3 py-1.5 rounded-lg transition-colors whitespace-nowrap"
                    >
                      Resetiraj lozinku
                    </button>
                  )}
                  {resetState === 'loading' && (
                    <span className="text-sm text-gi-text-subtle animate-pulse">Resetiranje...</span>
                  )}
                </div>
                {resetState === 'confirming' && (
                  <div className="bg-amber-950/20 border border-amber-800/40 rounded-lg px-4 py-3 space-y-2.5">
                    <p className="text-amber-200 text-sm font-medium">Resetirati lozinku za {name}?</p>
                    <p className="text-amber-300/70 text-xs">
                      Generirat će se nova privremena lozinka. Zaposlenik mora promijeniti lozinku pri sljedećoj prijavi. Prikazat će se samo jednom.
                    </p>
                    {resetError && <p className="text-red-400 text-xs">{resetError}</p>}
                    <div className="flex gap-2">
                      <button onClick={handleResetPassword} className="text-sm bg-amber-700 hover:bg-amber-600 text-white font-medium px-3 py-1.5 rounded-lg transition-colors">
                        Da, resetiraj
                      </button>
                      <button onClick={() => setResetState('idle')} className="text-sm border border-gi-border text-gi-text-muted hover:bg-gi-card-elevated px-3 py-1.5 rounded-lg transition-colors">
                        Odustani
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Activation / Deactivation */}
            {canManage && (
              <div className="space-y-2">
                <div className="flex items-center justify-between pt-1 border-t border-gi-border/60">
                  <div>
                    <p className="text-sm text-gi-text-muted font-medium">Status računa</p>
                    <p className="text-xs text-gi-text-subtle">
                      {emp.active ? 'Zaposlenik ima aktivan pristup sustavu' : 'Pristup je trenutno onemogućen'}
                    </p>
                  </div>
                  {deactivateState === 'idle' && (
                    <button
                      onClick={() => {
                        if (emp.active) {
                          setDeactivateState('confirming')
                          setDeactivateError(null)
                        } else {
                          handleActivate()
                        }
                      }}
                      className={`text-sm border px-3 py-1.5 rounded-lg transition-colors whitespace-nowrap ${
                        emp.active
                          ? 'border-red-800/60 text-red-400 hover:bg-red-950/40'
                          : 'border-green-800/60 text-green-400 hover:bg-green-950/40'
                      }`}
                    >
                      {emp.active ? 'Deaktiviraj' : 'Aktiviraj'}
                    </button>
                  )}
                  {deactivateState === 'loading' && (
                    <span className="text-sm text-gi-text-subtle animate-pulse">Obrada...</span>
                  )}
                </div>
                {deactivateState !== 'idle' && (
                  <div className="bg-red-950/20 border border-red-800/40 rounded-lg px-4 py-3 space-y-2.5">
                    <p className="text-red-200 text-sm font-medium">Deaktivirati zaposlenika?</p>
                    <p className="text-red-300/70 text-xs">
                      Zaposleniku će odmah biti ukinut pristup aplikaciji na svim uređajima. Više se neće moći prijaviti niti pristupati podacima tvrtke dok ponovno ne aktivirate račun.
                    </p>
                    {deactivateError && <p className="text-red-400 text-xs">{deactivateError}</p>}
                    <div className="flex gap-2">
                      <button
                        onClick={handleDeactivateConfirmed}
                        disabled={deactivateState === 'loading'}
                        className="text-sm bg-red-700 hover:bg-red-600 disabled:opacity-40 text-white font-medium px-3 py-1.5 rounded-lg transition-colors"
                      >
                        {deactivateState === 'loading' ? 'Deaktiviranje...' : 'Deaktiviraj i odjavi'}
                      </button>
                      <button
                        onClick={() => setDeactivateState('idle')}
                        disabled={deactivateState === 'loading'}
                        className="text-sm border border-gi-border text-gi-text-muted hover:bg-gi-card-elevated px-3 py-1.5 rounded-lg transition-colors disabled:opacity-40"
                      >
                        Odustani
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Hard delete */}
            {canHardDelete && (
              <div className="pt-1 border-t border-gi-border/60 space-y-2">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-red-500/80 font-medium">Trajno brisanje</p>
                    <p className="text-xs text-gi-text-subtle">Samo za testne ili greškom stvorene zapise</p>
                  </div>
                  <button
                    onClick={() => { setHardDeleteState('confirming'); setHardDeleteInput(''); setHardDeleteError(null) }}
                    className="text-xs border border-red-900/40 text-red-500/70 hover:bg-red-950/30 hover:text-red-400 px-3 py-1.5 rounded-lg transition-colors whitespace-nowrap"
                  >
                    Trajno izbriši
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Section 4: Assets ── */}
        <EmployeeAssetsList
          employeeID={id}
          assets={assets}
          canManage={canManage}
          onAssetChange={() => fetchAssets()}
        />
      </div>

      {/* Hard delete confirmation modal */}
      {hardDeleteState !== 'idle' && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="bg-gi-card border border-red-900/60 rounded-xl p-6 w-full max-w-md space-y-4 shadow-2xl">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-full bg-red-950/60 border border-red-900/60 flex items-center justify-center shrink-0">
                <svg className="w-4 h-4 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                </svg>
              </div>
              <p className="text-gi-text font-semibold">Trajno brisanje zaposlenika</p>
            </div>
            <div className="bg-red-950/20 border border-red-900/40 rounded-lg px-4 py-3 space-y-1">
              <p className="text-red-300 text-sm font-medium">Ova radnja je nepovratna.</p>
              <p className="text-gi-text-subtle text-xs">
                Koristite samo za testne ili slučajno stvorene zapise. Za stvarne zaposlenike koristite <strong className="text-gi-text-muted">deaktivaciju</strong>.
              </p>
            </div>
            <div>
              <p className="text-gi-text-muted text-sm mb-2">
                Upišite{' '}
                <span className="font-mono text-gi-text bg-gi-card-elevated px-1.5 py-0.5 rounded border border-gi-border">{hardDeleteConfirmPhrase}</span>
                {' '}ili{' '}
                <span className="font-mono text-gi-text bg-gi-card-elevated px-1.5 py-0.5 rounded border border-gi-border">IZBRIŠI</span>
                {' '}za potvrdu:
              </p>
              <input
                type="text"
                value={hardDeleteInput}
                onChange={e => setHardDeleteInput(e.target.value)}
                autoFocus
                className="w-full bg-gi-card-elevated border border-gi-border rounded-lg px-3 py-2 text-sm text-gi-text focus:outline-none focus:border-red-700 transition-colors"
                placeholder="Upišite potvrdu..."
                disabled={hardDeleteState === 'loading'}
              />
            </div>
            {hardDeleteError && <p className="text-red-400 text-sm">{hardDeleteError}</p>}
            <div className="flex gap-2">
              <button
                onClick={handleHardDelete}
                disabled={
                  hardDeleteState === 'loading' ||
                  (hardDeleteInput !== hardDeleteConfirmPhrase && hardDeleteInput !== 'IZBRIŠI')
                }
                className="flex-1 py-2.5 bg-red-700 hover:bg-red-600 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors"
              >
                {hardDeleteState === 'loading' ? 'Brisanje...' : 'Trajno izbriši'}
              </button>
              <button
                onClick={() => setHardDeleteState('idle')}
                disabled={hardDeleteState === 'loading'}
                className="flex-1 py-2.5 border border-gi-border text-gi-text-muted hover:bg-gi-card-elevated text-sm rounded-lg transition-colors disabled:opacity-40"
              >
                Odustani
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Reset password result modal */}
      {resetState === 'done' && resetPassword && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="bg-gi-card border border-gi-border rounded-xl p-6 w-full max-w-md space-y-4 shadow-2xl">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-full bg-amber-900/40 border border-amber-800/60 flex items-center justify-center shrink-0">
                <svg className="w-4 h-4 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                </svg>
              </div>
              <p className="text-gi-text font-semibold">Lozinka je resetirana</p>
            </div>
            <div className="bg-gi-card-elevated border border-gi-border rounded-lg px-4 py-3 space-y-2">
              <p className="text-gi-text-muted text-sm">Privremena lozinka za <span className="text-gi-text font-medium">{name}</span>:</p>
              <p className="font-mono text-lg text-gi-text bg-gi-card px-3 py-2 rounded border border-gi-border select-all break-all">
                {resetPassword}
              </p>
              <p className="text-amber-400 text-xs font-medium">
                Prikazano samo jednom — zabilježite i podijelite sigurnim kanalom.
              </p>
              <p className="text-gi-text-subtle text-xs">
                Zaposlenik mora promijeniti lozinku pri sljedećoj prijavi.
              </p>
            </div>
            <button
              onClick={() => { setResetState('idle'); setResetPassword(null) }}
              className="w-full py-2.5 px-4 bg-gi-yellow hover:bg-gi-yellow-hover text-gi-navy font-semibold rounded-lg text-sm transition-colors"
            >
              Zatvori
            </button>
          </div>
        </div>
      )}
    </AppShell>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider">{label}</p>
      <p className="text-sm text-gi-text mt-0.5 truncate">{value}</p>
    </div>
  )
}
