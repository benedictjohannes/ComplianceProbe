<script lang="ts">
  import UploadCloud from 'lucide-svelte/icons/upload-cloud';
  import Folder from 'lucide-svelte/icons/folder';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import AlertCircle from 'lucide-svelte/icons/alert-circle';
  import { Button } from '$lib/components/ui';
  import { cn } from '$lib/utils/cn';

  interface Props {
    disabled?: boolean;
    loading?: boolean;
    class?: string;
    onfile?: (file: File) => void;
    onerror?: (message: string) => void;
  }

  let {
    disabled = false,
    loading = false,
    class: className = '',
    onfile,
    onerror,
  }: Props = $props();

  let fileInputRef = $state<HTMLInputElement | null>(null);
  let isDragOver = $state<boolean>(false);
  let dragDepth = $state<number>(0);
  let localError = $state<string | null>(null);

  const ACCEPTED_EXTENSIONS = ['.yaml', '.yml', '.json'];

  function isValidPlaybookFile(file: File): boolean {
    const name = file.name.toLowerCase();
    return ACCEPTED_EXTENSIONS.some((ext) => name.endsWith(ext));
  }

  function handleFileSelection(files: FileList | File[] | null) {
    if (!files || files.length === 0 || disabled || loading) return;

    localError = null;
    const file = files[0];

    if (!isValidPlaybookFile(file)) {
      const msg = `Unsupported file type "${file.name}". Please provide a .yaml, .yml, or .json playbook.`;
      localError = msg;
      onerror?.(msg);
      return;
    }

    onfile?.(file);
  }

  function handleDragEnter(e: DragEvent) {
    e.preventDefault();
    if (disabled || loading) return;
    dragDepth += 1;
    isDragOver = true;
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    if (disabled || loading) return;
    isDragOver = true;
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    if (disabled || loading) return;
    dragDepth -= 1;
    if (dragDepth <= 0) {
      dragDepth = 0;
      isDragOver = false;
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragDepth = 0;
    isDragOver = false;
    if (disabled || loading) return;

    if (e.dataTransfer?.files) {
      handleFileSelection(e.dataTransfer.files);
    }
  }

  function triggerFilePicker() {
    if (disabled || loading || !fileInputRef) return;
    localError = null;
    fileInputRef.click();
  }

  function onInputChange(e: Event) {
    const target = e.target as HTMLInputElement;
    if (target.files) {
      handleFileSelection(target.files);
    }
    // Reset file input value so the same file can be selected again if needed
    target.value = '';
  }
</script>

<div class={cn('w-full max-w-xl mx-auto space-y-3', className)}>
  <div
    role="region"
    aria-label="Playbook dropzone"
    ondragenter={handleDragEnter}
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
    class={cn(
      'relative rounded-xl border border-dashed p-8 text-center transition-all select-none',
      isDragOver
        ? 'border-sky-500 bg-sky-500/10 shadow-lg shadow-sky-500/10 scale-[1.01]'
        : 'border-zinc-700 bg-zinc-900/40 hover:border-zinc-500',
      (disabled || loading) && 'opacity-60 pointer-events-none cursor-not-allowed',
      localError && 'border-rose-500/60 bg-rose-500/5'
    )}
  >
    <!-- Hidden File Input -->
    <input
      bind:this={fileInputRef}
      type="file"
      accept=".yaml,.yml,.json,application/x-yaml,text/yaml,application/json"
      class="hidden"
      onchange={onInputChange}
      {disabled}
    />

    <div class="space-y-4">
      <!-- Icon -->
      <div
        class={cn(
          'mx-auto flex h-14 w-14 items-center justify-center rounded-full transition-transform duration-200',
          isDragOver
            ? 'bg-sky-500/20 text-sky-400 scale-110'
            : 'bg-zinc-800/80 text-zinc-400',
          loading && 'text-sky-400 bg-sky-500/20'
        )}
      >
        {#if loading}
          <Loader2 class="h-7 w-7 animate-spin" />
        {:else}
          <UploadCloud class="h-7 w-7" />
        {/if}
      </div>

      <!-- Text Headings -->
      <div class="space-y-1.5">
        {#if loading}
          <div class="text-sm font-semibold text-zinc-200">
            Parsing and validating playbook schema...
          </div>
          <div class="text-xs font-mono text-zinc-500">
            Verifying assertion rules, commands, and target destinations
          </div>
        {:else if isDragOver}
          <div class="text-sm font-semibold text-sky-300">
            Drop file to load playbook...
          </div>
          <div class="text-xs font-mono text-sky-400/80">
            Release to ingest into compliance runner
          </div>
        {:else}
          <div class="text-base font-semibold text-zinc-200">
            Drag & drop your compliance playbook
          </div>
          <div class="text-xs font-mono text-zinc-500">
            Supports <span class="text-zinc-400">.yaml</span>, <span class="text-zinc-400">.yml</span>, and <span class="text-zinc-400">.json</span> files
          </div>
        {/if}
      </div>

      <!-- Action Button -->
      <div class="pt-2">
        <Button
          variant="secondary"
          size="sm"
          {disabled}
          {loading}
          onclick={triggerFilePicker}
        >
          <Folder class="h-4 w-4 mr-1.5 text-zinc-400" />
          Browse Files
        </Button>
      </div>
    </div>
  </div>

  {#if localError}
    <div class="flex items-center gap-2 rounded-md border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-300">
      <AlertCircle class="h-4 w-4 shrink-0 text-rose-400" />
      <span>{localError}</span>
    </div>
  {/if}
</div>
