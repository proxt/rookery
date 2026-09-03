<script>
  import { OpenURL, GetAppVersion } from '../../../wailsjs/go/main/App.js'
  import logo from '../../assets/images/logo.png'

  const REPO_URL = 'https://github.com/proxt/rookery'

  let version = $state('…')
  GetAppVersion().then((v) => (version = v))

  function open(url) {
    OpenURL(url)
  }
</script>

<div class="flex flex-1 flex-col items-center overflow-y-auto px-6 py-8 text-center">
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

  <button
    class="btn-secondary mt-4 w-full"
    onclick={() => open(REPO_URL)}
  >
    Репозиторий на GitHub
  </button>

  <p class="mt-6 text-[11px] leading-relaxed text-muted">
    Системный VPN-режим использует Wintun — © WireGuard LLC, распространяется
    по отдельной лицензии (см. client/gui/wintun/LICENSE.txt в репозитории).
  </p>
</div>
