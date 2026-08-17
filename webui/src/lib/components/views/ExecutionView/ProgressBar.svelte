<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { appState } from '$lib/state/appState.svelte';
  import Clock from 'lucide-svelte/icons/clock';
  import Check from 'lucide-svelte/icons/check';
  import X from 'lucide-svelte/icons/x';
  import Circle from 'lucide-svelte/icons/circle';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import { cn } from '$lib/utils/cn';

  interface Props {
    onCancelRequest?: () => void;
    class?: string;
  }

  let {
    onCancelRequest,
    class: className = '',
  }: Props = $props();

  let elapsedMs = $state(0);
  let startTime = $state<number>(Date.now());
  let timerInterval: ReturnType<typeof setInterval> | null = null;

  onMount(() => {
    startTime = Date.now();
    timerInterval = setInterval(() => {
      if (appState.isRunning) {
        elapsedMs = Date.now() - startTime;
      }
    }, 100);
  });

  onDestroy(() => {
    if (timerInterval) clearInterval(timerInterval);
  });

  // Derived stopwatch string "mm:ss.s"
  const formattedTime = $derived.by(() => {
    const totalMs = appState.execution?.duration_ms && !appState.isRunning
      ? appState.execution.duration_ms
      : elapsedMs;

    const totalSeconds = Math.floor(totalMs / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    const tenths = Math.floor((totalMs % 1000) / 100);

    const mStr = String(minutes).padStart(2, '0');
    const sStr = String(seconds).padStart(2, '0');
    return `${mStr}:${sStr}.${tenths}s`;
  });

  const pendingCount = $derived(
    Math.max(0, appState.totalAssertions - appState.completedAssertions)
  );
</script>

<div
  class={cn(
    'sticky top-14 z-20 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm p-4 shadow-md transition-all space-y-3',
    className
  )}
>
  <div class="flex flex-wrap items-center justify-between gap-3">
    <!-- Timer & Active Run Context -->
    <div class="flex items-center gap-3">
      <div class="flex items-center gap-1.5 text-zinc-900 dark:text-zinc-100 font-mono text-sm font-semibold">
        <Clock class="h-4 w-4 text-sky-500" />
        <span aria-label="Elapsed time">{formattedTime}</span>
      </div>

      {#if appState.execution?.run_id || appState.activeRunId}
        <span class="hidden sm:inline-flex px-2 py-0.5 rounded text-[11px] font-mono bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700">
          Run: {appState.execution?.run_id || appState.activeRunId}
        </span>
      {/if}
    </div>

    <!-- Status Badges & Percentage -->
    <div class="flex items-center gap-2 font-mono text-xs">
      <span class="text-zinc-600 dark:text-zinc-400 font-semibold">
        {appState.progressPercent}%
        <span class="font-normal text-zinc-400 dark:text-zinc-500">
          ({appState.completedAssertions}/{appState.totalAssertions})
        </span>
      </span>

      <!-- Passed Badge -->
      <span
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 text-[11px] font-medium"
        title="Passed assertions"
      >
        <Check class="h-3 w-3" />
        <span>{appState.passedAssertions}</span>
      </span>

      <!-- Failed Badge -->
      <span
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 text-[11px] font-medium"
        title="Failed assertions"
      >
        <X class="h-3 w-3" />
        <span>{appState.failedAssertions}</span>
      </span>

      <!-- Pending Badge -->
      <span
        class="hidden sm:inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700 text-[11px] font-medium"
        title="Pending assertions"
      >
        <Circle class="h-2.5 w-2.5" />
        <span>{pendingCount}</span>
      </span>

      <!-- Cancellation Action Button -->
      {#if appState.isCancelling}
        <button
          type="button"
          disabled
          class="ml-2 inline-flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-md bg-rose-500/10 text-rose-400 border border-rose-500/30 opacity-70 cursor-not-allowed"
        >
          <Loader2 class="h-3 w-3 animate-spin" />
          <span>Cancelling...</span>
        </button>
      {:else}
        <button
          type="button"
          onclick={onCancelRequest}
          class="ml-2 inline-flex items-center gap-1 px-3 py-1 text-xs font-semibold rounded-md border border-rose-500/40 text-rose-600 dark:text-rose-400 hover:bg-rose-500/10 transition-colors cursor-pointer"
        >
          <X class="h-3.5 w-3.5" />
          <span>Cancel Run</span>
        </button>
      {/if}
    </div>
  </div>

  <!-- Animated Progress Bar Track -->
  <div
    class="h-2 w-full rounded-full bg-zinc-100 dark:bg-zinc-800 overflow-hidden border border-zinc-200 dark:border-zinc-700/60"
    role="progressbar"
    aria-valuenow={appState.progressPercent}
    aria-valuemin={0}
    aria-valuemax={100}
    aria-label="Playbook execution progress"
  >
    <div
      class="h-full bg-sky-500 transition-all duration-300 ease-out rounded-full"
      style="width: {appState.progressPercent}%"
    ></div>
  </div>
</div>
