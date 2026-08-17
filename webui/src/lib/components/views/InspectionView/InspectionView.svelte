<script lang="ts">
  import Play from 'lucide-svelte/icons/play';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import Settings from 'lucide-svelte/icons/settings';
  import { Button } from '$lib/components/ui';
  import { ErrorBanner } from '$lib/components/common';
  import PlaybookHeader from './PlaybookHeader.svelte';
  import DestinationSummary from './DestinationSummary.svelte';
  import SectionsTree from './SectionsTree.svelte';
  import DestinationDrawer from './DestinationDrawer.svelte';
  import { appState as defaultAppState, AppState } from '$lib/state/appState.svelte';
  import type { DestinationUpdateRequest } from '$lib/api/types';

  interface Props {
    appStateInstance?: AppState;
    class?: string;
  }

  let { appStateInstance = defaultAppState, class: className = '' }: Props = $props();

  let isDestinationDrawerOpen = $state<boolean>(false);

  const hasValidationErrors = $derived.by(() => {
    return appStateInstance.errors.some(
      (err) =>
        err.code === 'PLAYBOOK_VALIDATION_FAILED' ||
        (typeof err.code === 'string' && err.code.includes('VALIDATION'))
    );
  });

  async function handleStartRun() {
    if (appStateInstance.isLoading || hasValidationErrors) return;
    try {
      await appStateInstance.startRun();
    } catch (e) {
      console.error('[InspectionView] Failed to start execution run:', e);
    }
  }

  async function handleUnload() {
    if (appStateInstance.isLoading) return;
    try {
      await appStateInstance.unloadPlaybook();
    } catch (e) {
      console.error('[InspectionView] Failed to unload playbook:', e);
    }
  }

  async function handleSaveDestination(req: DestinationUpdateRequest) {
    await appStateInstance.updateDestination(req);
  }
</script>

<div class="space-y-6 animate-in fade-in-50 duration-200 pb-20 {className}">
  <!-- Top Validation Diagnostics Banners (if any) -->
  {#if appStateInstance.errors.length > 0}
    <div class="space-y-3">
      {#each appStateInstance.errors as err, idx (idx)}
        <ErrorBanner
          code={typeof err.code === 'string' ? err.code : undefined}
          message={err.message}
          detail={err.detail}
          onunload={handleUnload}
          ondismiss={() => appStateInstance.dismissError(idx)}
        />
      {/each}
    </div>
  {/if}

  {#if appStateInstance.playbook}
    <!-- 1. Playbook Metadata Header Card -->
    <PlaybookHeader
      playbook={appStateInstance.playbook}
      destination={appStateInstance.reportDestination}
    />

    <!-- 2. Report Destination Status Summary -->
    <DestinationSummary
      destination={appStateInstance.reportDestination}
      onOpenDrawer={() => (isDestinationDrawerOpen = true)}
    />

    <!-- 3. Progressive 3-Level Sections & Assertion Hierarchy -->
    <div class="pt-2">
      <SectionsTree sections={appStateInstance.playbook.sections || []} />
    </div>

    <!-- Slide-over Destination Configuration Drawer -->
    <DestinationDrawer
      bind:open={isDestinationDrawerOpen}
      destination={appStateInstance.reportDestination}
      onSave={handleSaveDestination}
      onClose={() => (isDestinationDrawerOpen = false)}
    />

    <!-- Fixed/Sticky Bottom Action Bar -->
    <div class="fixed bottom-0 left-0 right-0 z-40 border-t border-zinc-800 bg-zinc-950/95 backdrop-blur-sm py-3 shadow-2xl">
      <div class="max-w-5xl mx-auto px-4 sm:px-6 flex items-center justify-between gap-4">
        <!-- Left: Unload Action -->
        <Button
          variant="ghost"
          size="sm"
          disabled={appStateInstance.isLoading}
          onclick={handleUnload}
          class="text-zinc-400 hover:text-rose-400 hover:bg-rose-500/10 gap-1.5"
        >
          <Trash2 class="h-4 w-4" />
          Unload Playbook
        </Button>

        <!-- Right Action Group -->
        <div class="flex items-center gap-3">
          <Button
            variant="secondary"
            size="sm"
            disabled={appStateInstance.isLoading}
            onclick={() => (isDestinationDrawerOpen = true)}
            class="hidden sm:inline-flex gap-1.5"
          >
            <Settings class="h-4 w-4 text-zinc-400" />
            Destination Settings
          </Button>

          <Button
            variant="primary"
            size="sm"
            loading={appStateInstance.isLoading}
            disabled={appStateInstance.isLoading || hasValidationErrors}
            onclick={handleStartRun}
            class="px-5 shadow-md shadow-emerald-950/50 gap-2 font-semibold"
            title={hasValidationErrors ? 'Cannot run playbook with validation errors' : 'Execute Playbook'}
          >
            <Play class="h-4 w-4 fill-white" />
            Execute Playbook ➜
          </Button>
        </div>
      </div>
    </div>
  {:else}
    <div class="rounded-lg border border-dashed border-zinc-800 p-8 text-center text-sm text-zinc-400 space-y-3">
      <p>No playbook loaded in state.</p>
      <Button variant="secondary" size="sm" onclick={() => appStateInstance.unloadPlaybook()}>
        Go to Load Screen
      </Button>
    </div>
  {/if}
</div>
