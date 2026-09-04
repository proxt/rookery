<script>
  import { fade } from 'svelte/transition'
  import { SaveGeneralSettings } from '../../../wailsjs/go/main/App.js'
  import Toggle from '../Toggle.svelte'

  let { settings = {}, onchange } = $props()

  let socksPort = $state(1080)
  let autoStart = $state(false)
  let startMinimized = $state(false)
  let systemWide = $state(false)
  let killSwitch = $state(false)
  let subAutoRefreshMinutes = $state(0)
  let subRefreshOnLaunch = $state(false)

  let saving = $state(false)
  let saveError = $state('')
  let saved = $state(false)

  $effect(() => {
    socksPort = settings.socksPort ?? 1080
    autoStart = settings.autoStart ?? false
    startMinimized = settings.startMinimized ?? false
    systemWide = settings.systemWide ?? false
    killSwitch = settings.killSwitch ?? false
    subAutoRefreshMinutes = settings.subAutoRefreshMinutes ?? 0
    subRefreshOnLaunch = settings.subRefreshOnLaunch ?? false
  })

  async function handleSave() {
    saving = true
    saveError = ''
    saved = false
    try {
      await SaveGeneralSettings(
        Number(socksPort),
        autoStart,
        startMinimized,
        systemWide,
        killSwitch && systemWide,
        Number(subAutoRefreshMinutes),
        subRefreshOnLaunch
      )
      await onchange()
      saved = true
      setTimeout(() => (saved = false), 1600)
    } catch (e) {
      saveError = String(e)
    } finally {
      saving = false
    }
  }
</script>

<div class="scroll-area flex flex-1 flex-col px-6 py-5">
  <h1 class="mb-4 text-xs font-semibold uppercase tracking-widest text-muted">Настройки</h1>

  <div class="mb-2 px-1 text-[10px] font-semibold uppercase tracking-widest text-muted/60">Приложение</div>
  <div class="card fade-in-up mb-3 divide-y divide-border p-1">
    <label class="flex items-center justify-between gap-3 px-3 py-3">
      <span class="text-sm">Автозапуск при входе в систему</span>
      <Toggle bind:checked={autoStart} />
    </label>
    <label class="flex items-center justify-between gap-3 px-3 py-3">
      <span class="text-sm">Запускать свёрнутым</span>
      <Toggle bind:checked={startMinimized} />
    </label>
  </div>

  <div class="mb-2 px-1 text-[10px] font-semibold uppercase tracking-widest text-muted/60">Сеть</div>
  <div class="card card-accent fade-in-up mb-3 p-4" style="animation-delay: 40ms">
    <label class="mb-1 block">
      <span class="mb-1 block text-xs text-muted">Порт SOCKS</span>
      <input class="input" type="number" min="1" max="65535" bind:value={socksPort} />
    </label>
  </div>

  <div class="card fade-in-up mb-3 p-4" style="animation-delay: 80ms">
    <label class="flex items-center justify-between gap-3">
      <span class="text-sm">Весь трафик ПК (системный VPN)</span>
      <Toggle bind:checked={systemWide} />
    </label>
    <p class="mt-2 text-xs text-muted">
      Заворачивает весь трафик системы через туннель, а не только приложения с
      настроенным SOCKS5. Требует прав администратора и завершает работу при
      одновременно включённом другом системном VPN.
    </p>

    <div class="mt-3 border-t border-border pt-3">
      <label class="flex items-center justify-between gap-3 {systemWide ? '' : 'opacity-40'}">
        <span class="text-sm">Kill switch — блокировать трафик при обрыве</span>
        <Toggle bind:checked={killSwitch} disabled={!systemWide} />
      </label>
      <p class="mt-2 text-xs text-muted">
        Пока туннель переподключается, весь трафик мимо него блокируется — ничего
        не утечёт напрямую. Если переподключиться не удаётся 10 минут, блокировка
        снимается автоматически, чтобы не остаться без интернета совсем. Работает
        только вместе с системным VPN.
      </p>
    </div>
  </div>

  <div class="mb-2 px-1 text-[10px] font-semibold uppercase tracking-widest text-muted/60">Подписки</div>
  <div class="card fade-in-up mb-3 p-4" style="animation-delay: 100ms">
    <div class="mb-1 text-xs font-semibold uppercase tracking-widest text-muted">Автообновление</div>
    <label class="mt-2 flex items-center justify-between gap-3">
      <span class="text-sm">Обновлять при запуске приложения</span>
      <Toggle bind:checked={subRefreshOnLaunch} />
    </label>
    <label class="mt-3 mb-1 block">
      <span class="mb-1 block text-xs text-muted">Автообновление каждые (минут, 0 — выключено)</span>
      <input class="input" type="number" min="0" max="1440" bind:value={subAutoRefreshMinutes} />
    </label>
    <p class="mt-1 text-xs text-muted">Список серверов у всех сохранённых подписок будет обновляться сам, без ручного «Обновить».</p>
  </div>

  {#if saveError}
    <p class="mb-3 text-xs text-state-error" transition:fade={{ duration: 150 }}>{saveError}</p>
  {/if}

  <div class="mb-3 flex items-center gap-3">
    <button class="btn-primary" onclick={handleSave} disabled={saving}>
      {saving ? 'Сохранение…' : 'Сохранить'}
    </button>
    {#if saved}
      <span class="text-xs text-state-connected" transition:fade={{ duration: 150 }}>Сохранено</span>
    {/if}
  </div>
</div>
