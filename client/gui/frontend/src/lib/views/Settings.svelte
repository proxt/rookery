<script>
  import { fade } from 'svelte/transition'
  import { SaveGeneralSettings, CheckForUpdate, DownloadAndInstallUpdate } from '../../../wailsjs/go/main/App.js'

  let { settings = {}, onchange } = $props()

  let socksPort = $state(1080)
  let autoStart = $state(false)
  let startMinimized = $state(false)
  let systemWide = $state(false)

  let saving = $state(false)
  let saveError = $state('')
  let saved = $state(false)

  let checkingUpdate = $state(false)
  let updateError = $state('')
  let updateInfo = $state(null)
  let installing = $state(false)
  let confirmingInstall = $state(false)

  async function checkUpdate() {
    checkingUpdate = true
    updateError = ''
    updateInfo = null
    try {
      updateInfo = await CheckForUpdate()
    } catch (e) {
      updateError = String(e)
    } finally {
      checkingUpdate = false
    }
  }

  async function installUpdate() {
    installing = true
    try {
      await DownloadAndInstallUpdate(updateInfo.downloadUrl)
      // App quits itself right after launching the installer — nothing
      // more to do here if this ever returns.
    } catch (e) {
      updateError = String(e)
      installing = false
      confirmingInstall = false
    }
  }

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

  <div class="mb-6 flex items-center gap-3">
    <button class="btn-primary" onclick={handleSave} disabled={saving}>
      {saving ? 'Сохранение…' : 'Сохранить'}
    </button>
    {#if saved}
      <span class="text-xs text-state-connected" transition:fade={{ duration: 150 }}>Сохранено</span>
    {/if}
  </div>

  <div class="card p-4">
    <div class="mb-1 text-xs font-semibold uppercase tracking-widest text-muted">Обновления</div>
    <p class="mb-3 text-xs text-muted">Версия проверяется по подписке, выбранной активной.</p>

    <button class="btn-secondary w-full" onclick={checkUpdate} disabled={checkingUpdate}>
      {checkingUpdate ? 'Проверка…' : 'Проверить обновления'}
    </button>

    {#if updateError}
      <p class="mt-2 text-xs text-state-error" transition:fade={{ duration: 150 }}>{updateError}</p>
    {/if}

    {#if updateInfo}
      <div class="mt-3 rounded-lg border border-border p-3 text-xs" transition:fade={{ duration: 150 }}>
        {#if updateInfo.available}
          <div class="mb-1 font-medium text-text">Доступна версия {updateInfo.version} (у вас {updateInfo.currentVersion})</div>
          {#if updateInfo.notes}<p class="mb-2 text-muted">{updateInfo.notes}</p>{/if}
          {#if !confirmingInstall}
            <button class="btn-primary w-full" onclick={() => (confirmingInstall = true)}>Скачать и установить</button>
          {:else}
            <p class="mb-2 text-muted">Приложение закроется и запустится установщик. Продолжить?</p>
            <div class="flex gap-2">
              <button class="btn-primary flex-1" onclick={installUpdate} disabled={installing}>
                {installing ? 'Скачивание…' : 'Да, установить'}
              </button>
              <button class="btn-secondary flex-1" onclick={() => (confirmingInstall = false)} disabled={installing}>Отмена</button>
            </div>
          {/if}
        {:else}
          <span class="text-muted">У вас последняя версия ({updateInfo.currentVersion})</span>
        {/if}
      </div>
    {/if}
  </div>
</div>
