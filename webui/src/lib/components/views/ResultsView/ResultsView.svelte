<script lang="ts">
  import { appState } from '$lib/state/appState.svelte';
  import { apiClient } from '$lib/api/client';
  import Scorecard from './Scorecard.svelte';
  import ReportTabs from './ReportTabs.svelte';
  import ExportDropdown from './ExportDropdown.svelte';
  import SubmissionPrompt from './SubmissionPrompt.svelte';
  import TextPreviewModal from './TextPreviewModal.svelte';
  import RotateCw from 'lucide-svelte/icons/rotate-cw';
  import FolderOpen from 'lucide-svelte/icons/folder-open';
  import Send from 'lucide-svelte/icons/send';
  import CheckCircle2 from 'lucide-svelte/icons/check-circle-2';
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import { cn } from '$lib/utils/cn';

  interface Props {
    class?: string;
  }

  let { class: className = '' }: Props = $props();

  let isReRunning = $state(false);
  let isUnloading = $state(false);

  // Dialog states
  let showSubmissionPrompt = $state(false);
  let showTextPreviewModal = $state(false);
  let previewModalTitle = $state('report.md');
  let previewModalContent = $state('');
  let previewModalDownloadUrl = $state<string | undefined>(undefined);
  let previewModalIsLog = $state(false);

  // Check if remote HTTPS destination is actively configured
  const hasRemoteDestination = $derived.by(() => {
    if (appState.reportDestination.https_source === 'custom' && appState.reportDestination.https?.url) {
      return true;
    }
    if (appState.reportDestination.https_source === 'playbook' && (appState.reportDestination.playbook_defaults?.has_https || appState.playbook?.reportDestinationHttps?.url)) {
      return true;
    }
    return false;
  });

  async function handleReRun() {
    isReRunning = true;
    try {
      await appState.startRun();
    } catch (e) {
      console.error('[ResultsView] Failed to restart execution:', e);
    } finally {
      isReRunning = false;
    }
  }

  async function handleLoadAnother() {
    isUnloading = true;
    try {
      await appState.unloadPlaybook();
    } catch (e) {
      console.error('[ResultsView] Failed to unload playbook:', e);
    } finally {
      isUnloading = false;
    }
  }

  async function handleOpenMarkdownModal(content?: string) {
    let docContent = content;
    if (!docContent) {
      try {
        docContent = await apiClient.getReportMarkdown();
      } catch {
        docContent = '# Compliance Report\n\nNo report available.';
      }
    }
    previewModalTitle = 'report.md';
    previewModalContent = docContent;
    previewModalDownloadUrl = '/api/report/md?download=1';
    previewModalIsLog = false;
    showTextPreviewModal = true;
  }

  async function handleOpenLogsModal(content?: string) {
    let logContent = content;
    if (!logContent) {
      try {
        logContent = await apiClient.getReportLog();
      } catch {
        logContent = appState.logs.join('\n') || 'No logs available.';
      }
    }
    previewModalTitle = 'report.log';
    previewModalContent = logContent;
    previewModalDownloadUrl = '/api/report/log?download=1';
    previewModalIsLog = true;
    showTextPreviewModal = true;
  }
</script>

<div class={cn('space-y-6 animate-in fade-in-50 pb-20', className)}>
  <!-- 1. Top Action Bar: Workflow & Lifecycle -->
  <div class="flex flex-wrap items-center justify-between gap-3 p-3 rounded-xl bg-white/70 dark:bg-zinc-900/60 border border-zinc-200 dark:border-zinc-800 shadow-xs">
    <div class="flex items-center gap-2">
      <!-- Re-run Playbook Button -->
      <button
        type="button"
        onclick={handleReRun}
        disabled={isReRunning || isUnloading}
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-700 text-zinc-900 dark:text-zinc-100 text-xs font-semibold shadow-xs transition cursor-pointer disabled:opacity-50"
      >
        <RotateCw class={cn('h-3.5 w-3.5', isReRunning && 'animate-spin')} />
        <span>{isReRunning ? 'Starting Run...' : 'Re-run Playbook'}</span>
      </button>

      <!-- Load Another Playbook Button -->
      <button
        type="button"
        onclick={handleLoadAnother}
        disabled={isReRunning || isUnloading}
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:bg-zinc-100 dark:hover:bg-zinc-700 text-zinc-700 dark:text-zinc-300 text-xs font-medium shadow-xs transition cursor-pointer disabled:opacity-50"
      >
        <FolderOpen class="h-3.5 w-3.5" />
        <span>Load Another Playbook</span>
      </button>
    </div>

    <!-- Right: Quick Status Indicator -->
    <div class="text-xs font-mono text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5">
      Status:
      <span
        class={cn(
          'font-semibold px-2 py-0.5 rounded text-[11px]',
          appState.isSubmitted && 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30',
          appState.isSubmissionError && 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/30',
          appState.isSubmitting && 'bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/30 animate-pulse',
          !appState.isSubmitted && !appState.isSubmissionError && !appState.isSubmitting && 'bg-zinc-100 dark:bg-zinc-800 text-zinc-800 dark:text-zinc-200 border border-zinc-200 dark:border-zinc-700'
        )}
      >
        {appState.status}
      </span>
    </div>
  </div>

  <!-- 2. Scorecard Hero Section -->
  <Scorecard />

  <!-- 3. Report & Audit Tabs Section -->
  <div class="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/90 shadow-sm p-6 space-y-4">
    <ReportTabs
      onFullscreenMarkdown={handleOpenMarkdownModal}
      onFullscreenLogs={handleOpenLogsModal}
    />
  </div>

  <!-- 4. Fixed/Sticky Bottom Action Bar: Inspection, Export & Delivery -->
  <div class="fixed bottom-0 left-0 right-0 z-40 border-t border-zinc-200 dark:border-zinc-800 bg-white/95 dark:bg-zinc-950/95 backdrop-blur-sm py-3 shadow-2xl transition-colors">
    <div class="max-w-5xl mx-auto px-4 sm:px-6 flex items-center justify-between gap-4">
      <!-- Left: Reports Dropup -->
      <div>
        <ExportDropdown
          onPreviewMarkdown={() => handleOpenMarkdownModal()}
          onPreviewLogs={() => handleOpenLogsModal()}
          side="top"
        />
      </div>

      <!-- Right: Conditional Submit Button -->
      <div class="flex items-center gap-3">
        {#if hasRemoteDestination}
          {#if appState.isSubmitted}
            <button
              type="button"
              onclick={() => (showSubmissionPrompt = true)}
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 text-xs font-semibold shadow-xs transition cursor-pointer select-none"
            >
              <CheckCircle2 class="h-3.5 w-3.5" />
              <span>Report Submitted</span>
            </button>
          {:else if appState.isSubmissionError}
            <button
              type="button"
              onclick={() => (showSubmissionPrompt = true)}
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold shadow-sm transition cursor-pointer select-none"
            >
              <AlertTriangle class="h-3.5 w-3.5" />
              <span>Retry Server Submission</span>
            </button>
          {:else if appState.isSubmitting}
            <button
              type="button"
              disabled
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-indigo-600 opacity-70 text-white text-xs font-semibold shadow-sm select-none cursor-not-allowed"
            >
              <Loader2 class="h-3.5 w-3.5 animate-spin" />
              <span>Submitting...</span>
            </button>
          {:else}
            <button
              type="button"
              onclick={() => (showSubmissionPrompt = true)}
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-sm transition cursor-pointer select-none"
            >
              <Send class="h-3.5 w-3.5" />
              <span>Submit Report to Server</span>
            </button>
          {/if}
        {/if}
      </div>
    </div>
  </div>

  <!-- 5. Modals & Dialogs -->
  <!-- Remote Submission Modal -->
  <SubmissionPrompt
    bind:open={showSubmissionPrompt}
  />

  <!-- Fullscreen Raw Text Preview Modal -->
  <TextPreviewModal
    bind:open={showTextPreviewModal}
    title={previewModalTitle}
    content={previewModalContent}
    downloadUrl={previewModalDownloadUrl}
    isLog={previewModalIsLog}
  />
</div>
