import axios from 'axios'

const api = axios.create({
  baseURL: '/mlk/platform/v1',
  timeout: 60000,
})

api.interceptors.request.use((config) => {
  const t = localStorage.getItem('mlk_admin_token')
  if (t) config.headers.Authorization = `Bearer ${t}`
  return config
})

export type ApiEnvelope<T> = { code: number; message: string; data?: T }

export default api
