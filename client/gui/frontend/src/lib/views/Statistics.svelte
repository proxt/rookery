<script>
  import { State } from '../constants.js'
  import { formatBytes, formatBytesPerSec, formatDurationNs } from '../format.js'
  import Sparkline from '../Sparkline.svelte'
  import StatIcon from '../StatIcon.svelte'

  let { status, history, currentUp, currentDown, activeLabel } = $props()

  const stats = $derived([
    { label: 'RTT', value: status.rttNs ? formatDurationNs(status.rttNs) : '—', icon: 'activity', bgClass: 'bg-up/12', textClass: 'text-up' },
    {
      label: 'Аптайм',
      value: status.state === State.CONNECTED ? formatDurationNs(status.uptimeNs) : '—',
      icon: 'clock',
      bgClass: 'bg-accent-2/12',
      textClass: 'text-accent-2',
    },
    { label: 'Активные потоки', value: String(status.activeStreams), icon: 'layers', bgClass: 'bg-accent-3/12', textClass: 'text-accent-3' },
    { label: 'Всего трафика', value: formatBytes(status.bytesUp + status.bytesDown), icon: 'swap', bgClass: 'bg-down/12', textClass: 'text-down' },
  ])
</script>

<div class="scroll-area flex flex-1 flex-col px-6 py-5">
  <h1 class="mb-1 text-xs font-semibold uppercase tracking-widest text-muted">Статистика</h1>
  <p class="mb-4 text-xs text-muted">
    {activeLabel ? `Текущая сессия — ${activeLabel}` : 'Нет активного подключения'}
  </p>

  <div class="card card-accent fade-in-up mb-4 p-4">
    <div class="mb-3 flex items-center justify-between text-xs text-muted">
      <span><span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-up shadow-[0_0_6px_var(--color-up)]"></span>Отдача {formatBytesPerSec(currentUp)}</span>
      <span><span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-down shadow-[0_0_6px_var(--color-down)]"></span>Приём {formatBytesPerSec(currentDown)}</span>
    </div>
    <Sparkline {history} heightClass="h-40" />
  </div>

  <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
    {#each stats as s (s.label)}
      <div class="card fade-in-up p-4 text-center">
        <div class="mx-auto mb-2 flex h-9 w-9 items-center justify-center rounded-full {s.bgClass}">
          <StatIcon name={s.icon} class="{s.textClass} h-4 w-4" />
        </div>
        <div class="text-[11px] uppercase tracking-wide text-muted">{s.label}</div>
        <div class="mt-0.5 text-base font-medium tabular-nums">{s.value}</div>
      </div>
    {/each}
  </div>

  <div class="mt-4 grid grid-cols-2 gap-3">
    <div class="card fade-in-up p-4">
      <div class="text-[11px] uppercase tracking-wide text-muted">Отдано за сессию</div>
      <div class="mt-0.5 text-lg font-semibold tabular-nums text-up">{formatBytes(status.bytesUp)}</div>
    </div>
    <div class="card fade-in-up p-4">
      <div class="text-[11px] uppercase tracking-wide text-muted">Принято за сессию</div>
      <div class="mt-0.5 text-lg font-semibold tabular-nums text-down">{formatBytes(status.bytesDown)}</div>
    </div>
  </div>

  {#if status.lastError}
    <p class="mt-4 text-xs text-state-error">{status.lastError}</p>
  {/if}
</div>
