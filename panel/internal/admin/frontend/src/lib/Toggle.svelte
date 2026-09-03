<script>
  let { checked = $bindable(false), disabled = false, onchange } = $props()

  function toggle() {
    if (disabled) return
    checked = !checked
    onchange?.(checked)
  }
</script>

<button
  type="button"
  role="switch"
  aria-checked={checked}
  {disabled}
  onclick={toggle}
  class="relative h-6 w-11 shrink-0 rounded-full transition-colors duration-200 cursor-pointer
         disabled:cursor-not-allowed disabled:opacity-40
         {checked ? 'bg-gradient-to-r from-up to-accent-2' : 'border border-border bg-surface-2'}"
>
  {#if checked}
    <span class="absolute inset-0 -z-10 rounded-full bg-up opacity-50 blur-md"></span>
  {/if}
  <span
    class="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,0.4)]
           transition-transform duration-200 ease-out"
    style="transform: translateX({checked ? '20px' : '0'})"
  ></span>
</button>
