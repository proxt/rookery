<script>
  import { fade } from 'svelte/transition'
  import { api } from '../api.js'

  let publicAddr = $state('')
  let adminUsername = $state('')
  let savingAddr = $state(false)
  let addrSaved = $state(false)

  let currentPassword = $state('')
  let newUsername = $state('')
  let newPassword = $state('')
  let savingCreds = $state(false)
  let credsError = $state('')
  let credsSaved = $state(false)

  async function load() {
    const s = await api.getSettings()
    publicAddr = s.public_addr
    adminUsername = s.admin_username
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

  async function saveCreds() {
    credsError = ''
    credsSaved = false
    savingCreds = true
    try {
      await api.updateCredentials(currentPassword, newUsername, newPassword)
      currentPassword = ''
      newUsername = ''
      newPassword = ''
      credsSaved = true
      await load()
      setTimeout(() => (credsSaved = false), 2000)
    } catch {
      credsError = 'Не удалось обновить — проверьте текущий пароль'
    } finally {
      savingCreds = false
    }
  }
</script>

<div class="mx-auto max-w-2xl">
  <h1 class="mb-6 text-lg font-semibold">Настройки</h1>

  <div class="card fade-in-up mb-4 p-5">
    <h2 class="mb-1 text-sm font-semibold">Публичный адрес панели</h2>
    <p class="mb-4 text-xs text-muted">Используется для формирования ссылок подписок (sub_url). Укажите домен, за которым стоит Caddy.</p>
    <div class="flex gap-2">
      <input class="input" bind:value={publicAddr} placeholder="https://panel.example.com" />
      <button class="btn-primary shrink-0" onclick={saveAddr} disabled={savingAddr}>
        {savingAddr ? '…' : addrSaved ? '✓ Сохранено' : 'Сохранить'}
      </button>
    </div>
  </div>

  <div class="card fade-in-up p-5" style="animation-delay: 60ms">
    <h2 class="mb-1 text-sm font-semibold">Логин и пароль администратора</h2>
    <p class="mb-4 text-xs text-muted">Текущий логин: <span class="text-text">{adminUsername}</span></p>

    <div class="space-y-3">
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="cred-current">Текущий пароль</label>
        <input id="cred-current" type="password" class="input" bind:value={currentPassword} />
      </div>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="mb-1 block text-xs font-medium text-muted" for="cred-username">Новый логин (необязательно)</label>
          <input id="cred-username" class="input" bind:value={newUsername} />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-muted" for="cred-password">Новый пароль (необязательно)</label>
          <input id="cred-password" type="password" class="input" bind:value={newPassword} />
        </div>
      </div>
      <div class="flex items-center justify-between pt-1">
        {#if credsError}<span class="text-xs text-danger" transition:fade>{credsError}</span>{:else}<span></span>{/if}
        <button class="btn-primary" onclick={saveCreds} disabled={savingCreds || !currentPassword}>
          {savingCreds ? '…' : credsSaved ? '✓ Обновлено' : 'Обновить'}
        </button>
      </div>
    </div>
  </div>
</div>
