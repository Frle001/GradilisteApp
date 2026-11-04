'use client'

import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { ArrowLeftRight } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
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
    <AppShell
      user={user}
      employee={employee}
      title="Stanje robe"
      backHref="/dashboard"
      onLogout={logout}
      action={
        showTransfers ? (
          <Link
            href="/dashboard/inventory/transfers"
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-gi-card-elevated border border-gi-border hover:border-gi-border/80 text-gi-text-muted hover:text-gi-text text-xs font-medium transition-colors"
          >
            <ArrowLeftRight className="w-3.5 h-3.5" />Povijest prijenosa
          </Link>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div>
          <h1 className="text-xl font-bold text-gi-text">Stanje robe</h1>
          <p className="text-sm text-gi-text-subtle mt-0.5">Osobno dodijeljeni resursi i materijali</p>
        </div>

        {showOthers && (
          <div className="bg-gi-card border border-gi-border rounded-xl p-4">
            <p className="text-[10px] text-gi-text-subtle uppercase tracking-wider font-medium mb-1.5">Pregled po zaposleniku</p>
            <p className="text-sm text-gi-text-muted">
              Kao menadžment možete pregledati stanje robe za svakog zaposlenika putem{' '}
              <Link href="/dashboard/employees" className="text-gi-yellow hover:text-gi-yellow-hover">
                popisa zaposlenika
              </Link>.
            </p>
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center h-32 border border-gi-border rounded-xl bg-gi-card">
            <p className="text-gi-text-subtle text-sm animate-pulse">Učitavanje stanja robe...</p>
          </div>
        ) : error ? (
          <div className="rounded-xl bg-red-900/30 border border-red-800/60 text-red-300 text-sm px-4 py-3 flex items-center gap-2">
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
    </AppShell>
  )
}
