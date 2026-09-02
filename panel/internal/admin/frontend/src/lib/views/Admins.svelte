<script>
  import { fade } from 'svelte/transition'
  import { flip } from 'svelte/animate'
  import { api } from '../api.js'
  import { formatDateTime } from '../format.js'
  import Pill from '../Pill.svelte'

  let admins = $state([])
  let loading = $state(true)

  let currentPassword = $state('')
  let newPassword = $state('')
  let savingPassword = $state(false)
  let passwordError = $state('')
  let passwordSaved = $state(false)

  let newUsername = $state('')
  let newAdminPassword = $state('')
  let creating = $state(false)
  let createError = $state('')

  async function load() {
    loading = true
    admins = (await api.listAdmins()) ?? []
    loading = false
  }
  load()

  async function changeOwnPassword() {
    passwordError = ''
    passwordSaved = false
    savingPassword = true
    try {
      await api.changeOwnPassword(currentPassword, newPassword)
      currentPassword = ''
      newPassword = ''
      passwordSaved = true
      setTimeout(() => (passwordSaved = false), 2000)
    } catch {
      passwordError = 'Не удалось обновить — проверьте текущий пароль'
    } finally {
      savingPassword = false
    }
  }

  async function createAdmin() {
    createError = ''
    if (!newUsername.trim() || newAdminPassword.length < 8) {
      createError = 'Логин обязателен, пароль — минимум 8 символов'
      return
    }
    creating = true
    try {
      await api.createAdmin(newUsername.trim(), newAdminPassword)
      newUsername = ''
      newAdminPassword = ''
      await load()
    } catch (e) {
      createError = e.status === 409 ? 'Такой логин уже занят' : 'Не удалось создать администратора'
    } finally {
      creating = false
    }
  }

  async function remove(id) {
    await api.deleteAdmin(id).catch(() => {})
    await load()
  }
</script>

<div class="mx-auto max-w-2xl">
  <h1 class="mb-6 text-lg font-semibold">Админы</h1>

  <div class="card fade-in-up mb-4 p-5">
    <h2 class="mb-1 text-sm font-semibold">Ваш пароль</h2>
    <div class="mt-3 space-y-3">
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="cred-current">Текущий пароль</label>
        <input id="cred-current" type="password" class="input" bind:value={currentPassword} />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="cred-new">Новый пароль</label>
        <input id="cred-new" type="password" class="input" bind:value={newPassword} />
      </div>
      <div class="flex items-center justify-between pt-1">
        {#if passwordError}<span class="text-xs text-danger" transition:fade>{passwordError}</span>{:else}<span></span>{/if}
        <button class="btn-primary" onclick={changeOwnPassword} disabled={savingPassword || !currentPassword || !newPassword}>
          {savingPassword ? '…' : passwordSaved ? '✓ Обновлено' : 'Обновить'}
        </button>
      </div>
    </div>
  </div>

  <div class="card fade-in-up p-5" style="animation-delay: 60ms">
    <h2 class="mb-1 text-sm font-semibold">Администраторы панели</h2>
    <p class="mb-4 text-xs text-muted">У каждого — свой логин и пароль для входа в эту панель.</p>

    <div class="mb-4 flex gap-2">
      <input class="input" placeholder="Логин" bind:value={newUsername} />
      <input class="input" type="password" placeholder="Пароль (мин. 8 символов)" bind:value={newAdminPassword} />
      <button class="btn-primary shrink-0" onclick={createAdmin} disabled={creating}>+ Добавить</button>
    </div>
    {#if createError}<p class="mb-3 text-xs text-danger" transition:fade>{createError}</p>{/if}

    {#if loading}
      <div class="py-8 text-center text-sm text-muted">Загрузка…</div>
    {:else}
      <div class="space-y-2">
        {#each admins as a (a.id)}
          <div class="flex items-center justify-between rounded-lg border border-border bg-surface-2 px-3 py-2.5" animate:flip={{ duration: 200 }} transition:fade={{ duration: 150 }}>
            <div class="flex items-center gap-2.5">
              <span class="text-sm font-medium">{a.username}</span>
              {#if a.is_you}<Pill tone="ok">вы</Pill>{/if}
              <span class="text-xs text-muted">с {formatDateTime(a.created_at)}</span>
            </div>
            {#if !a.is_you}
              <button class="icon-btn hover:text-danger" onclick={() => remove(a.id)} aria-label="Удалить администратора">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none">
                  <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2m-8 0 1 13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1l1-13"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </button>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
