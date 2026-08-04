'use client'

import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import apiClient from '@/lib/api-client'
import {
  clearAuth,
  getToken,
  getMustChangePassword,
  getUser as getAuthUser,
  setUser as cacheAuthUser,
  getCachedEmployee,
  setCachedEmployee,
  type CachedEmployee,
} from '@/lib/auth'
import { setServerReachable } from '@/lib/connection-state'

export interface MeUser {
  id: string
  company_id: string
  employee_id: string | null
  email: string
  role: string
  active: boolean
  email_verified: boolean
}

export interface MeEmployee {
  id: string
  first_name: string
  last_name: string
  role: string
}

function isAuthRejected(err: unknown): boolean {
  const status = (err as { response?: { status?: number } })?.response?.status
  return status === 401 || status === 403
}

export function useAuth() {
  const router = useRouter()
  const [user, setUserState] = useState<MeUser | null>(null)
  const [employee, setEmployeeState] = useState<MeEmployee | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isOffline, setIsOffline] = useState(false)

  const verifySession = useCallback(
    (onlyUpdateIfAlreadyLoaded = false) => {
      return apiClient
        .get('/auth/me')
        .then((res) => {
          const freshUser = res.data.user as MeUser
          const freshEmployee = (res.data.employee ?? null) as MeEmployee | null
          setUserState(freshUser)
          setEmployeeState(freshEmployee)
          cacheAuthUser(freshUser)
          setCachedEmployee(freshEmployee as CachedEmployee | null)
          setIsOffline(false)
          setServerReachable(true)
        })
        .catch((err) => {
          if (isAuthRejected(err)) {
            clearAuth()
            setServerReachable(true)
            router.replace('/login')
          } else {
            // Network failure, timeout, or 5xx — preserve the cached session.
            setIsOffline(true)
            setServerReachable(false)
          }
        })
        .finally(() => {
          if (!onlyUpdateIfAlreadyLoaded) setIsLoading(false)
        })
    },
    [router]
  )

  useEffect(() => {
    const token = getToken()
    if (!token) {
      setIsLoading(false)
      router.replace('/login')
      return
    }

    if (getMustChangePassword()) {
      setIsLoading(false)
      router.replace('/change-password')
      return
    }

    // Restore cached user immediately so the UI is available while offline.
    const cached = getAuthUser()
    const cachedEmp = getCachedEmployee()
    if (cached) {
      setUserState(cached as unknown as MeUser)
      setEmployeeState(cachedEmp as MeEmployee | null)
      setIsLoading(false)
    }

    // Verify session with the server in the background.
    apiClient
      .get('/auth/me')
      .then((res) => {
        const freshUser = res.data.user as MeUser
        const freshEmployee = (res.data.employee ?? null) as MeEmployee | null
        setUserState(freshUser)
        setEmployeeState(freshEmployee)
        cacheAuthUser(freshUser)
        setCachedEmployee(freshEmployee as CachedEmployee | null)
        setIsOffline(false)
        setServerReachable(true)
      })
      .catch((err) => {
        if (isAuthRejected(err)) {
          // Server explicitly rejected the session — clear and redirect.
          clearAuth()
          setServerReachable(true)
          router.replace('/login')
        } else if (cached) {
          // Network failure with a cached session — stay authenticated offline.
          setIsOffline(true)
          setServerReachable(false)
        } else {
          // No cached session and no server — nothing to show.
          router.replace('/login')
        }
      })
      .finally(() => {
        if (!cached) setIsLoading(false)
      })
  }, [router]) // eslint-disable-line react-hooks/exhaustive-deps

  // Re-verify when the browser reports connectivity restored.
  useEffect(() => {
    function handleOnline() {
      if (!getToken()) return
      verifySession(true)
    }
    window.addEventListener('online', handleOnline)
    return () => window.removeEventListener('online', handleOnline)
  }, [verifySession])

  function logout() {
    apiClient.post('/auth/logout').finally(() => {
      clearAuth()
      setServerReachable(true)
      router.push('/login')
    })
  }

  return { user, employee, isLoading, isOffline, logout }
}
