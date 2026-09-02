<script>
  import { fade } from 'svelte/transition'
  import { api } from './lib/api.js'
  import Sidebar from './lib/Sidebar.svelte'
  import Login from './lib/views/Login.svelte'
  import Dashboard from './lib/views/Dashboard.svelte'
  import Users from './lib/views/Users.svelte'
  import Nodes from './lib/views/Nodes.svelte'
  import Settings from './lib/views/Settings.svelte'

  let authed = $state(null) // null = checking, false = login, true = app
  let tab = $state('dashboard')
  let adminUsername = $state('')

  async function checkAuth() {
    try {
      await api.session()
      authed = true
      const s = await api.getSettings()
      adminUsername = s.admin_username
    } catch {
      authed = false
    }
  }
  checkAuth()

  async function onLoginSuccess() {
    authed = true
    const s = await api.getSettings()
    adminUsername = s.admin_username
  }

  async function logout() {
    await api.logout().catch(() => {})
    authed = false
  }

  const views = { dashboard: Dashboard, users: Users, nodes: Nodes, settings: Settings }
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
    <Sidebar active={tab} onselect={(t) => (tab = t)} onlogout={logout} {adminUsername} />
    <main class="flex-1 overflow-y-auto p-8">
      {#key tab}
        <div transition:fade={{ duration: 180 }}>
          <CurrentView />
        </div>
      {/key}
    </main>
  </div>
{/if}
