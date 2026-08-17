<script lang="ts">
  import Globe from 'lucide-svelte/icons/globe';
  import Shield from 'lucide-svelte/icons/shield';
  import { Button } from '$lib/components/ui';
  import { ErrorBanner } from '$lib/components/common';
  import Dropzone from './Dropzone.svelte';
  import RemoteUrlDialog from './RemoteUrlDialog.svelte';
  import { appState as defaultAppState, AppState } from '$lib/state/appState.svelte';
  import type { RemotePlaybookRequest } from '$lib/api/types';

  interface Props {
    appStateInstance?: AppState;
    class?: string;
  }

  let { appStateInstance = defaultAppState, class: className = '' }: Props = $props();

  let remoteDialogOpen = $state<boolean>(false);
  let remoteDialogError = $state<string | null>(null);

  async function handleFile(file: File) {
    appStateInstance.clearErrors();
    try {
      await appStateInstance.uploadPlaybookFile(file);
    } catch (e) {
      console.error('[LoadView] File upload failed:', e);
    }
  }

  async function handleRemoteSubmit(req: RemotePlaybookRequest) {
    remoteDialogError = null;
    try {
      await appStateInstance.loadRemotePlaybook(req);
      remoteDialogOpen = false;
    } catch (e: any) {
      console.error('[LoadView] Remote load failed:', e);
      remoteDialogError = e?.message || 'Failed to load remote playbook';
    }
  }
</script>

<div class="space-y-6 animate-in fade-in-50 duration-200 {className}">
  <!-- Centered Heading & Intro -->
  <div class="text-center space-y-2 pt-4">
    <div class="inline-flex h-10 w-10 items-center justify-center rounded-lg bg-sky-500/10 text-sky-400 border border-sky-500/20 mb-1">
      <Shield class="h-5 w-5" />
    </div>
    <h2 class="text-2xl font-bold tracking-tight text-zinc-100">
      Ingest Compliance Playbook
    </h2>
    <p class="text-sm text-zinc-400 max-w-md mx-auto">
      Upload or fetch a YAML or JSON compliance definition to inspect assertions and execute audits.
    </p>
  </div>

  <!-- Primary Dropzone -->
  <Dropzone
    loading={appStateInstance.isLoading}
    disabled={appStateInstance.isLoading}
    onfile={handleFile}
  />

  <!-- Divider -->
  <div class="max-w-xl mx-auto flex items-center gap-3">
    <div class="flex-1 h-px bg-zinc-800"></div>
    <span class="text-[11px] font-mono uppercase text-zinc-500">or</span>
    <div class="flex-1 h-px bg-zinc-800"></div>
  </div>

  <!-- Remote HTTPS Action Trigger -->
  <div class="text-center">
    <Button
      variant="outline"
      size="sm"
      disabled={appStateInstance.isLoading}
      onclick={() => {
        remoteDialogError = null;
        remoteDialogOpen = true;
      }}
    >
      <Globe class="h-3.5 w-3.5 mr-1.5 text-sky-400" />
      Fetch from HTTPS URL...
    </Button>
  </div>

  <!-- Remote URL Dialog Modal -->
  <RemoteUrlDialog
    bind:open={remoteDialogOpen}
    loading={appStateInstance.isLoading}
    error={remoteDialogError}
    onsubmit={handleRemoteSubmit}
    oncancel={() => {
      remoteDialogOpen = false;
      remoteDialogError = null;
    }}
  />

  <!-- Contextual Parsing / Loading Error Banners -->
  {#if appStateInstance.errors.length > 0}
    <div class="max-w-xl mx-auto space-y-3 pt-2">
      {#each appStateInstance.errors as err, idx (idx)}
        <ErrorBanner
          code={typeof err.code === 'string' ? err.code : undefined}
          message={err.message}
          detail={err.detail}
          ondismiss={() => appStateInstance.dismissError(idx)}
        />
      {/each}
    </div>
  {/if}
</div>
