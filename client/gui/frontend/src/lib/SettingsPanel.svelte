<script>
  import { fade, fly } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'

  let { open = false, settings = {}, onsave, onclose } = $props()

  let nodeAddr = $state('')
  let socksPort = $state(1080)
  let secret = $state('')
  let autoStart = $state(false)
  let startMinimized = $state(false)
  let saving = $state(false)
  let error = $state('')

  $effect(() => {
    if (open) {
      nodeAddr = settings.nodeAddr ?? ''
      socksPort = settings.socksPort ?? 1080
      secret = settings.secret ?? ''
      autoStart = settings.autoStart ?? false
      startMinimized = settings.startMinimized ?? false
      error = ''
    }
  })

  function handleKeydown(e) {
    if (open && e.key === 'Escape') onclose()
  }

  async function handleSave() {
    saving = true
    error = ''
    try {
      await onsave({
        nodeAddr,
        socksPort: Number(socksPort),
        secret,
        autoStart,
        startMinimized,
      })
      onclose()
    } catch (e) {
      error = String(e)
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

      <label class="mb-3 block">
        <span class="mb-1 block text-xs text-muted">Адрес ноды</span>
        <input
          class="input"
          type="text"
          placeholder="https://node.example.com"
          autocomplete="off"
          bind:value={nodeAddr}
        />
      </label>

      <label class="mb-3 block">
        <span class="mb-1 block text-xs text-muted">Порт SOCKS</span>
        <input class="input" type="number" min="1" max="65535" bind:value={socksPort} />
      </label>

      <label class="mb-4 block">
        <span class="mb-1 block text-xs text-muted">Секрет</span>
        <input class="input" type="password" autocomplete="off" bind:value={secret} />
      </label>

      <label class="mb-2 flex items-center justify-between">
        <span class="text-sm">Автозапуск при входе в систему</span>
        <input type="checkbox" class="h-4 w-4 accent-up" bind:checked={autoStart} />
      </label>

      <label class="mb-4 flex items-center justify-between">
        <span class="text-sm">Запускать свёрнутым</span>
        <input type="checkbox" class="h-4 w-4 accent-up" bind:checked={startMinimized} />
      </label>

      {#if error}
        <p class="mb-3 text-xs text-state-error" transition:fade={{ duration: 150 }}>{error}</p>
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
