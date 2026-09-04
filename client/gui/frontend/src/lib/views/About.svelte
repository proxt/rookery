<script>
  import { fade } from 'svelte/transition'
  import { OpenURL, GetAppVersion, CheckForUpdate, DownloadAndInstallUpdate } from '../../../wailsjs/go/main/App.js'
  import logo from '../../assets/images/logo.png'

  const REPO_URL = 'https://github.com/proxt/rookery'

  let version = $state('…')
  GetAppVersion().then((v) => (version = v))

  function open(url) {
    OpenURL(url)
  }

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
</script>

<div class="scroll-area flex flex-1 flex-col items-center px-6 py-8 text-center">
  <div class="relative mb-4">
    <span class="breathe absolute -inset-5 -z-10 rounded-full bg-gradient-to-br from-up to-accent-2 opacity-30 blur-2xl"></span>
    <img src={logo} alt="Rookery" class="h-20 w-20 rounded-full shadow-[0_8px_28px_-6px_var(--color-up)]" />
  </div>
  <h1 class="text-lg font-semibold">Rookery</h1>
  <p class="mt-1 text-xs text-muted">WebRTC-туннель, замаскированный под обычный p2p-трафик</p>

  <div class="card card-accent mt-6 w-full p-4 text-left text-xs">
    <div class="mb-2 flex items-center justify-between">
      <span class="text-muted">Версия</span>
      <span class="rounded-full bg-gradient-to-r from-up to-accent-2 px-2 py-0.5 font-mono text-[11px] font-semibold text-white">{version}</span>
    </div>
    <div class="flex items-center justify-between">
      <span class="text-muted">Транспорт</span>
      <span class="font-medium">WebRTC / smux</span>
    </div>
  </div>

  <div class="card fade-in-up mt-4 w-full p-4 text-left text-xs" style="animation-delay: 40ms">
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

  <button
    class="btn-secondary mt-3 w-full"
    onclick={() => open(REPO_URL)}
  >
    Репозиторий на GitHub
  </button>

  <p class="mt-6 text-[11px] leading-relaxed text-muted">
    Системный VPN-режим использует Wintun — © WireGuard LLC, распространяется
    по отдельной лицензии (см. client/gui/wintun/LICENSE.txt в репозитории).
    Маршрутизация по странам использует базу IP-адресов DB-IP (db-ip.com),
    распространяемую по лицензии CC BY 4.0.
  </p>
</div>
