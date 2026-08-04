export interface AuthUser {
  id: string
  company_id: string
  employee_id: string | null
  email: string
  role: string
  active?: boolean
  email_verified?: boolean
}

export interface CachedEmployee {
  id: string
  first_name: string
  last_name: string
  role: string
}

const TOKEN_KEY = 'gradiliste_token'
const USER_KEY = 'gradiliste_user'
const EMPLOYEE_KEY = 'gradiliste_employee'
const MUST_CHANGE_KEY = 'gradiliste_must_change_password'

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

export function getCachedEmployee(): CachedEmployee | null {
  if (typeof window === 'undefined') return null
  const raw = localStorage.getItem(EMPLOYEE_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as CachedEmployee
  } catch {
    return null
  }
}

export function setCachedEmployee(emp: CachedEmployee | null): void {
  if (typeof window === 'undefined') return
  if (emp === null) {
    localStorage.removeItem(EMPLOYEE_KEY)
  } else {
    localStorage.setItem(EMPLOYEE_KEY, JSON.stringify(emp))
  }
}

export function clearEmployee(): void {
  localStorage.removeItem(EMPLOYEE_KEY)
}

export function getMustChangePassword(): boolean {
  if (typeof window === 'undefined') return false
  return localStorage.getItem(MUST_CHANGE_KEY) === 'true'
}

export function setMustChangePassword(value: boolean): void {
  localStorage.setItem(MUST_CHANGE_KEY, value ? 'true' : 'false')
}

export function clearMustChangePassword(): void {
  localStorage.removeItem(MUST_CHANGE_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

export function clearAuth(): void {
  clearToken()
  clearUser()
  clearEmployee()
  clearMustChangePassword()
}
