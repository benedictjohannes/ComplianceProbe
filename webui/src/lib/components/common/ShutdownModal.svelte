<script lang="ts">
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';
  import PowerOff from 'lucide-svelte/icons/power-off';
  import X from 'lucide-svelte/icons/x';
  import { Dialog, Button } from '$lib/components/ui';

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
  let stopped = $state(false);

  async function handleConfirm() {
    stopping = true;
    try {
      if (onshutdown) {
        await onshutdown();
      } else {
        await fetch('/api/shutdown', { method: 'POST' });
      }
      stopped = true;
    } catch {
      // Server might immediately close connection on shutdown
      stopped = true;
    } finally {
      stopping = false;
    }
  }

  function handleCloseWindow() {
    if (typeof window !== 'undefined') {
      window.close();
    }
  }
</script>

<Dialog
  bind:open
  preventClose={stopped || stopping}
  maxWidth="md"
  {onOpenChange}
>
  {#if stopped}
    <div class="py-6 text-center space-y-4">
      <div class="inline-flex h-12 w-12 items-center justify-center rounded-full bg-rose-500/10 text-rose-400 border border-rose-500/20">
        <PowerOff class="h-6 w-6" />
      </div>
      <div class="space-y-1">
        <h3 class="text-lg font-semibold text-zinc-100">Server Stopped</h3>
        <p class="text-sm text-zinc-400 max-w-xs mx-auto">
          The <span class="font-mono text-zinc-300">crobe</span> local server process has terminated. You can safely close this browser tab.
        </p>
      </div>
      <div class="pt-4">
        <Button variant="secondary" onclick={handleCloseWindow} class="mx-auto">
          <X class="h-4 w-4 mr-1.5" />
          Close Window
        </Button>
      </div>
    </div>
  {:else}
    <div class="space-y-3">
      <div class="flex items-center gap-3 text-amber-400">
        <div class="p-2 rounded-lg bg-amber-500/10 border border-amber-500/20">
          <AlertTriangle class="h-5 w-5" />
        </div>
        <h3 class="text-base font-semibold text-zinc-100">Stop Compliance Probe Server?</h3>
      </div>

      <p class="text-sm text-zinc-400 leading-relaxed">
        This will terminate the local <span class="font-mono text-zinc-300 font-medium">crobe</span> backend process and close all active connections. Any in-flight executions will be halted.
      </p>
    </div>

    {#snippet footer()}
      <Button
        variant="ghost"
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
        Stop Server
      </Button>
    {/snippet}
  {/if}
</Dialog>
