<script>
  let { name, class: className = 'text-muted' } = $props()
</script>

<svg width="14" height="14" viewBox="0 0 24 24" fill="none" class={className}>
  {#if name === 'activity'}
    <path
      d="M2 12h4l2.5-7 4 14 2.5-7H22"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  {:else if name === 'clock'}
    <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="2" />
    <path d="M12 7v5l3.5 2" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
  {:else if name === 'layers'}
    <path
      d="M12 3 2 8l10 5 10-5-10-5Z"
      stroke="currentColor"
      stroke-width="2"
      stroke-linejoin="round"
    />
    <path d="M2 16l10 5 10-5" stroke="currentColor" stroke-width="2" stroke-linejoin="round" />
    <path d="M2 12l10 5 10-5" stroke="currentColor" stroke-width="2" stroke-linejoin="round" />
  {:else if name === 'swap'}
    <path d="M7 3v14" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    <path d="M3 13l4 4 4-4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
    <path d="M17 21V7" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    <path d="M21 11l-4-4-4 4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
  {/if}
</svg>
