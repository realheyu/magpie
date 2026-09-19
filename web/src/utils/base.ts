// 部署基础路径：生产环境由后端注入到 index.html（子路径部署如 "/magpie/"），
// 开发模式和根路径部署下占位符未替换，统一回退为 "/"。
function normalizeBase(value: unknown): string {
  if (typeof value !== 'string' || !value.startsWith('/')) return '/'
  return value.endsWith('/') ? value : `${value}/`
}

export const BASE_PATH = normalizeBase((globalThis as { __MAGPIE_BASE__?: unknown }).__MAGPIE_BASE__)

// 给以 "/" 开头的路径加上部署前缀：
// 根路径部署 withBase('/api/admin/me') === '/api/admin/me'
// 子路径部署 withBase('/api/admin/me') === '/magpie/api/admin/me'
export function withBase(path: string): string {
  if (!path.startsWith('/')) return path
  return BASE_PATH === '/' ? path : `${BASE_PATH.slice(0, -1)}${path}`
}
