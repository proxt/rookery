<script>
  import logo from '../assets/logo.png'
  let { active = 'dashboard', onselect, onlogout, adminUsername = '' } = $props()

  const items = [
    { id: 'dashboard', label: 'Обзор', icon: 'M4 11.5 12 4l8 7.5M6 10v9h12v-9' },
    { id: 'users', label: 'Пользователи', icon: 'M12 8a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4ZM5 20c1.2-3.6 4-5.4 7-5.4s5.8 1.8 7 5.4' },
    { id: 'nodes', label: 'Ноды', icon: 'M4 6h16M4 12h16M4 18h16' },
    { id: 'releases', label: 'Обновления', icon: 'M12 3v12m0 0-4-4m4 4 4-4M4 21h16' },
    { id: 'admins', label: 'Админы', icon: 'M12 3l7 3v6c0 4-3 7-7 8-4-1-7-4-7-8V6l7-3Z' },
    { id: 'settings', label: 'Настройки', icon: 'circle' },
  ]
</script>

<aside class="flex w-56 shrink-0 flex-col border-r border-border bg-surface/60 backdrop-blur-sm">
  <div class="flex items-center gap-2.5 px-5 py-5">
    <img src={logo} alt="" class="h-9 w-9 rounded-lg breathe" />
    <span class="text-sm font-semibold tracking-wide text-text">ROOKERY</span>
  </div>

  <nav class="flex flex-1 flex-col gap-0.5 px-3">
    {#each items as item (item.id)}
      <button
        class="group relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-150 cursor-pointer
               {active === item.id ? 'bg-surface-2 text-text' : 'text-muted hover:bg-surface-2/60 hover:text-text'}"
        onclick={() => onselect(item.id)}
      >
        {#if active === item.id}
          <span class="absolute left-0 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r-full bg-up"></span>
        {/if}
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" class="shrink-0">
          {#if item.icon === 'circle'}
            <circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1.08-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" stroke="currentColor" stroke-width="1.4" />
          {:else}
            <path d={item.icon} stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          {/if}
        </svg>
        {item.label}
      </button>
    {/each}
  </nav>

  <div class="border-t border-border p-3">
    <div class="flex items-center gap-2.5 rounded-xl px-2 py-2">
      <div class="flex h-8 w-8 items-center justify-center rounded-full bg-surface-2 text-xs font-semibold text-muted">
        {adminUsername.slice(0, 2).toUpperCase()}
      </div>
      <div class="min-w-0 flex-1">
        <div class="truncate text-xs font-medium text-text">{adminUsername}</div>
        <div class="text-[10px] text-muted">Администратор</div>
      </div>
      <button class="icon-btn" title="Выйти" onclick={onlogout}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
    </div>
  </div>
</aside>
