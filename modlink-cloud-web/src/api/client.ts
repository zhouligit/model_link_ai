import axios from 'axios'

const api = axios.create({
  baseURL: '/mlk/platform/v1',
  timeout: 60000,
})

api.interceptors.request.use((config) => {
  const t = localStorage.getItem('mlk_access_token')
  if (t) {
    config.headers.Authorization = `Bearer ${t}`
  }
  return config
})

export type ApiEnvelope<T> = {
  code: number
  message: string
  data?: T
  request_id?: string
}

export async function login(email: string, password: string) {
  const { data } = await api.post<ApiEnvelope<Record<string, unknown>>>('/auth/login', { email, password })
  return data
}

export async function fetchWallet() {
  const { data } = await api.get<ApiEnvelope<{ balance_cents: number; currency: string }>>('/wallet')
  return data
}

export async function createApiKey(name: string, scope: 'personal' | 'org', orgId?: number, test?: boolean) {
  const { data } = await api.post<ApiEnvelope<{ id: number; secret: string; key_prefix: string }>>('/api-keys', {
    name,
    scope,
    org_id: orgId,
    test: !!test,
  })
  return data
}

export async function mockPayOrder(orderId: string) {
  const { data } = await api.post<ApiEnvelope<{ ok: boolean }>>('/payment/mock/complete', { order_id: orderId })
  return data
}

export async function recharge(amountCents: number, channel = 'wechat') {
  const { data } = await api.post<ApiEnvelope<{ order_id: string; payment_params: Record<string, unknown> }>>(
    '/orders/recharge',
    { amount_cents: amountCents, channel },
  )
  return data
}

export default api
