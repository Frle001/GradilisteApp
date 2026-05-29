import axios, { type AxiosInstance } from 'axios'
import { clearAuth, getToken } from './auth'

const apiClient: AxiosInstance = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Attach Bearer token on every request if one is stored
apiClient.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// On 401, clear local auth state so the next navigation lands on /login
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      clearAuth()
      // Let the calling page handle the redirect
    }
    return Promise.reject(error)
  }
)

export default apiClient
