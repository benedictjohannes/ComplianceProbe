<script lang="ts">
  import { Accordion as BitsAccordion } from 'bits-ui';
  import type { Snippet } from 'svelte';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import { cn } from '$lib/utils/cn';

  export interface AccordionItemData {
    value: string;
    title?: string;
    disabled?: boolean;
  }

  interface Props {
    type?: 'single' | 'multiple';
    value?: string | string[];
    class?: string;
    children?: Snippet;
    onValueChange?: (v: any) => void;
  }

  let {
    type = 'single',
    value = $bindable(type === 'multiple' ? [] : ''),
    class: className = '',
    children,
    onValueChange,
  }: Props = $props();
</script>

{#if type === 'multiple'}
  <BitsAccordion.Root
    type="multiple"
    bind:value={value as string[]}
    onValueChange={(v) => {
      value = v;
      onValueChange?.(v);
    }}
    class={cn('w-full divide-y divide-zinc-800', className)}
  >
    {#if children}
      {@render children()}
    {/if}
  </BitsAccordion.Root>
{:else}
  <BitsAccordion.Root
    type="single"
    bind:value={value as string}
    onValueChange={(v) => {
      value = v;
      onValueChange?.(v);
    }}
    class={cn('w-full divide-y divide-zinc-800', className)}
  >
    {#if children}
      {@render children()}
    {/if}
  </BitsAccordion.Root>
{/if}
