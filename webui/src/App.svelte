<script lang="ts">
  import { onMount } from 'svelte';
  import { Header } from '$lib/components/common';
  import { LoadView, InspectionView, ExecutionView, ResultsView, TerminatedView } from '$lib/components/views';
  import { appState } from '$lib/state/appState.svelte';
  import { themeStore } from '$lib/state/theme.svelte';
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';

  // Extract query token if present on initial load
  onMount(() => {
    themeStore.init();

    let token: string | undefined;
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      token = params.get('token') || undefined;
    }

    appState.init(token);

    return () => {
      appState.destroy();
    };
  });

  const activeStep = $derived.by(() => {
    return appState.currentPipelineStep + 1;
  });

  const connectionState = $derived.by(() => {
    if (appState.isTerminated) return 'disconnected';
    if (appState.connectionStatus === 'connected') return 'connected';
    if (appState.connectionStatus === 'reconnecting') return 'reconnecting';
    return 'disconnected';
  });

  async function handleShutdown() {
    await appState.shutdownServer();
  }
</script>

<div class="min-h-screen flex flex-col bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100 font-sans selection:bg-sky-500/20 selection:text-sky-600 dark:selection:text-sky-300">
  <!-- Persistent Global Shell Header -->
  <Header
    {activeStep}
    playbookName={appState.playbook?.title}
    {connectionState}
    onshutdown={handleShutdown}
  />

  <!-- Main Shell Container -->
  <main class="flex-1 max-w-5xl w-full mx-auto px-4 py-6 sm:px-6">
    <!-- TERMINATED SERVER STATE -->
    {#if appState.isTerminated}
      <TerminatedView />

    <!-- STEP 1: LOAD VIEW (IDLE) -->
    {:else if appState.isIdle}
      <LoadView />

    <!-- STEP 2: INSPECTION VIEW (LOADED) -->
    {:else if appState.isLoaded}
      <InspectionView />

    <!-- STEP 3: EXECUTION VIEW (RUNNING) -->
    {:else if appState.isRunning}
      <ExecutionView />

    <!-- STEP 4: RESULTS VIEW (COMPLETED) -->
    {:else if appState.isCompleted}
      <ResultsView />

    <!-- FATAL SYSTEM ERROR STATE -->
    {:else if appState.isError}
      <div class="rounded-xl border border-rose-500/40 bg-rose-500/10 p-8 text-center space-y-4 animate-in fade-in-50">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-rose-500/20 text-rose-400">
          <AlertTriangle class="h-6 w-6" />
        </div>
        <div class="space-y-1">
          <h3 class="text-lg font-bold text-rose-200">Fatal System Error</h3>
          <p class="text-xs text-rose-300">An unrecoverable error occurred during execution.</p>
        </div>
        <div class="pt-2">
          <button
            type="button"
            onclick={() => appState.unloadPlaybook()}
            class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-medium rounded-md border border-zinc-700 cursor-pointer"
          >
            Reset Application State
          </button>
        </div>
      </div>
    {/if}
  </main>
</div>
