<script>
  import { api } from '../api.js'

  let publicAddr = $state('')
  let savingAddr = $state(false)
  let addrSaved = $state(false)

  async function load() {
    const s = await api.getSettings()
    publicAddr = s.public_addr
  }
  load()

  async function saveAddr() {
    savingAddr = true
    addrSaved = false
    try {
      await api.updateSettings(publicAddr)
      addrSaved = true
      setTimeout(() => (addrSaved = false), 2000)
    } finally {
      savingAddr = false
    }
  }
</script>

<div class="mx-auto max-w-2xl">
  <h1 class="mb-6 text-lg font-semibold">Настройки</h1>

  <div class="card fade-in-up p-5">
    <h2 class="mb-1 text-sm font-semibold">Публичный адрес панели</h2>
    <p class="mb-4 text-xs text-muted">Используется для формирования ссылок подписок (sub_url) и ссылок на обновления. Укажите домен, за которым стоит Caddy.</p>
    <div class="flex gap-2">
      <input class="input" bind:value={publicAddr} placeholder="https://panel.example.com" />
      <button class="btn-primary shrink-0" onclick={saveAddr} disabled={savingAddr}>
        {savingAddr ? '…' : addrSaved ? '✓ Сохранено' : 'Сохранить'}
      </button>
    </div>
  </div>

  <p class="mt-4 text-xs text-muted">
    Логин и пароль администратора — на вкладке «Админы».
  </p>
</div>
