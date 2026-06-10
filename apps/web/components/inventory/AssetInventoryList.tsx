'use client'

import AssetTypeBadge from './AssetTypeBadge'
import type { InventoryAssetItem } from '@/lib/types/inventory'

interface Props {
  assets: InventoryAssetItem[]
  canInitiateTransfer: boolean
  onTransfer: (asset: InventoryAssetItem) => void
}

export default function AssetInventoryList({ assets, canInitiateTransfer, onTransfer }: Props) {
  if (assets.length === 0) {
    return <p className="text-slate-500 text-sm">Nema dodijeljene opreme.</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-700">
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Vrsta</th>
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Naziv</th>
            <th className="text-right text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Kol.</th>
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">S/N</th>
            {canInitiateTransfer && <th className="pb-2" />}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {assets.map(a => (
            <tr key={a.id} className="hover:bg-slate-800/40 transition-colors">
              <td className="py-2.5 pr-4">
                <AssetTypeBadge type={a.asset_type} />
              </td>
              <td className="py-2.5 pr-4 text-white font-medium">
                {a.name}
                {a.notes && <span className="block text-xs text-slate-500 font-normal">{a.notes}</span>}
              </td>
              <td className="py-2.5 pr-4 text-right text-slate-300">
                {a.quantity.toFixed(2)}{a.unit ? ` ${a.unit}` : ''}
              </td>
              <td className="py-2.5 pr-4 text-slate-400 text-xs font-mono">
                {a.serial_number ?? '—'}
              </td>
              {canInitiateTransfer && (
                <td className="py-2.5 text-right">
                  <button
                    onClick={() => onTransfer(a)}
                    className="text-xs text-blue-400 hover:text-blue-300 transition-colors px-2 py-1 rounded hover:bg-slate-700"
                  >
                    Prenesi
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
