<script>
  import { fade, fly } from 'svelte/transition'
  import { api } from '../api.js'
  import logo from '../../assets/logo.png'

  let { onsuccess } = $props()

  let username = $state('')
  let password = $state('')
  let error = $state('')
  let loading = $state(false)

  async function submit(e) {
    e.preventDefault()
    error = ''
    loading = true
    try {
      await api.login(username, password)
      onsuccess()
    } catch {
      error = 'Неверный логин или пароль'
    } finally {
      loading = false
    }
  }
</script>

<div class="flex min-h-screen items-center justify-center p-6">
  <form class="card w-full max-w-sm p-8" in:fly={{ y: 12, duration: 400, easing: (t) => 1 - Math.pow(1 - t, 3) }} onsubmit={submit}>
    <div class="mb-8 flex flex-col items-center gap-3">
      <img src={logo} alt="" class="h-14 w-14 rounded-2xl breathe" />
      <div class="text-center">
        <div class="text-base font-semibold">Rookery Panel</div>
        <div class="text-xs text-muted">Вход для администратора</div>
      </div>
    </div>

    <label class="mb-1 block text-xs font-medium text-muted" for="username">Логин</label>
    <input id="username" class="input mb-4" autocomplete="username" bind:value={username} disabled={loading} />

    <label class="mb-1 block text-xs font-medium text-muted" for="password">Пароль</label>
    <input id="password" type="password" class="input mb-6" autocomplete="current-password" bind:value={password} disabled={loading} />

    <button type="submit" class="btn-primary w-full" disabled={loading || !username || !password}>
      {loading ? 'Входим…' : 'Войти'}
    </button>

    {#if error}
      <p class="mt-3 text-center text-xs text-danger" transition:fade={{ duration: 150 }}>{error}</p>
    {/if}
  </form>
</div>
