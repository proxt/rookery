<script>
  import { fade } from 'svelte/transition'
  import { api } from './lib/api.js'
  import Sidebar from './lib/Sidebar.svelte'
  import Login from './lib/views/Login.svelte'
  import Dashboard from './lib/views/Dashboard.svelte'
  import Users from './lib/views/Users.svelte'
  import Nodes from './lib/views/Nodes.svelte'
  import Releases from './lib/views/Releases.svelte'
  import Admins from './lib/views/Admins.svelte'
  import AuditLog from './lib/views/AuditLog.svelte'
  import Settings from './lib/views/Settings.svelte'

  let authed = $state(null) // null = checking, false = login, true = app
  let tab = $state('dashboard')
  let adminUsername = $state('')
  let buildTime = $state('')

  async function loadVersion() {
    try {
      const v = await api.version()
      buildTime = v.build_time && v.build_time !== 'unknown' ? v.build_time : ''
    } catch {
      buildTime = ''
    }
  }

  async function checkAuth() {
    try {
      const s = await api.session()
      authed = true
      adminUsername = s.username
      loadVersion()
    } catch {
      authed = false
    }
  }
  checkAuth()

  async function onLoginSuccess() {
    const s = await api.session()
    authed = true
    adminUsername = s.username
    loadVersion()
  }

  async function logout() {
    await api.logout().catch(() => {})
    authed = false
  }

  const views = { dashboard: Dashboard, users: Users, nodes: Nodes, releases: Releases, admins: Admins, audit: AuditLog, settings: Settings }
  const CurrentView = $derived(views[tab])
</script>

{#if authed === null}
  <div class="flex min-h-screen items-center justify-center">
    <div class="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-up"></div>
  </div>
{:else if authed === false}
  <Login onsuccess={onLoginSuccess} />
{:else}
  <div class="flex min-h-screen">
    <Sidebar active={tab} onselect={(t) => (tab = t)} onlogout={logout} {adminUsername} {buildTime} />
    <main class="flex-1 overflow-y-auto p-8">
      {#key tab}
        <div transition:fade={{ duration: 180 }}>
          <CurrentView />
        </div>
      {/key}
    </main>
  </div>
{/if}
