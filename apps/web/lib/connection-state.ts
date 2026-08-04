import { useState, useEffect } from 'react'

// Module-level signal — shared across all hook instances in one browser tab.
let _serverReachable = true
const _listeners = new Set<(v: boolean) => void>()

export function setServerReachable(v: boolean): void {
  if (_serverReachable === v) return
  _serverReachable = v
  _listeners.forEach(fn => fn(v))
}

export function getServerReachable(): boolean {
  return _serverReachable
}

export function useServerReachable(): boolean {
  const [reachable, setReachable] = useState(_serverReachable)
  useEffect(() => {
    // Sync with current module state in case it changed between render and effect.
    setReachable(_serverReachable)
    _listeners.add(setReachable)
    return () => { _listeners.delete(setReachable) }
  }, [])
  return reachable
}
