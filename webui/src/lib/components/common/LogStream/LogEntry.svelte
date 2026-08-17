<script lang="ts">
  import { cn } from '$lib/utils/cn';

  interface Props {
    message: string;
    lineNumber?: number;
    searchQuery?: string;
    class?: string;
  }

  let {
    message,
    lineNumber,
    searchQuery = '',
    class: className = '',
  }: Props = $props();

  // Highlight search query matches
  const segments = $derived.by(() => {
    if (!searchQuery || !searchQuery.trim()) {
      return [{ text: message, match: false }];
    }

    const query = searchQuery.trim();
    const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
    const parts = message.split(regex);

    return parts.map((part) => ({
      text: part,
      match: part.toLowerCase() === query.toLowerCase(),
    }));
  });

  // Determine line accent based on log message semantics
  const lineTone = $derived.by(() => {
    const lower = message.toLowerCase();
    if (lower.includes('error') || lower.includes('failed') || lower.includes('fatal')) {
      return 'text-rose-400 dark:text-rose-400 font-medium';
    }
    if (lower.includes('warn') || lower.includes('warning') || lower.includes('elevat')) {
      return 'text-amber-400 dark:text-amber-300';
    }
    if (lower.includes('passed') || lower.includes('success') || lower.includes('✓')) {
      return 'text-emerald-400 dark:text-emerald-400';
    }
    if (lower.includes('running') || lower.includes('executing') || lower.includes('cmd:')) {
      return 'text-sky-300 dark:text-sky-300';
    }
    return 'text-zinc-300 dark:text-zinc-300';
  });
</script>

<div
  class={cn(
    'group flex items-start gap-2 px-3 py-0.5 font-mono text-[11px] leading-relaxed hover:bg-zinc-800/50 transition-colors select-text',
    className
  )}
>
  {#if lineNumber !== undefined}
    <span class="w-8 shrink-0 text-right font-mono text-[10px] text-zinc-600 dark:text-zinc-600 select-none pr-1">
      {lineNumber}
    </span>
  {/if}

  <span class={cn('flex-1 break-all whitespace-pre-wrap', lineTone)}>
    {#each segments as segment, i (i)}
      {#if segment.match}
        <mark class="rounded-xs bg-amber-400 text-zinc-950 font-semibold px-0.5">{segment.text}</mark>
      {:else}
        {segment.text}
      {/if}
    {/each}
  </span>
</div>
