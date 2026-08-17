<script lang="ts">
  import Check from 'lucide-svelte/icons/check';
  import Copy from 'lucide-svelte/icons/copy';
  import { cn } from '$lib/utils/cn';

  export type CodeVariant = 'default' | 'amber' | 'purple' | 'sky' | 'emerald' | 'rose' | 'terminal';

  interface Props {
    code: string;
    beforeText?: string;
    language?: string;
    variant?: CodeVariant;
    copyable?: boolean;
    tryInline?: boolean;
    class?: string;
  }

  let {
    code,
    beforeText,
    language,
    variant = 'default',
    copyable = true,
    tryInline = false,
    class: className = '',
  }: Props = $props();

  let copied = $state(false);
  let timer: ReturnType<typeof setTimeout> | null = null;

  const isMultiline = $derived(code ? code.trim().includes('\n') : false);
  const renderInline = $derived(tryInline && !isMultiline);

  async function handleCopy() {
    if (!code) return;
    try {
      await navigator.clipboard.writeText(code);
      copied = true;
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => {
        copied = false;
      }, 2000);
    } catch (e) {
      console.error('[CodeBlock] Copy failed:', e);
    }
  }

  const variantClasses = $derived.by(() => {
    switch (variant) {
      case 'amber':
        return 'text-amber-800 dark:text-amber-200 bg-amber-500/10 dark:bg-amber-500/15 border-amber-500/20';
      case 'purple':
        return 'text-purple-800 dark:text-purple-200 bg-purple-500/10 dark:bg-purple-500/15 border-purple-500/20';
      case 'sky':
        return 'text-sky-800 dark:text-sky-200 bg-sky-500/10 dark:bg-sky-500/15 border-sky-500/20';
      case 'emerald':
        return 'text-emerald-800 dark:text-emerald-200 bg-emerald-500/10 dark:bg-emerald-500/15 border-emerald-500/20';
      case 'rose':
        return 'text-rose-800 dark:text-rose-200 bg-rose-500/10 dark:bg-rose-500/15 border-rose-500/20';
      case 'terminal':
        return 'bg-zinc-900 dark:bg-zinc-950 border-zinc-800 text-zinc-100 font-mono';
      case 'default':
      default:
        return 'text-zinc-800 dark:text-zinc-100 bg-zinc-100/90 dark:bg-zinc-900 border-zinc-200 dark:border-zinc-800';
    }
  });
</script>

{#if renderInline}
  <span class="inline-flex items-center gap-1.5 align-baseline">
    {#if beforeText}
      <span class="font-normal">{beforeText}</span>
    {/if}
    <code
      class={cn(
        'px-1 py-0.5 rounded border font-mono text-[11px] select-text transition-colors inline-block',
        variantClasses,
        className
      )}
    >
      {code}
    </code>
  </span>
{:else}
  <div class="space-y-1 w-full">
    {#if beforeText}
      <div class="text-[11px] font-semibold text-zinc-600 dark:text-zinc-400">
        {beforeText}
      </div>
    {/if}
    <div
      class={cn(
        'relative group rounded-md border p-2.5 font-mono text-xs overflow-x-auto select-text whitespace-pre-wrap transition-colors',
        variantClasses,
        className
      )}
    >
      {#if language}
        <span
          class="absolute top-1.5 right-8 text-[10px] text-zinc-400 dark:text-zinc-500 uppercase font-sans tracking-wider font-semibold pointer-events-none select-none opacity-60 group-hover:opacity-100 transition-opacity"
        >
          {language}
        </span>
      {/if}

      {#if copyable && code}
        <button
          type="button"
          onclick={handleCopy}
          aria-label="Copy code"
          class="absolute top-1.5 right-1.5 p-1 rounded bg-zinc-200/80 dark:bg-zinc-800/80 hover:bg-zinc-300 dark:hover:bg-zinc-700 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all cursor-pointer"
        >
          {#if copied}
            <Check class="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
          {:else}
            <Copy class="h-3 w-3" />
          {/if}
        </button>
      {/if}

      <code>{code}</code>
    </div>
  </div>
{/if}
