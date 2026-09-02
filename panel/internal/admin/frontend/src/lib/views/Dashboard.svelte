<script>
  import { onMount } from 'svelte'
  import { api } from '../api.js'
  import { formatBytes } from '../format.js'
  import StatTile from '../StatTile.svelte'
  import TrafficChart from '../TrafficChart.svelte'

  let overview = $state(null)
  let series = $state([])
  let rangeHours = $state(24)
  let loading = $state(true)

  const ranges = [
    { hours: 24, label: '24ч' },
    { hours: 24 * 7, label: '7д' },
    { hours: 24 * 30, label: '30д' },
  ]

  async function load() {
    loading = true
    const [o, s] = await Promise.all([api.statsOverview(), api.statsTimeSeries(rangeHours)])
    overview = o
    series = s
    loading = false
  }

  $effect(() => {
    rangeHours
    load()
  })

  onMount(() => {
    const t = setInterval(() => api.statsOverview().then((o) => (overview = o)).catch(() => {}), 15000)
    return () => clearInterval(t)
  })
</script>

<div class="mx-auto max-w-5xl">
  <h1 class="mb-6 text-lg font-semibold">Обзор</h1>

  {#if overview}
    <div class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      <StatTile label="Пользователи" value={overview.user_count} icon="👤" accent="up" delay={0} />
      <StatTile label="Ноды онлайн" value="{overview.nodes_online} / {overview.node_count}" icon="🖥️" accent="ok" delay={40} />
      <StatTile label="Трафик за 24ч" value={formatBytes(overview.today.bytes_up + overview.today.bytes_down)} icon="📈" accent="accent-2" delay={80} />
      <StatTile label="Трафик всего" value={formatBytes(overview.all_time.bytes_up + overview.all_time.bytes_down)} icon="🗄️" accent="warn" delay={120} />
    </div>
  {/if}

  <div class="card fade-in-up p-5" style="animation-delay: 160ms">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <div class="text-sm font-medium">Трафик</div>
        <div class="text-xs text-muted">Суммарно по всем пользователям и нодам</div>
      </div>
      <div class="flex gap-1 rounded-lg bg-surface-2 p-1">
        {#each ranges as r (r.hours)}
          <button
            class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors cursor-pointer {rangeHours === r.hours ? 'bg-up text-white' : 'text-muted hover:text-text'}"
            onclick={() => (rangeHours = r.hours)}
          >
            {r.label}
          </button>
        {/each}
      </div>
    </div>
    {#if !loading}
      <TrafficChart points={series} />
    {:else}
      <div class="flex h-56 items-center justify-center text-sm text-muted">Загрузка…</div>
    {/if}
  </div>
</div>
