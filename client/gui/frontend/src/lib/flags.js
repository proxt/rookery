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

// flag-icons CSS class for the parsed country code (e.g. "fi-cz"), or ''
// if tags has no recognizable country token. Windows doesn't render
// Unicode regional-indicator flag emoji as pictures (shows "CZ" as plain
// text instead, unlike macOS/iOS) — flag-icons draws a real SVG flag via
// CSS regardless of platform, so this is what's actually used in the UI;
// countryFlag() above is kept for contexts (docs, plain-text exports)
// where an embedded image isn't an option.
export function countryFlagClass(tags) {
  const code = countryCodeFromTags(tags)
  return code ? `fi fi-${code.toLowerCase()}` : ''
}

// The rest of a free-text tags string with the parsed country-code token
// removed — for displaying alongside a flag icon without repeating the
// code as text underneath it.
export function tagsWithoutCountry(tags) {
  if (!tags) return ''
  const code = countryCodeFromTags(tags)
  if (!code) return tags
  return tags
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t.toUpperCase() !== code)
    .join(', ')
}
