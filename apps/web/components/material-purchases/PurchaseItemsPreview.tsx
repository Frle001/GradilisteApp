'use client'

import { type PurchaseItemDraft } from '@/lib/types/material-purchases'

interface Props {
  items: PurchaseItemDraft[]
  onRemove: (tempId: string) => void
  disabled?: boolean
}

export default function PurchaseItemsPreview({ items, onRemove, disabled }: Props) {
  if (items.length === 0) {
    return (
      <div className="rounded-xl border border-slate-700 bg-slate-800/30 px-4 py-6 text-center">
        <p className="text-slate-500 text-sm">Lista kupljenog materijala je prazna.</p>
        <p className="text-slate-600 text-xs mt-1">Dodajte materijale gore.</p>
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-slate-700 overflow-hidden">
      <div className="bg-slate-800/80 border-b border-slate-700 px-4 py-2.5 flex items-center gap-2">
        <svg className="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
        </svg>
        <span className="text-sm font-medium text-slate-200">Lista kupljenog materijala</span>
        <span className="ml-auto inline-flex items-center justify-center w-5 h-5 rounded-full bg-blue-600 text-white text-xs font-bold">
          {items.length}
        </span>
      </div>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-700/60">
            <th className="text-left px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">Materijal</th>
            <th className="text-right px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">Količina</th>
            <th className="px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">Jed.</th>
            <th className="px-2 py-2.5 w-8" />
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {items.map(item => (
            <tr key={item._tempId} className="hover:bg-slate-800/40 transition-colors">
              <td className="px-4 py-2.5">
                <p className="text-slate-100 font-medium">{item.material_name}</p>
                {item.material_code && <p className="text-xs text-slate-500">{item.material_code}</p>}
              </td>
              <td className="px-4 py-2.5 text-right font-mono font-semibold text-slate-100 tabular-nums">
                {item.quantity.toLocaleString('hr-HR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
              </td>
              <td className="px-4 py-2.5 text-slate-400">{item.unit}</td>
              <td className="px-2 py-2.5">
                <button
                  type="button"
                  onClick={() => onRemove(item._tempId)}
                  disabled={disabled}
                  className="text-slate-500 hover:text-red-400 transition-colors disabled:opacity-40"
                  aria-label="Ukloni"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
