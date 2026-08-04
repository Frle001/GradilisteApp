'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { getPendingCount, getFailedCount } from '@/lib/offline/outbox'
import { runSync } from '@/lib/offline/sync-engine'
import { onRefresh } from '@/lib/refresh-events'

const POLL_INTERVAL_MS = 30_000

export function useOutboxSync(userId?: string, companyId?: string) {
  const [pendingCount, setPendingCount] = useState(0)
  const [failedCount, setFailedCount] = useState(0)
  const [isSyncing, setIsSyncing] = useState(false)
  const [isOnline, setIsOnline] = useState(true)
  const [lastSyncedAt, setLastSyncedAt] = useState<number | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const refreshCounts = useCallback(async () => {
    try {
      const [p, f] = await Promise.all([getPendingCount(), getFailedCount()])
      setPendingCount(p)
      setFailedCount(f)
    } catch {
      // IDB unavailable (SSR or private browsing)
    }
  }, [])

  const triggerSync = useCallback(async () => {
    if (isSyncing) return
    setIsSyncing(true)
    try {
      await runSync({ userId, companyId })
      setLastSyncedAt(Date.now())
    } finally {
      setIsSyncing(false)
      await refreshCounts()
    }
  }, [isSyncing, refreshCounts, userId, companyId])

  // Initial count + online state + startup sync
  useEffect(() => {
    setIsOnline(navigator.onLine)
    void refreshCounts()
    if (navigator.onLine) void triggerSync()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []) // run once on mount; triggerSync is stable after first render

  // Refresh counts whenever sync completes on this or other tabs/pages
  useEffect(() => {
    return onRefresh(() => { void refreshCounts() })
  }, [refreshCounts])

  // Auto-sync on online, visibilitychange, focus
  useEffect(() => {
    function handleOnline() {
      setIsOnline(true)
      void triggerSync()
    }
    function handleOffline() {
      setIsOnline(false)
    }
    function handleVisible() {
      if (!document.hidden) void triggerSync()
    }

    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
    document.addEventListener('visibilitychange', handleVisible)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
      document.removeEventListener('visibilitychange', handleVisible)
    }
  }, [triggerSync])

  // Background poll while online
  useEffect(() => {
    if (intervalRef.current) clearInterval(intervalRef.current)
    if (isOnline) {
      intervalRef.current = setInterval(() => {
        void triggerSync()
      }, POLL_INTERVAL_MS)
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [isOnline, triggerSync])

  return { pendingCount, failedCount, isSyncing, isOnline, lastSyncedAt, triggerSync, refreshCounts }
}
