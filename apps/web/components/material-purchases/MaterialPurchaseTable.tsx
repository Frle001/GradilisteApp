import Link from 'next/link'
import { type PurchaseListItem } from '@/lib/types/material-purchases'

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('hr-HR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

interface Props {
  rows: PurchaseListItem[]
  loading?: boolean
  showBuyer?: boolean
}

function LoadingSkeleton() {
  return (
    <div className="rounded-xl border border-slate-700 overflow-hidden">
      <div className="bg-slate-800/80 px-4 py-3 border-b border-slate-700 grid grid-cols-5 gap-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-3 rounded bg-slate-700 animate-pulse" />
        ))}
      </div>
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="px-4 py-3 border-b border-slate-800 grid grid-cols-5 gap-4 last:border-0">
          {Array.from({ length: 5 }).map((_, j) => (
            <div key={j} className="h-3 rounded bg-slate-800 animate-pulse" style={{ width: `${60 + (i * j * 7) % 40}%` }} />
          ))}
        </div>
      ))}
    </div>
  )
}

export default function MaterialPurchaseTable({ rows, loading, showBuyer = false }: Props) {
  if (loading) return <LoadingSkeleton />

  if (rows.length === 0) {
    return (
      <div className="rounded-xl border border-slate-700 bg-slate-800/30 flex flex-col items-center justify-center py-16 gap-3">
        <svg className="w-8 h-8 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 00-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 00-16.536-1.84M7.5 14.25L5.106 5.272M6 20.25a.75.75 0 11-1.5 0 .75.75 0 011.5 0zm12.75 0a.75.75 0 11-1.5 0 .75.75 0 011.5 0z" />
        </svg>
        <p className="text-slate-500 text-sm">Nema pronađenih upisa materijala</p>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-slate-700">
      <table className="w-full text-sm text-left">
        <thead>
          <tr className="bg-slate-800/80 border-b border-slate-700">
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Datum</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Gradilište</th>
            {showBuyer && (
              <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider whitespace-nowrap">Kupac</th>
            )}
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider text-center whitespace-nowrap">Stavke</th>
            <th className="px-4 py-3 text-xs font-semibold text-slate-400 uppercase tracking-wider">Račun</th>
            <th className="px-4 py-3" />
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800">
          {rows.map(row => (
            <tr key={row.id} className="hover:bg-slate-800/40 transition-colors">
              <td className="px-4 py-3 text-slate-300 whitespace-nowrap font-medium tabular-nums">
                {formatDate(row.purchased_at)}
              </td>
              <td className="px-4 py-3 max-w-[220px]">
                <p className="text-slate-100 font-medium truncate">{row.project_name}</p>
              </td>
              {showBuyer && (
                <td className="px-4 py-3 text-slate-300 whitespace-nowrap">{row.buyer_name}</td>
              )}
              <td className="px-4 py-3 text-center">
                <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-slate-700 text-slate-300 text-xs font-bold">
                  {row.items_count}
                </span>
              </td>
              <td className="px-4 py-3">
                {row.receipt_file_url ? (
                  <span className="inline-flex items-center gap-1 text-xs text-emerald-400 font-medium">
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M18.375 12.739l-7.693 7.693a4.5 4.5 0 01-6.364-6.364l10.94-10.94A3 3 0 1119.5 7.372L8.552 18.32m.009-.01l-.01.01m5.699-9.941l-7.81 7.81a1.5 1.5 0 002.112 2.13" />
                    </svg>
                    Priložen
                  </span>
                ) : (
                  <span className="text-xs text-slate-600">Nema računa</span>
                )}
              </td>
              <td className="px-4 py-3 text-right">
                <Link
                  href={`/dashboard/material-purchases/${row.id}`}
                  className="text-xs text-blue-400 hover:text-blue-300 font-medium whitespace-nowrap"
                >
                  Otvori →
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
