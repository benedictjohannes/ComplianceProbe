<script lang="ts">
  import Globe from 'lucide-svelte/icons/globe';
  import Folder from 'lucide-svelte/icons/folder';
  import Lock from 'lucide-svelte/icons/lock';
  import Eye from 'lucide-svelte/icons/eye';
  import EyeOff from 'lucide-svelte/icons/eye-off';
  import Clipboard from 'lucide-svelte/icons/clipboard';
  import Plus from 'lucide-svelte/icons/plus';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import AlertCircle from 'lucide-svelte/icons/alert-circle';
  import ShieldCheck from 'lucide-svelte/icons/shield-check';
  import { Dialog, Input, Button, Badge } from '$lib/components/ui';
  import type {
    ReportDestinationState,
    DestinationUpdateRequest,
    FolderSource,
    HttpsSource,
    ReportFormat,
  } from '$lib/api/types';

  interface Props {
    open?: boolean;
    destination: ReportDestinationState;
    onSave: (req: DestinationUpdateRequest) => Promise<void> | void;
    onClose?: () => void;
  }

  let {
    open = $bindable(false),
    destination,
    onSave,
    onClose,
  }: Props = $props();

  // Local editable state initialized from props
  let folderSource = $state<FolderSource>('default');
  let customFolder = $state<string>('');
  let httpsSource = $state<HttpsSource>('off');
  let customHttpsUrl = $state<string>('');
  let customHttpsFormat = $state<ReportFormat>('json');
  let customHttpsSecret = $state<string>('');
  let showSecret = $state<boolean>(false);
  let customHeaders = $state<Array<{ key: string; value: string }>>([]);
  let showCustomHeaders = $state<boolean>(false);

  let isSaving = $state<boolean>(false);
  let errorMessage = $state<string | null>(null);

  import { untrack } from 'svelte';

  // Sync state whenever dialog opens
  $effect(() => {
    if (open && destination) {
      untrack(() => {
        folderSource = destination.folder_source || 'default';
        customFolder = destination.folder || '';
        httpsSource = destination.https_source || 'off';

        if (destination.https) {
          customHttpsUrl = destination.https.url || '';
          customHttpsFormat = (destination.https.format as ReportFormat) || 'json';
          customHttpsSecret = destination.https.secret || '';
          if (destination.https.headers) {
            customHeaders = Object.entries(destination.https.headers).map(([key, value]) => ({
              key,
              value,
            }));
          } else {
            customHeaders = [];
          }
        } else {
          customHttpsUrl = '';
          customHttpsFormat = 'json';
          customHttpsSecret = '';
          customHeaders = [];
        }

        showSecret = false;
        errorMessage = null;
      });
    }
  });

  const isCliLocked = $derived(destination.folder_source === 'cli');
  const hasPlaybookHttps = $derived(!!destination.playbook_defaults?.has_https);

  async function pasteUrl() {
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard?.readText) {
        const text = await navigator.clipboard.readText();
        if (text && text.trim()) {
          customHttpsUrl = text.trim();
        }
      }
    } catch {
      // Ignore clipboard permission errors
    }
  }

  function addHeaderRow() {
    customHeaders = [...customHeaders, { key: '', value: '' }];
    showCustomHeaders = true;
  }

  function removeHeaderRow(index: number) {
    customHeaders = customHeaders.filter((_, i) => i !== index);
  }

  function validate(): boolean {
    errorMessage = null;

    if (folderSource === 'custom' && !customFolder.trim()) {
      errorMessage = 'Please specify a valid custom folder path.';
      return false;
    }

    if (httpsSource === 'custom') {
      const trimmedUrl = customHttpsUrl.trim();
      if (!trimmedUrl) {
        errorMessage = 'Please enter an HTTPS target endpoint URL.';
        return false;
      }

      try {
        const parsed = new URL(trimmedUrl);
        const isHttps = parsed.protocol === 'https:';
        const isLocalhost =
          parsed.hostname === 'localhost' ||
          parsed.hostname === '127.0.0.1' ||
          parsed.hostname === '::1';

        if (!isHttps && !isLocalhost) {
          errorMessage = 'Target URL must start with https:// (or localhost for dev).';
          return false;
        }
      } catch {
        errorMessage = 'Target URL is not a valid URL format.';
        return false;
      }
    }

    return true;
  }

  async function handleSave() {
    if (isSaving) return;
    if (!validate()) return;

    isSaving = true;
    try {
      const headerRecord: Record<string, string> = {};
      if (httpsSource === 'custom') {
        for (const h of customHeaders) {
          const k = h.key.trim();
          if (k) headerRecord[k] = h.value;
        }
      }

      const updateReq: DestinationUpdateRequest = {
        folder_source: folderSource,
        folder: folderSource === 'custom' ? customFolder.trim() : undefined,
        https_source: httpsSource,
        https:
          httpsSource === 'custom'
            ? {
                url: customHttpsUrl.trim(),
                format: customHttpsFormat,
                secret: customHttpsSecret.trim() ? customHttpsSecret.trim() : undefined,
                headers: Object.keys(headerRecord).length > 0 ? headerRecord : undefined,
              }
            : undefined,
      };

      await onSave(updateReq);
      open = false;
      onClose?.();
    } catch (e: any) {
      errorMessage = e?.message || 'Failed to update destination settings';
    } finally {
      isSaving = false;
    }
  }
</script>

<Dialog
  bind:open
  title="Destination Settings"
  description="Configure local file persistence and remote HTTPS report delivery."
  maxWidth="lg"
  preventClose={isSaving}
  onOpenChange={(v) => {
    if (!v) onClose?.();
  }}
>
  <div class="space-y-6">
    <!-- Error Alert if any -->
    {#if errorMessage}
      <div class="flex items-center gap-2 rounded-md border border-rose-500/40 bg-rose-500/10 p-3 text-xs text-rose-300">
        <AlertCircle class="h-4 w-4 shrink-0 text-rose-400" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    <!-- 1. LOCAL REPORT FOLDER SECTION -->
    <div class="space-y-3">
      <div class="flex items-center gap-2 pb-1 border-b border-zinc-200 dark:border-zinc-800">
        <Folder class="h-4 w-4 text-amber-500 dark:text-amber-400" />
        <h4 class="text-xs font-semibold uppercase tracking-wider text-zinc-700 dark:text-zinc-300">
          Local Report Storage
        </h4>
      </div>

      {#if isCliLocked}
        <div class="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-950 p-3 space-y-1">
          <div class="flex items-center gap-2">
            <Lock class="h-3.5 w-3.5 text-zinc-500 dark:text-zinc-400" />
            <span class="text-xs font-medium text-zinc-800 dark:text-zinc-200">Locked via CLI Flag</span>
            <Badge variant="info" size="sm" class="text-[10px]">--folder</Badge>
          </div>
          <p class="text-xs font-mono text-zinc-600 dark:text-zinc-400 select-all">
            {destination.folder || './reports/'}
          </p>
        </div>
      {:else}
        <div class="space-y-2">
          <!-- Folder Source Options -->
          <div class="grid grid-cols-3 gap-2">
            <label
              class="flex flex-col items-center justify-center rounded-lg border p-2.5 text-center cursor-pointer transition-colors {folderSource === 'default' ? 'border-sky-500 bg-sky-500/10 text-sky-700 dark:text-sky-300' : 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700'}"
            >
              <input
                type="radio"
                name="folder_source"
                value="default"
                bind:group={folderSource}
                class="sr-only"
              />
              <span class="text-xs font-semibold">Default</span>
              <span class="text-[10px] text-zinc-500 mt-0.5">./reports/</span>
            </label>

            <label
              class="flex flex-col items-center justify-center rounded-lg border p-2.5 text-center cursor-pointer transition-colors {folderSource === 'custom' ? 'border-sky-500 bg-sky-500/10 text-sky-700 dark:text-sky-300' : 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700'}"
            >
              <input
                type="radio"
                name="folder_source"
                value="custom"
                bind:group={folderSource}
                class="sr-only"
              />
              <span class="text-xs font-semibold">Custom Path</span>
              <span class="text-[10px] text-zinc-500 mt-0.5">Specify directory</span>
            </label>

            <label
              class="flex flex-col items-center justify-center rounded-lg border p-2.5 text-center cursor-pointer transition-colors {folderSource === 'off' ? 'border-sky-500 bg-sky-500/10 text-sky-700 dark:text-sky-300' : 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700'}"
            >
              <input
                type="radio"
                name="folder_source"
                value="off"
                bind:group={folderSource}
                class="sr-only"
              />
              <span class="text-xs font-semibold">Off</span>
              <span class="text-[10px] text-zinc-500 mt-0.5">In-memory only</span>
            </label>
          </div>

          {#if folderSource === 'custom'}
            <div class="pt-1 space-y-1">
              <label for="custom-folder-input" class="block text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Directory Path
              </label>
              <Input
                id="custom-folder-input"
                bind:value={customFolder}
                placeholder="/var/log/compliance-reports/"
                mono
                disabled={isSaving}
              />
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- 2. REMOTE HTTPS SUBMISSION SECTION -->
    <div class="space-y-3">
      <div class="flex items-center gap-2 pb-1 border-b border-zinc-200 dark:border-zinc-800">
        <Globe class="h-4 w-4 text-sky-600 dark:text-sky-400" />
        <h4 class="text-xs font-semibold uppercase tracking-wider text-zinc-700 dark:text-zinc-300">
          Remote HTTPS Submission
        </h4>
      </div>

      <!-- HTTPS Source Options -->
      <div class="grid grid-cols-3 gap-2">
        {#if hasPlaybookHttps}
          <label
            class="flex flex-col items-center justify-center rounded-lg border p-2.5 text-center cursor-pointer transition-colors {httpsSource === 'playbook' ? 'border-sky-500 bg-sky-500/10 text-sky-700 dark:text-sky-300' : 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700'}"
          >
            <input
              type="radio"
              name="https_source"
              value="playbook"
              bind:group={httpsSource}
              class="sr-only"
            />
            <span class="text-xs font-semibold">Playbook</span>
            <span class="text-[10px] text-zinc-500 mt-0.5">Default endpoint</span>
          </label>
        {/if}

        <label
          class="flex flex-col items-center justify-center rounded-lg border p-2.5 text-center cursor-pointer transition-colors {httpsSource === 'custom' ? 'border-sky-500 bg-sky-500/10 text-sky-700 dark:text-sky-300' : 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700'} {!hasPlaybookHttps ? 'col-span-1' : ''}"
        >
          <input
            type="radio"
            name="https_source"
            value="custom"
            bind:group={httpsSource}
            class="sr-only"
          />
          <span class="text-xs font-semibold">Custom</span>
          <span class="text-[10px] text-zinc-500 mt-0.5">Configure URL</span>
        </label>

        <label
          class="flex flex-col items-center justify-center rounded-lg border p-2.5 text-center cursor-pointer transition-colors {httpsSource === 'off' ? 'border-sky-500 bg-sky-500/10 text-sky-700 dark:text-sky-300' : 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700'} {!hasPlaybookHttps ? 'col-span-2' : ''}"
        >
          <input
            type="radio"
            name="https_source"
            value="off"
            bind:group={httpsSource}
            class="sr-only"
          />
          <span class="text-xs font-semibold">Off</span>
          <span class="text-[10px] text-zinc-500 mt-0.5">No remote submit</span>
        </label>
      </div>

      <!-- Playbook HTTPS Config Display -->
      {#if httpsSource === 'playbook' && destination.playbook_defaults?.https}
        <div class="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/80 p-3 space-y-2 text-xs font-mono">
          <div class="text-zinc-600 dark:text-zinc-400">
            Target Endpoint:
            <div class="text-zinc-900 dark:text-zinc-200 font-semibold select-all">
              {destination.playbook_defaults.https.url}
            </div>
          </div>
          <div class="flex items-center gap-3 text-[11px] text-zinc-500">
            <span>Format: {destination.playbook_defaults.https.format}</span>
            {#if destination.playbook_defaults.https.hasSignatureSecret}
              <span class="text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                <ShieldCheck class="h-3 w-3" />
                HMAC Signature: Active
              </span>
            {/if}
          </div>
        </div>
      {/if}

      <!-- Custom HTTPS Settings -->
      {#if httpsSource === 'custom'}
        <div class="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50/60 dark:bg-zinc-950/60 p-3.5 space-y-3.5">
          <!-- Target Endpoint URL -->
          <div class="space-y-1">
            <label for="custom-https-url" class="block text-xs font-medium text-zinc-700 dark:text-zinc-300">
              Target Endpoint URL
            </label>
            <div class="relative flex items-center">
              <Input
                id="custom-https-url"
                bind:value={customHttpsUrl}
                placeholder="https://sec-ops.corp.internal/ingest"
                mono
                disabled={isSaving}
                class="pr-16"
              />
              <Button
                type="button"
                variant="ghost"
                size="xs"
                onclick={pasteUrl}
                class="absolute right-1 text-[11px] text-zinc-600 dark:text-zinc-400 hover:text-sky-600 dark:hover:text-sky-400 gap-1 border border-zinc-300 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900"
              >
                <Clipboard class="h-3 w-3" />
                Paste
              </Button>
            </div>
          </div>

          <!-- Payload Format Selection -->
          <div class="space-y-1">
            <span class="block text-xs font-medium text-zinc-700 dark:text-zinc-300">
              Payload Format
            </span>
            <div class="inline-flex rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 p-0.5 text-xs font-mono">
              <button
                type="button"
                onclick={() => (customHttpsFormat = 'json')}
                class="px-3 py-1 rounded text-xs cursor-pointer transition-colors {customHttpsFormat === 'json' ? 'bg-sky-500/20 text-sky-700 dark:text-sky-400 font-semibold' : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'}"
              >
                JSON Payload
              </button>
              <button
                type="button"
                onclick={() => (customHttpsFormat = 'multipart')}
                class="px-3 py-1 rounded text-xs cursor-pointer transition-colors {customHttpsFormat === 'multipart' ? 'bg-sky-500/20 text-sky-700 dark:text-sky-400 font-semibold' : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'}"
              >
                Multipart Form (.zip)
              </button>
            </div>
          </div>

          <!-- HMAC Secret Field -->
          <div class="space-y-1">
            <label for="custom-https-secret" class="block text-xs font-medium text-zinc-700 dark:text-zinc-300">
              HMAC Signature Secret (Optional)
            </label>
            <div class="relative flex items-center">
              <Input
                id="custom-https-secret"
                type={showSecret ? 'text' : 'password'}
                bind:value={customHttpsSecret}
                placeholder="HMAC secret key for X-Signature-SHA256"
                mono
                disabled={isSaving}
                class="pr-16"
              />
              <Button
                type="button"
                variant="ghost"
                size="xs"
                onclick={() => (showSecret = !showSecret)}
                class="absolute right-1 text-[11px] text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200 gap-1 border border-zinc-300 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900"
              >
                {#if showSecret}
                  <EyeOff class="h-3 w-3" />
                  Hide
                {:else}
                  <Eye class="h-3 w-3" />
                  Show
                {/if}
              </Button>
            </div>
          </div>

          <!-- Custom Authorization Headers -->
          <div class="space-y-2 pt-1 border-t border-zinc-200 dark:border-zinc-800">
            <div class="flex items-center justify-between">
              <span class="text-xs font-medium text-zinc-600 dark:text-zinc-400">
                Custom Request Headers ({customHeaders.length})
              </span>
              <Button
                type="button"
                variant="outline"
                size="xs"
                onclick={addHeaderRow}
                class="text-[11px] text-zinc-700 dark:text-zinc-300 hover:text-sky-600 dark:hover:text-sky-400 gap-1"
              >
                <Plus class="h-3 w-3" />
                Add Header
              </Button>
            </div>

            {#if customHeaders.length > 0}
              <div class="space-y-2 max-h-36 overflow-y-auto pr-1">
                {#each customHeaders as header, idx (idx)}
                  <div class="flex items-center gap-2">
                    <Input
                      bind:value={header.key}
                      placeholder="Header Name"
                      mono
                      disabled={isSaving}
                      class="text-xs py-1 flex-1"
                    />
                    <Input
                      bind:value={header.value}
                      placeholder="Header Value"
                      mono
                      disabled={isSaving}
                      class="text-xs py-1 flex-1"
                    />
                    <button
                      type="button"
                      onclick={() => removeHeaderRow(idx)}
                      class="p-1 text-zinc-400 hover:text-rose-600 hover:bg-rose-500/10 rounded cursor-pointer"
                    >
                      <Trash2 class="h-3.5 w-3.5" />
                    </button>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  </div>

  {#snippet footer()}
    <Button
      type="button"
      variant="ghost"
      size="sm"
      disabled={isSaving}
      onclick={() => {
        open = false;
        onClose?.();
      }}
    >
      Cancel
    </Button>

    <Button
      type="button"
      variant="primary"
      size="sm"
      loading={isSaving}
      disabled={isSaving}
      onclick={handleSave}
    >
      Save Destination Settings
    </Button>
  {/snippet}
</Dialog>
