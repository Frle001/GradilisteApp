'use client'

import type { InventoryMaterialItem } from '@/lib/types/inventory'

interface Props {
  materials: InventoryMaterialItem[]
  canInitiateTransfer: boolean
  onTransfer: (material: InventoryMaterialItem) => void
}

export default function MaterialResponsibilityList({ materials, canInitiateTransfer, onTransfer }: Props) {
  if (materials.length === 0) {
    return <p className="text-slate-500 text-sm">Nema materijala u odgovornosti.</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-700">
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Materijal</th>
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Gradilište</th>
            <th className="text-right text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Količina</th>
            {canInitiateTransfer && <th className="pb-2" />}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {materials.map(m => (
            <tr key={m.project_material_id + m.project_id} className="hover:bg-slate-800/40 transition-colors">
              <td className="py-2.5 pr-4 text-white font-medium">
                {m.material_name}
                {m.material_code && (
                  <span className="block text-xs text-slate-500 font-mono font-normal">{m.material_code}</span>
                )}
              </td>
              <td className="py-2.5 pr-4 text-slate-300 text-sm">{m.project_name}</td>
              <td className="py-2.5 pr-4 text-right text-slate-300">
                {m.total_quantity.toFixed(2)} {m.unit}
              </td>
              {canInitiateTransfer && (
                <td className="py-2.5 text-right">
                  <button
                    onClick={() => onTransfer(m)}
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
