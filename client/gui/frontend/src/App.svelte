<script>
  import { onMount } from 'svelte'
  import { Connect, Disconnect, GetStatus, GetAppSettings } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import { State, EventType } from './lib/constants.js'
  import BottomNav from './lib/BottomNav.svelte'
  import Dashboard from './lib/views/Dashboard.svelte'
  import Profiles from './lib/views/Profiles.svelte'
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

  let settings = $state({ profiles: [], activeProfileId: '', socksPort: 1080 })

  const activeProfileName = $derived(
    settings.profiles.find((p) => p.id === settings.activeProfileId)?.name ?? ''
  )

  async function reloadSettings() {
    const s = await GetAppSettings()
    settings = { ...s, profiles: s.profiles ?? [] }
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
    <Dashboard {status} {history} {currentUp} {currentDown} {activeProfileName} ontoggle={toggle} />
  {:else if activeTab === 'profiles'}
    <Profiles {settings} onchange={reloadSettings} />
  {:else if activeTab === 'settings'}
    <Settings {settings} onchange={reloadSettings} />
  {:else}
    <About />
  {/if}

  <BottomNav active={activeTab} onselect={(id) => (activeTab = id)} />
</div>
