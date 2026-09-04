<script>
  let { active = 'servers', onselect } = $props()

  const items = [
    { id: 'servers', label: 'Серверы', icon: 'home' },
    { id: 'routing', label: 'Маршруты', icon: 'routing' },
    { id: 'stats', label: 'Статистика', icon: 'stats' },
    { id: 'logs', label: 'Логи', icon: 'logs' },
    { id: 'settings', label: 'Настройки', icon: 'settings' },
    { id: 'about', label: 'О программе', icon: 'about' },
  ]
</script>

<nav class="flex w-[76px] shrink-0 flex-col items-center gap-1 border-r border-border bg-surface/60 py-4 backdrop-blur-sm">
  {#each items as item (item.id)}
    <button
      class="group relative flex w-[64px] flex-col items-center gap-1 rounded-xl py-2.5 text-[10px] font-medium
             transition-colors duration-150 cursor-pointer
             {active === item.id ? 'bg-surface-2 text-text' : 'text-muted hover:bg-surface-2/60 hover:text-text'}"
      onclick={() => onselect(item.id)}
    >
      {#if active === item.id}
        <span class="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-r-full bg-gradient-to-b from-up to-accent-2"></span>
      {/if}
      <svg width="19" height="19" viewBox="0 0 24 24" fill="none">
        {#if item.icon === 'home'}
          <path d="M4 11.5 12 4l8 7.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M6 10v9h12v-9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        {:else if item.icon === 'routing'}
          <path d="M4 6h6a2 2 0 0 1 2 2v8a2 2 0 0 0 2 2h6M16 6l4 4-4 4M4 18h6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        {:else if item.icon === 'stats'}
          <path d="M4 20V10M11 20V4M18 20v-7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        {:else if item.icon === 'logs'}
          <path d="M5 4h14v16l-3-2-3 2-3-2-3 2-2-2V4Z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" />
          <path d="M8 9h8M8 13h5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
        {:else if item.icon === 'settings'}
          <circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2" />
          <path
            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1.08-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z"
            stroke="currentColor"
            stroke-width="1.5"
          />
        {:else}
          <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="2" />
          <path d="M12 11v6" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          <circle cx="12" cy="7.5" r="1" fill="currentColor" />
        {/if}
      </svg>
      {item.label}
    </button>
  {/each}
</nav>
