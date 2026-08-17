<script lang="ts">
  import { onMount } from 'svelte';
  import { Header } from '$lib/components/common';
  import { LoadView, InspectionView } from '$lib/components/views';
  import { appState } from '$lib/state/appState.svelte';
  import { themeStore } from '$lib/state/theme.svelte';
  import { apiClient } from '$lib/api/client';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Play from 'lucide-svelte/icons/play';
  import Award from 'lucide-svelte/icons/award';
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

  const maxAccessibleStep = $derived.by(() => {
    if (appState.isIdle) return 1;
    if (appState.isLoaded) return 2;
    if (appState.isRunning) return 3;
    if (appState.isCompleted) return 4;
    return 1;
  });

  const connectionState = $derived.by(() => {
    if (appState.connectionStatus === 'connected') return 'connected';
    if (appState.connectionStatus === 'reconnecting') return 'reconnecting';
    return 'disconnected';
  });

  async function handleShutdown() {
    try {
      await apiClient.shutdown();
    } catch (e) {
      console.error('[App] Shutdown request failed:', e);
    }
  }
</script>

<div class="min-h-screen flex flex-col bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100 font-sans selection:bg-sky-500/20 selection:text-sky-600 dark:selection:text-sky-300">
  <!-- Persistent Global Shell Header -->
  <Header
    {activeStep}
    playbookName={appState.playbook?.title}
    {connectionState}
    {maxAccessibleStep}
    onshutdown={handleShutdown}
  />

  <!-- Main Shell Container -->
  <main class="flex-1 max-w-5xl w-full mx-auto px-4 py-6 sm:px-6">
    <!-- STEP 1: LOAD VIEW (IDLE) -->
    {#if appState.isIdle}
      <LoadView />

    <!-- STEP 2: INSPECTION VIEW (LOADED) -->
    {:else if appState.isLoaded}
      <InspectionView />

    <!-- STEP 3: EXECUTION VIEW (RUNNING) (Phase 3D Placeholder) -->
    {:else if appState.isRunning}
      <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-8 text-center space-y-4 animate-in fade-in-50">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-sky-500/10 text-sky-400">
          <Loader2 class="h-6 w-6 animate-spin" />
        </div>
        <div class="space-y-1">
          <h3 class="text-lg font-bold text-zinc-100">Live Execution in Progress</h3>
          <p class="text-xs font-mono text-zinc-400">
            Run ID: {appState.execution?.run_id || appState.activeRunId || 'active'}
          </p>
        </div>
        <div class="text-xs text-zinc-400 max-w-md mx-auto">
          Executing assertions and streaming audit logs in real-time.
        </div>
      </div>

    <!-- STEP 4: RESULTS VIEW (COMPLETED) (Phase 3E Placeholder) -->
    {:else if appState.isCompleted}
      <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-8 text-center space-y-4 animate-in fade-in-50">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-400">
          <Award class="h-6 w-6" />
        </div>
        <div class="space-y-1">
          <h3 class="text-lg font-bold text-zinc-100">Audit Execution Completed</h3>
          <p class="text-xs font-mono text-zinc-400">
            Passed: {appState.passedAssertions} / {appState.totalAssertions}
          </p>
        </div>
        <div class="pt-2">
          <button
            type="button"
            onclick={() => appState.unloadPlaybook()}
            class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-medium rounded-md border border-zinc-700 cursor-pointer"
          >
            Load Another Playbook
          </button>
        </div>
      </div>

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
