<script lang="ts">
  import Copy from 'lucide-svelte/icons/copy';
  import Check from 'lucide-svelte/icons/check';
  import Download from 'lucide-svelte/icons/download';
  import Search from 'lucide-svelte/icons/search';
  import ChevronUp from 'lucide-svelte/icons/chevron-up';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import Maximize2 from 'lucide-svelte/icons/maximize-2';
  import { cn } from '$lib/utils/cn';

  interface Props {
    content: string;
    filename?: string;
    downloadUrl?: string;
    showFullscreenButton?: boolean;
    onFullscreen?: () => void;
    class?: string;
  }

  let {
    content,
    filename = 'report.md',
    downloadUrl,
    showFullscreenButton = false,
    onFullscreen,
    class: className = '',
  }: Props = $props();

  let searchQuery = $state('');
  let currentMatchIndex = $state(0);
  let copied = $state(false);

  // Split lines
  const lines = $derived(content.split('\n'));

  // Calculate matches
  const matches = $derived.by(() => {
    if (!searchQuery.trim()) return [];
    const query = searchQuery.toLowerCase();
    const result: Array<{ lineIndex: number; charIndex: number }> = [];
    lines.forEach((line, lineIndex) => {
      let startIndex = 0;
      const lowerLine = line.toLowerCase();
      while (startIndex < lowerLine.length) {
        const found = lowerLine.indexOf(query, startIndex);
        if (found === -1) break;
        result.push({ lineIndex, charIndex: found });
        startIndex = found + query.length;
      }
    });
    return result;
  });

  const totalMatches = $derived(matches.length);

  function handleNextMatch() {
    if (totalMatches === 0) return;
    currentMatchIndex = (currentMatchIndex + 1) % totalMatches;
    scrollToMatch(currentMatchIndex);
  }

  function handlePrevMatch() {
    if (totalMatches === 0) return;
    currentMatchIndex = (currentMatchIndex - 1 + totalMatches) % totalMatches;
    scrollToMatch(currentMatchIndex);
  }

  function scrollToMatch(index: number) {
    if (!matches[index]) return;
    const match = matches[index];
    const el = document.getElementById(`doc-line-${match.lineIndex}`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(content);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 2000);
    } catch {
      // Fallback
      const textarea = document.createElement('textarea');
      textarea.value = content;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 2000);
    }
  }

  function handleDownload() {
    if (downloadUrl) {
      const a = document.createElement('a');
      a.href = downloadUrl;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    } else {
      const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }
  }
</script>

<div
  class={cn(
    'flex flex-col rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-950 text-zinc-300 font-mono text-xs overflow-hidden shadow-inner',
    className
  )}
>
  <!-- Sticky Viewer Header Toolbar -->
  <div class="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5 bg-zinc-900 border-b border-zinc-800 text-zinc-400 select-none">
    <!-- Left: Document Meta -->
    <div class="flex items-center gap-3">
      <span class="font-semibold text-zinc-200">
        {filename}
      </span>
      <span class="text-[11px] text-zinc-500">
        {lines.length} lines • {(new Blob([content]).size / 1024).toFixed(1)} KB
      </span>
    </div>

    <!-- Right: Search & Actions -->
    <div class="flex items-center gap-2">
      <!-- Search Input -->
      <div class="relative flex items-center">
        <Search class="absolute left-2.5 h-3.5 w-3.5 text-zinc-500 pointer-events-none" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search document..."
          class="h-7 w-36 sm:w-48 pl-8 pr-16 rounded bg-zinc-950/90 border border-zinc-700/80 text-[11px] text-zinc-200 placeholder:text-zinc-500 focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500/30"
          onkeydown={(e) => {
            if (e.key === 'Enter') {
              if (e.shiftKey) handlePrevMatch();
              else handleNextMatch();
            }
          }}
        />

        <!-- Match Counter & Steppers -->
        {#if searchQuery.trim()}
          <div class="absolute right-1.5 flex items-center gap-1 text-[10px] text-zinc-400">
            <span>
              {totalMatches > 0 ? `${currentMatchIndex + 1}/${totalMatches}` : '0'}
            </span>
            <button
              type="button"
              onclick={handlePrevMatch}
              disabled={totalMatches === 0}
              class="p-0.5 rounded hover:bg-zinc-800 disabled:opacity-40 cursor-pointer"
              title="Previous match (Shift+Enter)"
            >
              <ChevronUp class="h-3 w-3" />
            </button>
            <button
              type="button"
              onclick={handleNextMatch}
              disabled={totalMatches === 0}
              class="p-0.5 rounded hover:bg-zinc-800 disabled:opacity-40 cursor-pointer"
              title="Next match (Enter)"
            >
              <ChevronDown class="h-3 w-3" />
            </button>
          </div>
        {/if}
      </div>

      <!-- Copy Button -->
      <button
        type="button"
        onclick={handleCopy}
        class="h-7 inline-flex items-center gap-1.5 px-2.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 text-[11px] font-medium transition cursor-pointer"
        title="Copy document content"
      >
        {#if copied}
          <Check class="h-3.5 w-3.5 text-emerald-400" />
          <span class="text-emerald-400">Copied!</span>
        {:else}
          <Copy class="h-3.5 w-3.5" />
          <span>Copy</span>
        {/if}
      </button>

      <!-- Download Button -->
      <button
        type="button"
        onclick={handleDownload}
        class="h-7 inline-flex items-center gap-1.5 px-2.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 text-[11px] font-medium transition cursor-pointer"
        title="Download file"
      >
        <Download class="h-3.5 w-3.5" />
        <span class="hidden sm:inline">Download</span>
      </button>

      <!-- Fullscreen / Modal Trigger -->
      {#if showFullscreenButton && onFullscreen}
        <button
          type="button"
          onclick={onFullscreen}
          class="h-7 inline-flex items-center justify-center p-1.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 transition cursor-pointer"
          title="Open fullscreen preview"
        >
          <Maximize2 class="h-3.5 w-3.5" />
        </button>
      {/if}
    </div>
  </div>

  <!-- Document Line-by-Line Gutter Canvas -->
  <div class="overflow-x-auto overflow-y-auto max-h-[560px] p-4 select-text leading-relaxed font-mono">
    <table class="w-full border-collapse">
      <tbody>
        {#each lines as line, i}
          <tr id="doc-line-{i}" class="hover:bg-zinc-900/60 transition-colors group">
            <!-- Line Number Gutter -->
            <td class="w-12 select-none text-right pr-4 text-zinc-600 group-hover:text-zinc-400 font-mono text-[11px] border-r border-zinc-800/80 align-top">
              {i + 1}
            </td>

            <!-- Line Content -->
            <td class="pl-4 whitespace-pre text-zinc-300 font-mono text-[11px] align-top">
              {#if searchQuery.trim() && line.toLowerCase().includes(searchQuery.toLowerCase())}
                {@const parts = line.split(new RegExp(`(${searchQuery.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&')})`, 'gi'))}
                {#each parts as part}
                  {#if part.toLowerCase() === searchQuery.toLowerCase()}
                    <mark class="bg-amber-400/30 text-amber-200 rounded-xs px-0.5 font-bold">
                      {part}
                    </mark>
                  {:else}
                    {part}
                  {/if}
                {/each}
              {:else}
                {line || '\n'}
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
