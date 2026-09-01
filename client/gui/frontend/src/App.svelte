<script>
  import { onMount } from 'svelte'
  import { fade } from 'svelte/transition'
  import { Connect, Disconnect, GetStatus, GetSettings, SaveSettings } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import { State, EventType, STATE_LABEL } from './lib/constants.js'
  import { formatBytes, formatBytesPerSec, formatDurationNs } from './lib/format.js'
  import ToggleButton from './lib/ToggleButton.svelte'
  import Sparkline from './lib/Sparkline.svelte'
  import SettingsPanel from './lib/SettingsPanel.svelte'
  import StatIcon from './lib/StatIcon.svelte'
  import logo from './assets/images/logo.png'

  const HISTORY_LEN = 60

  let status = $state({
    state: State.DISCONNECTED,
    uptimeNs: 0,
    rttNs: 0,
    activeStreams: 0,
    bytesUp: 0,
    bytesDown: 0,
    lastError: '',
  })

  let history = $state([])
  let currentUp = $state(0)
  let currentDown = $state(0)

  let settingsOpen = $state(false)
  let settings = $state({})

  onMount(() => {
    GetStatus().then((s) => (status = s))
    GetSettings().then((s) => (settings = s))

    EventsOn('tunnel:event', (ev) => {
      if (ev.type === EventType.STATE_CHANGED) {
        status = { ...status, state: ev.state, lastError: ev.err }
        if (ev.state !== State.CONNECTED) {
          history = []
          currentUp = 0
          currentDown = 0
        }
      } else if (ev.type === EventType.STATS_TICK) {
        currentUp = ev.bytesUpPerSec
        currentDown = ev.bytesDownPerSec
        history = [...history, { up: ev.bytesUpPerSec, down: ev.bytesDownPerSec }].slice(-HISTORY_LEN)
        GetStatus().then((s) => (status = s))
      }
    })
  })

  async function toggle() {
    if (status.state === State.CONNECTED || status.state === State.CONNECTING) {
      Disconnect()
      return
    }
    try {
      await Connect()
    } catch (e) {
      status = { ...status, state: State.ERROR, lastError: String(e) }
    }
  }

  async function handleSaveSettings(newSettings) {
    await SaveSettings(newSettings)
    settings = newSettings
  }

  const stats = $derived([
    {
      label: 'RTT',
      value: status.rttNs ? formatDurationNs(status.rttNs) : '—',
      icon: 'activity',
    },
    {
      label: 'Аптайм',
      value: status.state === State.CONNECTED ? formatDurationNs(status.uptimeNs) : '—',
      icon: 'clock',
    },
    {
      label: 'Потоки',
      value: String(status.activeStreams),
      icon: 'layers',
    },
    {
      label: 'Трафик',
      value: formatBytes(status.bytesUp + status.bytesDown),
      icon: 'swap',
    },
  ])
</script>

<div class="flex h-screen flex-col px-6 py-5">
  <header class="flex items-center justify-between fade-in-up">
    <img src={logo} alt="Rookery" class="h-7 w-7 rounded-full opacity-90" />
    <button
      class="rounded-lg p-1.5 text-muted transition-all duration-300 hover:text-text hover:rotate-45 cursor-pointer"
      onclick={() => (settingsOpen = true)}
      aria-label="Настройки"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
        <circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2" />
        <path
          d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1.08-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z"
          stroke="currentColor"
          stroke-width="1.5"
        />
      </svg>
    </button>
  </header>

  <div class="flex flex-1 flex-col items-center justify-center gap-4">
    <ToggleButton state={status.state} onclick={toggle} />
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
      <span class="flex items-center gap-1.5">
        <span class="h-2 w-2 rounded-full bg-up shadow-[0_0_6px_var(--color-up)]"></span>
        Отдача {formatBytesPerSec(currentUp)}
      </span>
      <span class="flex items-center gap-1.5">
        <span class="h-2 w-2 rounded-full bg-down shadow-[0_0_6px_var(--color-down)]"></span>
        Приём {formatBytesPerSec(currentDown)}
      </span>
    </div>
    <Sparkline {history} />
  </div>

  <div class="card mb-4 grid grid-cols-4 gap-2 p-3 text-center fade-in-up" style="animation-delay: 120ms">
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

<SettingsPanel
  open={settingsOpen}
  {settings}
  onsave={handleSaveSettings}
  onclose={() => (settingsOpen = false)}
/>
