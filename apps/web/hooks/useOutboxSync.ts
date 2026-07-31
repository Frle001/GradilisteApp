'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { getPendingCount } from '@/lib/offline/outbox'
import { runSync } from '@/lib/offline/sync-engine'

const POLL_INTERVAL_MS = 30_000 // background poll when online

export function useOutboxSync() {
  const [pendingCount, setPendingCount] = useState(0)
  const [isSyncing, setIsSyncing] = useState(false)
  const [isOnline, setIsOnline] = useState(true)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const refreshCount = useCallback(async () => {
    try {
      const n = await getPendingCount()
      setPendingCount(n)
    } catch {
      // IDB may not be available (SSR or private browsing)
    }
  }, [])

  const triggerSync = useCallback(async () => {
    if (isSyncing) return
    setIsSyncing(true)
    try {
      await runSync()
    } finally {
      setIsSyncing(false)
      await refreshCount()
    }
  }, [isSyncing, refreshCount])

  // Initial count + online state
  useEffect(() => {
    setIsOnline(navigator.onLine)
    refreshCount()
  }, [refreshCount])

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

  return { pendingCount, isSyncing, isOnline, triggerSync, refreshCount }
}
