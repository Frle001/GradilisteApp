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
    return <p className="text-gi-text-subtle text-sm">Nema dodijeljene opreme.</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gi-border">
            <th className="text-left text-xs font-medium text-gi-text-subtle uppercase tracking-wider pb-2.5 pr-4">Vrsta</th>
            <th className="text-left text-xs font-medium text-gi-text-subtle uppercase tracking-wider pb-2.5 pr-4">Naziv</th>
            <th className="text-right text-xs font-medium text-gi-text-subtle uppercase tracking-wider pb-2.5 pr-4">Kol.</th>
            <th className="text-left text-xs font-medium text-gi-text-subtle uppercase tracking-wider pb-2.5 pr-4">S/N</th>
            {canInitiateTransfer && <th className="pb-2.5" />}
          </tr>
        </thead>
        <tbody className="divide-y divide-gi-border/60">
          {assets.map(a => (
            <tr key={a.id} className="hover:bg-gi-card-elevated/40 transition-colors">
              <td className="py-2.5 pr-4">
                <AssetTypeBadge type={a.asset_type} />
              </td>
              <td className="py-2.5 pr-4 text-gi-text font-medium">
                {a.name}
                {a.notes && <span className="block text-xs text-gi-text-subtle font-normal">{a.notes}</span>}
              </td>
              <td className="py-2.5 pr-4 text-right text-gi-text-muted tabular-nums">
                {a.quantity.toFixed(2)}{a.unit ? ` ${a.unit}` : ''}
              </td>
              <td className="py-2.5 pr-4 text-gi-text-subtle text-xs font-mono">
                {a.serial_number ?? '—'}
              </td>
              {canInitiateTransfer && (
                <td className="py-2.5 text-right">
                  <button
                    onClick={() => onTransfer(a)}
                    className="text-xs text-gi-text-muted hover:text-gi-yellow transition-colors px-2 py-1 rounded hover:bg-gi-card-elevated"
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
