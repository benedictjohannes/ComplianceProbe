<script lang="ts">
  import Dialog from '$lib/components/ui/Dialog.svelte';
  import ShieldAlert from 'lucide-svelte/icons/shield-alert';
  import Loader2 from 'lucide-svelte/icons/loader-2';

  interface Props {
    open?: boolean;
  }

  let { open = false }: Props = $props();
</script>

<Dialog
  {open}
  preventClose={true}
  maxWidth="md"
>
  {#snippet header()}
    <div class="text-center space-y-3 pt-2">
      <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-amber-500/10 border border-amber-500/20 text-amber-500 dark:text-amber-400">
        <ShieldAlert class="h-7 w-7" />
      </div>
      <div class="space-y-1">
        <h3 class="text-lg font-bold text-zinc-900 dark:text-zinc-100">
          Administrator Privileges Required
        </h3>
        <p class="text-xs text-zinc-500 dark:text-zinc-400">
          Root or elevated permissions requested by compliance probe
        </p>
      </div>
    </div>
  {/snippet}

  {#snippet children()}
    <div class="space-y-4 py-2 text-center">
      <p class="text-xs text-zinc-600 dark:text-zinc-300 leading-relaxed max-w-sm mx-auto">
        <strong class="text-zinc-800 dark:text-zinc-200">crobe</strong> is preparing system-level probes that require root/sudo privileges.
        Please accept the elevation prompt (UAC / polkit / sudo dialogue) on your screen to proceed.
      </p>

      <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20 text-xs font-mono font-medium animate-pulse">
        <Loader2 class="h-3.5 w-3.5 animate-spin" />
        <span>Waiting for system authorization...</span>
      </div>
    </div>
  {/snippet}
</Dialog>
