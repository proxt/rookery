<script>
  import { fade } from 'svelte/transition'
  import { SaveGeneralSettings } from '../../../wailsjs/go/main/App.js'

  let { settings = {}, onchange } = $props()

  let socksPort = $state(1080)
  let autoStart = $state(false)
  let startMinimized = $state(false)
  let systemWide = $state(false)

  let saving = $state(false)
  let saveError = $state('')
  let saved = $state(false)

  $effect(() => {
    socksPort = settings.socksPort ?? 1080
    autoStart = settings.autoStart ?? false
    startMinimized = settings.startMinimized ?? false
    systemWide = settings.systemWide ?? false
  })

  async function handleSave() {
    saving = true
    saveError = ''
    saved = false
    try {
      await SaveGeneralSettings(Number(socksPort), autoStart, startMinimized, systemWide)
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

<div class="flex flex-1 flex-col px-6 py-5">
  <h1 class="mb-4 text-xs font-semibold uppercase tracking-widest text-muted">Настройки</h1>

  <label class="mb-4 block">
    <span class="mb-1 block text-xs text-muted">Порт SOCKS</span>
    <input class="input" type="number" min="1" max="65535" bind:value={socksPort} />
  </label>

  <label class="mb-2 flex items-center justify-between">
    <span class="text-sm">Автозапуск при входе в систему</span>
    <input type="checkbox" class="h-4 w-4 accent-up" bind:checked={autoStart} />
  </label>

  <label class="mb-2 flex items-center justify-between">
    <span class="text-sm">Запускать свёрнутым</span>
    <input type="checkbox" class="h-4 w-4 accent-up" bind:checked={startMinimized} />
  </label>

  <label class="mb-1 flex items-center justify-between">
    <span class="text-sm">Весь трафик ПК (системный VPN)</span>
    <input type="checkbox" class="h-4 w-4 accent-up" bind:checked={systemWide} />
  </label>
  <p class="mb-6 text-xs text-muted">
    Заворачивает весь трафик системы через туннель, а не только приложения с
    настроенным SOCKS5. Требует прав администратора и завершает работу при
    одновременно включённом другом системном VPN.
  </p>

  {#if saveError}
    <p class="mb-3 text-xs text-state-error" transition:fade={{ duration: 150 }}>{saveError}</p>
  {/if}

  <div class="flex items-center gap-3">
    <button class="btn-primary" onclick={handleSave} disabled={saving}>
      {saving ? 'Сохранение…' : 'Сохранить'}
    </button>
    {#if saved}
      <span class="text-xs text-state-connected" transition:fade={{ duration: 150 }}>Сохранено</span>
    {/if}
  </div>
</div>
