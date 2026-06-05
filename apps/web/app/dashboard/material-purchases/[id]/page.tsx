'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import DashboardShell from '@/components/layout/DashboardShell'
import ReceiptPreview from '@/components/material-purchases/ReceiptPreview'
import apiClient from '@/lib/api-client'
import { type PurchaseDetail } from '@/lib/types/material-purchases'

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('hr-HR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export default function MaterialPurchaseDetailPage() {
  const { user, employee, isLoading, logout } = useAuth()
  const params = useParams()
  const id = params.id as string

  const [purchase, setPurchase] = useState<PurchaseDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    apiClient.get(`/material-purchases/${id}`)
      .then(res => setPurchase(res.data))
      .catch(err => {
        const e = err as { response?: { status?: number; data?: { error?: string } } }
        if (e?.response?.status === 404) setError('Upis nije pronađen.')
        else if (e?.response?.status === 403) setError('Nemate pristup ovom upisu.')
        else setError(e?.response?.data?.error ?? 'Greška pri učitavanju.')
      })
      .finally(() => setLoading(false))
  }, [id])

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <p className="text-slate-400 text-sm">Učitavanje...</p>
      </div>
    )
  }
  if (!user) return null

  return (
    <DashboardShell
      user={user}
      employee={employee}
      title="Detalji upisa"
      backHref="/dashboard/material-purchases"
      onLogout={logout}
    >
      {loading && (
        <div className="flex justify-center py-16">
          <p className="text-slate-500 text-sm">Učitavanje…</p>
        </div>
      )}

      {error && (
        <div className="rounded-xl bg-red-950 border border-red-800 text-red-300 text-sm px-4 py-3">
          {error}
        </div>
      )}

      {purchase && (
        <div className="max-w-3xl space-y-6">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight">Upis materijala</h1>
            <p className="text-sm text-slate-400 mt-1">{purchase.project.name} · {formatDate(purchase.purchased_at)}</p>
          </div>

          {/* Summary card */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 grid grid-cols-2 gap-x-8 gap-y-4 sm:grid-cols-3">
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Gradilište</p>
              <p className="text-slate-100 font-medium mt-0.5">{purchase.project.name}</p>
              {purchase.project.address && (
                <p className="text-xs text-slate-500 mt-0.5">{purchase.project.address}</p>
              )}
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Kupac</p>
              <p className="text-slate-100 font-medium mt-0.5">{purchase.buyer.full_name}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Datum kupnje</p>
              <p className="text-slate-100 font-medium mt-0.5">{formatDate(purchase.purchased_at)}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Broj stavki</p>
              <p className="text-slate-100 font-medium mt-0.5">{purchase.items.length}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Uneseno</p>
              <p className="text-slate-100 font-medium mt-0.5">{formatDate(purchase.created_at)}</p>
            </div>
            {purchase.notes && (
              <div className="col-span-2 sm:col-span-3">
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wider">Napomena</p>
                <p className="text-slate-300 mt-0.5">{purchase.notes}</p>
              </div>
            )}
          </div>

          {/* Items table */}
          <div>
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">Kupljeni materijali</h2>
            <div className="overflow-x-auto rounded-xl border border-slate-700">
              <table className="w-full text-sm text-left">
                <thead>
                  <tr className="bg-slate-800/80 border-b border-slate-700">
                    <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Materijal</th>
                    <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider text-right">Količina</th>
                    <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Jed.</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800">
                  {purchase.items.map(item => (
                    <tr key={item.id} className="hover:bg-slate-800/40 transition-colors">
                      <td className="px-4 py-3">
                        <p className="text-slate-100 font-medium">{item.material_name}</p>
                        {item.material_code && <p className="text-xs text-slate-500">{item.material_code}</p>}
                      </td>
                      <td className="px-4 py-3 text-right font-mono font-semibold text-slate-100 tabular-nums">
                        {item.quantity.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                      </td>
                      <td className="px-4 py-3 text-slate-400">{item.unit}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Receipt */}
          <div>
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-3">Račun</h2>
            {purchase.receipt_file_url ? (
              <ReceiptPreview purchaseId={purchase.id} originalFilename={purchase.receipt_original_filename} />
            ) : (
              <div className="rounded-xl border border-slate-700 bg-slate-800/30 px-4 py-6 text-center">
                <p className="text-slate-500 text-sm">Nema priloženog računa za ovaj upis.</p>
              </div>
            )}
          </div>
        </div>
      )}
    </DashboardShell>
  )
}
