<script>
  let { active = 'dashboard', onselect } = $props()

  const tabs = [
    { id: 'dashboard', label: 'Главная', icon: 'home' },
    { id: 'subscriptions', label: 'Подписки', icon: 'profiles' },
    { id: 'settings', label: 'Настройки', icon: 'settings' },
    { id: 'about', label: 'О программе', icon: 'about' },
  ]
</script>

<nav class="flex shrink-0 border-t border-border bg-surface/80 backdrop-blur-sm">
  {#each tabs as tab (tab.id)}
    <button
      class="flex flex-1 flex-col items-center gap-1 py-2.5 text-[10px] font-medium
             transition-colors cursor-pointer {active === tab.id ? 'text-up' : 'text-muted hover:text-text'}"
      onclick={() => onselect(tab.id)}
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
        {#if tab.icon === 'home'}
          <path d="M4 11.5 12 4l8 7.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M6 10v9h12v-9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        {:else if tab.icon === 'profiles'}
          <circle cx="12" cy="8" r="3.2" stroke="currentColor" stroke-width="2" />
          <path d="M5 20c1.2-3.6 4-5.4 7-5.4s5.8 1.8 7 5.4" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
        {:else if tab.icon === 'settings'}
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
      {tab.label}
    </button>
  {/each}
</nav>
