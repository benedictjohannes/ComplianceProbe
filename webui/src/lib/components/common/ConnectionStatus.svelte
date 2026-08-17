<script lang="ts">
  import { Tooltip } from '$lib/components/ui';
  import { cn } from '$lib/utils/cn';

  export type ConnectionState = 'connected' | 'reconnecting' | 'disconnected';

  interface Props {
    state?: ConnectionState;
    class?: string;
    onreconnect?: () => void;
  }

  let {
    state = 'connected',
    class: className = '',
    onreconnect,
  }: Props = $props();
</script>

<div class={cn('inline-flex items-center gap-1.5 font-mono text-xs select-none', className)}>
  {#if state === 'connected'}
    <span class="inline-flex h-2 w-2 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]"></span>
    <span class="text-zinc-400 hidden sm:inline">Connected</span>
  {:else if state === 'reconnecting'}
    <span class="inline-flex h-2 w-2 rounded-full bg-amber-500 animate-ping"></span>
    <span class="text-amber-400 flex items-center gap-1 font-medium">
      Reconnecting...
    </span>
  {:else}
    <span class="inline-flex h-2 w-2 rounded-full bg-rose-500"></span>
    <span class="text-rose-400 font-medium">Disconnected</span>
    {#if onreconnect}
      <button
        type="button"
        onclick={onreconnect}
        class="text-xs text-sky-400 underline hover:text-sky-300 ml-1 cursor-pointer"
      >
        Retry
      </button>
    {/if}
  {/if}
</div>
