<script>
  import { fade } from 'svelte/transition'
  import { State, STATE_LABEL } from '../constants.js'
  import { formatBytes, formatBytesPerSec, formatDurationNs } from '../format.js'
  import ToggleButton from '../ToggleButton.svelte'
  import Sparkline from '../Sparkline.svelte'
  import StatIcon from '../StatIcon.svelte'

  let { status, history, currentUp, currentDown, activeLabel, ontoggle } = $props()

  // Full literal class strings (not built from a short key) so Tailwind's
  // static scanner — which only picks up whole tokens it can see verbatim
  // in the source — actually generates these utilities.
  const stats = $derived([
    { label: 'RTT', value: status.rttNs ? formatDurationNs(status.rttNs) : '—', icon: 'activity', bgClass: 'bg-up/12', textClass: 'text-up' },
    {
      label: 'Аптайм',
      value: status.state === State.CONNECTED ? formatDurationNs(status.uptimeNs) : '—',
      icon: 'clock',
      bgClass: 'bg-accent-2/12',
      textClass: 'text-accent-2',
    },
    { label: 'Потоки', value: String(status.activeStreams), icon: 'layers', bgClass: 'bg-accent-3/12', textClass: 'text-accent-3' },
    { label: 'Трафик', value: formatBytes(status.bytesUp + status.bytesDown), icon: 'swap', bgClass: 'bg-down/12', textClass: 'text-down' },
  ])

  const stateLabel = $derived(status.killSwitchEngaged ? 'Заблокировано (kill switch)' : STATE_LABEL[status.state])
  const live = $derived(status.state === State.CONNECTED)
</script>

<div class="flex flex-1 flex-col px-6 py-5">
  <header class="mb-1 flex justify-center fade-in-up">
    <span
      class="inline-flex items-center gap-2 rounded-full border border-border bg-surface-2/70 px-3 py-1.5 text-xs font-medium text-text backdrop-blur-sm"
    >
      <span class="relative flex h-1.5 w-1.5">
        {#if live}<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-state-connected opacity-75"></span>{/if}
        <span class="relative inline-flex h-1.5 w-1.5 rounded-full {live ? 'bg-state-connected' : 'bg-muted'}"></span>
      </span>
      {activeLabel || 'Подписка не выбрана'}
    </span>
  </header>

  <div class="flex flex-1 flex-col items-center justify-center gap-4">
    <ToggleButton state={status.state} onclick={ontoggle} />
    {#key status.state}
      <span
        class="text-xs uppercase tracking-widest {status.killSwitchEngaged ? 'text-state-error' : 'text-muted'}"
        in:fade={{ duration: 250, delay: 100 }}
        out:fade={{ duration: 100 }}
      >
        {stateLabel}
      </span>
    {/key}
  </div>

  <div class="card card-accent mb-4 p-4 fade-in-up" style="animation-delay: 60ms">
    <div class="mb-2 flex items-center justify-between text-xs text-muted">
      <span><span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-up shadow-[0_0_6px_var(--color-up)]"></span>Отдача {formatBytesPerSec(currentUp)}</span>
      <span><span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-down shadow-[0_0_6px_var(--color-down)]"></span>Приём {formatBytesPerSec(currentDown)}</span>
    </div>
    <Sparkline {history} />
  </div>

  <div class="card mb-2 grid grid-cols-4 gap-2 p-3 text-center fade-in-up" style="animation-delay: 120ms">
    {#each stats as s (s.label)}
      <div class="transition-transform duration-200 hover:-translate-y-0.5">
        <div class="mx-auto mb-1 flex h-7 w-7 items-center justify-center rounded-full {s.bgClass}">
          <StatIcon name={s.icon} class={s.textClass} />
        </div>
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
