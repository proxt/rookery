<script>
  import { OpenURL } from '../../../wailsjs/go/main/App.js'
  import logo from '../../assets/images/logo.png'

  const REPO_URL = 'https://github.com/proxt/rookery'

  function open(url) {
    OpenURL(url)
  }
</script>

<div class="flex flex-1 flex-col items-center px-6 py-8 text-center">
  <img src={logo} alt="Rookery" class="mb-4 h-20 w-20 rounded-full shadow-lg" />
  <h1 class="text-lg font-semibold">Rookery</h1>
  <p class="mt-1 text-xs text-muted">WebRTC-туннель, замаскированный под обычный p2p-трафик</p>

  <div class="card mt-6 w-full p-4 text-left text-xs">
    <div class="mb-2 flex items-center justify-between">
      <span class="text-muted">Версия</span>
      <span class="font-medium">0.1.0</span>
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
