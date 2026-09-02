<script>
  // tone: 'ok' | 'warn' | 'danger' | 'muted'
  let { tone = 'muted', children } = $props()
  const toneClass = {
    ok: 'bg-ok/15 text-ok',
    warn: 'bg-warn/15 text-warn',
    danger: 'bg-danger/15 text-danger',
    muted: 'bg-surface-3 text-muted',
  }
</script>

<span class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium {toneClass[tone]}">
  <span class="h-1.5 w-1.5 rounded-full" style="background: currentColor"></span>
  {@render children()}
</span>
