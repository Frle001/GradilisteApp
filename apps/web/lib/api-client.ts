import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'
import { clearAuth, getToken, setToken } from './auth'

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://gradiliste-api.onrender.com'

const apiClient: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
  withCredentials: true, // required to send/receive the HTTP-only refresh cookie
  headers: {
    'Content-Type': 'application/json',
  },
})

// Attach Bearer access token on every request
apiClient.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// ── 401 handling: attempt one refresh, then give up ──────────────────────────

let isRefreshing = false
let pendingQueue: Array<{
  resolve: (token: string) => void
  reject: (err: unknown) => void
}> = []

function drainQueue(token: string | null, err: unknown) {
  pendingQueue.forEach(({ resolve, reject }) => {
    if (token) resolve(token)
    else reject(err)
  })
  pendingQueue = []
}

apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    if (error.response?.status !== 401 || original._retry) {
      return Promise.reject(error)
    }

    // Avoid refresh-loop if the refresh call itself returns 401
    if (original.url?.includes('/auth/refresh')) {
      clearAuth()
      return Promise.reject(error)
    }

    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        pendingQueue.push({
          resolve: (token) => {
            original.headers.Authorization = `Bearer ${token}`
            resolve(apiClient(original))
          },
          reject,
        })
      })
    }

    original._retry = true
    isRefreshing = true

    try {
      const res = await axios.post<{ access_token: string }>(
        `${BASE_URL}/auth/refresh`,
        {},
        { withCredentials: true }
      )
      const newToken = res.data.access_token
      setToken(newToken)
      apiClient.defaults.headers.common.Authorization = `Bearer ${newToken}`
      drainQueue(newToken, null)
      original.headers.Authorization = `Bearer ${newToken}`
      return apiClient(original)
    } catch (refreshErr) {
      drainQueue(null, refreshErr)
      clearAuth()
      return Promise.reject(refreshErr)
    } finally {
      isRefreshing = false
    }
  }
)

export default apiClient
