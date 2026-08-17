<script lang="ts">
  import Folder from 'lucide-svelte/icons/folder';
  import Globe from 'lucide-svelte/icons/globe';
  import Settings from 'lucide-svelte/icons/settings';
  import Lock from 'lucide-svelte/icons/lock';
  import ShieldCheck from 'lucide-svelte/icons/shield-check';
  import { Badge, Button, Card } from '$lib/components/ui';
  import type { ReportDestinationState } from '$lib/api/types';

  interface Props {
    destination: ReportDestinationState;
    onOpenDrawer?: () => void;
    class?: string;
  }

  let { destination, onOpenDrawer, class: className = '' }: Props = $props();

  const isFolderOff = $derived(destination.folder_source === 'off');
  const isHttpsOff = $derived(destination.https_source === 'off');

  const folderBadgeVariant = $derived.by(() => {
    if (destination.folder_source === 'off') return 'neutral';
    if (destination.folder_source === 'cli') return 'info';
    return 'default';
  });

  const httpsBadgeVariant = $derived.by(() => {
    if (destination.https_source === 'off') return 'neutral';
    if (destination.https_source === 'playbook') return 'passed';
    return 'info';
  });

  const destinationFolderDisplay = $derived.by(() => {
    if (destination.folder_source === 'off') return 'Disabled (reports retained in memory only)';
    if (destination.folder_source === 'cli') return destination.folder || 'Configured via CLI flag';
    if (destination.folder_source === 'custom') return destination.folder || 'Custom directory path';
    if (destination.folder_source === 'playbook') {
      return destination.playbook_defaults?.folder_path || 'Playbook default path';
    }
    return '~/.local/state/crobe/runs/';
  });

  const httpsEndpointDisplay = $derived.by(() => {
    if (destination.https_source === 'off') return 'Disabled (no remote submission)';
    if (destination.https_source === 'playbook') {
      return destination.playbook_defaults?.https?.url || 'Playbook configured endpoint';
    }
    if (destination.https_source === 'custom') {
      return destination.https?.url || 'Custom HTTPS endpoint';
    }
    return 'Not configured';
  });
</script>

<div class="space-y-2.5 {className}">
  <div class="flex items-center justify-between">
    <span class="text-xs font-semibold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
      Report Destinations
    </span>

    {#if onOpenDrawer}
      <Button
        variant="outline"
        size="xs"
        onclick={onOpenDrawer}
        class="text-xs text-zinc-700 dark:text-zinc-300 hover:text-zinc-900 dark:hover:text-white gap-1.5"
      >
        <Settings class="h-3.5 w-3.5 text-zinc-500 dark:text-zinc-400" />
        Configure Destinations
      </Button>
    {/if}
  </div>

  <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
    <!-- Local Report Folder Card -->
    <Card class="p-3.5 space-y-2">
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <Folder class="h-4 w-4 text-amber-500 dark:text-amber-400 shrink-0" />
          <span class="text-xs font-semibold text-zinc-800 dark:text-zinc-200">Local Folder Storage</span>
        </div>

        <Badge variant={folderBadgeVariant} size="sm" class="font-mono text-[10px] uppercase">
          {#if destination.folder_source === 'cli'}
            <Lock class="h-2.5 w-2.5 mr-0.5" />
            CLI-Locked
          {:else}
            {destination.folder_source}
          {/if}
        </Badge>
      </div>

      <div class="font-mono text-xs text-zinc-600 dark:text-zinc-400 truncate select-all" title={destinationFolderDisplay}>
        {destinationFolderDisplay}
      </div>
    </Card>

    <!-- Remote HTTPS Submission Card -->
    <Card class="p-3.5 space-y-2">
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <Globe class="h-4 w-4 text-sky-600 dark:text-sky-400 shrink-0" />
          <span class="text-xs font-semibold text-zinc-800 dark:text-zinc-200">Remote HTTPS Submission</span>
        </div>

        <Badge variant={httpsBadgeVariant} size="sm" class="font-mono text-[10px] uppercase">
          {destination.https_source}
        </Badge>
      </div>

      <div class="space-y-1">
        <div class="font-mono text-xs text-zinc-600 dark:text-zinc-400 truncate select-all" title={httpsEndpointDisplay}>
          {httpsEndpointDisplay}
        </div>

        {#if !isHttpsOff}
          <div class="flex items-center gap-2 text-[11px] font-mono text-zinc-500">
            <span>
              Format: <span class="text-zinc-700 dark:text-zinc-400">{destination.https?.format || destination.playbook_defaults?.https?.format || 'json'}</span>
            </span>
            {#if destination.playbook_defaults?.https?.hasSignatureSecret || destination.https?.secret}
              <span>•</span>
              <span class="inline-flex items-center text-emerald-600 dark:text-emerald-400/90 gap-0.5">
                <ShieldCheck class="h-3 w-3" />
                HMAC Signed
              </span>
            {/if}
          </div>
        {/if}
      </div>
    </Card>
  </div>
</div>
