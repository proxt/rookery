export function formatBytes(n) {
  n = Number(n) || 0
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${n.toFixed(i === 0 ? 0 : n < 10 ? 2 : 1)} ${units[i]}`
}

export function formatDateTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d)) return iso
  return d.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// For <input type="datetime-local">, which needs "YYYY-MM-DDTHH:mm" in
// local time with no timezone suffix.
export function toDatetimeLocal(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d)) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Inverse of toDatetimeLocal: local "YYYY-MM-DDTHH:mm" -> RFC3339Nano UTC,
// or "" if empty.
export function fromDatetimeLocal(local) {
  if (!local) return ''
  const d = new Date(local)
  if (isNaN(d)) return ''
  return d.toISOString().replace('Z', '000Z') // pad to nanosecond-ish precision the Go side parses
}

export function relativeTime(iso) {
  if (!iso) return 'никогда'
  const d = new Date(iso)
  if (isNaN(d)) return iso
  const diffSec = Math.round((Date.now() - d.getTime()) / 1000)
  if (diffSec < 5) return 'только что'
  if (diffSec < 60) return `${diffSec} с назад`
  const diffMin = Math.round(diffSec / 60)
  if (diffMin < 60) return `${diffMin} мин назад`
  const diffHour = Math.round(diffMin / 60)
  if (diffHour < 24) return `${diffHour} ч назад`
  const diffDay = Math.round(diffHour / 24)
  return `${diffDay} дн назад`
}
