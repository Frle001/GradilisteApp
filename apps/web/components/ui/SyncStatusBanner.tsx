'use client'

import { useOutboxSync } from '@/hooks/useOutboxSync'

/**
 * A slim banner rendered inside DashboardShell that shows offline/pending state.
 * Renders nothing when everything is synced and the user is online.
 */
export default function SyncStatusBanner() {
  const { pendingCount, isSyncing, isOnline, triggerSync } = useOutboxSync()

  if (isOnline && pendingCount === 0) return null

  return (
    <div
      className={[
        'w-full px-4 py-2 text-xs font-medium flex items-center justify-between gap-3',
        !isOnline
          ? 'bg-slate-800 border-b border-slate-700 text-slate-400'
          : 'bg-amber-950/60 border-b border-amber-800/60 text-amber-300',
      ].join(' ')}
    >
      <span className="flex items-center gap-2">
        {!isOnline ? (
          <>
            <svg className="w-3.5 h-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round"
                d="M3 3l18 18M8.111 8.111A7.5 7.5 0 0119.5 12M4.929 4.929A15 15 0 0121 12m-5.07 5.064A7.5 7.5 0 014.5 12m-.44 4.56A15 15 0 003 12" />
            </svg>
            Niste povezani
            {pendingCount > 0 && ` · ${pendingCount} ${pendingCount === 1 ? 'unos čeka' : 'unosa čekaju'} slanje`}
          </>
        ) : (
          <>
            {isSyncing ? (
              <svg className="w-3.5 h-3.5 shrink-0 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
            ) : (
              <svg className="w-3.5 h-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round"
                  d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
            )}
            {isSyncing
              ? 'Slanje podataka…'
              : `${pendingCount} ${pendingCount === 1 ? 'unos čeka' : 'unosa čekaju'} slanje`}
          </>
        )}
      </span>

      {isOnline && !isSyncing && pendingCount > 0 && (
        <button
          onClick={() => void triggerSync()}
          className="shrink-0 underline underline-offset-2 hover:text-amber-200 transition"
        >
          Pošalji sada
        </button>
      )}
    </div>
  )
}
