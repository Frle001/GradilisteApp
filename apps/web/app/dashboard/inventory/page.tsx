'use client'

import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
import InventoryOverview from '@/components/inventory/InventoryOverview'
import apiClient from '@/lib/api-client'
import type { InventoryResponse } from '@/lib/types/inventory'
import { canTransfer, canViewOtherInventory } from '@/lib/types/inventory'

export default function MyInventoryPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const [inventory, setInventory] = useState<InventoryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchInventory = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await apiClient.get('/inventory/me')
      setInventory(res.data)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? 'Greška pri učitavanju stanja.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchInventory() }, [fetchInventory])

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const showTransfers = canTransfer(user.role)
  const showOthers = canViewOtherInventory(user.role)

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Moje stanje robe"
      backHref="/dashboard"
      onLogout={logout}
      action={
        showTransfers ? (
          <Link
            href="/dashboard/inventory/transfers"
            className="px-4 py-2 bg-slate-700 text-white font-semibold rounded-lg text-sm hover:bg-slate-600 transition"
          >
            Povijest prijenosa
          </Link>
        ) : undefined
      }
    >
      <div className="max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Moje stanje robe</h1>
          <p className="text-sm text-slate-400 mt-1">Osobno dodijeljeni resursi i materijali</p>
        </div>

        {showOthers && (
          <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
            <p className="text-xs text-slate-400 uppercase tracking-wider font-medium mb-2">Pregled po zaposleniku</p>
            <p className="text-sm text-slate-300">
              Kao menadžment možete pregledati stanje robe za svakog zaposlenika putem{' '}
              <Link href="/dashboard/employees" className="text-blue-400 hover:underline">
                popisa zaposlenika
              </Link>.
            </p>
          </div>
        )}

        {loading ? (
          <LoadingOverlay />
        ) : error ? (
          <div className="rounded-xl bg-red-950 border border-red-800 text-red-300 text-sm px-4 py-3 flex items-center gap-2">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            {error}
          </div>
        ) : inventory ? (
          <InventoryOverview
            inventory={inventory}
            canInitiateTransfer={showTransfers}
            onTransferSuccess={fetchInventory}
          />
        ) : null}
      </div>
    </DashboardShell>
  )
}
