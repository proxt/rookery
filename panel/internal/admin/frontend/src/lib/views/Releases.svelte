<script>
  import { fade } from 'svelte/transition'
  import { flip } from 'svelte/animate'
  import { api } from '../api.js'
  import { formatBytes, formatDateTime } from '../format.js'
  import Pill from '../Pill.svelte'

  let releases = $state([])
  let loading = $state(true)

  let version = $state('')
  let notes = $state('')
  let fileInput = $state(null)
  let selectedFile = $state(null)
  let uploading = $state(false)
  let uploadProgress = $state(0)
  let uploadError = $state('')

  async function load() {
    loading = true
    releases = (await api.listReleases()) ?? []
    loading = false
  }
  load()

  function onFileChange(e) {
    selectedFile = e.target.files?.[0] ?? null
  }

  async function upload() {
    uploadError = ''
    if (!version.trim() || !selectedFile) {
      uploadError = 'Укажите версию и выберите файл'
      return
    }
    uploading = true
    uploadProgress = 0
    try {
      await api.uploadRelease(version.trim(), notes.trim(), selectedFile, (p) => (uploadProgress = p))
      version = ''
      notes = ''
      selectedFile = null
      if (fileInput) fileInput.value = ''
      await load()
    } catch (e) {
      uploadError = 'Не удалось загрузить файл'
    } finally {
      uploading = false
    }
  }

  async function remove(id) {
    await api.deleteRelease(id).catch(() => {})
    await load()
  }
</script>

<div class="mx-auto max-w-3xl">
  <h1 class="mb-6 text-lg font-semibold">Обновления</h1>

  <div class="card fade-in-up mb-4 p-5">
    <h2 class="mb-1 text-sm font-semibold">Загрузить новую версию</h2>
    <p class="mb-4 text-xs text-muted">Клиент проверяет эту версию при запросе обновлений; последняя загруженная — всегда актуальная.</p>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="rel-version">Версия</label>
        <input id="rel-version" class="input" placeholder="0.3.0" bind:value={version} />
      </div>
      <div>
        <label class="mb-1 block text-xs font-medium text-muted" for="rel-file">Файл установщика</label>
        <input id="rel-file" bind:this={fileInput} type="file" class="input" onchange={onFileChange} />
      </div>
    </div>
    <div class="mt-3">
      <label class="mb-1 block text-xs font-medium text-muted" for="rel-notes">Заметки (необязательно)</label>
      <textarea id="rel-notes" class="input" rows="2" bind:value={notes}></textarea>
    </div>

    {#if uploading}
      <div class="mt-3 h-1.5 overflow-hidden rounded-full bg-surface-2">
        <div class="h-full bg-up transition-all duration-150" style="width: {Math.round(uploadProgress * 100)}%"></div>
      </div>
    {/if}
    {#if uploadError}<p class="mt-2 text-xs text-danger" transition:fade>{uploadError}</p>{/if}

    <div class="mt-3 flex justify-end">
      <button class="btn-primary" onclick={upload} disabled={uploading}>
        {uploading ? `Загрузка… ${Math.round(uploadProgress * 100)}%` : 'Загрузить'}
      </button>
    </div>
  </div>

  {#if loading}
    <div class="py-8 text-center text-sm text-muted">Загрузка…</div>
  {:else if releases.length === 0}
    <div class="card fade-in-up py-12 text-center text-sm text-muted">Пока нет ни одного загруженного релиза</div>
  {:else}
    <div class="space-y-2">
      {#each releases as rel, i (rel.id)}
        <div class="card fade-in-up flex items-center justify-between p-4" style="animation-delay: {i * 40}ms" animate:flip={{ duration: 200 }} transition:fade={{ duration: 150 }}>
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="text-sm font-semibold">v{rel.version}</span>
              {#if i === 0}<Pill tone="ok">актуальная</Pill>{/if}
            </div>
            {#if rel.notes}<p class="mt-0.5 truncate text-xs text-muted">{rel.notes}</p>{/if}
            <p class="mt-1 text-[11px] text-muted">{rel.filename} · {formatBytes(rel.size)} · {formatDateTime(rel.created_at)}</p>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <a class="btn-secondary" href={rel.url} download>Скачать</a>
            <button class="icon-btn hover:text-danger" onclick={() => remove(rel.id)} aria-label="Удалить релиз">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none">
                <path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2m-8 0 1 13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1l1-13"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
