<script>
  import Subscriptions from './Subscriptions.svelte'
  import Dashboard from './Dashboard.svelte'

  let {
    settings,
    onchange,
    status,
    history,
    currentUp,
    currentDown,
    activeLabel,
    ontoggle,
    systemWide,
    onmodechange,
  } = $props()
</script>

<div class="flex flex-1 overflow-hidden">
  <div class="flex w-[300px] shrink-0 flex-col overflow-hidden border-r border-border">
    <Subscriptions {settings} {onchange} />
  </div>
  <Dashboard {status} {history} {currentUp} {currentDown} {activeLabel} {ontoggle} {systemWide} {onmodechange} />
</div>
