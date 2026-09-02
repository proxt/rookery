<script>
  import { flip } from 'svelte/animate'
  import { fade } from 'svelte/transition'
  import { api } from '../api.js'
  import { formatDateTime, relativeTime } from '../format.js'
  import Pill from '../Pill.svelte'
  import UserModal from '../UserModal.svelte'

  const ONLINE_WINDOW_MS = 2 * 60 * 1000

  let users = $state([])
  let nodes = $state([])
  let loading = $state(true)
  let newName = $state('')
  let creating = $state(false)
  let openUserId = $state(null)
  let search = $state('')

  async function load() {
    loading = true
    const [u, n] = await Promise.all([api.listUsers(), api.listNodes()])
    users = u ?? []
    nodes = n ?? []
    loading = false
  }
  load()

  const filtered = $derived(
    search.trim() ? users.filter((u) => u.name.toLowerCase().includes(search.trim().toLowerCase())) : users
  )

  function statusOf(u) {
    if (!u.enabled) return { tone: 'muted', label: 'Отключён' }
    if (u.expires_at && new Date(u.expires_at) < new Date()) return { tone: 'danger', label: 'Истёк' }
    if (u.starts_at && new Date(u.starts_at) > new Date()) return { tone: 'warn', label: 'Ожидает' }
    return { tone: 'ok', label: 'Активен' }
  }

  function isOnline(u) {
    return u.last_active_at && Date.now() - new Date(u.last_active_at).getTime() < ONLINE_WINDOW_MS
  }

  function expiryOf(u) {
    if (!u.expires_at) return 'Бессрочно'
    return formatDateTime(u.expires_at)
  }

  async function createUser() {
    if (!newName.trim()) return
    creating = true
    try {
      const u = await api.createUser(newName.trim())
      newName = ''
      users = [u, ...users]
      openUserId = u.id
    } finally {
      creating = false
    }
  }

  async function copyLink(u, e) {
    e.stopPropagation()
    await navigator.clipboard.writeText(u.sub_url)
  }

  async function onUserChanged() {
    await load()
  }

  async function onUserDeleted(id) {
    users = users.filter((u) => u.id !== id)
    openUserId = null
  }
</script>

<div class="mx-auto max-w-5xl">
  <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
    <h1 class="text-lg font-semibold">Пользователи</h1>
    <div class="flex gap-2">
      <input class="input w-48" placeholder="Поиск…" bind:value={search} />
      <input class="input w-48" placeholder="Имя нового пользователя" bind:value={newName}
        onkeydown={(e) => e.key === 'Enter' && createUser()} />
      <button class="btn-primary shrink-0" onclick={createUser} disabled={creating || !newName.trim()}>
        + Добавить
      </button>
    </div>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-muted">Загрузка…</div>
  {:else if filtered.length === 0}
    <div class="card fade-in-up py-16 text-center text-sm text-muted">
      {users.length === 0 ? 'Пока нет ни одного пользователя' : 'Ничего не найдено'}
    </div>
  {:else}
    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border text-left text-xs uppercase tracking-wide text-muted">
            <th class="px-4 py-3 font-medium">Имя</th>
            <th class="px-4 py-3 font-medium">Статус</th>
            <th class="px-4 py-3 font-medium">Онлайн</th>
            <th class="px-4 py-3 font-medium">Ноды</th>
            <th class="px-4 py-3 font-medium">Окончание</th>
            <th class="px-4 py-3 font-medium"></th>
          </tr>
        </thead>
        <tbody>
          {#each filtered as u (u.id)}
            {@const status = statusOf(u)}
            {@const online = isOnline(u)}
            <tr
              class="cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-surface-2/60"
              onclick={() => (openUserId = u.id)}
              animate:flip={{ duration: 200 }}
              transition:fade={{ duration: 150 }}
            >
              <td class="px-4 py-3 font-medium">{u.name}</td>
              <td class="px-4 py-3"><Pill tone={status.tone}>{status.label}</Pill></td>
              <td class="px-4 py-3">
                <Pill tone={online ? 'ok' : 'muted'}>{online ? 'В сети' : 'Не в сети'}</Pill>
                {#if !online && u.last_active_at}
                  <div class="mt-0.5 text-[11px] text-muted">{relativeTime(u.last_active_at)}</div>
                {/if}
              </td>
              <td class="px-4 py-3 text-muted">
                {u.nodes.length === 0 ? '—' : u.nodes.length === 1 ? u.nodes[0].name : `${u.nodes.length} нод`}
              </td>
              <td class="px-4 py-3 text-muted">{expiryOf(u)}</td>
              <td class="px-4 py-3 text-right">
                <button class="icon-btn" title="Скопировать ссылку" onclick={(e) => copyLink(u, e)}>
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none">
                    <rect x="9" y="9" width="12" height="12" rx="2" stroke="currentColor" stroke-width="2" />
                    <path d="M5 15V5a2 2 0 0 1 2-2h10" stroke="currentColor" stroke-width="2" />
                  </svg>
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if openUserId}
  {#key openUserId}
    <UserModal userId={openUserId} allNodes={nodes} onclose={() => (openUserId = null)} onchanged={onUserChanged} ondeleted={onUserDeleted} />
  {/key}
{/if}
