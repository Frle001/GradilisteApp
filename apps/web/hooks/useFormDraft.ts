'use client'

import { useState, useEffect, useCallback, useRef } from 'react'

const DEBOUNCE_MS = 800

export function useFormDraft<T>(
  key: string,
  initialValue: T
): [T, (value: T | ((prev: T) => T)) => void, () => void] {
  const storageKey = `gradiliste_draft_${key}`
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const [value, setValueState] = useState<T>(() => {
    if (typeof window === 'undefined') return initialValue
    try {
      const stored = localStorage.getItem(storageKey)
      if (stored) return JSON.parse(stored) as T
    } catch {
      // ignore parse errors
    }
    return initialValue
  })

  const setValue = useCallback(
    (update: T | ((prev: T) => T)) => {
      setValueState(prev => {
        const next = typeof update === 'function' ? (update as (p: T) => T)(prev) : update
        if (debounceRef.current) clearTimeout(debounceRef.current)
        debounceRef.current = setTimeout(() => {
          try {
            localStorage.setItem(storageKey, JSON.stringify(next))
          } catch {
            // ignore quota errors
          }
        }, DEBOUNCE_MS)
        return next
      })
    },
    [storageKey]
  )

  const clearDraft = useCallback(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    try {
      localStorage.removeItem(storageKey)
    } catch {
      // ignore
    }
  }, [storageKey])

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [])

  return [value, setValue, clearDraft]
}
