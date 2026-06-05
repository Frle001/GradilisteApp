import AssetTypeBadge from './AssetTypeBadge'
import type { TransferListItem } from '@/lib/types/inventory'

interface Props {
  items: TransferListItem[]
  loading: boolean
  currentEmployeeId?: string
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('hr-HR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export default function TransferHistoryTable({ items, loading, currentEmployeeId }: Props) {
  if (loading) {
    return (
      <div className="py-12 text-center text-slate-500 text-sm">Učitavanje...</div>
    )
  }
  if (items.length === 0) {
    return (
      <div className="py-12 text-center text-slate-500 text-sm">Nema evidentiranih prijenosa.</div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-700">
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Datum</th>
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Predao</th>
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Primio</th>
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Vrsta</th>
            <th className="text-left text-xs font-medium text-slate-400 uppercase tracking-wider pb-2 pr-4">Stavka</th>
            <th className="text-right text-xs font-medium text-slate-400 uppercase tracking-wider pb-2">Kol.</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {items.map(t => {
            const isFrom = currentEmployeeId && t.from_employee_id === currentEmployeeId
            const isTo   = currentEmployeeId && t.to_employee_id === currentEmployeeId
            return (
              <tr key={t.id} className="hover:bg-slate-800/40 transition-colors">
                <td className="py-2.5 pr-4 text-slate-400 whitespace-nowrap">{formatDate(t.transferred_at)}</td>
                <td className={`py-2.5 pr-4 ${isFrom ? 'text-red-400 font-medium' : 'text-slate-300'}`}>
                  {t.from_employee_name}
                </td>
                <td className={`py-2.5 pr-4 ${isTo ? 'text-emerald-400 font-medium' : 'text-slate-300'}`}>
                  {t.to_employee_name}
                </td>
                <td className="py-2.5 pr-4">
                  <AssetTypeBadge type={t.asset_type} />
                </td>
                <td className="py-2.5 pr-4 text-white">
                  {t.item_name || '—'}
                  {t.notes && <span className="block text-xs text-slate-500">{t.notes}</span>}
                </td>
                <td className="py-2.5 text-right text-slate-300">{t.quantity.toFixed(2)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
