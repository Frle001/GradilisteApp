'use client'

import { useCallback, useEffect, useState } from 'react'
import { useOutboxSync } from '@/hooks/useOutboxSync'
import { getFailedEntries, discardEntry, requeueFailed, getPendingCountsByType } from '@/lib/offline/outbox'
import type { OutboxEntry, OutboxEntryType } from '@/lib/offline/types'
import { onRefresh } from '@/lib/refresh-events'

const TYPE_LABELS: Record<OutboxEntryType, string> = {
  'daily-report': 'Dnevni izvještaj',
  'worker-hours': 'Sati',
  'photo-upload': 'Fotografija',
}

function formatTime(ts: number | null): string {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString('hr-HR', { hour: '2-digit', minute: '2-digit' })
}

interface Props {
  userId?: string
  companyId?: string
}

export default function SyncStatusBanner({ userId, companyId }: Props) {
  const { pendingCount, failedCount, isSyncing, isOnline, lastSyncedAt, triggerSync, refreshCounts } =
    useOutboxSync(userId, companyId)

  const [failedEntries, setFailedEntries] = useState<OutboxEntry[]>([])
  const [showFailed, setShowFailed] = useState(false)
  const [confirmDiscard, setConfirmDiscard] = useState<string | null>(null)
  const [pendingByType, setPendingByType] = useState<Record<OutboxEntryType, number>>({
    'daily-report': 0, 'worker-hours': 0, 'photo-upload': 0,
  })

  const refreshFailed = useCallback(async () => {
    try {
      const [entries, counts] = await Promise.all([
        getFailedEntries(),
        getPendingCountsByType(),
      ])
      setFailedEntries(entries)
      setPendingByType(counts)
    } catch { /* IDB unavailable */ }
  }, [])

  useEffect(() => { void refreshFailed() }, [refreshFailed])

  // Keep failed list in sync with outbox state changes
  useEffect(() => onRefresh(() => { void refreshFailed(); void refreshCounts() }), [refreshFailed, refreshCounts])

  const handleRetry = async (id: string) => {
    await requeueFailed(id)
    await refreshCounts()
    await refreshFailed()
    void triggerSync()
  }

  const handleDiscard = async (id: string) => {
    await discardEntry(id)
    await refreshCounts()
    await refreshFailed()
    setConfirmDiscard(null)
  }

  const visible = !isOnline || pendingCount > 0 || failedCount > 0
  if (!visible) return null

  const pendingReports = pendingByType['daily-report']
  const pendingHours = pendingByType['worker-hours']
  const pendingPhotos = pendingByType['photo-upload']

  return (
    <div className="w-full border-b text-xs">
      {/* Main status row */}
      <div
        className={[
          'px-4 py-2 flex items-center justify-between gap-3 font-medium',
          !isOnline
            ? 'bg-slate-800 border-slate-700 text-slate-400'
            : failedCount > 0
              ? 'bg-red-950/60 border-red-800/60 text-red-300'
              : 'bg-amber-950/60 border-amber-800/60 text-amber-300',
        ].join(' ')}
      >
        <span className="flex items-center gap-2 min-w-0">
          {!isOnline ? (
            <>
              <svg className="w-3.5 h-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round"
                  d="M3 3l18 18M8.111 8.111A7.5 7.5 0 0119.5 12M4.929 4.929A15 15 0 0121 12m-5.07 5.064A7.5 7.5 0 014.5 12m-.44 4.56A15 15 0 003 12" />
              </svg>
              <span>
                Niste povezani
                {pendingCount > 0 && ` · ${pendingCount} ${pendingCount === 1 ? 'unos čeka' : 'unosa čekaju'} slanje`}
              </span>
            </>
          ) : isSyncing ? (
            <>
              <svg className="w-3.5 h-3.5 shrink-0 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
              <span>Slanje podataka…</span>
            </>
          ) : failedCount > 0 ? (
            <>
              <svg className="w-3.5 h-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round"
                  d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
              <span>
                {failedCount} {failedCount === 1 ? 'unos nije poslan' : 'unosa nisu poslana'}
                {pendingCount > 0 && ` · ${pendingCount} čeka`}
              </span>
            </>
          ) : (
            <>
              <svg className="w-3.5 h-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round"
                  d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
              <span>
                {pendingCount} {pendingCount === 1 ? 'unos čeka' : 'unosa čekaju'} slanje
                {pendingReports > 0 && ` · ${pendingReports} izvještaj`}
                {pendingHours > 0 && ` · ${pendingHours} sati`}
                {pendingPhotos > 0 && ` · ${pendingPhotos} foto`}
              </span>
            </>
          )}
          {lastSyncedAt && !isSyncing && (
            <span className="text-slate-500 hidden sm:inline">· zadnje: {formatTime(lastSyncedAt)}</span>
          )}
        </span>

        <span className="flex items-center gap-3 shrink-0">
          {isOnline && !isSyncing && pendingCount > 0 && (
            <button
              onClick={() => void triggerSync()}
              className="underline underline-offset-2 hover:opacity-80 transition"
            >
              Pošalji sada
            </button>
          )}
          {failedCount > 0 && (
            <button
              onClick={() => setShowFailed(v => !v)}
              className="underline underline-offset-2 hover:opacity-80 transition"
            >
              {showFailed ? 'Sakrij' : 'Detalji'}
            </button>
          )}
        </span>
      </div>

      {/* Failed entries detail panel */}
      {showFailed && failedEntries.length > 0 && (
        <div className="bg-red-950/40 border-t border-red-900/40 px-4 py-3 space-y-3">
          {failedEntries.map(entry => (
            <div key={entry.id} className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="font-medium text-red-200">
                  {TYPE_LABELS[entry.type] ?? entry.type}
                </p>
                {entry.lastError && (
                  <p className="text-red-400 truncate mt-0.5">{entry.lastError}</p>
                )}
                <p className="text-slate-500 mt-0.5">
                  {entry.attempts} {entry.attempts === 1 ? 'pokušaj' : 'pokušaja'}
                  {entry.lastAttemptAt && ` · ${formatTime(entry.lastAttemptAt)}`}
                </p>
              </div>
              <div className="flex gap-3 shrink-0">
                <button
                  onClick={() => void handleRetry(entry.id)}
                  className="text-amber-400 underline underline-offset-2 hover:text-amber-300 transition"
                >
                  Pokušaj ponovo
                </button>
                {confirmDiscard === entry.id ? (
                  <span className="flex gap-2">
                    <button
                      onClick={() => void handleDiscard(entry.id)}
                      className="text-red-400 underline underline-offset-2 hover:text-red-300 transition"
                    >
                      Potvrdi
                    </button>
                    <button
                      onClick={() => setConfirmDiscard(null)}
                      className="text-slate-400 underline underline-offset-2 hover:text-slate-300 transition"
                    >
                      Odustani
                    </button>
                  </span>
                ) : (
                  <button
                    onClick={() => setConfirmDiscard(entry.id)}
                    className="text-slate-400 underline underline-offset-2 hover:text-slate-300 transition"
                  >
                    Odbaci
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
