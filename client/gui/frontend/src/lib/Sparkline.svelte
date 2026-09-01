<script>
  // history: array of { up, down } byte/s samples, oldest first.
  let { history = [] } = $props()

  const width = 100
  const height = 100
  const pad = 6

  // Builds a smooth (Catmull-Rom-ish via midpoint cubic) line + closed area
  // path, plus the coordinates of the last point for the endpoint marker.
  function buildPath(values, max) {
    const n = values.length
    if (n === 0) return { line: '', area: '', last: null }

    const pts = values.map((v, i) => {
      const x = n === 1 ? width : (i / (n - 1)) * width
      const y = height - pad - (v / max) * (height - pad * 2)
      return [x, y]
    })

    if (n === 1) {
      const [x, y] = pts[0]
      return { line: `M${x},${y} L${x},${y}`, area: `M${x},${height} L${x},${y} L${x},${height} Z`, last: [x, y] }
    }

    let line = `M${pts[0][0].toFixed(2)},${pts[0][1].toFixed(2)}`
    for (let i = 0; i < pts.length - 1; i++) {
      const [x0, y0] = pts[i]
      const [x1, y1] = pts[i + 1]
      const mx = (x0 + x1) / 2
      line += ` C${mx.toFixed(2)},${y0.toFixed(2)} ${mx.toFixed(2)},${y1.toFixed(2)} ${x1.toFixed(2)},${y1.toFixed(2)}`
    }

    const last = pts[pts.length - 1]
    const first = pts[0]
    const area = `${line} L${last[0].toFixed(2)},${height} L${first[0].toFixed(2)},${height} Z`
    return { line, area, last }
  }

  const maxVal = $derived(Math.max(1, ...history.map((h) => Math.max(h.up, h.down))))
  const downPath = $derived(buildPath(history.map((h) => h.down), maxVal))
  const upPath = $derived(buildPath(history.map((h) => h.up), maxVal))
</script>

<svg viewBox="0 0 {width} {height}" preserveAspectRatio="none" class="h-16 w-full overflow-visible">
  <defs>
    <linearGradient id="down-fill" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="var(--color-down)" stop-opacity="0.35" />
      <stop offset="100%" stop-color="var(--color-down)" stop-opacity="0" />
    </linearGradient>
    <linearGradient id="up-fill" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="var(--color-up)" stop-opacity="0.35" />
      <stop offset="100%" stop-color="var(--color-up)" stop-opacity="0" />
    </linearGradient>
  </defs>

  <path d={downPath.area} fill="url(#down-fill)" class="path-anim" />
  <path d={upPath.area} fill="url(#up-fill)" class="path-anim" />

  <path
    d={downPath.line}
    fill="none"
    stroke="var(--color-down)"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    vector-effect="non-scaling-stroke"
    class="path-anim"
  />
  <path
    d={upPath.line}
    fill="none"
    stroke="var(--color-up)"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    vector-effect="non-scaling-stroke"
    class="path-anim"
  />

  {#if downPath.last}
    <circle cx={downPath.last[0]} cy={downPath.last[1]} r="2.2" fill="var(--color-down)" class="dot-anim" />
  {/if}
  {#if upPath.last}
    <circle cx={upPath.last[0]} cy={upPath.last[1]} r="2.2" fill="var(--color-up)" class="dot-anim" />
  {/if}
</svg>

<style>
  .path-anim {
    transition: d 0.4s cubic-bezier(0.16, 1, 0.3, 1);
  }
  .dot-anim {
    transition: cx 0.4s cubic-bezier(0.16, 1, 0.3, 1), cy 0.4s cubic-bezier(0.16, 1, 0.3, 1);
    filter: drop-shadow(0 0 3px currentColor);
  }
</style>
