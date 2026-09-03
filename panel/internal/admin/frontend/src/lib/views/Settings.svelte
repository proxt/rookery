<script>
  import { api } from '../api.js'
  import Toggle from '../Toggle.svelte'

  let publicAddr = $state('')
  let autoUpdateEnabled = $state(true)
  let savingAddr = $state(false)
  let addrSaved = $state(false)

  async function load() {
    const s = await api.getSettings()
    publicAddr = s.public_addr
    autoUpdateEnabled = s.auto_update_enabled
  }
  load()

  async function saveAddr() {
    savingAddr = true
    addrSaved = false
    try {
      await api.updateSettings(publicAddr, autoUpdateEnabled)
      addrSaved = true
      setTimeout(() => (addrSaved = false), 2000)
    } finally {
      savingAddr = false
    }
  }

  async function onAutoUpdateChange(value) {
    autoUpdateEnabled = value
    await api.updateSettings(publicAddr, value)
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

  <div class="card fade-in-up mt-4 p-5" style="animation-delay: 60ms">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-sm font-semibold">Автоматическое обновление</h2>
        <p class="mt-1 max-w-md text-xs text-muted">
          Панель сама проверяет и накатывает новые образы через Watchtower каждые
          5 минут. Документация (эта страница на /docs) обновляется тем же
          образом — вместе с панелью. Выключение не затрагивает уже
          запущенный процесс, только будущие проверки.
        </p>
      </div>
      <Toggle bind:checked={autoUpdateEnabled} onchange={onAutoUpdateChange} />
    </div>
  </div>

  <p class="mt-4 text-xs text-muted">
    Логин и пароль администратора — на вкладке «Админы».
  </p>
</div>
