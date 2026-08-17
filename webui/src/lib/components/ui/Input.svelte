<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLInputAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils/cn';

  interface Props extends Omit<HTMLInputAttributes, 'value'> {
    value?: string | number;
    mono?: boolean;
    error?: boolean | string;
    class?: string;
    leading?: Snippet;
    trailing?: Snippet;
  }

  let {
    value = $bindable(''),
    mono = false,
    error = false,
    class: className = '',
    disabled = false,
    leading,
    trailing,
    type = 'text',
    ...restProps
  }: Props = $props();
</script>

<div class="relative flex items-center w-full">
  {#if leading}
    <div class="absolute left-3 flex items-center pointer-events-none text-zinc-500">
      {@render leading()}
    </div>
  {/if}

  <input
    {type}
    bind:value
    {disabled}
    class={cn(
      'w-full bg-zinc-900 dark:bg-zinc-900 border border-zinc-700 dark:border-zinc-800 rounded-md text-sm text-zinc-100 placeholder:text-zinc-500 py-2 px-3 transition-colors',
      'focus-visible:outline-none focus-visible:border-sky-500 focus-visible:ring-1 focus-visible:ring-sky-500',
      'disabled:opacity-50 disabled:cursor-not-allowed',
      mono && 'font-mono text-xs',
      error && 'border-rose-500 focus-visible:border-rose-500 focus-visible:ring-rose-500',
      leading && 'pl-9',
      trailing && 'pr-9',
      className
    )}
    {...restProps}
  />

  {#if trailing}
    <div class="absolute right-3 flex items-center text-zinc-400">
      {@render trailing()}
    </div>
  {/if}
</div>
