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
  lastLoginAt?: string
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
  expiresAt?: string
  lastUsedAt?: string
  createdAt: string
  updatedAt: string
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
  const resp = await fetch(path, { ...init, headers })
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
  createApp: (app: Partial<AppConfig> & { appName: string; changeSummary?: string }) =>
    request<AppConfig>('/api/admin/apps', { method: 'POST', body: JSON.stringify(app) }),
  updateApp: (appName: string, app: Partial<AppConfig> & { changeSummary?: string }) =>
    request<AppConfig>(`/api/admin/apps/${encodeURIComponent(appName)}`, { method: 'PUT', body: JSON.stringify(app) }),
  deleteApp: (appName: string) => request<null>(`/api/admin/apps/${encodeURIComponent(appName)}`, { method: 'DELETE' }),
  revisions: (appName: string) => request<PageData<Revision>>(`/api/admin/apps/${encodeURIComponent(appName)}/revisions`),
  rollback: (appName: string, version: number) =>
    request<AppConfig>(`/api/admin/apps/${encodeURIComponent(appName)}/rollback`, { method: 'POST', body: JSON.stringify({ version }) }),
  listUsers: (query = '', page = 1, pageSize = 20) =>
    request<PageData<User>>(`/api/admin/users?query=${encodeURIComponent(query)}&page=${page}&pageSize=${pageSize}`),
  createUser: (user: { username: string; displayName?: string; password: string; role: string; status: string }) =>
    request<User>('/api/admin/users', { method: 'POST', body: JSON.stringify(user) }),
  updateUser: (id: number, user: { displayName?: string; password?: string; role?: string; status?: string }) =>
    request<User>(`/api/admin/users/${id}`, { method: 'PUT', body: JSON.stringify(user) }),
  getPermissions: (id: number) => request<{ permissions: { appName: string; permission: string }[] }>(`/api/admin/users/${id}/permissions`),
  setPermissions: (id: number, permissions: { appName: string; permission: string }[]) =>
    request<null>(`/api/admin/users/${id}/permissions`, { method: 'PUT', body: JSON.stringify({ permissions }) }),
  listAPIKeys: (page = 1, pageSize = 20) =>
    request<PageData<APIKeyItem>>(`/api/admin/api-keys?page=${page}&pageSize=${pageSize}`),
  createAPIKey: (body: { name: string; appNames: string[]; expiresAt?: string }) =>
    request<{ apiKey: string; item: APIKeyItem }>('/api/admin/api-keys', { method: 'POST', body: JSON.stringify(body) }),
  updateAPIKeyApps: (id: number, appNames: string[]) =>
    request<null>(`/api/admin/api-keys/${id}/apps`, { method: 'PUT', body: JSON.stringify({ appNames }) }),
  deleteAPIKey: (id: number) => request<null>(`/api/admin/api-keys/${id}`, { method: 'DELETE' }),
}
