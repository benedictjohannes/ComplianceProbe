<script lang="ts">
  import { DropdownMenu as BitsDropdown } from 'bits-ui';
  import type { Snippet } from 'svelte';
  import { cn } from '$lib/utils/cn';

  export interface DropdownItem {
    id: string;
    label: string;
    icon?: any;
    danger?: boolean;
    disabled?: boolean;
    separator?: boolean;
    onclick?: () => void;
  }

  interface Props {
    open?: boolean;
    items?: DropdownItem[];
    side?: 'top' | 'right' | 'bottom' | 'left';
    align?: 'start' | 'center' | 'end';
    sideOffset?: number;
    class?: string;
    trigger: Snippet;
    children?: Snippet;
    onOpenChange?: (open: boolean) => void;
  }

  let {
    open = $bindable(false),
    items = [],
    side = 'bottom',
    align = 'end',
    sideOffset = 6,
    class: className = '',
    trigger,
    children,
    onOpenChange,
  }: Props = $props();
</script>

<BitsDropdown.Root
  bind:open
  onOpenChange={(v) => {
    open = v;
    onOpenChange?.(v);
  }}
>
  <BitsDropdown.Trigger class="inline-flex">
    {@render trigger()}
  </BitsDropdown.Trigger>

  <BitsDropdown.Portal>
    <BitsDropdown.Content
      {side}
      {align}
      {sideOffset}
      class={cn(
        'z-50 min-w-[180px] rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-1.5 shadow-xl text-zinc-800 dark:text-zinc-200 animate-in fade-in-0 zoom-in-95',
        className
      )}
    >
      {#if children}
        {@render children()}
      {:else}
        {#each items as item (item.id)}
          {#if item.separator}
            <BitsDropdown.Separator class="my-1 h-px bg-zinc-200 dark:bg-zinc-800" />
          {:else}
            <BitsDropdown.Item
              disabled={item.disabled}
              onSelect={() => item.onclick?.()}
              class={cn(
                'relative flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs font-medium outline-none select-none cursor-pointer transition-colors',
                'hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-zinc-900 dark:hover:text-zinc-100 focus:bg-zinc-100 dark:focus:bg-zinc-800 focus:text-zinc-900 dark:focus:text-zinc-100',
                item.danger && 'text-rose-600 dark:text-rose-400 hover:text-rose-700 dark:hover:text-rose-300 hover:bg-rose-50 dark:hover:bg-rose-500/10 focus:bg-rose-50 dark:focus:bg-rose-500/10',
                item.disabled && 'opacity-40 cursor-not-allowed pointer-events-none'
              )}
            >
              {#if item.icon}
                {@const Icon = item.icon}
                <Icon class="h-3.5 w-3.5 shrink-0" />
              {/if}
              <span>{item.label}</span>
            </BitsDropdown.Item>
          {/if}
        {/each}
      {/if}
    </BitsDropdown.Content>
  </BitsDropdown.Portal>
</BitsDropdown.Root>
