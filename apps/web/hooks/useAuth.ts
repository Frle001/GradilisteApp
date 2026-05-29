'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import apiClient from '@/lib/api-client'
import { clearAuth, getToken, getMustChangePassword } from '@/lib/auth'

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

export function useAuth() {
  const router = useRouter()
  const [user, setUser] = useState<MeUser | null>(null)
  const [employee, setEmployee] = useState<MeEmployee | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!getToken()) {
      router.replace('/login')
      return
    }

    if (getMustChangePassword()) {
      router.replace('/change-password')
      return
    }

    apiClient
      .get('/auth/me')
      .then((res) => {
        setUser(res.data.user)
        setEmployee(res.data.employee ?? null)
      })
      .catch(() => {
        clearAuth()
        router.replace('/login')
      })
      .finally(() => setIsLoading(false))
  }, [router])

  function logout() {
    apiClient.post('/auth/logout').finally(() => {
      clearAuth()
      router.push('/login')
    })
  }

  return { user, employee, isLoading, logout }
}
