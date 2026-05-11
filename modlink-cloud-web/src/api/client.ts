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

export async function register(email: string, password: string, displayName?: string) {
  const { data } = await api.post<ApiEnvelope<Record<string, unknown>>>('/auth/register', {
    email,
    password,
    display_name: displayName,
  })
  return data
}

export type TokenPayload = {
  access_token?: string
  refresh_token?: string
}

/** 写入登录 / refresh / switch 组织 返回的新 token */
export function applySessionTokens(payload: TokenPayload) {
  if (payload.access_token) localStorage.setItem('mlk_access_token', payload.access_token)
  if (payload.refresh_token) localStorage.setItem('mlk_refresh_token', payload.refresh_token)
}

export async function fetchMe() {
  const { data } = await api.get<
    ApiEnvelope<{
      id: number
      email?: string
      display_name?: string
      role?: string
      current_org_id?: number | null
    }>
  >('/auth/me')
  return data
}

export async function listOrgs() {
  const { data } = await api.get<ApiEnvelope<{ items: { id: number; name: string; slug?: string; status?: string }[] }>>(
    '/orgs',
  )
  return data
}

export async function switchOrg(orgId: number) {
  const { data } = await api.post<ApiEnvelope<TokenPayload>>(`/orgs/${orgId}/switch`, {})
  return data
}

export async function fetchWallet(orgId?: number | null) {
  const { data } = await api.get<
    ApiEnvelope<{ balance_cents: number; currency: string; credit_status?: string }>
  >('/wallet', orgId == null ? {} : { params: { org_id: orgId } })
  return data
}

export async function listApiKeys() {
  const { data } = await api.get<ApiEnvelope<{ items: { id: number; name: string; scope: string; key_prefix: string; status: string; org_id?: number }[] }>>(
    '/api-keys',
  )
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

export async function deleteApiKey(keyId: number) {
  const { data } = await api.delete<ApiEnvelope<{ ok?: boolean }>>(`/api-keys/${keyId}`)
  return data
}

export async function mockPayOrder(orderId: string) {
  const { data } = await api.post<ApiEnvelope<{ ok: boolean }>>('/payment/mock/complete', { order_id: orderId })
  return data
}

export async function listOrders() {
  const { data } = await api.get<ApiEnvelope<{ items: Record<string, unknown>[] }>>('/orders')
  return data
}

export async function getOrder(orderId: string | number) {
  const { data } = await api.get<ApiEnvelope<Record<string, unknown>>>(`/orders/${orderId}`)
  return data
}

export async function recharge(amountCents: number, channel = 'wechat', orgId?: number | null) {
  const body: Record<string, unknown> = { amount_cents: amountCents, channel }
  if (orgId != null) body.org_id = orgId
  const { data } = await api.post<ApiEnvelope<{ order_id: string; payment_params: Record<string, unknown> }>>(
    '/orders/recharge',
    body,
  )
  return data
}

/** 调用平台注销并清除本地 access / refresh（服务端会撤销 refresh token） */
export async function logoutAndClearLocal() {
  try {
    await api.post<ApiEnvelope<Record<string, unknown>>>('/auth/logout', {})
  } catch {
    /* 仍清除本地，避免卡在已登出状态 */
  }
  localStorage.removeItem('mlk_access_token')
  localStorage.removeItem('mlk_refresh_token')
}

export default api
