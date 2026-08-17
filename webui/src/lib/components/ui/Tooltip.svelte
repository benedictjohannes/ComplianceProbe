<script lang="ts">
  import { Tooltip as BitsTooltip } from 'bits-ui';
  import type { Snippet } from 'svelte';
  import { cn } from '$lib/utils/cn';

  interface Props {
    content?: string;
    side?: 'top' | 'right' | 'bottom' | 'left';
    sideOffset?: number;
    delayDuration?: number;
    children?: Snippet;
    tooltipContent?: Snippet;
    class?: string;
  }

  let {
    content,
    side = 'top',
    sideOffset = 6,
    delayDuration = 200,
    children,
    tooltipContent,
    class: className = '',
  }: Props = $props();
</script>

<BitsTooltip.Provider {delayDuration}>
  <BitsTooltip.Root>
    <BitsTooltip.Trigger class="inline-flex">
      {#if children}
        {@render children()}
      {/if}
    </BitsTooltip.Trigger>
    <BitsTooltip.Portal>
      <BitsTooltip.Content
        {side}
        {sideOffset}
        class={cn(
          'z-50 rounded-md border border-zinc-700 dark:border-zinc-700 bg-zinc-900 px-2.5 py-1 text-xs text-zinc-200 shadow-md',
          className
        )}
      >
        {#if tooltipContent}
          {@render tooltipContent()}
        {:else if content}
          {content}
        {/if}
      </BitsTooltip.Content>
    </BitsTooltip.Portal>
  </BitsTooltip.Root>
</BitsTooltip.Provider>
