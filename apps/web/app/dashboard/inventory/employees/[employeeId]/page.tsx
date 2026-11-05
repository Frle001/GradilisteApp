'use client'

import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import AppShell from '@/components/layout/AppShell'
import InventoryOverview from '@/components/inventory/InventoryOverview'
import apiClient from '@/lib/api-client'
import type { InventoryResponse } from '@/lib/types/inventory'
import { employeeFullName, ROLE_LABELS } from '@/lib/types/inventory'

export default function EmployeeInventoryPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams()
  const employeeId = params.employeeId as string

  const [inventory, setInventory] = useState<InventoryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchInventory = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await apiClient.get(`/inventory/employees/${employeeId}`)
      setInventory(res.data)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? 'Greška pri učitavanju stanja.')
    } finally {
      setLoading(false)
    }
  }, [employeeId])

  useEffect(() => { fetchInventory() }, [fetchInventory])

  if (isLoading) return <LoadingScreen />
  if (!user) return null

  const empName = inventory ? employeeFullName(inventory.employee) : 'Zaposlenik'

  return (
    <AppShell
      user={user}
      employee={employee}
      title={`Stanje robe — ${empName}`}
      backHref="/dashboard/employees"
      onLogout={logout}
    >
      <div className="space-y-5">
        <div>
          <h1 className="text-xl font-bold text-gi-text">
            {inventory ? employeeFullName(inventory.employee) : 'Stanje robe'}
          </h1>
          {inventory && (
            <p className="text-sm text-gi-text-subtle mt-0.5">
              {ROLE_LABELS[inventory.employee.role] ?? inventory.employee.role} · Osobno zaduženi materijali i oprema
            </p>
          )}
        </div>

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
            canInitiateTransfer={false}
            onTransferSuccess={fetchInventory}
          />
        ) : null}
      </div>
    </AppShell>
  )
}
