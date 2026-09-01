<script>
  import { fade } from 'svelte/transition'
  import { AddProfileFromLink, DeleteProfile, SetActiveProfile } from '../../../wailsjs/go/main/App.js'

  let { settings = { profiles: [], activeProfileId: '' }, onchange } = $props()

  let linkInput = $state('')
  let adding = $state(false)
  let addError = $state('')

  async function handleAdd() {
    const link = linkInput.trim()
    if (!link) return
    adding = true
    addError = ''
    try {
      await AddProfileFromLink(link)
      linkInput = ''
      await onchange()
    } catch (e) {
      addError = 'Не удалось разобрать ссылку — проверьте, что она скопирована целиком'
    } finally {
      adding = false
    }
  }

  async function activate(id) {
    await SetActiveProfile(id)
    await onchange()
  }

  async function remove(id) {
    await DeleteProfile(id)
    await onchange()
  }
</script>

<div class="flex flex-1 flex-col overflow-hidden px-6 py-5">
  <h1 class="mb-4 text-xs font-semibold uppercase tracking-widest text-muted">Профили</h1>

  <div class="mb-4 flex-1 overflow-y-auto">
    {#if settings.profiles.length === 0}
      <div class="card p-6 text-center text-xs text-muted">
        Пока нет ни одного профиля — вставьте ссылку из панели администратора ноды ниже
      </div>
    {:else}
      <div class="space-y-2">
        {#each settings.profiles as p (p.id)}
          {@const isActive = p.id === settings.activeProfileId}
          <div
            class="card flex items-center justify-between gap-2 p-3"
            transition:fade={{ duration: 150 }}
          >
            <button
              class="flex min-w-0 flex-1 items-center gap-2.5 text-left cursor-pointer"
              onclick={() => activate(p.id)}
            >
              <span
                class="h-2.5 w-2.5 shrink-0 rounded-full {isActive
                  ? 'bg-state-connected shadow-[0_0_6px_var(--color-state-connected)]'
                  : 'border border-border'}"
              ></span>
              <span class="min-w-0">
                <div class="truncate text-sm font-medium">{p.name}</div>
                <div class="truncate text-xs text-muted">{p.nodeAddr}</div>
              </span>
            </button>
            <button
              class="shrink-0 rounded-lg p-1.5 text-muted transition-colors hover:text-state-error cursor-pointer"
              onclick={() => remove(p.id)}
              aria-label="Удалить профиль"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
                <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2m-8 0 1 13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1l1-13"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <div class="shrink-0">
    <span class="mb-1 block text-xs text-muted">Ссылка профиля (rookery://…)</span>
    <div class="flex gap-2">
      <input
        class="input"
        type="text"
        placeholder="rookery://…"
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
