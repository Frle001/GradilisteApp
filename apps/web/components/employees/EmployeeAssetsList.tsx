'use client'

import { useState } from 'react'
import { type Asset, ASSET_TYPE_LABELS } from '@/lib/types/employees'
import { StatusBadge } from './EmployeeBadge'
import EmployeeAssetForm from './EmployeeAssetForm'
import apiClient from '@/lib/api-client'

interface EmployeeAssetsListProps {
  employeeID: string
  assets: Asset[]
  canManage: boolean
  onAssetChange: () => void
}

export default function EmployeeAssetsList({
  employeeID,
  assets,
  canManage,
  onAssetChange,
}: EmployeeAssetsListProps) {
  const [showForm, setShowForm] = useState(false)
  const [editingAsset, setEditingAsset] = useState<Asset | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function handleDeactivate(assetID: string) {
    setError(null)
    try {
      await apiClient.patch(`/employees/${employeeID}/assets/${assetID}/deactivate`)
      onAssetChange()
    } catch {
      setError('Greška pri deaktivaciji. Pokušajte ponovo.')
    }
  }

  function handleFormSuccess() {
    setShowForm(false)
    setEditingAsset(null)
    onAssetChange()
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">
          Zaduženja i oprema
        </h2>
        {canManage && !showForm && !editingAsset && (
          <button
            onClick={() => setShowForm(true)}
            className="text-xs text-white border border-slate-700 hover:border-slate-500 px-3 py-1.5 rounded-lg transition"
          >
            + Dodaj zaduženje
          </button>
        )}
      </div>

      {error && (
        <div className="bg-red-950 border border-red-800 rounded-lg px-4 py-3">
          <p className="text-red-300 text-sm">{error}</p>
        </div>
      )}

      {showForm && (
        <div className="bg-slate-900 border border-slate-700 rounded-xl p-5">
          <h3 className="text-sm font-medium text-white mb-4">Novo zaduženje</h3>
          <EmployeeAssetForm
            employeeID={employeeID}
            onSuccess={handleFormSuccess}
            onCancel={() => setShowForm(false)}
          />
        </div>
      )}

      {editingAsset && (
        <div className="bg-slate-900 border border-slate-700 rounded-xl p-5">
          <h3 className="text-sm font-medium text-white mb-4">Uredi zaduženje</h3>
          <EmployeeAssetForm
            employeeID={employeeID}
            asset={editingAsset}
            onSuccess={handleFormSuccess}
            onCancel={() => setEditingAsset(null)}
          />
        </div>
      )}

      {assets.length === 0 && !showForm ? (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 text-center">
          <p className="text-slate-400 text-sm">Nema evidentiranih zaduženja.</p>
        </div>
      ) : (
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <div className="divide-y divide-slate-800">
            {assets.map((asset) => (
              <div key={asset.id} className="px-5 py-4 flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm font-medium text-white">{asset.name}</span>
                    <span className="text-xs text-slate-500">
                      {ASSET_TYPE_LABELS[asset.asset_type] ?? asset.asset_type}
                    </span>
                    <StatusBadge active={asset.active} />
                  </div>
                  <div className="text-xs text-slate-400 mt-1 space-x-3">
                    <span>
                      Količina: {asset.quantity}
                      {asset.unit ? ` ${asset.unit}` : ''}
                    </span>
                    {asset.serial_number && <span>S/N: {asset.serial_number}</span>}
                  </div>
                  {asset.notes && (
                    <p className="text-xs text-slate-500 mt-1 truncate">{asset.notes}</p>
                  )}
                </div>
                {canManage && asset.active && (
                  <div className="flex items-center gap-3 shrink-0">
                    <button
                      onClick={() => setEditingAsset(asset)}
                      className="text-xs text-slate-400 hover:text-white transition"
                    >
                      Uredi
                    </button>
                    <button
                      onClick={() => handleDeactivate(asset.id)}
                      className="text-xs text-slate-500 hover:text-red-400 transition"
                    >
                      Razduženje
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
