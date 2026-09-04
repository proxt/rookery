<script>
  import { onMount, onDestroy } from 'svelte'
  import { GetLogs } from '../../../wailsjs/go/main/App.js'

  let lines = $state([])
  let autoRefresh = $state(true)
  let timer = null

  async function load() {
    lines = await GetLogs()
  }

  onMount(() => {
    load()
    timer = setInterval(() => {
      if (autoRefresh) load()
    }, 2000)
  })
  onDestroy(() => clearInterval(timer))

  function levelClass(line) {
    if (line.includes('level=ERROR')) return 'text-state-error'
    if (line.includes('level=WARN')) return 'text-state-connecting'
    return 'text-muted'
  }

  async function copyAll() {
    await navigator.clipboard.writeText(lines.join('\n'))
  }
</script>

<div class="flex flex-1 flex-col px-6 py-5">
  <div class="mb-3 flex items-center justify-between">
    <h1 class="text-xs font-semibold uppercase tracking-widest text-muted">Логи</h1>
    <div class="flex items-center gap-3">
      <label class="flex cursor-pointer items-center gap-1.5 text-xs text-muted">
        <input type="checkbox" class="h-3.5 w-3.5 accent-[var(--color-up)]" bind:checked={autoRefresh} />
        Автообновление
      </label>
      <button class="btn-secondary" onclick={load}>Обновить</button>
      <button class="btn-secondary" onclick={copyAll} disabled={lines.length === 0}>Копировать</button>
    </div>
  </div>

  <div class="card scroll-area flex-1 overflow-y-auto p-3 font-mono text-[11px] leading-relaxed">
    {#if lines.length === 0}
      <p class="py-8 text-center text-muted">Записей пока нет</p>
    {:else}
      {#each lines as line, i (i)}
        <div class="whitespace-pre-wrap break-all {levelClass(line)}">{line}</div>
      {/each}
    {/if}
  </div>
</div>
