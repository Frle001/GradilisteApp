'use client'

import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import LoadingScreen from '@/components/ui/LoadingScreen'
import LoadingOverlay from '@/components/ui/LoadingOverlay'
import DashboardShell from '@/components/layout/DashboardShell'
import InventoryOverview from '@/components/inventory/InventoryOverview'
import apiClient from '@/lib/api-client'
import type { InventoryResponse } from '@/lib/types/inventory'
import { employeeFullName } from '@/lib/types/inventory'

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
    <DashboardShell
      user={user}
      employee={employee}
      title={`Stanje robe — ${empName}`}
      backHref="/dashboard/employees"
      onLogout={logout}
    >
      <div className="max-w-4xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">
            {inventory ? `Stanje robe — ${employeeFullName(inventory.employee)}` : 'Stanje robe'}
          </h1>
          {inventory && (
            <p className="text-sm text-slate-400 mt-1">
              Uloga: {inventory.employee.role}
            </p>
          )}
        </div>

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
            canInitiateTransfer={false}
            onTransferSuccess={fetchInventory}
          />
        ) : null}
      </div>
    </DashboardShell>
  )
}
