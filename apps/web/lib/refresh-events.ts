const REFRESH_EVENT = 'gradiliste:refresh'

export type RefreshDomain =
  | 'daily-reports'
  | 'worker-hours'
  | 'gradevinski-dnevnik'
  | 'project-materials'
  | 'project-summary'

export function broadcastRefresh(domains: RefreshDomain[]): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent(REFRESH_EVENT, { detail: { domains } }))
}

/**
 * Subscribe to refresh events. Returns an unsubscribe function.
 * Safe to call during SSR — returns a no-op unsubscriber.
 */
export function onRefresh(handler: (domains: RefreshDomain[]) => void): () => void {
  if (typeof window === 'undefined') return () => {}
  const listener = (e: Event) => {
    const detail = (e as CustomEvent<{ domains: RefreshDomain[] }>).detail
    handler(detail?.domains ?? [])
  }
  window.addEventListener(REFRESH_EVENT, listener)
  return () => window.removeEventListener(REFRESH_EVENT, listener)
}
