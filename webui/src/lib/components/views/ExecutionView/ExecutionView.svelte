<script lang="ts">
  import { appState } from '$lib/state/appState.svelte';
  import ErrorBanner from '$lib/components/common/ErrorBanner.svelte';
  import { LogStream } from '$lib/components/common/LogStream';
  import ProgressBar from './ProgressBar.svelte';
  import AssertionList from './AssertionList.svelte';
  import CancelConfirmModal from './CancelConfirmModal.svelte';
  import ElevationPrompt from './ElevationPrompt.svelte';
  import { cn } from '$lib/utils/cn';

  interface Props {
    class?: string;
  }

  let { class: className = '' }: Props = $props();

  let showCancelModal = $state(false);

  function handleOpenCancelModal() {
    showCancelModal = true;
  }

  async function handleConfirmCancel() {
    try {
      await appState.cancelRun();
    } catch (e) {
      console.error('[ExecutionView] Cancel run failed:', e);
    }
  }
</script>

<div class={cn('space-y-6 pb-28', className)}>
  <!-- Error Banners (if any runtime errors occur) -->
  {#if appState.hasErrors}
    <div class="space-y-2">
      {#each appState.errors as error, i (i)}
        <ErrorBanner
          code={error.code}
          message={error.message}
          detail={error.detail}
          ondismiss={() => appState.dismissError(i)}
        />
      {/each}
    </div>
  {/if}

  <!-- Sticky Execution Metric Header & Progress Bar -->
  <ProgressBar
    onCancelRequest={handleOpenCancelModal}
  />

  <!-- Main Assertion Progression Tree -->
  <AssertionList />

  <!-- Cancellation Confirmation Dialog -->
  <CancelConfirmModal
    bind:open={showCancelModal}
    onConfirm={handleConfirmCancel}
  />

  <!-- OS Elevation Pending Backdrop/Modal -->
  <ElevationPrompt
    open={appState.isElevating}
  />

  <!-- Bottom Docked Terminal Log Drawer -->
  <LogStream />
</div>
