<script lang="ts">
  import { Dialog as BitsDialog } from 'bits-ui';
  import type { Snippet } from 'svelte';
  import X from 'lucide-svelte/icons/x';
  import { cn } from '$lib/utils/cn';

  interface Props {
    open?: boolean;
    title?: string;
    description?: string;
    preventClose?: boolean;
    maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | 'full';
    class?: string;
    trigger?: Snippet;
    header?: Snippet;
    children?: Snippet;
    footer?: Snippet;
    onOpenChange?: (open: boolean) => void;
  }

  let {
    open = $bindable(false),
    title,
    description,
    preventClose = false,
    maxWidth = 'md',
    class: className = '',
    trigger,
    header,
    children,
    footer,
    onOpenChange,
  }: Props = $props();

  const maxWidthStyles = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-2xl',
    full: 'max-w-5xl w-[95vw]',
  };
</script>

<BitsDialog.Root
  bind:open
  onOpenChange={(v) => {
    open = v;
    onOpenChange?.(v);
  }}
>
  {#if trigger}
    <BitsDialog.Trigger class="inline-flex">
      {@render trigger()}
    </BitsDialog.Trigger>
  {/if}

  <BitsDialog.Portal>
    <BitsDialog.Overlay
      class="fixed inset-0 z-50 bg-black/70 transition-opacity"
    />
    <BitsDialog.Content
      interactOutsideBehavior={preventClose ? 'ignore' : 'close'}
      escapeKeydownBehavior={preventClose ? 'ignore' : 'close'}
      class={cn(
        'fixed left-[50%] top-[50%] z-50 translate-x-[-50%] translate-y-[-50%]',
        'w-[92vw] rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6 shadow-2xl transition-all text-zinc-900 dark:text-zinc-100',
        maxWidthStyles[maxWidth],
        className
      )}
    >
      {#if !preventClose}
        <BitsDialog.Close
          class="absolute right-4 top-4 rounded-md p-1 text-zinc-500 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-zinc-900 dark:hover:text-zinc-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 cursor-pointer"
        >
          <X class="h-4 w-4" />
          <span class="sr-only">Close</span>
        </BitsDialog.Close>
      {/if}

      {#if header}
        {@render header()}
      {:else if title || description}
        <div class="mb-4 space-y-1.5 pr-6">
          {#if title}
            <BitsDialog.Title class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
              {title}
            </BitsDialog.Title>
          {/if}
          {#if description}
            <BitsDialog.Description class="text-sm text-zinc-600 dark:text-zinc-400">
              {description}
            </BitsDialog.Description>
          {/if}
        </div>
      {/if}

      {#if children}
        <div class="py-2">
          {@render children()}
        </div>
      {/if}

      {#if footer}
        <div class="mt-6 flex items-center justify-end gap-3 pt-3 border-t border-zinc-200 dark:border-zinc-800">
          {@render footer()}
        </div>
      {/if}
    </BitsDialog.Content>
  </BitsDialog.Portal>
</BitsDialog.Root>
