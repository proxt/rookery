<script>
  import { onMount } from 'svelte'
  import { Connect, Disconnect, GetStatus, GetAppSettings, SetSystemWideMode } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import { State, EventType } from './lib/constants.js'
  import BottomNav from './lib/BottomNav.svelte'
  import Dashboard from './lib/views/Dashboard.svelte'
  import Subscriptions from './lib/views/Subscriptions.svelte'
  import Settings from './lib/views/Settings.svelte'
  import About from './lib/views/About.svelte'

  const HISTORY_LEN = 60

  let activeTab = $state('dashboard')

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

  let settings = $state({ subscriptions: [], activeSubscriptionId: '', socksPort: 1080 })
  let killSwitchWarning = $state('')

  const activeLabel = $derived.by(() => {
    const sub = settings.subscriptions.find((s) => s.id === settings.activeSubscriptionId)
    if (!sub) return ''
    const node = sub.nodes?.find((n) => n.id === sub.activeNodeId) ?? sub.nodes?.[0]
    return node ? `${sub.name} · ${node.name}` : sub.name
  })

  async function reloadSettings() {
    const s = await GetAppSettings()
    settings = { ...s, subscriptions: (s.subscriptions ?? []).map((sub) => ({ ...sub, nodes: sub.nodes ?? [] })) }
  }

  onMount(() => {
    GetStatus().then((s) => (status = s))
    reloadSettings()

    EventsOn('tunnel:event', (ev) => {
      if (ev.type === EventType.STATE_CHANGED) {
        status = { ...status, state: ev.state, lastError: ev.err }
        if (ev.state !== State.CONNECTED) {
          history = []
          currentUp = 0
          currentDown = 0
        }
        // Refetch for killSwitchEngaged — it's decided synchronously
        // alongside the state transition on the Go side, not carried on
        // the event itself.
        GetStatus().then((s) => (status = s))
      } else if (ev.type === EventType.STATS_TICK) {
        currentUp = ev.bytesUpPerSec
        currentDown = ev.bytesDownPerSec
        history = [...history, { up: ev.bytesUpPerSec, down: ev.bytesDownPerSec }].slice(-HISTORY_LEN)
        GetStatus().then((s) => (status = s))
      } else if (ev.type === EventType.KILL_SWITCH_WARNING) {
        killSwitchWarning = ev.err
        GetStatus().then((s) => (status = s))
      }
    })

    // Fired when a rookery://... link launches (or is forwarded to) this
    // process — e.g. clicking "Установить в приложение" on a subscription
    // page — after it's been added as a new subscription.
    EventsOn('subscription:added', () => {
      activeTab = 'subscriptions'
      reloadSettings()
    })

    // Fired after a background auto-refresh of subscriptions — same reload,
    // but without yanking the user over to the Subscriptions tab.
    EventsOn('settings:updated', () => {
      reloadSettings()
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

  // Mode (proxy vs. TUN) dropdown on the dashboard — a lighter-weight path
  // than the full Settings form. If a session is already running, the new
  // mode can't apply mid-session (SOCKS5-vs-TUN capture is set up once at
  // Start), so this reconnects; otherwise it just saves for next Connect.
  async function changeMode(systemWide) {
    await SetSystemWideMode(systemWide)
    await reloadSettings()
    if (status.state === State.CONNECTED || status.state === State.CONNECTING) {
      Disconnect()
      try {
        await Connect()
      } catch (e) {
        status = { ...status, state: State.ERROR, lastError: String(e) }
      }
    }
  }
</script>

<div class="flex h-screen flex-col">
  {#if killSwitchWarning}
    <div class="flex items-start justify-between gap-2 bg-state-error/15 px-4 py-2 text-xs text-state-error">
      <span>{killSwitchWarning}</span>
      <button class="shrink-0 cursor-pointer font-medium hover:underline" onclick={() => (killSwitchWarning = '')}>Скрыть</button>
    </div>
  {/if}
  {#if activeTab === 'dashboard'}
    <Dashboard
      {status}
      {history}
      {currentUp}
      {currentDown}
      {activeLabel}
      ontoggle={toggle}
      systemWide={settings.systemWide}
      onmodechange={changeMode}
    />
  {:else if activeTab === 'subscriptions'}
    <Subscriptions {settings} onchange={reloadSettings} />
  {:else if activeTab === 'settings'}
    <Settings {settings} onchange={reloadSettings} />
  {:else}
    <About />
  {/if}

  <BottomNav active={activeTab} onselect={(id) => (activeTab = id)} />
</div>
