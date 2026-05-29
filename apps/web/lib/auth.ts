export interface AuthUser {
  id: string
  company_id: string
  employee_id: string | null
  email: string
  role: string
}

const TOKEN_KEY = 'gradiliste_token'
const USER_KEY = 'gradiliste_user'

export function getToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function getUser(): AuthUser | null {
  if (typeof window === 'undefined') return null
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthUser
  } catch {
    return null
  }
}

export function setUser(user: AuthUser): void {
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function clearUser(): void {
  localStorage.removeItem(USER_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

export function clearAuth(): void {
  clearToken()
  clearUser()
}
