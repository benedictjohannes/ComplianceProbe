<script lang="ts">
  import { Accordion as BitsAccordion } from 'bits-ui';
  import type { Snippet } from 'svelte';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import { cn } from '$lib/utils/cn';

  interface Props {
    value: string;
    title?: string;
    disabled?: boolean;
    class?: string;
    headerClass?: string;
    trigger?: Snippet;
    badge?: Snippet;
    children?: Snippet;
  }

  let {
    value,
    title,
    disabled = false,
    class: className = '',
    headerClass = '',
    trigger,
    badge,
    children,
  }: Props = $props();
</script>

<BitsAccordion.Item {value} {disabled} class={cn('border-b border-zinc-800 last:border-b-0', className)}>
  <BitsAccordion.Header class={cn('flex items-center justify-between', headerClass)}>
    <BitsAccordion.Trigger
      class="flex flex-1 items-center justify-between py-3 px-2 text-sm font-medium text-zinc-200 transition-all hover:text-white cursor-pointer select-none group focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-500 rounded"
    >
      {#if trigger}
        {@render trigger()}
      {:else}
        <div class="flex items-center gap-2">
          <span>{title}</span>
          {#if badge}
            {@render badge()}
          {/if}
        </div>
      {/if}
      <ChevronDown
        class="h-4 w-4 shrink-0 text-zinc-400 transition-transform duration-200 group-data-[state=open]:rotate-180"
      />
    </BitsAccordion.Trigger>
  </BitsAccordion.Header>
  <BitsAccordion.Content
    class="overflow-hidden text-sm text-zinc-300 transition-all px-2 pb-3"
  >
    {#if children}
      {@render children()}
    {/if}
  </BitsAccordion.Content>
</BitsAccordion.Item>
