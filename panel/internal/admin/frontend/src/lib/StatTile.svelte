<script>
  let { label, value, icon = null, accent = 'up', delay = 0 } = $props()
</script>

<div class="card fade-in-up flex items-center gap-4 p-4" style="animation-delay: {delay}ms">
  {#if icon}
    <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-xl"
      style="background: color-mix(in srgb, var(--color-{accent}) 15%, transparent); color: var(--color-{accent})">
      {icon}
    </div>
  {/if}
  <div class="min-w-0">
    <div class="truncate text-xl font-semibold">{value}</div>
    <div class="truncate text-xs text-muted">{label}</div>
  </div>
</div>
