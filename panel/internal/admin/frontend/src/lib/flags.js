// Node "tags" is free text with no enforced structure (UI convention is
// something like "cz, cheap" — a country code among other comma-separated
// labels, never parsed anywhere else). This pulls the first 2-letter token
// out and renders it as a flag via Unicode regional indicator symbols — no
// image assets, no new dependency.
const REGIONAL_INDICATOR_BASE = 0x1f1e6 // 'A'

export function countryCodeFromTags(tags) {
  if (!tags) return null
  for (const raw of tags.split(',')) {
    const token = raw.trim()
    if (/^[a-zA-Z]{2}$/.test(token)) return token.toUpperCase()
  }
  return null
}

export function countryFlag(tags) {
  const code = countryCodeFromTags(tags)
  if (!code) return ''
  return String.fromCodePoint(
    REGIONAL_INDICATOR_BASE + (code.charCodeAt(0) - 65),
    REGIONAL_INDICATOR_BASE + (code.charCodeAt(1) - 65)
  )
}
