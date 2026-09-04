<script>
  import { fade, fly } from 'svelte/transition'
  import { api } from './api.js'
  import { formatBytes, formatDateTime, toDatetimeLocal, fromDatetimeLocal } from './format.js'
  import Pill from './Pill.svelte'
  import TrafficChart from './TrafficChart.svelte'
  import { countryFlag } from './flags.js'

  let { userId, allNodes = [], onclose, onchanged, ondeleted } = $props()

  let user = $state(null)
  let tab = $state('info')
  let saving = $state(false)
  let confirmingDelete = $state(false)

  // Info tab form state
  let name = $state('')
  let enabled = $state(true)
  let startsAt = $state('')
  let expiresAt = $state('')
  let unlimited = $state(true)

  // Nodes tab
  let selectedNodeIds = $state(new Set())

  // Stats tab
  let totals = $state(null)
  let series = $state([])
  let statsLoading = $state(true)

  async function load() {
    user = await api.getUser(userId)
    name = user.name
    enabled = user.enabled
    startsAt = toDatetimeLocal(user.starts_at)
    expiresAt = toDatetimeLocal(user.expires_at)
    unlimited = !user.starts_at && !user.expires_at
    selectedNodeIds = new Set(user.nodes.map((n) => n.id))
  }
  load()

  function onUnlimitedChange() {
    if (unlimited) {
      startsAt = ''
      expiresAt = ''
    }
  }

  async function loadStats() {
    statsLoading = true
    const [t, s] = await Promise.all([api.statsUser(userId), api.statsUserSeries(userId, 24 * 7)])
    totals = t
    series = s
    statsLoading = false
  }

  $effect(() => {
    if (tab === 'stats' && !totals) loadStats()
  })

  async function saveInfo() {
    saving = true
    try {
      await api.updateUser(userId, { name, enabled, startsAt: fromDatetimeLocal(startsAt), expiresAt: fromDatetimeLocal(expiresAt) })
      await load()
      onchanged()
    } finally {
      saving = false
    }
  }

  function toggleNode(id) {
    const next = new Set(selectedNodeIds)
    next.has(id) ? next.delete(id) : next.add(id)
    selectedNodeIds = next
  }

  async function saveNodes() {
    saving = true
    try {
      await api.setUserNodes(userId, [...selectedNodeIds])
      await load()
      onchanged()
    } finally {
      saving = false
    }
  }

  async function copyLink() {
    await navigator.clipboard.writeText(user.sub_url)
  }

  async function deleteUser() {
    await api.deleteUser(userId)
    ondeleted(userId)
  }

  function onKeydown(e) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" role="presentation" transition:fade={{ duration: 150 }} onclick={onclose}>
  <div
    class="card flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden"
    transition:fly={{ y: 16, duration: 250, easing: (t) => 1 - Math.pow(1 - t, 3) }}
    onclick={(e) => e.stopPropagation()} role="presentation"
  >
    {#if !user}
      <div class="p-10 text-center text-sm text-muted">Загрузка…</div>
    {:else}
      <div class="flex items-center justify-between border-b border-border px-5 py-4">
        <div class="flex items-center gap-3">
          <div class="flex h-9 w-9 items-center justify-center rounded-full bg-surface-2 text-sm font-semibold">
            {user.name.slice(0, 2).toUpperCase()}
          </div>
          <div>
            <div class="text-sm font-semibold">{user.name}</div>
            <div class="text-xs text-muted">создан {formatDateTime(user.created_at)}</div>
          </div>
        </div>
        <button class="icon-btn" onclick={onclose} aria-label="Закрыть">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" /></svg>
        </button>
      </div>

      <div class="flex gap-1 border-b border-border px-3 pt-2">
        {#each [['info', 'Инфо'], ['nodes', 'Ноды'], ['stats', 'Статистика']] as [id, label] (id)}
          <button
            class="relative px-3 pb-2.5 text-sm font-medium transition-colors cursor-pointer {tab === id ? 'text-text' : 'text-muted hover:text-text'}"
            onclick={() => (tab = id)}
          >
            {label}
            {#if tab === id}<span class="absolute bottom-0 left-0 right-0 h-0.5 rounded-full bg-up"></span>{/if}
          </button>
        {/each}
      </div>

      <div class="flex-1 overflow-y-auto p-5">
        {#if tab === 'info'}
          <div in:fade={{ duration: 150 }} class="space-y-4">
            <div>
              <label class="mb-1 block text-xs font-medium text-muted" for="u-name">Имя</label>
              <input id="u-name" class="input" bind:value={name} />
            </div>

            <div class="flex items-center justify-between rounded-lg border border-border bg-surface-2 px-3 py-2.5">
              <span class="text-sm">Подписка активна</span>
              <button
                class="flex h-6 w-11 shrink-0 items-center rounded-full p-0.5 transition-colors cursor-pointer {enabled ? 'justify-end bg-up' : 'justify-start bg-surface-3'}"
                onclick={() => (enabled = !enabled)}
                aria-label="Переключить"
              >
                <span class="h-5 w-5 rounded-full bg-white shadow transition-transform"></span>
              </button>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="mb-1 block text-xs font-medium text-muted" for="u-starts">Начало</label>
                <input id="u-starts" type="datetime-local" class="input" bind:value={startsAt} disabled={unlimited} />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-muted" for="u-expires">Окончание</label>
                <input id="u-expires" type="datetime-local" class="input" bind:value={expiresAt} disabled={unlimited} />
              </div>
            </div>

            <label class="flex cursor-pointer items-center gap-2 text-xs text-muted">
              <input type="checkbox" class="h-3.5 w-3.5 accent-[var(--color-up)]" bind:checked={unlimited} onchange={onUnlimitedChange} />
              Без ограничений по времени (бессрочная подписка)
            </label>

            <div>
              <label class="mb-1 block text-xs font-medium text-muted" for="u-link">Ссылка подписки</label>
              <div class="flex gap-2">
                <input id="u-link" class="input truncate" readonly value={user.sub_url} />
                <button class="btn-secondary shrink-0" onclick={copyLink}>Копировать</button>
              </div>
            </div>

            <div class="flex items-center justify-between pt-2">
              {#if confirmingDelete}
                <div class="flex items-center gap-2 text-xs" transition:fade={{ duration: 120 }}>
                  <span class="text-muted">Удалить безвозвратно?</span>
                  <button class="text-danger underline" onclick={deleteUser}>Да, удалить</button>
                  <button class="text-muted underline" onclick={() => (confirmingDelete = false)}>Отмена</button>
                </div>
              {:else}
                <button class="btn-danger" onclick={() => (confirmingDelete = true)}>Удалить пользователя</button>
              {/if}
              <button class="btn-primary" onclick={saveInfo} disabled={saving}>{saving ? 'Сохранение…' : 'Сохранить'}</button>
            </div>
          </div>
        {:else if tab === 'nodes'}
          <div in:fade={{ duration: 150 }} class="space-y-2">
            {#if allNodes.length === 0}
              <p class="py-8 text-center text-sm text-muted">Нет ни одной ноды — добавьте на вкладке «Ноды».</p>
            {:else}
              {#each allNodes as n (n.id)}
                <label class="flex cursor-pointer items-center justify-between rounded-lg border border-border bg-surface-2 px-3 py-2.5 transition-colors hover:border-up/40">
                  <div>
                    <div class="text-sm font-medium">
                      {#if countryFlag(n.tags)}<span class="mr-1">{countryFlag(n.tags)}</span>{/if}{n.name}
                    </div>
                    {#if n.tags}<div class="text-xs text-muted">{n.tags}</div>{/if}
                  </div>
                  <input type="checkbox" class="h-4 w-4 accent-[var(--color-up)]" checked={selectedNodeIds.has(n.id)} onchange={() => toggleNode(n.id)} />
                </label>
              {/each}
              <div class="flex justify-end pt-2">
                <button class="btn-primary" onclick={saveNodes} disabled={saving}>{saving ? 'Сохранение…' : 'Сохранить'}</button>
              </div>
            {/if}
          </div>
        {:else if tab === 'stats'}
          <div in:fade={{ duration: 150 }}>
            {#if statsLoading}
              <div class="py-16 text-center text-sm text-muted">Загрузка…</div>
            {:else}
              <div class="mb-4 grid grid-cols-2 gap-3">
                <div class="rounded-lg border border-border bg-surface-2 px-3 py-2.5">
                  <div class="text-xs text-muted">Загружено ↑</div>
                  <div class="text-base font-semibold text-up">{formatBytes(totals.bytes_up)}</div>
                </div>
                <div class="rounded-lg border border-border bg-surface-2 px-3 py-2.5">
                  <div class="text-xs text-muted">Скачано ↓</div>
                  <div class="text-base font-semibold text-down">{formatBytes(totals.bytes_down)}</div>
                </div>
              </div>
              <div class="mb-2 text-xs text-muted">За последние 7 дней</div>
              <TrafficChart points={series} />
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
