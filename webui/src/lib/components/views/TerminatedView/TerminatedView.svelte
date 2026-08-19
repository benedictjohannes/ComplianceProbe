<script lang="ts">
  import PowerOff from 'lucide-svelte/icons/power-off';
  import Terminal from 'lucide-svelte/icons/terminal';
  import X from 'lucide-svelte/icons/x';
  import Button from '$lib/components/ui/Button.svelte';

  function handleCloseWindow() {
    if (typeof window !== 'undefined') {
      window.close();
    }
  }
</script>

<div class="min-h-[70vh] flex items-center justify-center px-4">
  <div class="w-full max-w-md rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/90 p-8 shadow-xl text-center space-y-6 animate-in fade-in-50 zoom-in-95">
    <!-- Visual Icon -->
    <div class="mx-auto inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-rose-500/10 text-rose-500 border border-rose-500/20 shadow-inner">
      <PowerOff class="h-8 w-8" />
    </div>

    <!-- Title & Explanation -->
    <div class="space-y-2">
      <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100">
        Server Terminated
      </h2>
      <p class="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed max-w-sm mx-auto">
        The <span class="font-mono text-zinc-800 dark:text-zinc-200 font-semibold">crobe</span> local process has stopped. All active sessions and background monitors have been cleanly closed.
      </p>
    </div>

    <!-- Instructions / Next Steps -->
    <div class="rounded-xl border border-zinc-200 dark:border-zinc-800/80 bg-zinc-50 dark:bg-zinc-950/60 p-4 text-left space-y-2">
      <div class="flex items-center gap-2 text-xs font-semibold text-zinc-700 dark:text-zinc-300">
        <Terminal class="h-3.5 w-3.5 text-zinc-500" />
        <span>To start Compliance Probe again:</span>
      </div>
      <pre class="text-xs font-mono text-sky-600 dark:text-sky-400 bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded px-2.5 py-1.5 select-all">crobe --ui</pre>
      <p class="text-[11px] text-zinc-500 dark:text-zinc-500">
        You can safely close this browser window or tab.
      </p>
    </div>

    <!-- Actions -->
    <div class="pt-2 flex justify-center">
      <Button
        variant="secondary"
        size="sm"
        onclick={handleCloseWindow}
        class="gap-1.5 border border-zinc-200 dark:border-zinc-700"
      >
        <X class="h-4 w-4" />
        Close Window
      </Button>
    </div>
  </div>
</div>
