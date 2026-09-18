const pad = (n: number) => String(n).padStart(2, '0')

// 后端返回 UTC 的 ISO 时间（如 2026-09-09T06:10:36.536Z），
// new Date 解析后用本地时区的 getter 输出，即按浏览器时区显示。
export function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}
