<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { appState } from '$lib/state/appState.svelte';
  import LogControls from './LogControls.svelte';
  import LogEntry from './LogEntry.svelte';
  import { cn } from '$lib/utils/cn';

  interface Props {
    logs?: string[];
    initialExpanded?: boolean;
    class?: string;
  }

  let {
    logs,
    initialExpanded = false,
    class: className = '',
  }: Props = $props();

  const activeLogs = $derived(logs ?? appState.logs);

  // svelte-ignore state_referenced_locally
  let isExpanded = $state(initialExpanded);
  let isMaximized = $state(false);
  let autoScroll = $state(true);
  let searchQuery = $state('');
  let currentMatchIndex = $state(0);
  let isCopied = $state(false);
  let copyTimer: ReturnType<typeof setTimeout> | null = null;

  let viewportEl: HTMLDivElement | null = $state(null);

  // Filter matching line indices
  const matchingIndices = $derived.by(() => {
    if (!searchQuery || !searchQuery.trim()) return [];
    const query = searchQuery.toLowerCase().trim();
    const indices: number[] = [];
    activeLogs.forEach((line, index) => {
      if (line.toLowerCase().includes(query)) {
        indices.push(index);
      }
    });
    return indices;
  });

  const matchCount = $derived(matchingIndices.length);

  // Keep currentMatchIndex in bounds
  $effect(() => {
    if (matchingIndices.length === 0) {
      currentMatchIndex = 0;
    } else if (currentMatchIndex >= matchingIndices.length) {
      currentMatchIndex = matchingIndices.length - 1;
    }
  });

  // Auto-scroll when logs change if auto-scroll is enabled
  $effect(() => {
    // Read length to establish dependency
    const _len = activeLogs.length;
    if (autoScroll && isExpanded && viewportEl) {
      tick().then(() => {
        if (viewportEl) {
          viewportEl.scrollTop = viewportEl.scrollHeight;
        }
      });
    }
  });

  function handleScroll() {
    if (!viewportEl) return;
    const { scrollTop, scrollHeight, clientHeight } = viewportEl;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 25;
    if (!isAtBottom && autoScroll) {
      autoScroll = false;
    } else if (isAtBottom && !autoScroll) {
      autoScroll = true;
    }
  }

  function toggleAutoScroll() {
    autoScroll = !autoScroll;
    if (autoScroll && viewportEl) {
      viewportEl.scrollTop = viewportEl.scrollHeight;
    }
  }

  function handleToggleExpand() {
    isExpanded = !isExpanded;
    if (!isExpanded) {
      isMaximized = false;
    } else if (autoScroll) {
      tick().then(() => {
        if (viewportEl) {
          viewportEl.scrollTop = viewportEl.scrollHeight;
        }
      });
    }
  }

  function handleToggleMaximize() {
    isMaximized = !isMaximized;
    if (isMaximized) {
      isExpanded = true;
    }
  }

  function scrollToMatchedLine(targetLineIndex: number) {
    if (!viewportEl) return;
    const lineEl = viewportEl.querySelector(`[data-line-index="${targetLineIndex}"]`);
    if (lineEl && typeof lineEl.scrollIntoView === 'function') {
      autoScroll = false;
      lineEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }

  function handlePrevMatch() {
    if (matchingIndices.length === 0) return;
    currentMatchIndex = (currentMatchIndex - 1 + matchingIndices.length) % matchingIndices.length;
    scrollToMatchedLine(matchingIndices[currentMatchIndex]);
  }

  function handleNextMatch() {
    if (matchingIndices.length === 0) return;
    currentMatchIndex = (currentMatchIndex + 1) % matchingIndices.length;
    scrollToMatchedLine(matchingIndices[currentMatchIndex]);
  }

  async function handleCopy() {
    if (activeLogs.length === 0) return;
    try {
      await navigator.clipboard.writeText(activeLogs.join('\n'));
      isCopied = true;
      if (copyTimer) clearTimeout(copyTimer);
      copyTimer = setTimeout(() => {
        isCopied = false;
      }, 2000);
    } catch (e) {
      console.error('[LogStream] Copy failed:', e);
    }
  }

  // Keyboard shortcut: Escape to collapse/un-maximize
  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape' && (isMaximized || isExpanded)) {
      if (isMaximized) {
        isMaximized = false;
      } else {
        isExpanded = false;
      }
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

<div
  class={cn(
    'fixed inset-x-0 bottom-0 z-30 transition-all duration-200 border-t border-zinc-300 dark:border-zinc-800 bg-white dark:bg-zinc-950 shadow-2xl flex flex-col',
    isMaximized ? 'h-[75vh]' : isExpanded ? 'h-64 sm:h-72' : 'h-10',
    className
  )}
>
  <!-- Sticky Header Controls -->
  <LogControls
    lineCount={activeLogs.length}
    {autoScroll}
    onToggleAutoScroll={toggleAutoScroll}
    {searchQuery}
    onSearchChange={(q) => (searchQuery = q)}
    {matchCount}
    {currentMatchIndex}
    onPrevMatch={handlePrevMatch}
    onNextMatch={handleNextMatch}
    onCopy={handleCopy}
    {isCopied}
    {isMaximized}
    onToggleMaximize={handleToggleMaximize}
    {isExpanded}
    onToggleExpand={handleToggleExpand}
  />

  <!-- Terminal Canvas Body -->
  {#if isExpanded}
    <div
      bind:this={viewportEl}
      onscroll={handleScroll}
      class="flex-1 overflow-y-auto bg-zinc-950 text-zinc-300 font-mono text-xs select-text p-2 scroll-smooth"
    >
      {#if activeLogs.length === 0}
        <div class="h-full flex items-center justify-center text-zinc-600 dark:text-zinc-600 text-xs font-mono py-8 select-none">
          No logs available yet. Execution logs will stream here in real-time.
        </div>
      {:else}
        {#each activeLogs as logLine, i (i)}
          <div data-line-index={i}>
            <LogEntry
              message={logLine}
              lineNumber={i + 1}
              {searchQuery}
            />
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>
