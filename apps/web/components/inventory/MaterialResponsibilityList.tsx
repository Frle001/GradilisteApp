'use client'

import type { InventoryMaterialItem } from '@/lib/types/inventory'

interface Props {
  materials: InventoryMaterialItem[]
  canInitiateTransfer: boolean
  onTransfer: (material: InventoryMaterialItem) => void
}

export default function MaterialResponsibilityList({ materials, canInitiateTransfer, onTransfer }: Props) {
  if (materials.length === 0) {
    return <p className="text-gi-text-subtle text-sm">Nema materijala u odgovornosti.</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gi-border">
            <th className="text-left text-xs font-medium text-gi-text-subtle uppercase tracking-wider pb-2.5 pr-4">Materijal</th>
            <th className="text-left text-xs font-medium text-gi-text-subtle uppercase tracking-wider pb-2.5 pr-4">Gradilište</th>
            <th className="text-right text-xs font-medium text-gi-text-subtle uppercase tracking-wider pb-2.5 pr-4">Količina</th>
            {canInitiateTransfer && <th className="pb-2.5" />}
          </tr>
        </thead>
        <tbody className="divide-y divide-gi-border/60">
          {materials.map(m => (
            <tr key={m.project_material_id + m.project_id} className="hover:bg-gi-card-elevated/40 transition-colors">
              <td className="py-2.5 pr-4 text-gi-text font-medium">
                {m.material_name}
                {m.material_code && (
                  <span className="block text-xs text-gi-text-subtle font-mono font-normal">{m.material_code}</span>
                )}
              </td>
              <td className="py-2.5 pr-4 text-gi-text-muted text-sm">{m.project_name}</td>
              <td className="py-2.5 pr-4 text-right text-gi-text-muted tabular-nums">
                <span className={m.total_quantity <= 0 ? 'text-red-400' : ''}>
                  {m.total_quantity.toFixed(2)}
                </span>
                {' '}{m.unit}
              </td>
              {canInitiateTransfer && (
                <td className="py-2.5 text-right">
                  <button
                    onClick={() => onTransfer(m)}
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
