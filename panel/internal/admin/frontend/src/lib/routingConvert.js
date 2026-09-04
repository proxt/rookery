// Converts a routing rule set { id, name, rules: [{id,type,value,action}] }
// to/from Rookery's own JSON shape and a v2ray-core routing config (also
// what Happ exports for custom routing rules). Mirrors
// client/internal/routing/v2ray.go exactly — kept as plain JS here rather
// than shared with the Go module, per the same reasoning already applied to
// flags.js: two separate frontend projects, no shared JS package between
// them, and the conversion is short enough that duplicating it is cheaper
// than introducing one.

export function toRookeryJSON(ruleSet) {
  return JSON.stringify({ id: ruleSet.id, name: ruleSet.name, rules: ruleSet.rules }, null, 2)
}

export function fromRookeryJSON(text) {
  const parsed = JSON.parse(text)
  if (!Array.isArray(parsed.rules)) throw new Error('некорректный формат')
  return { name: parsed.name || '', rules: parsed.rules.map((r) => ({ type: r.type, value: r.value, action: r.action })), skipped: 0 }
}

export function toV2rayJSON(ruleSet) {
  let skipped = 0
  const rules = []
  for (const r of ruleSet.rules) {
    const tag = r.action === 'direct' ? 'direct' : 'proxy'
    if (r.type === 'domain') {
      rules.push({ type: 'field', domain: [`domain:${r.value}`], outboundTag: tag })
    } else if (r.type === 'geoip') {
      rules.push({ type: 'field', ip: [`geoip:${String(r.value).toLowerCase()}`], outboundTag: tag })
    } else {
      skipped++
    }
  }
  const data = JSON.stringify({ routing: { domainStrategy: 'AsIs', rules } }, null, 2)
  return { data, skipped }
}

export function fromV2rayJSON(text, name) {
  const parsed = JSON.parse(text)
  const routing = parsed.routing || parsed
  const rules = []
  let skipped = 0
  for (const vr of routing.rules || []) {
    const action = vr.outboundTag === 'direct' ? 'direct' : 'proxy'
    for (const d of vr.domain || []) {
      if (d.startsWith('domain:')) rules.push({ type: 'domain', value: d.slice('domain:'.length), action })
      else skipped++
    }
    for (const ip of vr.ip || []) {
      if (ip.startsWith('geoip:')) rules.push({ type: 'geoip', value: ip.slice('geoip:'.length).toUpperCase(), action })
      else skipped++
    }
  }
  return { name: name || '', rules, skipped }
}

export function downloadJSON(filename, content) {
  const blob = new Blob([content], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

export function readFile(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = () => reject(reader.error)
    reader.readAsText(file)
  })
}
