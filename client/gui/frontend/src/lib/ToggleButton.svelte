<script>
  import { State } from './constants.js'

  let { state = State.DISCONNECTED, onclick } = $props()

  const stateClass = $derived({
    [State.DISCONNECTED]: 'bg-state-disconnected',
    [State.CONNECTING]: 'bg-state-connecting',
    [State.CONNECTED]: 'bg-state-connected',
    [State.ERROR]: 'bg-state-error',
  }[state])

  const glowVar = $derived({
    [State.DISCONNECTED]: 'var(--color-state-disconnected)',
    [State.CONNECTING]: 'var(--color-state-connecting)',
    [State.CONNECTED]: 'var(--color-state-connected)',
    [State.ERROR]: 'var(--color-state-error)',
  }[state])

  const label = $derived({
    [State.DISCONNECTED]: 'Подключить',
    [State.CONNECTING]: 'Подключение',
    [State.CONNECTED]: 'Отключить',
    [State.ERROR]: 'Повторить',
  }[state])

  const busy = $derived(state === State.CONNECTING)
  const live = $derived(state === State.CONNECTED)
</script>

<button
  class="group relative flex h-40 w-40 items-center justify-center rounded-full
         transition-transform duration-150 active:scale-[0.96] cursor-pointer"
  style="--glow: {glowVar}"
  onclick={onclick}
  disabled={busy}
>
  {#if busy}
    <span class="pulse-ring absolute inset-0 rounded-full {stateClass} opacity-40"></span>
    <span class="pulse-ring-delay absolute inset-0 rounded-full {stateClass} opacity-30"></span>
  {/if}

  {#if live}
    <span
      class="breathe absolute -inset-4 rounded-full blur-2xl"
      style="background: var(--glow)"
    ></span>
  {/if}

  <span
    class="absolute inset-0 rounded-full {stateClass} transition-[background-color] duration-500 ease-out
           group-hover:brightness-110"
    style="box-shadow: 0 0 0 1px rgba(255,255,255,0.08) inset, 0 14px 36px -10px var(--glow), 0 0 28px -6px var(--glow);"
  ></span>

  <span class="relative flex flex-col items-center gap-1.5 text-white/95">
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" class="opacity-90">
      <path d="M12 2.5v8.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      <path
        d="M6.5 6.5a8 8 0 1 0 11 0"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        fill="none"
      />
    </svg>
    <span class="text-base font-semibold tracking-wide">{label}</span>
  </span>
</button>
