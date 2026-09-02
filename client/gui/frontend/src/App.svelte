<script>
  import { onMount } from 'svelte'
  import { Connect, Disconnect, GetStatus, GetAppSettings } from '../wailsjs/go/main/App.js'
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
      } else if (ev.type === EventType.STATS_TICK) {
        currentUp = ev.bytesUpPerSec
        currentDown = ev.bytesDownPerSec
        history = [...history, { up: ev.bytesUpPerSec, down: ev.bytesDownPerSec }].slice(-HISTORY_LEN)
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
</script>

<div class="flex h-screen flex-col">
  {#if activeTab === 'dashboard'}
    <Dashboard {status} {history} {currentUp} {currentDown} {activeLabel} ontoggle={toggle} />
  {:else if activeTab === 'subscriptions'}
    <Subscriptions {settings} onchange={reloadSettings} />
  {:else if activeTab === 'settings'}
    <Settings {settings} onchange={reloadSettings} />
  {:else}
    <About />
  {/if}

  <BottomNav active={activeTab} onselect={(id) => (activeTab = id)} />
</div>
