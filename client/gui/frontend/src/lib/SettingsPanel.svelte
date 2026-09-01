<script>
  import { fade, fly } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { ParseLink } from '../../wailsjs/go/main/App.js'

  let { open = false, settings = {}, onsave, onclose } = $props()

  let profileName = $state('')
  let nodeAddr = $state('')
  let userId = $state('')
  let secret = $state('')
  let socksPort = $state(1080)
  let autoStart = $state(false)
  let startMinimized = $state(false)

  let linkInput = $state('')
  let importError = $state('')
  let saveError = $state('')
  let saving = $state(false)
  let importing = $state(false)

  const hasProfile = $derived(!!nodeAddr && !!userId && !!secret)

  $effect(() => {
    if (open) {
      profileName = settings.profileName ?? ''
      nodeAddr = settings.nodeAddr ?? ''
      userId = settings.userId ?? ''
      secret = settings.secret ?? ''
      socksPort = settings.socksPort ?? 1080
      autoStart = settings.autoStart ?? false
      startMinimized = settings.startMinimized ?? false
      linkInput = ''
      importError = ''
      saveError = ''
    }
  })

  function handleKeydown(e) {
    if (open && e.key === 'Escape') onclose()
  }

  async function handleImport() {
    const link = linkInput.trim()
    if (!link) return
    importError = ''
    importing = true
    try {
      const parsed = await ParseLink(link)
      profileName = parsed.profileName
      nodeAddr = parsed.nodeAddr
      userId = parsed.userId
      secret = parsed.secret
      linkInput = ''
    } catch (e) {
      importError = 'Не удалось разобрать ссылку — проверьте, что она скопирована целиком'
    } finally {
      importing = false
    }
  }

  async function handleSave() {
    saving = true
    saveError = ''
    try {
      await onsave({
        profileName,
        nodeAddr,
        socksPort: Number(socksPort),
        userId,
        secret,
        autoStart,
        startMinimized,
      })
      onclose()
    } catch (e) {
      saveError = String(e)
    } finally {
      saving = false
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-10 flex items-center justify-center bg-black/50 backdrop-blur-sm"
    role="presentation"
    onclick={onclose}
    transition:fade={{ duration: 180 }}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="card w-80 p-5 shadow-2xl"
      role="presentation"
      onclick={(e) => e.stopPropagation()}
      transition:fly={{ y: 16, duration: 220, easing: cubicOut }}
    >
      <h2 class="mb-4 text-xs font-semibold uppercase tracking-widest text-muted">
        Настройки
      </h2>

      <div class="mb-3">
        <span class="mb-1 block text-xs text-muted">Профиль</span>
        {#if hasProfile}
          <div class="rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm">
            <div class="font-medium">{profileName || 'Без названия'}</div>
            <div class="truncate text-xs text-muted">{nodeAddr}</div>
          </div>
        {:else}
          <div class="rounded-lg border border-dashed border-border px-3 py-2 text-xs text-muted">
            Профиль ещё не добавлен — вставьте ссылку ниже
          </div>
        {/if}
      </div>

      <label class="mb-4 block">
        <span class="mb-1 block text-xs text-muted">Ссылка профиля (rookery://…)</span>
        <div class="flex gap-2">
          <input
            class="input"
            type="text"
            placeholder="rookery://…"
            autocomplete="off"
            bind:value={linkInput}
          />
          <button class="btn-secondary shrink-0" onclick={handleImport} disabled={importing}>
            Вставить
          </button>
        </div>
        {#if importError}
          <p class="mt-2 text-xs text-state-error" transition:fade={{ duration: 150 }}>{importError}</p>
        {/if}
      </label>

      <label class="mb-4 block">
        <span class="mb-1 block text-xs text-muted">Порт SOCKS</span>
        <input class="input" type="number" min="1" max="65535" bind:value={socksPort} />
      </label>

      <label class="mb-2 flex items-center justify-between">
        <span class="text-sm">Автозапуск при входе в систему</span>
        <input type="checkbox" class="h-4 w-4 accent-up" bind:checked={autoStart} />
      </label>

      <label class="mb-4 flex items-center justify-between">
        <span class="text-sm">Запускать свёрнутым</span>
        <input type="checkbox" class="h-4 w-4 accent-up" bind:checked={startMinimized} />
      </label>

      {#if saveError}
        <p class="mb-3 text-xs text-state-error">{saveError}</p>
      {/if}

      <div class="flex justify-end gap-2">
        <button class="btn-secondary" onclick={onclose}>Отмена</button>
        <button class="btn-primary" onclick={handleSave} disabled={saving}>
          {saving ? 'Сохранение…' : 'Сохранить'}
        </button>
      </div>
    </div>
  </div>
{/if}
