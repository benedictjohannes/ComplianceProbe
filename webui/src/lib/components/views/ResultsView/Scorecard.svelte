<script lang="ts">
  import { appState } from '$lib/state/appState.svelte';
  import Check from 'lucide-svelte/icons/check';
  import X from 'lucide-svelte/icons/x';
  import Ban from 'lucide-svelte/icons/ban';
  import Clock from 'lucide-svelte/icons/clock';
  import FileText from 'lucide-svelte/icons/file-text';
  import Server from 'lucide-svelte/icons/server';
  import ShieldCheck from 'lucide-svelte/icons/shield-check';
  import ShieldAlert from 'lucide-svelte/icons/shield-alert';
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';
  import { cn } from '$lib/utils/cn';

  interface Props {
    class?: string;
  }

  let { class: className = '' }: Props = $props();

  const totalAssertions = $derived(appState.totalAssertions);
  const passedAssertions = $derived(appState.passedAssertions);
  const failedAssertions = $derived(appState.failedAssertions);
  const isCancelled = $derived(
    appState.execution?.status === 'cancelled' ||
    appState.status === 'running.cancelling'
  );

  const completedCount = $derived(appState.completedAssertions);
  const skippedAssertions = $derived(
    Math.max(0, totalAssertions - completedCount)
  );

  // Compliance Pass Rate calculation (strictly: (passed / total) * 100)
  const passRate = $derived.by(() => {
    if (totalAssertions <= 0) return 0;
    return Math.round((passedAssertions / totalAssertions) * 100);
  });

  // Gauge colors & status
  const isExcellent = $derived(passRate >= 90);
  const isModerate = $derived(passRate >= 70 && passRate < 90);
  const isPoor = $derived(passRate < 70);

  const overallStatus = $derived.by(() => {
    if (isCancelled) return 'ABORTED';
    if (failedAssertions === 0 && passedAssertions > 0 && passedAssertions === totalAssertions) {
      return 'PASSED';
    }
    return 'FAILED';
  });

  // SVG Gauge Math (r = 44, C = 2 * PI * 44 ≈ 276.46)
  const radius = 44;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = $derived(
    circumference - (passRate / 100) * circumference
  );

  // Formatted duration
  const formattedDuration = $derived.by(() => {
    const totalMs = appState.execution?.duration_ms || 0;
    if (totalMs < 1000) return `${totalMs}ms`;
    const totalSecs = (totalMs / 1000).toFixed(2);
    return `${totalSecs}s`;
  });
</script>

<div
  class={cn(
    'rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/90 shadow-sm p-6 sm:p-7 space-y-6',
    className
  )}
>
  <!-- Top Hero Grid: Gauge & Key Metrics -->
  <div class="flex flex-col md:flex-row items-start md:items-center justify-between gap-6 sm:gap-8 pb-6 border-b border-zinc-200 dark:border-zinc-800">
    <!-- Left: Radial Gauge Ring & Pass Rate -->
    <div class="flex items-center gap-6 min-w-0 flex-1">
      <div class="relative flex items-center justify-center shrink-0">
        <svg
          class="w-28 h-28 transform -rotate-90"
          viewBox="0 0 100 100"
          aria-label="Compliance Pass Rate Gauge"
          role="img"
        >
          <!-- Background Track -->
          <circle
            cx="50"
            cy="50"
            r={radius}
            class="stroke-zinc-100 dark:stroke-zinc-800/80"
            stroke-width="9"
            fill="transparent"
          />
          <!-- Value Ring -->
          <circle
            cx="50"
            cy="50"
            r={radius}
            class={cn(
              'transition-all duration-700 ease-out',
              isExcellent
                ? 'stroke-emerald-500'
                : isModerate
                ? 'stroke-amber-500'
                : 'stroke-rose-500'
            )}
            stroke-width="9"
            stroke-linecap="round"
            stroke-dasharray={circumference}
            stroke-dashoffset={strokeDashoffset}
            fill="transparent"
          />
        </svg>

        <!-- Center Rate Text -->
        <div class="absolute inset-0 flex flex-col items-center justify-center text-center select-none">
          <span class={cn(
            'text-2xl font-bold tracking-tight',
            isExcellent
              ? 'text-emerald-600 dark:text-emerald-400'
              : isModerate
              ? 'text-amber-600 dark:text-amber-400'
              : 'text-rose-600 dark:text-rose-400'
          )}>
            {passRate}%
          </span>
          <span class="text-[10px] font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
            Pass Rate
          </span>
        </div>
      </div>

      <!-- Playbook & Overall Status Title -->
      <div class="space-y-2 min-w-0 flex-1">
        <div class="flex items-center gap-2.5 flex-wrap">
          <!-- Overall Status Badge -->
          {#if overallStatus === 'PASSED'}
            <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-bold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 shrink-0">
              <ShieldCheck class="h-3.5 w-3.5" />
              PASSED
            </span>
          {:else if overallStatus === 'ABORTED'}
            <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-bold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30 shrink-0">
              <Ban class="h-3.5 w-3.5" />
              ABORTED
            </span>
          {:else}
            <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-bold bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/30 shrink-0">
              <ShieldAlert class="h-3.5 w-3.5" />
              FAILED
            </span>
          {/if}

          <span class="text-xs font-mono text-zinc-400 dark:text-zinc-500 truncate">
            Run: {appState.execution?.run_id || 'run'}
          </span>
        </div>

        <h2 class="text-xl sm:text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-100 break-words line-clamp-2" title={appState.playbook?.title || 'Compliance Audit'}>
          {appState.playbook?.title || 'Compliance Audit'}
        </h2>

        <p class="text-xs text-zinc-500 dark:text-zinc-400 max-w-md">
          {#if overallStatus === 'PASSED'}
            All security baseline assertions satisfied full policy criteria.
          {:else if overallStatus === 'ABORTED'}
            Execution was cancelled before completing all assertions.
          {:else}
            {failedAssertions} assertion{failedAssertions === 1 ? '' : 's'} failed compliance criteria. Review audit breakdown below.
          {/if}
        </p>
      </div>
    </div>

    <!-- Right: Execution Meta Info Card -->
    <div class="w-full md:w-auto grid grid-cols-2 sm:grid-cols-2 gap-3 p-3.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800/80 text-xs shrink-0">
      <!-- Duration -->
      <div class="space-y-1">
        <div class="flex items-center gap-1.5 text-zinc-400 dark:text-zinc-500">
          <Clock class="h-3.5 w-3.5" />
          <span>Duration</span>
        </div>
        <div class="font-mono font-semibold text-zinc-800 dark:text-zinc-200">
          {formattedDuration}
        </div>
      </div>

      <!-- Total Rules -->
      <div class="space-y-1">
        <div class="flex items-center gap-1.5 text-zinc-400 dark:text-zinc-500">
          <FileText class="h-3.5 w-3.5" />
          <span>Total Assertions</span>
        </div>
        <div class="font-mono font-semibold text-zinc-800 dark:text-zinc-200">
          {totalAssertions} rules
        </div>
      </div>
    </div>
  </div>

  <!-- Strict Assertion Counters Cards -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3.5">
    <!-- Total Assertions -->
    <div class="p-3.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/50 border border-zinc-200 dark:border-zinc-800/80 space-y-1">
      <div class="text-[11px] font-medium text-zinc-500 dark:text-zinc-400">
        Total Assertions
      </div>
      <div class="text-xl font-bold font-mono text-zinc-900 dark:text-zinc-100">
        {totalAssertions}
      </div>
    </div>

    <!-- Passed -->
    <div class="p-3.5 rounded-xl bg-emerald-500/5 border border-emerald-500/20 space-y-1">
      <div class="text-[11px] font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
        <Check class="h-3.5 w-3.5 stroke-[2.5]" />
        <span>Passed</span>
      </div>
      <div class="text-xl font-bold font-mono text-emerald-600 dark:text-emerald-400">
        {passedAssertions}
      </div>
    </div>

    <!-- Failed -->
    <div class="p-3.5 rounded-xl bg-rose-500/5 border border-rose-500/20 space-y-1">
      <div class="text-[11px] font-semibold text-rose-600 dark:text-rose-400 flex items-center gap-1">
        <X class="h-3.5 w-3.5 stroke-[2.5]" />
        <span>Failed</span>
      </div>
      <div class="text-xl font-bold font-mono text-rose-600 dark:text-rose-400">
        {failedAssertions}
      </div>
    </div>

    <!-- Skipped / Aborted -->
    <div class="p-3.5 rounded-xl bg-zinc-50 dark:bg-zinc-950/50 border border-zinc-200 dark:border-zinc-800/80 space-y-1">
      <div class="text-[11px] font-medium text-zinc-500 dark:text-zinc-400 flex items-center gap-1">
        <Ban class="h-3.5 w-3.5" />
        <span>Skipped</span>
      </div>
      <div class="text-xl font-bold font-mono text-zinc-700 dark:text-zinc-300">
        {skippedAssertions}
      </div>
    </div>
  </div>
</div>
