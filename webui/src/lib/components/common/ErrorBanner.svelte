<script lang="ts">
  import AlertCircle from 'lucide-svelte/icons/alert-circle';
  import X from 'lucide-svelte/icons/x';
  import RotateCcw from 'lucide-svelte/icons/rotate-ccw';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import { Badge, Button } from '$lib/components/ui';
  import { cn } from '$lib/utils/cn';

  export interface ValidationErrorItem {
    path?: string;
    code?: string;
    message: string;
  }

  interface Props {
    code?: string;
    message: string;
    detail?: ValidationErrorItem[] | unknown;
    class?: string;
    onretry?: () => void;
    onunload?: () => void;
    ondismiss?: () => void;
  }

  let {
    code,
    message,
    detail,
    class: className = '',
    onretry,
    onunload,
    ondismiss,
  }: Props = $props();

  const validationErrors = $derived.by<ValidationErrorItem[]>(() => {
    if (Array.isArray(detail)) {
      return detail.filter((item) => typeof item === 'object' && item !== null && 'message' in item);
    }
    return [];
  });
</script>

<div
  role="alert"
  class={cn(
    'relative rounded-lg border border-rose-500/40 bg-rose-500/10 p-4 text-sm text-rose-200 shadow-sm transition-all',
    className
  )}
>
  <div class="flex items-start justify-between gap-3">
    <div class="flex items-start gap-3">
      <AlertCircle class="h-5 w-5 text-rose-400 shrink-0 mt-0.5" />
      <div class="space-y-1.5 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          {#if code}
            <Badge variant="failed" size="sm" class="font-mono uppercase tracking-wider font-semibold">
              {code}
            </Badge>
          {/if}
          <span class="font-semibold text-rose-100">{message}</span>
        </div>

        {#if validationErrors.length > 0}
          <div class="mt-2 space-y-1 rounded bg-rose-950/60 p-2.5 border border-rose-500/20 text-xs font-mono">
            <div class="text-rose-300 font-semibold mb-1">Validation Diagnostics ({validationErrors.length}):</div>
            {#each validationErrors as err, i (i)}
              <div class="flex items-start gap-1.5 text-rose-300/90">
                <span class="text-rose-400">•</span>
                {#if err.path}
                  <span class="text-rose-200 font-semibold">{err.path}:</span>
                {/if}
                <span>{err.message}</span>
              </div>
            {/each}
          </div>
        {/if}

        {#if onretry || onunload}
          <div class="mt-3 flex items-center gap-2 pt-1">
            {#if onretry}
              <Button variant="secondary" size="xs" onclick={onretry} class="border-rose-500/30 text-rose-200 hover:bg-rose-500/20">
                <RotateCcw class="h-3 w-3 mr-1" />
                Retry
              </Button>
            {/if}
            {#if onunload}
              <Button variant="secondary" size="xs" onclick={onunload} class="border-rose-500/30 text-rose-200 hover:bg-rose-500/20">
                <Trash2 class="h-3 w-3 mr-1" />
                Unload Playbook
              </Button>
            {/if}
          </div>
        {/if}
      </div>
    </div>

    {#if ondismiss}
      <button
        type="button"
        onclick={ondismiss}
        aria-label="Dismiss error"
        class="rounded p-1 text-rose-400 hover:bg-rose-500/20 hover:text-rose-200 cursor-pointer"
      >
        <X class="h-4 w-4" />
      </button>
    {/if}
  </div>
</div>
