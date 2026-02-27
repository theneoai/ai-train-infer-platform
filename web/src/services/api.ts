import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const API_BASE_URL = (import.meta as any).env?.VITE_API_URL || 'http://localhost:8080'

export const api = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    return Promise.reject(error.response?.data?.error?.message || 'Unknown error')
  }
)

// Auth API
export const authApi = {
  register: (data: { email: string; username: string; password: string }) =>
    api.post('/auth/register', data),
  login: (data: { email: string; password: string }) =>
    api.post('/auth/login', data),
  getMe: () => api.get('/auth/me'),
}

// Dataset API
export const datasetApi = {
  list: () => api.get('/datasets'),
  create: (data: FormData) =>
    api.post('/datasets', data, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  get: (id: string) => api.get(`/datasets/${id}`),
  delete: (id: string) => api.delete(`/datasets/${id}`),
}

// Training API
export const trainingApi = {
  list: () => api.get('/training/jobs'),
  create: (data: {
    name: string
    dataset_id: string
    config: Record<string, unknown>
    gpu_count: number
  }) => api.post('/training/jobs', data),
  get: (id: string) => api.get(`/training/jobs/${id}`),
  delete: (id: string) => api.delete(`/training/jobs/${id}`),
  stop: (id: string) => api.post(`/training/jobs/${id}/stop`),
  getLogs: (id: string) => `${API_BASE_URL}/api/v1/training/jobs/${id}/logs`,
}

// Inference API
export const inferenceApi = {
  list: () => api.get('/inference/services'),
  create: (data: { name: string; model_id: string; config: Record<string, unknown> }) =>
    api.post('/inference/services', data),
  get: (id: string) => api.get(`/inference/services/${id}`),
  delete: (id: string) => api.delete(`/inference/services/${id}`),
}
