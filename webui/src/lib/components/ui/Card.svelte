<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';
  import { cn } from '$lib/utils/cn';

  interface Props extends HTMLAttributes<HTMLDivElement> {
    hover?: boolean;
    class?: string;
    children?: Snippet;
  }

  let {
    hover = false,
    class: className = '',
    children,
    onclick,
    ...restProps
  }: Props = $props();
</script>

<div
  class={cn(
    'bg-white dark:bg-zinc-900/70 border border-zinc-200 dark:border-zinc-800/90 text-zinc-900 dark:text-zinc-100 rounded-lg p-4 transition-all shadow-xs',
    hover && 'hover:border-zinc-300 dark:hover:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-900/90 cursor-pointer',
    className
  )}
  {onclick}
  {...restProps}
>
  {#if children}
    {@render children()}
  {/if}
</div>
