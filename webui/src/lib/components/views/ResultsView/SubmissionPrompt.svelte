<script lang="ts">
  import { appState } from '$lib/state/appState.svelte';
  import { Dialog, Button } from '$lib/components/ui';
  import Globe from 'lucide-svelte/icons/globe';
  import FileText from 'lucide-svelte/icons/file-text';
  import Key from 'lucide-svelte/icons/key';
  import Send from 'lucide-svelte/icons/send';
  import CheckCircle2 from 'lucide-svelte/icons/check-circle-2';
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';
  import RotateCw from 'lucide-svelte/icons/rotate-cw';

  interface Props {
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
  }

  let {
    open = $bindable(false),
    onOpenChange,
  }: Props = $props();

  let submitting = $derived(appState.isSubmitting);
  let submissionSuccess = $state(false);

  // Check if destination is configured
  const destinationUrl = $derived.by(() => {
    if (appState.reportDestination.https_source === 'custom' && appState.reportDestination.https) {
      return appState.reportDestination.https.url;
    }
    if (appState.reportDestination.playbook_defaults?.https?.url) {
      return appState.reportDestination.playbook_defaults.https.url;
    }
    if (appState.playbook?.reportDestinationHttps?.url) {
      return appState.playbook.reportDestinationHttps.url;
    }
    return 'https://... (configured destination)';
  });

  const reportFormat = $derived.by(() => {
    if (appState.reportDestination.https_source === 'custom' && appState.reportDestination.https) {
      return (appState.reportDestination.https.format || 'json').toUpperCase();
    }
    if (appState.reportDestination.playbook_defaults?.https?.format) {
      return appState.reportDestination.playbook_defaults.https.format.toUpperCase();
    }
    if (appState.playbook?.reportDestinationHttps?.format) {
      return appState.playbook.reportDestinationHttps.format.toUpperCase();
    }
    return 'JSON';
  });

  const hasSecret = $derived.by(() => {
    if (appState.reportDestination.https_source === 'custom' && appState.reportDestination.https) {
      return Boolean(appState.reportDestination.https.secret);
    }
    if (appState.reportDestination.playbook_defaults?.https) {
      return Boolean(appState.reportDestination.playbook_defaults.https.hasSignatureSecret);
    }
    if (appState.playbook?.reportDestinationHttps) {
      return Boolean(appState.playbook.reportDestinationHttps.hasSignatureSecret);
    }
    return false;
  });

  const configuredHeaders = $derived.by(() => {
    if (appState.reportDestination.https_source === 'custom' && appState.reportDestination.https?.headers) {
      return Object.keys(appState.reportDestination.https.headers);
    }
    if (appState.reportDestination.playbook_defaults?.https?.configuredHeaders) {
      return appState.reportDestination.playbook_defaults.https.configuredHeaders;
    }
    if (appState.playbook?.reportDestinationHttps?.configuredHeaders) {
      return appState.playbook.reportDestinationHttps.configuredHeaders;
    }
    return [];
  });

  const submissionError = $derived.by(() => {
    return appState.errors.find(
      (e) =>
        e.code === 'REMOTE_SUBMISSION_FAILED' ||
        e.code === 'REMOTE_SUBMISSION_TIMEOUT'
    );
  });

  async function handleSubmit() {
    try {
      await appState.submitRemoteReport();
      if (!appState.errors.some((e) => e.code.includes('SUBMISSION'))) {
        submissionSuccess = true;
      }
    } catch {
      // Error handled in appState.errors
    }
  }
</script>

<Dialog
  bind:open
  preventClose={submitting}
  maxWidth="md"
  {onOpenChange}
>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center gap-3">
      <div class="p-2 rounded-lg bg-indigo-500/10 border border-indigo-500/20 text-indigo-500 dark:text-indigo-400">
        <Send class="h-5 w-5" />
      </div>
      <div>
        <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
          Remote Report Submission
        </h3>
        <p class="text-xs text-zinc-500 dark:text-zinc-400">
          Transmit certified audit report to the central compliance service.
        </p>
      </div>
    </div>

    <!-- Succeeded State -->
    {#if submissionSuccess}
      <div class="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-700 dark:text-emerald-300 space-y-2">
        <div class="flex items-center gap-2 font-semibold text-sm">
          <CheckCircle2 class="h-4 w-4 text-emerald-500" />
          <span>Report Delivered Successfully</span>
        </div>
        <p class="text-xs text-emerald-600 dark:text-emerald-400 leading-relaxed">
          The compliance report was delivered and accepted with HTTP 200 OK at the configured endpoint.
        </p>
      </div>

    <!-- Form & Config Preview -->
    {:else}
      <div class="p-3.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800/80 space-y-2.5 text-xs">
        <!-- Endpoint -->
        <div class="flex items-center justify-between gap-3">
          <span class="text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 shrink-0">
            <Globe class="h-3.5 w-3.5 text-zinc-400" />
            Destination
          </span>
          <span class="font-mono text-zinc-800 dark:text-zinc-200 truncate max-w-[240px] text-right font-medium" title={destinationUrl}>
            {destinationUrl}
          </span>
        </div>

        <!-- Format -->
        <div class="flex items-center justify-between gap-3">
          <span class="text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 shrink-0">
            <FileText class="h-3.5 w-3.5 text-zinc-400" />
            Payload Format
          </span>
          <span class="font-mono text-zinc-800 dark:text-zinc-200 font-medium">
            {reportFormat}
          </span>
        </div>

        <!-- Signature -->
        <div class="flex items-center justify-between gap-3">
          <span class="text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 shrink-0">
            <Key class="h-3.5 w-3.5 text-zinc-400" />
            HMAC Signature
          </span>
          <span class={hasSecret ? 'text-emerald-600 dark:text-emerald-400 font-medium' : 'text-zinc-500'}>
            {hasSecret ? 'Yes (Secret Configured)' : 'Unsigned'}
          </span>
        </div>

        <!-- Custom Headers -->
        {#if configuredHeaders.length > 0}
          <div class="pt-1.5 border-t border-zinc-200 dark:border-zinc-800/60 text-[11px] space-y-1">
            <span class="text-zinc-500 dark:text-zinc-400 font-medium">Custom Headers:</span>
            <div class="flex flex-wrap gap-1">
              {#each configuredHeaders as header}
                <span class="px-1.5 py-0.5 rounded bg-zinc-200 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-mono text-[10px]">
                  {header}
                </span>
              {/each}
            </div>
          </div>
        {/if}
      </div>

      <!-- Submission Error / Retry Alert -->
      {#if submissionError}
        <div class="p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-700 dark:text-rose-300 space-y-1.5 text-xs">
          <div class="flex items-center gap-1.5 font-bold">
            <AlertTriangle class="h-4 w-4 text-rose-500 shrink-0" />
            <span>Submission Failed</span>
          </div>
          <p class="text-[11px] text-rose-600 dark:text-rose-400 leading-relaxed font-mono">
            {submissionError.message}
          </p>
        </div>
      {/if}
    {/if}
  </div>

  {#snippet footer()}
    {#if submissionSuccess}
      <Button
        variant="secondary"
        size="sm"
        onclick={() => {
          open = false;
        }}
      >
        Close
      </Button>
    {:else}
      <Button
        variant="ghost"
        size="sm"
        disabled={submitting}
        onclick={() => {
          open = false;
        }}
      >
        Cancel
      </Button>

      {#if submissionError}
        <Button
          variant="indigo"
          size="sm"
          loading={submitting}
          onclick={handleSubmit}
        >
          <RotateCw class="h-3.5 w-3.5 mr-1.5" />
          Retry Submission
        </Button>
      {:else}
        <Button
          variant="indigo"
          size="sm"
          loading={submitting}
          onclick={handleSubmit}
        >
          <Send class="h-3.5 w-3.5 mr-1.5" />
          Submit Report
        </Button>
      {/if}
    {/if}
  {/snippet}
</Dialog>
