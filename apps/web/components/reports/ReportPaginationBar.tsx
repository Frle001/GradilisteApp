'use client'

import { type ReportPagination } from '@/lib/types/reports'

interface Props {
  pagination: ReportPagination
  onPageChange: (page: number) => void
}

export default function ReportPaginationBar({ pagination, onPageChange }: Props) {
  const { page, total_pages, total, per_page } = pagination
  const from = total === 0 ? 0 : (page - 1) * per_page + 1
  const to = Math.min(page * per_page, total)

  return (
    <div className="flex items-center justify-between px-1 py-2">
      <p className="text-sm text-slate-400">
        {total === 0 ? (
          'Nema rezultata'
        ) : (
          <>
            <span className="font-medium text-slate-200">{from}–{to}</span>
            {' '}od{' '}
            <span className="font-medium text-slate-200">{total}</span>
            {' '}zapisa
          </>
        )}
      </p>

      <div className="flex items-center gap-1.5">
        <button
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
          className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg border border-slate-700 bg-slate-800 hover:bg-slate-700 hover:border-slate-600 disabled:opacity-35 disabled:cursor-not-allowed text-slate-300 text-sm transition-colors"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
          </svg>
          Prethodno
        </button>

        <span className="px-3 py-1.5 text-sm text-slate-400 tabular-nums">
          {page} / {total_pages}
        </span>

        <button
          disabled={page >= total_pages}
          onClick={() => onPageChange(page + 1)}
          className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg border border-slate-700 bg-slate-800 hover:bg-slate-700 hover:border-slate-600 disabled:opacity-35 disabled:cursor-not-allowed text-slate-300 text-sm transition-colors"
        >
          Sljedeće
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
          </svg>
        </button>
      </div>
    </div>
  )
}
