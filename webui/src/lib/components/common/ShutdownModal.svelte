<script lang="ts">
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';
  import PowerOff from 'lucide-svelte/icons/power-off';
  import { Dialog, Button } from '$lib/components/ui';
  import { apiClient } from '$lib/api/client';

  interface Props {
    open?: boolean;
    onshutdown?: () => Promise<void> | void;
    onOpenChange?: (open: boolean) => void;
  }

  let {
    open = $bindable(false),
    onshutdown,
    onOpenChange,
  }: Props = $props();

  let stopping = $state(false);

  async function handleConfirm() {
    stopping = true;
    try {
      if (onshutdown) {
        await onshutdown();
      } else {
        await apiClient.shutdown();
      }
    } catch {
      // Server may drop connection immediately
    } finally {
      stopping = false;
      open = false;
    }
  }
</script>

<Dialog
  bind:open
  preventClose={stopping}
  maxWidth="md"
  {onOpenChange}
>
  {#snippet children()}
    <div class="space-y-3">
      <div class="flex items-center gap-3 text-amber-500 dark:text-amber-400">
        <div class="p-2 rounded-lg bg-amber-500/10 border border-amber-500/20">
          <AlertTriangle class="h-5 w-5" />
        </div>
        <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">Stop crobe process?</h3>
      </div>

      <p class="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed">
        This will terminate the local <span class="font-mono text-zinc-800 dark:text-zinc-200 font-medium">crobe</span> backend process and close all active connections. Any in-flight executions will be halted.
      </p>
    </div>
  {/snippet}

  {#snippet footer()}
    <Button
      variant="outline"
      size="sm"
      disabled={stopping}
      onclick={() => {
        open = false;
      }}
    >
      Keep Running
    </Button>
    <Button
      variant="destructive"
      size="sm"
      loading={stopping}
      onclick={handleConfirm}
    >
      <PowerOff class="h-3.5 w-3.5 mr-1.5" />
      Stop Process
    </Button>
  {/snippet}
</Dialog>
