<script lang="ts">
  import Terminal from 'lucide-svelte/icons/terminal';
  import Search from 'lucide-svelte/icons/search';
  import ChevronUp from 'lucide-svelte/icons/chevron-up';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Copy from 'lucide-svelte/icons/copy';
  import Check from 'lucide-svelte/icons/check';
  import Maximize2 from 'lucide-svelte/icons/maximize-2';
  import Minimize2 from 'lucide-svelte/icons/minimize-2';
  import ArrowDownToLine from 'lucide-svelte/icons/arrow-down-to-line';
  import X from 'lucide-svelte/icons/x';
  import { cn } from '$lib/utils/cn';

  interface Props {
    lineCount: number;
    autoScroll: boolean;
    onToggleAutoScroll: () => void;
    searchQuery: string;
    onSearchChange: (query: string) => void;
    matchCount: number;
    currentMatchIndex: number;
    onPrevMatch: () => void;
    onNextMatch: () => void;
    onCopy: () => void;
    isCopied: boolean;
    isMaximized: boolean;
    onToggleMaximize: () => void;
    isExpanded: boolean;
    onToggleExpand: () => void;
    class?: string;
  }

  let {
    lineCount,
    autoScroll,
    onToggleAutoScroll,
    searchQuery,
    onSearchChange,
    matchCount,
    currentMatchIndex,
    onPrevMatch,
    onNextMatch,
    onCopy,
    isCopied,
    isMaximized,
    onToggleMaximize,
    isExpanded,
    onToggleExpand,
    class: className = '',
  }: Props = $props();
</script>

<div
  class={cn(
    'flex items-center justify-between gap-2 px-3 py-1.5 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900/90 select-none text-xs text-zinc-700 dark:text-zinc-300',
    className
  )}
>
  <!-- Left: Drawer Header & Line Count Trigger -->
  <button
    type="button"
    onclick={onToggleExpand}
    class="flex items-center gap-2 font-mono text-xs font-semibold text-zinc-800 dark:text-zinc-200 hover:text-sky-600 dark:hover:text-sky-400 transition-colors cursor-pointer"
  >
    <Terminal class="h-3.5 w-3.5 text-zinc-500 dark:text-zinc-400" />
    <span>Console Logs</span>
    <span class="text-[11px] font-normal text-zinc-500 dark:text-zinc-400">
      ({lineCount} {lineCount === 1 ? 'line' : 'lines'})
    </span>
  </button>

  <!-- Center: Search & Navigation (Visible when expanded) -->
  {#if isExpanded}
    <div class="flex items-center gap-1.5 flex-1 max-w-xs mx-2">
      <div class="relative w-full flex items-center">
        <Search class="absolute left-2 h-3.5 w-3.5 text-zinc-400 pointer-events-none" />
        <input
          type="text"
          value={searchQuery}
          oninput={(e) => onSearchChange(e.currentTarget.value)}
          placeholder="Search logs..."
          aria-label="Search logs"
          class="w-full pl-7 pr-7 py-1 text-[11px] font-mono rounded bg-white dark:bg-zinc-950 border border-zinc-300 dark:border-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 focus:outline-none focus:ring-1 focus:ring-sky-500 transition-all"
        />
        {#if searchQuery}
          <button
            type="button"
            onclick={() => onSearchChange('')}
            aria-label="Clear search"
            class="absolute right-2 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
          >
            <X class="h-3 w-3" />
          </button>
        {/if}
      </div>

      {#if searchQuery.trim()}
        <div class="flex items-center gap-1 text-[10px] text-zinc-500 dark:text-zinc-400 shrink-0 font-mono">
          {#if matchCount > 0}
            <span>{currentMatchIndex + 1}/{matchCount}</span>
            <button
              type="button"
              onclick={onPrevMatch}
              aria-label="Previous match"
              class="p-0.5 rounded hover:bg-zinc-200 dark:hover:bg-zinc-800 text-zinc-600 dark:text-zinc-300 cursor-pointer"
            >
              <ChevronLeft class="h-3 w-3" />
            </button>
            <button
              type="button"
              onclick={onNextMatch}
              aria-label="Next match"
              class="p-0.5 rounded hover:bg-zinc-200 dark:hover:bg-zinc-800 text-zinc-600 dark:text-zinc-300 cursor-pointer"
            >
              <ChevronRight class="h-3 w-3" />
            </button>
          {:else}
            <span class="text-rose-500 dark:text-rose-400">0 matches</span>
          {/if}
        </div>
      {/if}
    </div>
  {/if}

  <!-- Right: Control Actions -->
  <div class="flex items-center gap-1 shrink-0">
    <!-- Auto-Scroll Toggle -->
    <button
      type="button"
      onclick={onToggleAutoScroll}
      aria-label="Toggle auto scroll"
      class={cn(
        'flex items-center gap-1 px-2 py-1 rounded text-[11px] font-medium border transition-colors cursor-pointer',
        autoScroll
          ? 'bg-sky-500/10 text-sky-600 dark:text-sky-400 border-sky-500/30'
          : 'bg-zinc-200/60 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border-zinc-300 dark:border-zinc-700 hover:bg-zinc-200 dark:hover:bg-zinc-700'
      )}
      title={autoScroll ? 'Auto-scroll is ON' : 'Auto-scroll is PAUSED'}
    >
      <ArrowDownToLine class="h-3 w-3" />
      <span class="hidden sm:inline">Auto-scroll: {autoScroll ? 'ON' : 'OFF'}</span>
    </button>

    <!-- Copy All Logs -->
    <button
      type="button"
      onclick={onCopy}
      aria-label="Copy logs"
      class="flex items-center gap-1 px-2 py-1 rounded text-[11px] font-medium bg-zinc-200/60 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-300 dark:hover:bg-zinc-700 border border-zinc-300 dark:border-zinc-700 transition-colors cursor-pointer"
      title="Copy all logs to clipboard"
    >
      {#if isCopied}
        <Check class="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
        <span class="text-emerald-600 dark:text-emerald-400 font-semibold">Copied!</span>
      {:else}
        <Copy class="h-3 w-3" />
        <span class="hidden sm:inline">Copy</span>
      {/if}
    </button>

    <!-- Maximize / Restore Toggle (Visible when expanded) -->
    {#if isExpanded}
      <button
        type="button"
        onclick={onToggleMaximize}
        aria-label={isMaximized ? 'Restore console height' : 'Maximize console height'}
        class="p-1 rounded text-zinc-600 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-800 hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors cursor-pointer"
        title={isMaximized ? 'Restore height' : 'Maximize drawer'}
      >
        {#if isMaximized}
          <Minimize2 class="h-3.5 w-3.5" />
        {:else}
          <Maximize2 class="h-3.5 w-3.5" />
        {/if}
      </button>
    {/if}

    <!-- Expand / Collapse Toggle -->
    <button
      type="button"
      onclick={onToggleExpand}
      aria-label={isExpanded ? 'Collapse console drawer' : 'Expand console drawer'}
      class="p-1 rounded text-zinc-600 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-800 hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors cursor-pointer"
      title={isExpanded ? 'Collapse drawer' : 'Expand drawer'}
    >
      {#if isExpanded}
        <ChevronDown class="h-4 w-4" />
      {:else}
        <ChevronUp class="h-4 w-4" />
      {/if}
    </button>
  </div>
</div>
