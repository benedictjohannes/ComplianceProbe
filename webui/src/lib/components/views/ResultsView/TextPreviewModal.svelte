<script lang="ts">
  import { Dialog } from '$lib/components/ui';
  import MarkdownViewer from './MarkdownViewer.svelte';
  import FileText from 'lucide-svelte/icons/file-text';
  import Terminal from 'lucide-svelte/icons/terminal';

  interface Props {
    open?: boolean;
    title?: string;
    content: string;
    downloadUrl?: string;
    isLog?: boolean;
    onOpenChange?: (open: boolean) => void;
  }

  let {
    open = $bindable(false),
    title = 'report.md',
    content,
    downloadUrl,
    isLog = false,
    onOpenChange,
  }: Props = $props();
</script>

<Dialog
  bind:open
  maxWidth="full"
  class="max-h-[90vh] flex flex-col p-4 sm:p-6 bg-zinc-950 border-zinc-800 text-zinc-100"
  {onOpenChange}
>
  {#snippet header()}
    <div class="flex items-center gap-2.5 mb-3 pr-6 text-zinc-200">
      <div class="p-1.5 rounded-lg bg-zinc-800 border border-zinc-700 text-sky-400">
        {#if isLog}
          <Terminal class="h-4 w-4" />
        {:else}
          <FileText class="h-4 w-4" />
        {/if}
      </div>
      <div>
        <h3 class="text-base font-semibold text-zinc-100">
          {title}
        </h3>
        <p class="text-xs text-zinc-400">
          Raw document inspection with line numbering and in-file search.
        </p>
      </div>
    </div>
  {/snippet}

  <div class="flex-1 min-h-0 pt-1">
    <MarkdownViewer
      {content}
      filename={title}
      {downloadUrl}
      showFullscreenButton={false}
      class="max-h-[65vh] h-[65vh]"
    />
  </div>
</Dialog>
