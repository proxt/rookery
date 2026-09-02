<script>
  import { formatBytes } from './format.js'

  // points: [{ bucket_hour, bytes_up, bytes_down }], oldest first.
  let { points = [] } = $props()

  const width = 600
  const height = 220
  const padTop = 12
  const padBottom = 24
  const padLeft = 4
  const padRight = 4

  let hoverIndex = $state(null)

  function buildPath(values, max, key) {
    const n = values.length
    if (n === 0) return { line: '', area: '', points: [] }
    const usableH = height - padTop - padBottom
    const usableW = width - padLeft - padRight
    const pts = values.map((v, i) => {
      const x = padLeft + (n === 1 ? usableW / 2 : (i / (n - 1)) * usableW)
      const y = padTop + usableH - (max > 0 ? (v / max) * usableH : 0)
      return [x, y]
    })
    let line = `M${pts[0][0].toFixed(2)},${pts[0][1].toFixed(2)}`
    for (let i = 0; i < pts.length - 1; i++) {
      const [x0, y0] = pts[i]
      const [x1, y1] = pts[i + 1]
      const mx = (x0 + x1) / 2
      line += ` C${mx.toFixed(2)},${y0.toFixed(2)} ${mx.toFixed(2)},${y1.toFixed(2)} ${x1.toFixed(2)},${y1.toFixed(2)}`
    }
    const area = `${line} L${pts[pts.length - 1][0].toFixed(2)},${height - padBottom} L${pts[0][0].toFixed(2)},${height - padBottom} Z`
    return { line, area, points: pts }
  }

  const maxVal = $derived(Math.max(1, ...points.map((p) => Math.max(p.bytes_up, p.bytes_down))))
  const upPath = $derived(buildPath(points.map((p) => p.bytes_up), maxVal))
  const downPath = $derived(buildPath(points.map((p) => p.bytes_down), maxVal))

  function onMove(e) {
    if (points.length === 0) return
    const rect = e.currentTarget.getBoundingClientRect()
    const x = ((e.clientX - rect.left) / rect.width) * width
    const idx = Math.round(((x - padLeft) / (width - padLeft - padRight)) * (points.length - 1))
    hoverIndex = Math.min(points.length - 1, Math.max(0, idx))
  }

  function labelFor(bucketHour) {
    // "2006-01-02T15" -> local hour label
    const d = new Date(bucketHour + ':00:00Z')
    if (isNaN(d)) return bucketHour
    return d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  }
</script>

<div class="relative">
  <svg
    viewBox="0 0 {width} {height}"
    class="h-56 w-full overflow-visible"
    onmousemove={onMove}
    onmouseleave={() => (hoverIndex = null)}
    role="img"
  >
    <defs>
      <linearGradient id="chart-up-fill" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="var(--color-up)" stop-opacity="0.3" />
        <stop offset="100%" stop-color="var(--color-up)" stop-opacity="0" />
      </linearGradient>
      <linearGradient id="chart-down-fill" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="var(--color-down)" stop-opacity="0.3" />
        <stop offset="100%" stop-color="var(--color-down)" stop-opacity="0" />
      </linearGradient>
    </defs>

    {#each [0.25, 0.5, 0.75] as f}
      <line x1={padLeft} x2={width - padRight} y1={padTop + (height - padTop - padBottom) * f} y2={padTop + (height - padTop - padBottom) * f}
        stroke="var(--color-border)" stroke-width="1" stroke-dasharray="3,4" opacity="0.5" />
    {/each}

    <path d={downPath.area} fill="url(#chart-down-fill)" class="path-anim" />
    <path d={upPath.area} fill="url(#chart-up-fill)" class="path-anim" />
    <path d={downPath.line} fill="none" stroke="var(--color-down)" stroke-width="2" stroke-linecap="round" vector-effect="non-scaling-stroke" class="path-anim" />
    <path d={upPath.line} fill="none" stroke="var(--color-up)" stroke-width="2" stroke-linecap="round" vector-effect="non-scaling-stroke" class="path-anim" />

    {#if hoverIndex !== null && upPath.points[hoverIndex]}
      <line x1={upPath.points[hoverIndex][0]} x2={upPath.points[hoverIndex][0]} y1={padTop} y2={height - padBottom}
        stroke="var(--color-muted)" stroke-width="1" opacity="0.4" />
      <circle cx={upPath.points[hoverIndex][0]} cy={upPath.points[hoverIndex][1]} r="3" fill="var(--color-up)" />
      <circle cx={downPath.points[hoverIndex][0]} cy={downPath.points[hoverIndex][1]} r="3" fill="var(--color-down)" />
    {/if}
  </svg>

  {#if hoverIndex !== null && points[hoverIndex]}
    <div class="pointer-events-none absolute top-0 rounded-lg border border-border bg-surface-2 px-2.5 py-1.5 text-xs shadow-lg fade-in-up"
      style="left: {Math.min(85, Math.max(0, (upPath.points[hoverIndex][0] / width) * 100))}%">
      <div class="mb-0.5 text-muted">{labelFor(points[hoverIndex].bucket_hour)}</div>
      <div class="flex items-center gap-1.5"><span class="h-1.5 w-1.5 rounded-full bg-up"></span>↑ {formatBytes(points[hoverIndex].bytes_up)}</div>
      <div class="flex items-center gap-1.5"><span class="h-1.5 w-1.5 rounded-full bg-down"></span>↓ {formatBytes(points[hoverIndex].bytes_down)}</div>
    </div>
  {/if}
</div>

<style>
  .path-anim { transition: d 0.4s cubic-bezier(0.16, 1, 0.3, 1); }
</style>
