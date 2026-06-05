'use client'

import { type MaterialListItem } from '@/lib/types/project-materials'

interface Props {
  materials: MaterialListItem[]
  canManage: boolean
  onEdit: (m: MaterialListItem) => void
  onDeactivate: (id: string) => void
  showInactive: boolean
}

export default function ProjectMaterialTable({ materials, canManage, onEdit, onDeactivate, showInactive }: Props) {
  const visible = showInactive ? materials : materials.filter((m) => m.active)

  if (visible.length === 0) {
    return (
      <div className="text-center py-10 text-slate-500 text-sm">
        Nema materijala za prikaz.
      </div>
    )
  }

  return (
    <>
      {/* Desktop table */}
      <div className="hidden sm:block overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800 text-slate-500 text-xs uppercase tracking-wide">
              <th className="text-left py-2 pr-4">Naziv</th>
              <th className="text-left py-2 pr-4">Šifra</th>
              <th className="text-right py-2 pr-4">Planirano</th>
              <th className="text-right py-2 pr-4">Utrošeno</th>
              <th className="text-right py-2 pr-4">Dostupno</th>
              <th className="text-left py-2 pr-4">JM</th>
              <th className="text-left py-2 pr-4">Izvor</th>
              {canManage && <th className="text-right py-2">Akcije</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60">
            {visible.map((m) => (
              <tr key={m.id} className={`${!m.active ? 'opacity-50' : ''}`}>
                <td className="py-2.5 pr-4 text-white font-medium">{m.material_name}</td>
                <td className="py-2.5 pr-4 text-slate-400">{m.material_code ?? '—'}</td>
                <td className="py-2.5 pr-4 text-slate-300 text-right">{fmtQty(m.planned_quantity)}</td>
                <td className="py-2.5 pr-4 text-slate-300 text-right">{fmtQty(m.used_quantity)}</td>
                <td className={`py-2.5 pr-4 text-right font-medium ${m.available_quantity <= 0 ? 'text-red-400' : 'text-green-400'}`}>
                  {fmtQty(m.available_quantity)}
                </td>
                <td className="py-2.5 pr-4 text-slate-400">{m.unit}</td>
                <td className="py-2.5 pr-4 text-slate-500 text-xs">{m.source ?? '—'}</td>
                {canManage && (
                  <td className="py-2.5 text-right space-x-2 whitespace-nowrap">
                    {m.active && (
                      <>
                        <button
                          onClick={() => onEdit(m)}
                          className="text-xs text-slate-400 hover:text-white transition"
                        >
                          Uredi
                        </button>
                        <button
                          onClick={() => onDeactivate(m.id)}
                          className="text-xs text-red-500 hover:text-red-400 transition"
                        >
                          Deaktiviraj
                        </button>
                      </>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile cards */}
      <div className="sm:hidden space-y-3">
        {visible.map((m) => (
          <div
            key={m.id}
            className={`bg-slate-800/50 border border-slate-700 rounded-lg p-4 ${!m.active ? 'opacity-50' : ''}`}
          >
            <div className="flex justify-between items-start gap-2 mb-2">
              <div>
                <p className="text-white font-medium text-sm">{m.material_name}</p>
                {m.material_code && <p className="text-slate-500 text-xs">{m.material_code}</p>}
              </div>
              <span className="text-slate-400 text-xs">{m.unit}</span>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs mb-3">
              <div>
                <p className="text-slate-500">Planirano</p>
                <p className="text-slate-300">{fmtQty(m.planned_quantity)}</p>
              </div>
              <div>
                <p className="text-slate-500">Utrošeno</p>
                <p className="text-slate-300">{fmtQty(m.used_quantity)}</p>
              </div>
              <div>
                <p className="text-slate-500">Dostupno</p>
                <p className={m.available_quantity <= 0 ? 'text-red-400 font-medium' : 'text-green-400 font-medium'}>
                  {fmtQty(m.available_quantity)}
                </p>
              </div>
            </div>
            {canManage && m.active && (
              <div className="flex gap-3 pt-2 border-t border-slate-700">
                <button onClick={() => onEdit(m)} className="text-xs text-slate-400 hover:text-white transition">
                  Uredi
                </button>
                <button onClick={() => onDeactivate(m.id)} className="text-xs text-red-500 hover:text-red-400 transition">
                  Deaktiviraj
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
    </>
  )
}

function fmtQty(n: number): string {
  return n % 1 === 0 ? n.toString() : n.toFixed(2)
}
