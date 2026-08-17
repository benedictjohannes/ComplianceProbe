<script lang="ts">
  import Globe from 'lucide-svelte/icons/globe';
  import Clipboard from 'lucide-svelte/icons/clipboard';
  import X from 'lucide-svelte/icons/x';
  import Plus from 'lucide-svelte/icons/plus';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import AlertCircle from 'lucide-svelte/icons/alert-circle';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import { Dialog, Input, Button } from '$lib/components/ui';
  import type { RemotePlaybookRequest, AppError } from '$lib/api/types';

  interface Props {
    open?: boolean;
    loading?: boolean;
    error?: AppError | string | null;
    onsubmit?: (req: RemotePlaybookRequest) => void | Promise<void>;
    oncancel?: () => void;
  }

  let {
    open = $bindable(false),
    loading = false,
    error = null,
    onsubmit,
    oncancel,
  }: Props = $props();

  let url = $state<string>('');
  let headers = $state<Array<{ key: string; value: string }>>([]);
  let showAdvancedHeaders = $state<boolean>(false);
  let validationError = $state<string | null>(null);

  function resetForm() {
    url = '';
    headers = [];
    showAdvancedHeaders = false;
    validationError = null;
  }

  function handleClose() {
    if (loading) return;
    open = false;
    oncancel?.();
  }

  async function pasteFromClipboard() {
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard?.readText) {
        const text = await navigator.clipboard.readText();
        if (text && text.trim()) {
          url = text.trim();
          validationError = null;
        }
      }
    } catch {
      // Graceful fallback if clipboard permission is denied
    }
  }

  function addHeaderRow() {
    headers = [...headers, { key: '', value: '' }];
    showAdvancedHeaders = true;
  }

  function removeHeaderRow(index: number) {
    headers = headers.filter((_, i) => i !== index);
  }

  function validateUrl(inputUrl: string): boolean {
    const trimmed = inputUrl.trim();
    if (!trimmed) {
      validationError = 'Please enter a playbook URL.';
      return false;
    }

    try {
      const parsed = new URL(trimmed);
      const isHttps = parsed.protocol === 'https:';
      const isLocalhost =
        parsed.hostname === 'localhost' ||
        parsed.hostname === '127.0.0.1' ||
        parsed.hostname === '::1';

      if (!isHttps && !isLocalhost) {
        validationError = 'Only HTTPS URLs (or localhost for development) are permitted.';
        return false;
      }
    } catch {
      validationError = 'Invalid URL format. Please provide a valid HTTPS URL.';
      return false;
    }

    validationError = null;
    return true;
  }

  async function handleSubmit(e?: Event) {
    e?.preventDefault();
    if (loading) return;

    if (!validateUrl(url)) return;

    // Filter valid headers
    const headerRecord: Record<string, string> = {};
    for (const h of headers) {
      const trimmedKey = h.key.trim();
      if (trimmedKey) {
        headerRecord[trimmedKey] = h.value;
      }
    }

    const payload: RemotePlaybookRequest = {
      url: url.trim(),
      ...(Object.keys(headerRecord).length > 0 ? { headers: headerRecord } : {}),
    };

    await onsubmit?.(payload);
  }
</script>

<Dialog
  bind:open
  title="Fetch Remote Playbook"
  description="Enter an HTTPS endpoint serving a valid YAML or JSON playbook."
  maxWidth="lg"
  preventClose={loading}
  onOpenChange={(v) => {
    if (!v) handleClose();
  }}
>
  <form onsubmit={handleSubmit} class="space-y-4">
    <!-- URL Input -->
    <div class="space-y-1.5">
      <label for="remote-url-input" class="block text-xs font-semibold text-zinc-700 dark:text-zinc-300">
        Playbook HTTPS URL
      </label>

      <div class="relative flex items-center">
        <Input
          id="remote-url-input"
          bind:value={url}
          placeholder="https://example.com/playbook.yaml"
          mono
          disabled={loading}
          error={!!validationError}
          class="pr-20"
        >
          {#snippet leading()}
            <Globe class="h-4 w-4 text-zinc-400 dark:text-zinc-500" />
          {/snippet}
        </Input>

        <div class="absolute right-2 flex items-center gap-1">
          {#if url}
            <button
              type="button"
              onclick={() => {
                url = '';
                validationError = null;
              }}
              disabled={loading}
              class="rounded p-1 text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 cursor-pointer disabled:opacity-50"
              title="Clear"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          {/if}

          <Button
            type="button"
            variant="ghost"
            size="xs"
            disabled={loading}
            onclick={pasteFromClipboard}
            class="text-[11px] px-1.5 py-0.5 text-zinc-600 dark:text-zinc-400 hover:text-sky-600 dark:hover:text-sky-400 gap-1 border border-zinc-300 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-950"
            title="Paste from clipboard"
          >
            <Clipboard class="h-3 w-3" />
            Paste
          </Button>
        </div>
      </div>

      {#if validationError}
        <p class="text-xs text-rose-600 dark:text-rose-400 flex items-center gap-1 mt-1 font-mono">
          <AlertCircle class="h-3.5 w-3.5 shrink-0" />
          <span>{validationError}</span>
        </p>
      {/if}
    </div>

    <!-- Advanced Headers Accordion -->
    <div class="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50/60 dark:bg-zinc-950/40 p-3 space-y-3">
      <button
        type="button"
        onclick={() => (showAdvancedHeaders = !showAdvancedHeaders)}
        class="flex w-full items-center justify-between text-xs font-semibold text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200 cursor-pointer select-none"
      >
        <span class="flex items-center gap-1.5">
          {#if showAdvancedHeaders}
            <ChevronDown class="h-3.5 w-3.5 text-zinc-400 dark:text-zinc-500" />
          {:else}
            <ChevronRight class="h-3.5 w-3.5 text-zinc-400 dark:text-zinc-500" />
          {/if}
          Advanced: Request Headers
          <span class="rounded-full bg-zinc-200 dark:bg-zinc-800 px-1.5 py-0.2 text-[10px] font-mono text-zinc-700 dark:text-zinc-400">
            {headers.length}
          </span>
        </span>
      </button>

      {#if showAdvancedHeaders}
        <div class="space-y-2 pt-1">
          <p class="text-[11px] text-zinc-500">
            Configure custom request headers for endpoints requiring authentication or bearer tokens.
          </p>

          {#if headers.length > 0}
            <div class="space-y-2 max-h-48 overflow-y-auto pr-1">
              {#each headers as header, index (index)}
                <div class="flex items-center gap-2">
                  <Input
                    bind:value={header.key}
                    placeholder="Header Name (e.g. Authorization)"
                    mono
                    disabled={loading}
                    class="flex-1 text-xs py-1.5"
                  />
                  <Input
                    bind:value={header.value}
                    placeholder="Value (e.g. Bearer token_...)"
                    mono
                    disabled={loading}
                    class="flex-1 text-xs py-1.5"
                  />
                  <button
                    type="button"
                    onclick={() => removeHeaderRow(index)}
                    disabled={loading}
                    class="p-1.5 rounded text-zinc-500 hover:text-rose-400 hover:bg-rose-500/10 cursor-pointer disabled:opacity-50"
                    title="Remove header"
                  >
                    <Trash2 class="h-3.5 w-3.5" />
                  </button>
                </div>
              {/each}
            </div>
          {/if}

          <Button
            type="button"
            variant="outline"
            size="xs"
            disabled={loading}
            onclick={addHeaderRow}
            class="text-xs text-zinc-400 hover:text-sky-400 mt-1"
          >
            <Plus class="h-3 w-3 mr-1" />
            Add Custom Header
          </Button>
        </div>
      {/if}
    </div>

    <!-- Error Display (if server returned error) -->
    {#if error}
      <div class="rounded-md border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-300 flex items-start gap-2">
        <AlertCircle class="h-4 w-4 shrink-0 text-rose-400 mt-0.5" />
        <div class="space-y-1">
          <div class="font-semibold text-rose-200">
            {typeof error === 'string' ? error : error.message}
          </div>
          {#if typeof error === 'object' && error?.code}
            <div class="font-mono text-[10px] text-rose-400">
              Code: {error.code}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </form>

  {#snippet footer()}
    <Button
      type="button"
      variant="ghost"
      size="sm"
      disabled={loading}
      onclick={handleClose}
    >
      Cancel
    </Button>

    <Button
      type="button"
      variant="primary"
      size="sm"
      {loading}
      disabled={loading || !url.trim()}
      onclick={handleSubmit}
    >
      Fetch & Load ➜
    </Button>
  {/snippet}
</Dialog>
