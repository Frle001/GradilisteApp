'use client'

import { useOnlineStatus } from '@/hooks/useOnlineStatus'

export default function OfflineIndicator() {
  const isOnline = useOnlineStatus()

  if (isOnline) return null

  return (
    <div
      role="status"
      aria-live="polite"
      className="fixed top-0 left-0 right-0 z-50 bg-amber-500 text-slate-900 text-sm font-semibold text-center py-2 px-4 pt-safe"
    >
      Nema internetske veze
    </div>
  )
}
