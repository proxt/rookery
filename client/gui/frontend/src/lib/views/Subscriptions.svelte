<script>
  import { fade, slide } from 'svelte/transition'
  import {
    AddSubscriptionFromLink,
    DeleteSubscription,
    SetActiveSubscription,
    SetActiveNode,
    RefreshSubscription,
    MeasureNodeLatencies,
  } from '../../../wailsjs/go/main/App.js'

  let { settings = { subscriptions: [], activeSubscriptionId: '' }, onchange } = $props()

  let linkInput = $state('')
  let adding = $state(false)
  let addError = $state('')
  let expandedId = $state(null)
  let refreshingId = $state(null)
  // Ping results are transient (network conditions change instantly) —
  // kept only in local UI state, never persisted to settings. Keyed by
  // subscription id, then node id -> ms (-1 means measured & unreachable;
  // absent means not measured yet).
  let pings = $state({})
  let measuringId = $state(null)

  async function measureLatencies(subId) {
    measuringId = subId
    try {
      pings[subId] = await MeasureNodeLatencies(subId)
    } catch {
      // leave whatever was there — a failed batch shouldn't wipe prior results
    } finally {
      measuringId = null
    }
  }

  async function handleAdd() {
    const link = linkInput.trim()
    if (!link) return
    adding = true
    addError = ''
    try {
      const sub = await AddSubscriptionFromLink(link)
      linkInput = ''
      expandedId = sub.id
      await onchange()
    } catch (e) {
      addError = 'Не удалось добавить подписку — проверьте ссылку и подключение к сети'
    } finally {
      adding = false
    }
  }

  async function activate(id) {
    await SetActiveSubscription(id)
    await onchange()
  }

  async function remove(id, e) {
    e.stopPropagation()
    await DeleteSubscription(id)
    await onchange()
  }

  function toggleExpand(id, e) {
    e.stopPropagation()
    const opening = expandedId !== id
    expandedId = opening ? id : null
    if (opening) measureLatencies(id)
  }

  async function pickNode(subId, nodeId) {
    await SetActiveNode(subId, nodeId)
    await onchange()
  }

  async function refresh(id, e) {
    e.stopPropagation()
    refreshingId = id
    try {
      await RefreshSubscription(id)
      await onchange()
    } finally {
      refreshingId = null
    }
  }
</script>

<div class="flex flex-1 flex-col overflow-hidden px-6 py-5">
  <h1 class="mb-4 text-xs font-semibold uppercase tracking-widest text-muted">Подписки</h1>

  <div class="mb-4 flex-1 overflow-y-auto">
    {#if settings.subscriptions.length === 0}
      <div class="card p-6 text-center text-xs text-muted">
        Пока нет ни одной подписки — вставьте ссылку rookery://sub/… ниже
      </div>
    {:else}
      <div class="space-y-2">
        {#each settings.subscriptions as sub (sub.id)}
          {@const isActive = sub.id === settings.activeSubscriptionId}
          {@const activeNode = sub.nodes.find((n) => n.id === sub.activeNodeId) ?? sub.nodes[0]}
          <div class="card overflow-hidden" transition:fade={{ duration: 150 }}>
            <div class="flex w-full items-center justify-between gap-2 p-3">
              <button
                class="flex min-w-0 flex-1 items-center gap-2.5 text-left cursor-pointer"
                onclick={() => activate(sub.id)}
              >
                <span
                  class="h-2.5 w-2.5 shrink-0 rounded-full {isActive
                    ? 'bg-state-connected shadow-[0_0_6px_var(--color-state-connected)]'
                    : 'border border-border'}"
                ></span>
                <span class="min-w-0">
                  <div class="truncate text-sm font-medium">{sub.name}</div>
                  <div class="truncate text-xs text-muted">
                    {activeNode ? activeNode.name : 'нет серверов'} · {sub.nodes.length} серв.
                  </div>
                </span>
              </button>
              <span class="flex shrink-0 items-center gap-0.5">
                <button
                  class="rounded-lg p-1.5 text-muted transition-colors hover:text-text cursor-pointer {refreshingId === sub.id ? 'animate-spin' : ''}"
                  onclick={(e) => refresh(sub.id, e)}
                  aria-label="Обновить список серверов"
                >
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none">
                    <path d="M4 4v5h5M20 20v-5h-5M4.5 9a8 8 0 0 1 14.6-3M19.5 15a8 8 0 0 1-14.6 3"
                      stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
                <button
                  class="rounded-lg p-1.5 text-muted transition-colors cursor-pointer {expandedId === sub.id ? 'text-up' : 'hover:text-text'}"
                  onclick={(e) => toggleExpand(sub.id, e)}
                  aria-label="Выбрать сервер"
                >
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" style="transform: rotate({expandedId === sub.id ? 180 : 0}deg); transition: transform .2s">
                    <path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
                <button
                  class="rounded-lg p-1.5 text-muted transition-colors hover:text-state-error cursor-pointer"
                  onclick={(e) => remove(sub.id, e)}
                  aria-label="Удалить подписку"
                >
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none">
                    <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2m-8 0 1 13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1l1-13"
                      stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
              </span>
            </div>

            {#if expandedId === sub.id}
              <div class="border-t border-border px-3 py-2" transition:slide={{ duration: 180 }}>
                {#if sub.nodes.length === 0}
                  <p class="py-2 text-center text-xs text-muted">Нет доступных серверов</p>
                {:else}
                  <div class="space-y-1">
                    {#each sub.nodes as node (node.id)}
                      {@const picked = node.id === (sub.activeNodeId || sub.nodes[0]?.id)}
                      {@const ms = pings[sub.id]?.[node.id]}
                      <button
                        class="flex w-full items-center justify-between rounded-lg px-2.5 py-2 text-left text-xs transition-colors cursor-pointer {picked ? 'bg-up/10 text-text' : 'text-muted hover:bg-surface-2'}"
                        onclick={() => pickNode(sub.id, node.id)}
                      >
                        <span class="flex items-center gap-2">
                          <span class="h-1.5 w-1.5 shrink-0 rounded-full {picked ? 'bg-up' : 'border border-border'}"></span>
                          {node.name}
                          {#if node.tags}<span class="text-muted">· {node.tags}</span>{/if}
                          {#if ms !== undefined}
                            {#if ms < 0}
                              <span class="text-state-error">· недоступна</span>
                            {:else}
                              <span class="text-muted">· {ms} мс</span>
                            {/if}
                          {:else if measuringId === sub.id}
                            <span class="text-muted">· …</span>
                          {/if}
                        </span>
                        {#if picked}<span class="text-up">✓</span>{/if}
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <div class="shrink-0">
    <span class="mb-1 block text-xs text-muted">Ссылка подписки (rookery://sub/…)</span>
    <div class="flex gap-2">
      <input
        class="input"
        type="text"
        placeholder="rookery://sub/…"
        autocomplete="off"
        bind:value={linkInput}
        onkeydown={(e) => e.key === 'Enter' && handleAdd()}
      />
      <button class="btn-primary shrink-0" onclick={handleAdd} disabled={adding}>
        Добавить
      </button>
    </div>
    {#if addError}
      <p class="mt-2 text-xs text-state-error" transition:fade={{ duration: 150 }}>{addError}</p>
    {/if}
  </div>
</div>
