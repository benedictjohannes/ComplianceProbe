<script lang="ts">
  import type { Assertion, AssertionSnapshot, AssertionExecutionStatus } from '$lib/api/types';
  import Check from 'lucide-svelte/icons/check';
  import X from 'lucide-svelte/icons/x';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Circle from 'lucide-svelte/icons/circle';
  import Ban from 'lucide-svelte/icons/ban';
  import Shield from 'lucide-svelte/icons/shield';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';
  import CodeBlock from '$lib/components/ui/CodeBlock.svelte';
  import { cn } from '$lib/utils/cn';

  interface Props {
    assertion: Partial<Assertion> & { code: string; title: string };
    snapshot?: AssertionSnapshot;
    isActive?: boolean;
    class?: string;
  }

  let {
    assertion,
    snapshot,
    isActive = false,
    class: className = '',
  }: Props = $props();

  const status: AssertionExecutionStatus = $derived.by(() => {
    if (snapshot?.status) return snapshot.status;
    if (isActive) return 'running';
    return 'pending';
  });

  const isRunning = $derived(status === 'running');
  const isPassed = $derived(status === 'passed');
  const isFailed = $derived(status === 'failed');
  const isCancelled = $derived(status === 'cancelled');
  const isPending = $derived(status === 'pending');

  let isManuallyToggled = $state<boolean | null>(null);

  // Auto-expand on failure or while running, unless user explicitly toggled
  const isExpanded = $derived.by(() => {
    if (isManuallyToggled !== null) return isManuallyToggled;
    if (isFailed || isRunning) return true;
    return false;
  });

  function handleToggle() {
    isManuallyToggled = !isExpanded;
  }

  const requiresElevation = $derived.by(() => {
    return Boolean(
      assertion.cmds?.some((c) => c.exec?.requireElevation) ||
      assertion.preCmds?.some((e) => e?.requireElevation) ||
      assertion.postCmds?.some((e) => e?.requireElevation)
    );
  });
</script>

<div
  id="assertion-{assertion.code}"
  class={cn(
    'rounded-xl border transition-all duration-200 overflow-hidden',
    isRunning
      ? 'border-sky-500/60 bg-sky-500/5 ring-1 ring-sky-500/30'
      : isPassed
      ? 'border-emerald-500/30 bg-emerald-500/5'
      : isFailed
      ? 'border-rose-500/40 bg-rose-500/5'
      : isCancelled
      ? 'border-amber-500/30 bg-amber-500/5'
      : 'border-zinc-200 dark:border-zinc-800 bg-white/70 dark:bg-zinc-900/60 opacity-80',
    className
  )}
>
  <!-- Interactive Header -->
  <button
    type="button"
    onclick={handleToggle}
    class="w-full text-left p-3.5 flex items-center justify-between gap-3 hover:bg-zinc-500/5 transition-colors cursor-pointer select-none"
    aria-expanded={isExpanded}
  >
    <!-- Left: Status Icon, Code, Title -->
    <div class="flex items-center gap-2.5 min-w-0 flex-1">
      <div class="shrink-0">
        {#if isRunning}
          <Loader2 class="h-4 w-4 text-sky-500 animate-spin" />
        {:else if isPassed}
          <div class="h-4 w-4 rounded-full bg-emerald-500/20 text-emerald-500 flex items-center justify-center">
            <Check class="h-3 w-3 stroke-[3]" />
          </div>
        {:else if isFailed}
          <div class="h-4 w-4 rounded-full bg-rose-500/20 text-rose-500 flex items-center justify-center">
            <X class="h-3 w-3 stroke-[3]" />
          </div>
        {:else if isCancelled}
          <Ban class="h-4 w-4 text-amber-500" />
        {:else}
          <Circle class="h-4 w-4 text-zinc-400 dark:text-zinc-600" />
        {/if}
      </div>

      <!-- Code Badge -->
      <span class={cn(
        'font-mono text-xs font-semibold px-2 py-0.5 rounded shrink-0',
        isRunning ? 'bg-sky-500/15 text-sky-600 dark:text-sky-300' :
        isPassed ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300' :
        isFailed ? 'bg-rose-500/15 text-rose-700 dark:text-rose-300' :
        'bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400'
      )}>
        {assertion.code}
      </span>

      <!-- Title -->
      <span class={cn(
        'text-sm font-medium truncate',
        isPending ? 'text-zinc-500 dark:text-zinc-400' : 'text-zinc-900 dark:text-zinc-100'
      )}>
        {assertion.title}
      </span>
    </div>

    <!-- Right: Badges & Accordion Trigger -->
    <div class="flex items-center gap-2 shrink-0 font-mono text-xs">
      {#if requiresElevation}
        <span class="hidden sm:inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-sans font-medium bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
          <Shield class="h-2.5 w-2.5" />
          <span>sudo</span>
        </span>
      {/if}

      <!-- Status / Score Pill -->
      {#if isPassed && snapshot}
        <span class="px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 text-[11px] font-semibold">
          +{snapshot.score} {snapshot.score === 1 ? 'pt' : 'pts'}
        </span>
      {:else if isFailed && snapshot}
        <span class="px-2 py-0.5 rounded-full bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/30 text-[11px] font-semibold">
          {snapshot.score}/{snapshot.min_score} pts
        </span>
      {:else if isRunning}
        <span class="px-2 py-0.5 rounded-full bg-sky-500/15 text-sky-600 dark:text-sky-400 border border-sky-500/30 text-[11px] font-medium animate-pulse">
          Running
        </span>
      {:else if isCancelled}
        <span class="px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30 text-[11px] font-medium">
          Cancelled
        </span>
      {:else}
        <span class="px-2 py-0.5 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-500 text-[11px]">
          Pending
        </span>
      {/if}

      <!-- Duration -->
      {#if snapshot?.duration_ms && snapshot.duration_ms > 0}
        <span class="text-zinc-400 dark:text-zinc-500 text-[11px]">
          {snapshot.duration_ms}ms
        </span>
      {/if}

      <!-- Chevron -->
      <div class="text-zinc-400 dark:text-zinc-500">
        {#if isExpanded}
          <ChevronDown class="h-4 w-4" />
        {:else}
          <ChevronRight class="h-4 w-4" />
        {/if}
      </div>
    </div>
  </button>

  <!-- Expanded Details Body -->
  {#if isExpanded}
    <div class="px-4 pb-4 pt-1 border-t border-zinc-200 dark:border-zinc-800/80 space-y-3 text-xs">
      <!-- Description -->
      {#if assertion.description}
        <p class="text-zinc-600 dark:text-zinc-300 leading-relaxed">
          {assertion.description}
        </p>
      {/if}

      <!-- Expectations Grid -->
      {#if assertion.passDescription || assertion.failDescription}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px] p-2.5 rounded-lg bg-zinc-100/80 dark:bg-zinc-950/60 border border-zinc-200 dark:border-zinc-800">
          {#if assertion.passDescription}
            <div class="space-y-0.5">
              <span class="font-semibold text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                <Check class="h-3 w-3" /> Pass Criteria:
              </span>
              <p class="text-zinc-600 dark:text-zinc-400">{assertion.passDescription}</p>
            </div>
          {/if}
          {#if assertion.failDescription}
            <div class="space-y-0.5">
              <span class="font-semibold text-rose-600 dark:text-rose-400 flex items-center gap-1">
                <X class="h-3 w-3" /> Fail Criteria:
              </span>
              <p class="text-zinc-600 dark:text-zinc-400">{assertion.failDescription}</p>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Failure Diagnostic Banner -->
      {#if isFailed}
        <div class="flex items-start gap-2 p-2.5 rounded-lg bg-rose-500/10 border border-rose-500/30 text-rose-800 dark:text-rose-300 text-xs">
          <AlertTriangle class="h-4 w-4 text-rose-500 shrink-0 mt-0.5" />
          <div class="space-y-1">
            <span class="font-bold">Execution Failed</span>
            <p class="text-[11px] text-rose-700 dark:text-rose-400">
              Assertion criteria were not satisfied. Review the probe output below for diagnostics.
            </p>
          </div>
        </div>
      {/if}

      <!-- Probe Output CodeBlock -->
      {#if snapshot?.output}
        <div class="space-y-1">
          <div class="text-[11px] font-semibold text-zinc-700 dark:text-zinc-300">
            Probe Output & Rules Diagnostic:
          </div>
          <CodeBlock
            code={snapshot.output}
            variant="terminal"
            copyable={true}
          />
        </div>
      {/if}
    </div>
  {/if}
</div>
