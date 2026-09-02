<script>
  import { fade, fly } from 'svelte/transition'
  import { api } from './api.js'
  import { formatBytes, relativeTime } from './format.js'
  import Pill from './Pill.svelte'

  // node: existing store.Node-shaped object, or null to create.
  let { node = null, onclose, onsaved, ondeleted } = $props()

  let name = $state(node?.name ?? '')
  let address = $state(node?.address ?? '')
  let tags = $state(node?.tags ?? '')
  let enabled = $state(node?.enabled ?? true)
  let saving = $state(false)
  let confirmingDelete = $state(false)
  let totals = $state(null)

  if (node) api.statsNode(node.id).then((t) => (totals = t))

  const online = $derived(node?.last_seen_at && Date.now() - new Date(node.last_seen_at).getTime() < 120000)

  async function save() {
    if (!name.trim() || !address.trim()) return
    saving = true
    try {
      if (node) {
        await api.updateNode(node.id, { name: name.trim(), address: address.trim(), tags: tags.trim(), enabled })
      } else {
        await api.createNode(name.trim(), address.trim(), tags.trim())
      }
      onsaved()
    } finally {
      saving = false
    }
  }

  async function del() {
    await api.deleteNode(node.id)
    ondeleted(node.id)
  }

  async function copyKey() {
    await navigator.clipboard.writeText(node.api_key)
  }

  function onKeydown(e) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" role="presentation" transition:fade={{ duration: 150 }} onclick={onclose}>
  <div
    class="card w-full max-w-md p-5"
    transition:fly={{ y: 16, duration: 250, easing: (t) => 1 - Math.pow(1 - t, 3) }}
    onclick={(e) => e.stopPropagation()} role="presentation"
  >
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-sm font-semibold">{node ? 'Нода' : 'Новая нода'}</h2>
      {#if node}<Pill tone={online ? 'ok' : 'muted'}>{online ? 'online' : 'offline'}</Pill>{/if}
    </div>

    <div class="space-y-3">
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="n-name">Название</label>
        <input id="n-name" class="input" bind:value={name} placeholder="de-frankfurt-1" />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="n-addr">Публичный адрес</label>
        <input id="n-addr" class="input" bind:value={address} placeholder="https://node.example.com" />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="n-tags">Теги (страна, метка)</label>
        <input id="n-tags" class="input" bind:value={tags} placeholder="de, cheap" />
      </div>

      {#if node}
        <div class="flex items-center justify-between rounded-lg border border-border bg-surface-2 px-3 py-2.5">
          <span class="text-sm">Включена</span>
          <button class="relative h-6 w-11 rounded-full transition-colors cursor-pointer {enabled ? 'bg-up' : 'bg-surface-3'}" onclick={() => (enabled = !enabled)} aria-label="Переключить">
            <span class="absolute top-0.5 h-5 w-5 rounded-full bg-white transition-transform {enabled ? 'translate-x-5' : 'translate-x-0.5'}"></span>
          </button>
        </div>

        <div>
          <label class="mb-1 block text-xs font-medium text-muted" for="n-key">API-ключ (для node.yaml)</label>
          <div class="flex gap-2">
            <input id="n-key" class="input truncate font-mono text-xs" readonly value={node.api_key} />
            <button class="btn-secondary shrink-0" onclick={copyKey}>Копировать</button>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3 text-xs">
          <div class="rounded-lg border border-border bg-surface-2 px-3 py-2">
            <div class="text-muted">Последний heartbeat</div>
            <div class="mt-0.5 font-medium">{relativeTime(node.last_seen_at)}</div>
          </div>
          <div class="rounded-lg border border-border bg-surface-2 px-3 py-2">
            <div class="text-muted">Трафик всего</div>
            <div class="mt-0.5 font-medium">{totals ? formatBytes(totals.bytes_up + totals.bytes_down) : '…'}</div>
          </div>
        </div>
      {/if}
    </div>

    <div class="mt-5 flex items-center justify-between">
      {#if node}
        {#if confirmingDelete}
          <div class="flex items-center gap-2 text-xs">
            <span class="text-muted">Удалить безвозвратно?</span>
            <button class="text-danger underline" onclick={del}>Да, удалить</button>
            <button class="text-muted underline" onclick={() => (confirmingDelete = false)}>Отмена</button>
          </div>
        {:else}
          <button class="btn-danger" onclick={() => (confirmingDelete = true)}>Удалить</button>
        {/if}
      {:else}
        <span></span>
      {/if}
      <button class="btn-primary" onclick={save} disabled={saving || !name.trim() || !address.trim()}>
        {saving ? 'Сохранение…' : node ? 'Сохранить' : 'Создать'}
      </button>
    </div>
  </div>
</div>
