import axios, { AxiosInstance } from 'axios'

const apiClient: AxiosInstance = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add interceptor to include auth token in future
apiClient.interceptors.request.use((config) => {
  // TODO: Add token from localStorage or cookie
  return config
})

// Handle errors globally
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    // TODO: Handle auth errors, token refresh, etc.
    return Promise.reject(error)
  }
)

export default apiClient
