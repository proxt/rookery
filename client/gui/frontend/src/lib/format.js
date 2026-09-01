export function formatBytesPerSec(bps) {
  return formatBytes(bps) + '/с'
}

export function formatBytes(b) {
  if (b < 1024) return `${b} Б`
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} КБ`
  if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(1)} МБ`
  return `${(b / 1024 / 1024 / 1024).toFixed(2)} ГБ`
}

export function formatDurationNs(ns) {
  if (!ns) return '0с'
  const ms = ns / 1e6
  if (ms < 1000) return `${ms.toFixed(0)}мс`

  const totalSec = Math.floor(ns / 1e9)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60

  if (h > 0) return `${h}ч ${m}м`
  if (m > 0) return `${m}м ${s}с`
  return `${s}с`
}
