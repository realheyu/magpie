import { withBase } from '@/utils/base'

export interface Result<T> {
  code: number
  msg: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
}

export interface User {
  id: number
  username: string
  displayName: string
  role: string
  status: string
  lastLoginAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface AppConfig {
  id: number
  appName: string
  description: string
  format: string
  content: string
  sensitive: boolean
  version: number
  status: string
  permission: string
  createdAt: string
  updatedAt: string
}

export interface AppOption {
  appName: string
  description: string
  format: string
  sensitive: boolean
  version: number
  status: string
}

export interface Revision {
  id: number
  appName: string
  version: number
  format: string
  content: string
  sensitive: boolean
  changeSummary: string
  createdByUserId?: number
  createdAt: string
}

export interface APIKeyItem {
  id: number
  name: string
  keyPreview: string
  status: string
  appNames: string[]
  expiresAt?: string | null
  lastUsedAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface AuditLog {
  id: number
  actorType: string
  actorId?: number | null
  action: string
  resourceType: string
  resourceId: string
  metadata: string
  createdAt: string
}

const tokenKey = 'magpieToken'

export function getToken() {
  return localStorage.getItem(tokenKey) || ''
}

export function setToken(token: string) {
  localStorage.setItem(tokenKey, token)
}

export function clearToken() {
  localStorage.removeItem(tokenKey)
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && init.body) headers.set('Content-Type', 'application/json')
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const resp = await fetch(withBase(path), { ...init, headers })
  // 会话过期统一处理：清掉本地 token 并回到登录页。
  // 登录接口自身的 401（密码错误）除外，此时已经在登录页。
  const loginPath = withBase('/login')
  if (resp.status === 401 && path !== '/api/admin/login' && window.location.pathname !== loginPath) {
    clearToken()
    window.location.href = loginPath
    throw new Error('登录已过期，请重新登录')
  }
  const payload = (await resp.json()) as Result<T>
  if (!resp.ok || payload.code !== 0) throw new Error(payload.msg || `请求失败：${resp.status}`)
  return payload.data
}

export const api = {
  login: (username: string, password: string) =>
    request<{ token: string; expiresAt: string; user: User }>('/api/admin/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () => request<null>('/api/admin/logout', { method: 'POST' }),
  me: () => request<User>('/api/admin/me'),
  listApps: (query = '', page = 1, pageSize = 20) =>
    request<PageData<AppConfig>>(`/api/admin/apps?query=${encodeURIComponent(query)}&page=${page}&pageSize=${pageSize}`),
  listAppOptions: (query = '') => request<AppOption[]>(`/api/admin/apps/options?query=${encodeURIComponent(query)}`),
  getApp: (appName: string) => request<AppConfig>(`/api/admin/apps/${encodeURIComponent(appName)}`),
  createApp: (app: Partial<AppConfig> & { appName: string; changeSummary?: string }) =>
    request<AppConfig>('/api/admin/apps', { method: 'POST', body: JSON.stringify(app) }),
  updateApp: (appName: string, app: Partial<AppConfig> & { changeSummary?: string }) =>
    request<AppConfig>(`/api/admin/apps/${encodeURIComponent(appName)}`, { method: 'PUT', body: JSON.stringify(app) }),
  deleteApp: (appName: string) => request<null>(`/api/admin/apps/${encodeURIComponent(appName)}`, { method: 'DELETE' }),
  revisions: (appName: string, page = 1, pageSize = 20) =>
    request<PageData<Revision>>(`/api/admin/apps/${encodeURIComponent(appName)}/revisions?page=${page}&pageSize=${pageSize}`),
  rollback: (appName: string, version: number) =>
    request<AppConfig>(`/api/admin/apps/${encodeURIComponent(appName)}/rollback`, { method: 'POST', body: JSON.stringify({ version }) }),
  restoreApp: (appName: string, version?: number) =>
    request<AppConfig>(`/api/admin/apps/${encodeURIComponent(appName)}/restore`, { method: 'POST', body: JSON.stringify({ version: version || 0 }) }),
  listUsers: (query = '', page = 1, pageSize = 20) =>
    request<PageData<User>>(`/api/admin/users?query=${encodeURIComponent(query)}&page=${page}&pageSize=${pageSize}`),
  createUser: (user: { username: string; displayName?: string; password: string; role: string; status: string }) =>
    request<User>('/api/admin/users', { method: 'POST', body: JSON.stringify(user) }),
  updateUser: (id: number, user: { displayName?: string; password?: string; role?: string; status?: string }) =>
    request<User>(`/api/admin/users/${id}`, { method: 'PUT', body: JSON.stringify(user) }),
  deleteUser: (id: number) => request<null>(`/api/admin/users/${id}`, { method: 'DELETE' }),
  getPermissions: (id: number) => request<{ permissions: { appName: string; permission: string }[] }>(`/api/admin/users/${id}/permissions`),
  setPermissions: (id: number, permissions: { appName: string; permission: string }[]) =>
    request<null>(`/api/admin/users/${id}/permissions`, { method: 'PUT', body: JSON.stringify({ permissions }) }),
  listAPIKeys: (page = 1, pageSize = 20) =>
    request<PageData<APIKeyItem>>(`/api/admin/api-keys?page=${page}&pageSize=${pageSize}`),
  createAPIKey: (body: { name: string; appNames: string[]; expiresAt?: string }) =>
    request<{ apiKey: string; item: APIKeyItem }>('/api/admin/api-keys', { method: 'POST', body: JSON.stringify(body) }),
  updateAPIKeyApps: (id: number, appNames: string[]) =>
    request<null>(`/api/admin/api-keys/${id}/apps`, { method: 'PUT', body: JSON.stringify({ appNames }) }),
  updateAPIKeyStatus: (id: number, status: string) =>
    request<APIKeyItem>(`/api/admin/api-keys/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  deleteAPIKey: (id: number) => request<null>(`/api/admin/api-keys/${id}`, { method: 'DELETE' }),
  listAuditLogs: (query = '', page = 1, pageSize = 20) =>
    request<PageData<AuditLog>>(`/api/admin/audit-logs?query=${encodeURIComponent(query)}&page=${page}&pageSize=${pageSize}`),
}
