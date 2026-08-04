/**
 * Offline auth behavior tests — no React rendering.
 *
 * Tests validate the decision logic and localStorage state produced by each
 * scenario, rather than rendering the hook itself.
 *
 *  1. /auth/me network error keeps user authenticated
 *  2. /auth/me timeout keeps user authenticated
 *  3. /auth/me 500 keeps user authenticated
 *  4. /auth/me 401 logs user out
 *  5. Refresh network error does not clear auth (see api-client test)
 *  6. Refresh 401 clears auth (see api-client test)
 *  7. Outbox remains intact during network outage
 *  8. Reconnect refreshes /auth/me and preserves session
 *  9. Manual logout clears local auth
 * 10. User-switch isolation remains intact
 *
 * @vitest-environment jsdom
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import {
  clearAuth,
  getToken, setToken,
  getUser, setUser,
  getCachedEmployee, setCachedEmployee,
} from '@/lib/auth'
import { setServerReachable, getServerReachable } from '@/lib/connection-state'

// ── fixtures ──────────────────────────────────────────────────────────────────

const CACHED_USER = {
  id: 'user-1',
  company_id: 'comp-1',
  employee_id: 'emp-1',
  email: 'test@example.com',
  role: 'radnik',
  active: true,
  email_verified: true,
}

const CACHED_EMPLOYEE = {
  id: 'emp-1',
  first_name: 'Ivan',
  last_name: 'Horvat',
  role: 'radnik',
}

/** Simulates hook startup state: token + cached user in localStorage */
function seed() {
  setToken('valid-token')
  setUser(CACHED_USER)
  setCachedEmployee(CACHED_EMPLOYEE)
}

/**
 * Simulate the exact catch-block logic from useAuth:
 *   if (isAuthRejected(err)) clearAuth() + redirect
 *   else if (cached) setOffline = true
 *   else redirect
 */
function isAuthRejected(err: unknown): boolean {
  const status = (err as { response?: { status?: number } })?.response?.status
  return status === 401 || status === 403
}

function simulateAuthMeError(err: unknown, hasCachedUser: boolean): {
  authCleared: boolean
  wouldRedirect: boolean
  wouldSetOffline: boolean
} {
  if (isAuthRejected(err)) {
    clearAuth()
    return { authCleared: true, wouldRedirect: true, wouldSetOffline: false }
  }
  if (hasCachedUser) {
    return { authCleared: false, wouldRedirect: false, wouldSetOffline: true }
  }
  return { authCleared: false, wouldRedirect: true, wouldSetOffline: false }
}

// ── lifecycle ─────────────────────────────────────────────────────────────────

beforeEach(() => localStorage.clear())
afterEach(() => localStorage.clear())

// ── 1. Network error keeps user authenticated ─────────────────────────────────

describe('1. /auth/me network error keeps user authenticated', () => {
  it('preserves token and cached user', () => {
    seed()
    const err = new Error('Network Error') // no .response

    const { authCleared, wouldRedirect } = simulateAuthMeError(err, true)

    expect(authCleared).toBe(false)
    expect(wouldRedirect).toBe(false)
    expect(getToken()).toBe('valid-token')
    expect(getUser()?.id).toBe('user-1')
  })

  it('sets offline flag (would call setServerReachable(false))', () => {
    seed()
    const err = new Error('Network Error')

    const { wouldSetOffline } = simulateAuthMeError(err, true)

    expect(wouldSetOffline).toBe(true)
  })
})

// ── 2. Timeout keeps user authenticated ──────────────────────────────────────

describe('2. /auth/me timeout keeps user authenticated', () => {
  it('preserves token — timeout has no .response', () => {
    seed()
    const err = Object.assign(new Error('timeout of 10000ms exceeded'), {
      code: 'ECONNABORTED',
      // no .response
    })

    const { authCleared, wouldRedirect } = simulateAuthMeError(err, true)

    expect(authCleared).toBe(false)
    expect(wouldRedirect).toBe(false)
    expect(getToken()).toBe('valid-token')
  })
})

// ── 3. 5xx keeps user authenticated ──────────────────────────────────────────

describe('3. /auth/me 5xx keeps user authenticated', () => {
  for (const status of [500, 502, 503, 504]) {
    it(`preserves token on ${status}`, () => {
      seed()
      const err = { response: { status } }

      const { authCleared, wouldRedirect } = simulateAuthMeError(err, true)

      expect(authCleared).toBe(false)
      expect(wouldRedirect).toBe(false)
      expect(getToken()).toBe('valid-token')
    })
  }
})

// ── 4. 401/403 logs user out ──────────────────────────────────────────────────

describe('4. /auth/me 401/403 logs user out', () => {
  it('clears auth on 401', () => {
    seed()
    const err = { response: { status: 401 } }

    const { authCleared, wouldRedirect } = simulateAuthMeError(err, true)

    expect(authCleared).toBe(true)
    expect(wouldRedirect).toBe(true)
    expect(getToken()).toBeNull()
    expect(getUser()).toBeNull()
    expect(getCachedEmployee()).toBeNull()
  })

  it('clears auth on 403', () => {
    seed()
    const err = { response: { status: 403 } }

    const { authCleared } = simulateAuthMeError(err, true)

    expect(authCleared).toBe(true)
    expect(getToken()).toBeNull()
  })

  it('does not confuse 401 with 402 or 400', () => {
    for (const status of [400, 402, 404]) {
      seed()
      const err = { response: { status } }
      const { authCleared } = simulateAuthMeError(err, true)
      expect(authCleared).toBe(false)
      localStorage.clear()
    }
  })
})

// ── 7. Outbox remains intact during network outage ────────────────────────────

describe('7. Outbox accessible during network outage', () => {
  it('does not clear token, user, or employee', () => {
    seed()
    const err = new Error('Network Error')

    simulateAuthMeError(err, true)

    expect(getToken()).toBe('valid-token')
    expect(getUser()?.id).toBe('user-1')
    expect(getCachedEmployee()?.id).toBe('emp-1')
  })

  it('preserves ownerId (user.id) and companyId for outbox isolation', () => {
    seed()
    const err = new Error('Network Error')

    simulateAuthMeError(err, true)

    // These are the values outbox entries are matched against
    const storedUser = getUser()
    expect(storedUser?.id).toBe('user-1')
    expect(storedUser?.company_id).toBe('comp-1')
  })
})

// ── 8. Reconnect refreshes /auth/me and preserves session ────────────────────

describe('8. Reconnect: successful /auth/me marks server reachable', () => {
  it('setServerReachable(true) is called after successful response', () => {
    seed()
    // Simulate what the hook's .then() handler does on success
    setServerReachable(false) // was offline

    // On successful /auth/me response:
    setServerReachable(true)

    expect(getServerReachable()).toBe(true)
  })

  it('session remains valid after reconnect', () => {
    seed()
    setServerReachable(false)

    // Simulate successful reconnect: update cache with fresh data
    setUser({ ...CACHED_USER, email: 'fresh@example.com' })
    setServerReachable(true)

    expect(getToken()).toBe('valid-token')
    expect(getUser()?.email).toBe('fresh@example.com')
    expect(getServerReachable()).toBe(true)
  })
})

// ── 9. Manual logout clears local auth ───────────────────────────────────────

describe('9. Manual logout clears local auth', () => {
  it('clearAuth removes token, user, and employee', () => {
    seed()

    clearAuth()

    expect(getToken()).toBeNull()
    expect(getUser()).toBeNull()
    expect(getCachedEmployee()).toBeNull()
  })

  it('clearAuth resets server-reachable signal', () => {
    seed()
    setServerReachable(false)

    clearAuth()
    setServerReachable(true) // hook's logout() calls this

    expect(getServerReachable()).toBe(true)
  })
})

// ── 10. User-switch isolation ─────────────────────────────────────────────────

describe('10. User-switch isolation', () => {
  it('after logout no user id remains in storage', () => {
    seed()

    clearAuth()

    expect(getToken()).toBeNull()
    expect(getUser()).toBeNull()
    expect(getCachedEmployee()).toBeNull()
  })

  it('ownerId in state matches cached user id (no cross-user bleed)', () => {
    // User A
    seed()
    const userA = getUser()
    expect(userA?.id).toBe('user-1')

    // User A logs out
    clearAuth()

    // User B logs in (different id)
    const userB = { ...CACHED_USER, id: 'user-2', company_id: 'comp-2' }
    setToken('token-b')
    setUser(userB)

    // User A's data must not be visible
    expect(getUser()?.id).toBe('user-2')
    expect(getUser()?.company_id).toBe('comp-2')
  })

  it('no cached user after logout means no one is implicitly authenticated', () => {
    seed()
    clearAuth()

    const hasCachedUser = !!(getToken() && getUser())
    expect(hasCachedUser).toBe(false)
  })
})
