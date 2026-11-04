'use client'

import { type MaterialListItem, TRACKING_TYPE_LABELS } from '@/lib/types/project-materials'

interface Props {
  materials: MaterialListItem[]
  canManage: boolean
  onEdit: (m: MaterialListItem) => void
  onDelete: (id: string) => void
  showInactive: boolean
}

function fmtQty(n: number): string {
  return n % 1 === 0 ? n.toString() : n.toFixed(2)
}

function StockIndicator({ available, planned }: { available: number; planned: number }) {
  if (available > 0 && available >= planned) {
    return <span className="text-green-400 font-semibold">{fmtQty(available)}</span>
  }
  if (available > 0) {
    return <span className="text-gi-yellow font-semibold">{fmtQty(available)}</span>
  }
  return <span className="text-red-400 font-semibold">{fmtQty(available)}</span>
}

export default function ProjectMaterialTable({ materials, canManage, onEdit, onDelete, showInactive }: Props) {
  const visible = showInactive ? materials : materials.filter((m) => m.active)

  if (visible.length === 0) {
    return (
      <div className="text-center py-12 text-gi-text-subtle text-sm">
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
            <tr className="border-b border-gi-border text-gi-text-subtle text-xs uppercase tracking-wider">
              <th className="text-left py-2.5 pr-4 font-medium">Naziv</th>
              <th className="text-left py-2.5 pr-4 font-medium">Šifra</th>
              <th className="text-right py-2.5 pr-4 font-medium">Planirano</th>
              <th className="text-right py-2.5 pr-4 font-medium">Utrošeno</th>
              <th className="text-right py-2.5 pr-4 font-medium">Dostupno</th>
              <th className="text-left py-2.5 pr-4 font-medium">JM</th>
              <th className="text-left py-2.5 pr-4 font-medium">Tip</th>
              <th className="text-left py-2.5 pr-4 font-medium">Izvor</th>
              {canManage && <th className="text-right py-2.5 font-medium">Akcije</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-gi-border/50">
            {visible.map((m) => (
              <tr key={m.id} className={`hover:bg-gi-card-elevated/40 transition-colors ${!m.active ? 'opacity-40' : ''}`}>
                <td className="py-2.5 pr-4 text-gi-text font-medium">{m.material_name}</td>
                <td className="py-2.5 pr-4 text-gi-text-subtle">{m.material_code ?? '—'}</td>
                <td className="py-2.5 pr-4 text-gi-text-muted text-right">{fmtQty(m.planned_quantity)}</td>
                <td className="py-2.5 pr-4 text-gi-text-muted text-right">{fmtQty(m.used_quantity)}</td>
                <td className="py-2.5 pr-4 text-right">
                  <StockIndicator available={m.available_quantity} planned={m.planned_quantity} />
                </td>
                <td className="py-2.5 pr-4 text-gi-text-subtle">{m.unit}</td>
                <td className="py-2.5 pr-4">
                  <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                    m.tracking_type === 'work'
                      ? 'bg-emerald-500/15 text-emerald-300 border border-emerald-600/40'
                      : 'bg-gi-card-elevated text-gi-text-muted border border-gi-border'
                  }`}>
                    {TRACKING_TYPE_LABELS[m.tracking_type] ?? m.tracking_type}
                  </span>
                </td>
                <td className="py-2.5 pr-4 text-gi-text-subtle text-xs">{m.source ?? '—'}</td>
                {canManage && (
                  <td className="py-2.5 text-right space-x-3 whitespace-nowrap">
                    {m.active && (
                      <>
                        <button
                          onClick={() => onEdit(m)}
                          className="text-xs text-gi-text-muted hover:text-gi-text transition-colors"
                        >
                          Uredi
                        </button>
                        <button
                          onClick={() => onDelete(m.id)}
                          className="text-xs text-red-500 hover:text-red-400 transition-colors"
                        >
                          Ukloni
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
      <div className="sm:hidden space-y-2.5">
        {visible.map((m) => (
          <div
            key={m.id}
            className={`bg-gi-card-elevated border border-gi-border rounded-xl p-4 ${!m.active ? 'opacity-40' : ''}`}
          >
            <div className="flex justify-between items-start gap-2 mb-2">
              <div>
                <p className="text-gi-text font-medium text-sm">{m.material_name}</p>
                {m.material_code && <p className="text-gi-text-subtle text-xs mt-0.5">{m.material_code}</p>}
                <span className={`inline-flex items-center mt-1.5 px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                  m.tracking_type === 'work'
                    ? 'bg-emerald-500/15 text-emerald-300 border border-emerald-600/40'
                    : 'bg-gi-card text-gi-text-muted border border-gi-border'
                }`}>
                  {TRACKING_TYPE_LABELS[m.tracking_type] ?? m.tracking_type}
                </span>
              </div>
              <span className="text-gi-text-subtle text-xs flex-shrink-0">{m.unit}</span>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs mb-3">
              <div>
                <p className="text-gi-text-subtle mb-0.5">Planirano</p>
                <p className="text-gi-text-muted font-medium">{fmtQty(m.planned_quantity)}</p>
              </div>
              <div>
                <p className="text-gi-text-subtle mb-0.5">Utrošeno</p>
                <p className="text-gi-text-muted font-medium">{fmtQty(m.used_quantity)}</p>
              </div>
              <div>
                <p className="text-gi-text-subtle mb-0.5">Dostupno</p>
                <StockIndicator available={m.available_quantity} planned={m.planned_quantity} />
              </div>
            </div>
            {canManage && m.active && (
              <div className="flex gap-3 pt-2.5 border-t border-gi-border">
                <button onClick={() => onEdit(m)} className="text-xs text-gi-text-muted hover:text-gi-text transition-colors">
                  Uredi
                </button>
                <button onClick={() => onDelete(m.id)} className="text-xs text-red-500 hover:text-red-400 transition-colors">
                  Ukloni
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
    </>
  )
}
