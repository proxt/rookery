<script>
  import { flip } from 'svelte/animate'
  import { fade, slide } from 'svelte/transition'
  import { api } from '../api.js'

  const TYPE_LABEL = { domain: 'Домен', app: 'Приложение', geoip: 'Страна (GeoIP)' }
  const VALUE_PLACEHOLDER = { domain: 'example.com', app: 'chrome.exe', geoip: 'RU' }

  let sets = $state([])
  let loading = $state(true)
  let newName = $state('')
  let creating = $state(false)
  let expandedId = $state(null)

  // Local edit buffers, keyed by set id — only populated for the
  // currently-expanded set, discarded (not saved) on collapse without
  // an explicit save.
  let editName = $state('')
  let editRules = $state([])
  let saving = $state(false)

  async function load() {
    loading = true
    sets = (await api.listRoutingRuleSets()) ?? []
    loading = false
  }
  load()

  async function createSet() {
    if (!newName.trim()) return
    creating = true
    try {
      const rs = await api.createRoutingRuleSet(newName.trim())
      newName = ''
      sets = [rs, ...sets]
      expand(rs)
    } finally {
      creating = false
    }
  }

  function expand(rs) {
    expandedId = rs.id
    editName = rs.name
    editRules = rs.rules.map((r) => ({ ...r }))
  }

  function toggleExpand(rs) {
    expandedId === rs.id ? (expandedId = null) : expand(rs)
  }

  function addRule() {
    editRules = [...editRules, { id: '', type: 'domain', value: '', action: 'direct' }]
  }
  function removeRule(i) {
    editRules = editRules.filter((_, idx) => idx !== i)
  }

  async function saveSet(id) {
    saving = true
    try {
      await api.updateRoutingRuleSet(id, editName.trim() || 'Без названия', editRules.filter((r) => r.value.trim()))
      await load()
    } finally {
      saving = false
    }
  }

  async function deleteSet(id, e) {
    e.stopPropagation()
    await api.deleteRoutingRuleSet(id).catch(() => {})
    if (expandedId === id) expandedId = null
    await load()
  }
</script>

<div class="mx-auto max-w-3xl">
  <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-lg font-semibold">Маршрутизация</h1>
      <p class="mt-0.5 text-xs text-muted">Наборы правил (домены/приложения/страны) — назначаются пользователю и приезжают вместе с подпиской.</p>
    </div>
    <div class="flex gap-2">
      <input class="input w-56" placeholder="Название нового набора" bind:value={newName}
        onkeydown={(e) => e.key === 'Enter' && createSet()} />
      <button class="btn-primary shrink-0" onclick={createSet} disabled={creating || !newName.trim()}>+ Создать</button>
    </div>
  </div>

  {#if loading}
    <div class="py-16 text-center text-sm text-muted">Загрузка…</div>
  {:else if sets.length === 0}
    <div class="card fade-in-up py-16 text-center text-sm text-muted">Пока нет ни одного набора правил</div>
  {:else}
    <div class="space-y-2">
      {#each sets as rs (rs.id)}
        <div class="card overflow-hidden" animate:flip={{ duration: 200 }} transition:fade={{ duration: 150 }}>
          <div class="flex w-full items-center justify-between p-3 text-left cursor-pointer" role="button" tabindex="0"
            onclick={() => toggleExpand(rs)} onkeydown={(e) => e.key === 'Enter' && toggleExpand(rs)}>
            <div class="min-w-0">
              <div class="truncate text-sm font-medium">{rs.name}</div>
              <div class="text-xs text-muted">{rs.rules.length} правил</div>
            </div>
            <div class="flex shrink-0 items-center gap-1">
              <button class="icon-btn hover:text-danger" onclick={(e) => deleteSet(rs.id, e)} aria-label="Удалить набор">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none">
                  <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2m-8 0 1 13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1l1-13"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </button>
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" style="transform: rotate({expandedId === rs.id ? 180 : 0}deg); transition: transform .2s">
                <path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </div>
          </div>

          {#if expandedId === rs.id}
            <div class="border-t border-border p-3" transition:slide={{ duration: 180 }}>
              <label class="mb-3 block">
                <span class="mb-1 block text-xs text-muted">Название</span>
                <input class="input" bind:value={editName} />
              </label>

              <div class="space-y-2">
                {#each editRules as rule, i}
                  <div class="flex items-center gap-2">
                    <select class="input w-36" bind:value={rule.type}>
                      <option value="domain">Домен</option>
                      <option value="app">Приложение</option>
                      <option value="geoip">Страна (GeoIP)</option>
                    </select>
                    <input class="input flex-1" placeholder={VALUE_PLACEHOLDER[rule.type]} bind:value={rule.value} />
                    <select class="input w-32" bind:value={rule.action}>
                      <option value="direct">Напрямую</option>
                      <option value="proxy">Через туннель</option>
                    </select>
                    <button class="icon-btn hover:text-danger shrink-0" onclick={() => removeRule(i)} aria-label="Удалить правило">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none">
                        <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
                      </svg>
                    </button>
                  </div>
                {/each}
              </div>

              <div class="mt-3 flex items-center justify-between">
                <button class="btn-secondary" onclick={addRule}>+ Правило</button>
                <button class="btn-primary" onclick={() => saveSet(rs.id)} disabled={saving}>
                  {saving ? 'Сохранение…' : 'Сохранить'}
                </button>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
