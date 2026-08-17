<script lang="ts">
  import Shield from 'lucide-svelte/icons/shield';
  import Lock from 'lucide-svelte/icons/lock';
  import Folder from 'lucide-svelte/icons/folder';
  import Globe from 'lucide-svelte/icons/globe';
  import { Badge, Card } from '$lib/components/ui';
  import type { PlaybookInspection, ReportDestinationState } from '$lib/api/types';

  interface Props {
    playbook: PlaybookInspection;
    destination?: ReportDestinationState;
    class?: string;
  }

  let { playbook, destination, class: className = '' }: Props = $props();

  const totalAssertions = $derived.by(() => {
    if (!playbook.sections) return 0;
    return playbook.sections.reduce((acc, s) => acc + (s.assertions?.length || 0), 0);
  });

  const frontmatterEntries = $derived.by(() => {
    if (!playbook.reportFrontmatter) return [];
    return Object.entries(playbook.reportFrontmatter).filter(
      ([key, val]) =>
        typeof val === 'string' ||
        typeof val === 'number' ||
        typeof val === 'boolean'
    );
  });

  const folderStatusLabel = $derived.by(() => {
    if (!destination) return 'Default';
    switch (destination.folder_source) {
      case 'cli':
        return 'CLI-Locked';
      case 'custom':
        return 'Custom Folder';
      case 'off':
        return 'Folder Off';
      default:
        return 'Default Folder';
    }
  });

  const httpsStatusLabel = $derived.by(() => {
    if (!destination) return 'Off';
    switch (destination.https_source) {
      case 'playbook':
        return 'Playbook HTTPS';
      case 'custom':
        return 'Custom HTTPS';
      default:
        return 'HTTPS Off';
    }
  });
</script>

<Card class="space-y-3 {className}">
  <!-- Title, Description & Main Badges -->
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div class="space-y-1.5 max-w-2xl">
      <div class="flex items-center gap-2">
        <div class="flex h-7 w-7 items-center justify-center rounded bg-sky-500/10 text-sky-400 border border-sky-500/20">
          <Shield class="h-4 w-4" />
        </div>
        <h2 class="text-lg font-bold text-zinc-100 tracking-tight">
          {playbook.title || 'Untitled Playbook'}
        </h2>
      </div>

      {#if playbook.reportFrontmatter?.description}
        <p class="text-sm text-zinc-400">
          {String(playbook.reportFrontmatter.description)}
        </p>
      {/if}
    </div>

    <div class="flex flex-wrap items-center gap-2">
      {#if playbook.requiresElevation}
        <Badge variant="warning" size="sm" class="gap-1 font-medium">
          <Lock class="h-3 w-3" />
          Requires Sudo
        </Badge>
      {/if}

      <Badge variant="neutral" size="sm">
        {totalAssertions} {totalAssertions === 1 ? 'Assertion' : 'Assertions'}
      </Badge>

      <Badge variant="default" size="sm">
        {playbook.sections?.length || 0} {playbook.sections?.length === 1 ? 'Section' : 'Sections'}
      </Badge>
    </div>
  </div>

  <!-- Destination & Frontmatter Metadata Badges -->
  <div class="flex flex-wrap items-center gap-2 pt-2 border-t border-zinc-800/80 text-xs">
    <!-- Destination pills -->
    <Badge variant="outline" size="sm" class="gap-1 font-mono text-[11px] text-zinc-400">
      <Folder class="h-3 w-3 text-zinc-500" />
      {folderStatusLabel}
    </Badge>

    <Badge variant="outline" size="sm" class="gap-1 font-mono text-[11px] text-zinc-400">
      <Globe class="h-3 w-3 text-zinc-500" />
      {httpsStatusLabel}
    </Badge>

    <!-- Key-Value Frontmatter Badges -->
    {#each frontmatterEntries as [key, value]}
      {#if key !== 'description'}
        <span class="inline-flex items-center rounded bg-zinc-800/60 border border-zinc-700/50 px-2 py-0.5 text-[11px] font-mono text-zinc-300">
          <span class="text-zinc-500 mr-1">{key}:</span>
          {String(value)}
        </span>
      {/if}
    {/each}
  </div>
</Card>
