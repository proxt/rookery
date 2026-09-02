<script>
  import { flip } from 'svelte/animate'
  import { fade } from 'svelte/transition'
  import { api } from '../api.js'
  import { relativeTime } from '../format.js'
  import Pill from '../Pill.svelte'
  import NodeModal from '../NodeModal.svelte'

  let nodes = $state([])
  let loading = $state(true)
  let openNode = $state(null) // 'new' | node object | null

  async function load() {
    loading = true
    nodes = (await api.listNodes()) ?? []
    loading = false
  }
  load()

  function isOnline(n) {
    return n.last_seen_at && Date.now() - new Date(n.last_seen_at).getTime() < 120000
  }

  async function onsaved() {
    openNode = null
    await load()
  }
  async function ondeleted(id) {
    openNode = null
    nodes = nodes.filter((n) => n.id !== id)
  }
</script>

<div class="mx-auto max-w-5xl">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-lg font-semibold">Ноды</h1>
    <button class="btn-primary" onclick={() => (openNode = 'new')}>+ Добавить ноду</button>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-muted">Загрузка…</div>
  {:else if nodes.length === 0}
    <div class="card fade-in-up py-16 text-center text-sm text-muted">
      Пока нет ни одной ноды. Добавьте её здесь, затем впишите выданные <code>node_id</code>/<code>api_key</code> в конфиг relay-сервера.
    </div>
  {:else}
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      {#each nodes as n (n.id)}
        <button
          class="card fade-in-up flex items-center justify-between p-4 text-left transition-colors hover:border-up/40 cursor-pointer"
          animate:flip={{ duration: 200 }}
          transition:fade={{ duration: 150 }}
          onclick={() => (openNode = n)}
        >
          <div class="min-w-0">
            <div class="truncate text-sm font-semibold">{n.name}</div>
            <div class="truncate text-xs text-muted">{n.address}</div>
            {#if n.tags}<div class="mt-1 text-[11px] text-muted">{n.tags}</div>{/if}
          </div>
          <div class="ml-3 flex shrink-0 flex-col items-end gap-1.5">
            <Pill tone={isOnline(n) ? 'ok' : 'muted'}>{isOnline(n) ? 'online' : 'offline'}</Pill>
            <span class="text-[10px] text-muted">{relativeTime(n.last_seen_at)}</span>
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>

{#if openNode}
  {#key openNode === 'new' ? 'new' : openNode.id}
    <NodeModal node={openNode === 'new' ? null : openNode} onclose={() => (openNode = null)} {onsaved} {ondeleted} />
  {/key}
{/if}
