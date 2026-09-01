<script>
  import { fade } from 'svelte/transition'
  import { State, STATE_LABEL } from '../constants.js'
  import { formatBytes, formatBytesPerSec, formatDurationNs } from '../format.js'
  import ToggleButton from '../ToggleButton.svelte'
  import Sparkline from '../Sparkline.svelte'
  import StatIcon from '../StatIcon.svelte'

  let { status, history, currentUp, currentDown, activeProfileName, ontoggle } = $props()

  const stats = $derived([
    { label: 'RTT', value: status.rttNs ? formatDurationNs(status.rttNs) : '—', icon: 'activity' },
    {
      label: 'Аптайм',
      value: status.state === State.CONNECTED ? formatDurationNs(status.uptimeNs) : '—',
      icon: 'clock',
    },
    { label: 'Потоки', value: String(status.activeStreams), icon: 'layers' },
    { label: 'Трафик', value: formatBytes(status.bytesUp + status.bytesDown), icon: 'swap' },
  ])
</script>

<div class="flex flex-1 flex-col px-6 py-5">
  <header class="mb-1 text-center fade-in-up">
    <span class="text-xs uppercase tracking-widest text-muted">
      {activeProfileName || 'Профиль не выбран'}
    </span>
  </header>

  <div class="flex flex-1 flex-col items-center justify-center gap-4">
    <ToggleButton state={status.state} onclick={ontoggle} />
    {#key status.state}
      <span
        class="text-xs uppercase tracking-widest text-muted"
        in:fade={{ duration: 250, delay: 100 }}
        out:fade={{ duration: 100 }}
      >
        {STATE_LABEL[status.state]}
      </span>
    {/key}
  </div>

  <div class="card mb-4 p-4 fade-in-up" style="animation-delay: 60ms">
    <div class="mb-2 flex items-center justify-between text-xs text-muted">
      <span><span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-up shadow-[0_0_6px_var(--color-up)]"></span>Отдача {formatBytesPerSec(currentUp)}</span>
      <span><span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-down shadow-[0_0_6px_var(--color-down)]"></span>Приём {formatBytesPerSec(currentDown)}</span>
    </div>
    <Sparkline {history} />
  </div>

  <div class="card mb-2 grid grid-cols-4 gap-2 p-3 text-center fade-in-up" style="animation-delay: 120ms">
    {#each stats as s (s.label)}
      <div class="transition-transform duration-200 hover:-translate-y-0.5">
        <StatIcon name={s.icon} />
        <div class="text-[10px] uppercase tracking-wide text-muted">{s.label}</div>
        <div class="text-sm font-medium tabular-nums">{s.value}</div>
      </div>
    {/each}
  </div>

  {#if status.lastError}
    <footer class="truncate text-xs text-state-error" transition:fade={{ duration: 200 }}>
      {status.lastError}
    </footer>
  {/if}
</div>
