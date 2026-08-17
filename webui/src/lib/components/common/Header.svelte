<script lang="ts">
  import Shield from 'lucide-svelte/icons/shield';
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import PowerOff from 'lucide-svelte/icons/power-off';
  import Check from 'lucide-svelte/icons/check';
  import { Badge, Button } from '$lib/components/ui';
  import ConnectionStatus, { type ConnectionState } from './ConnectionStatus.svelte';
  import ThemeToggle from './ThemeToggle.svelte';
  import ShutdownModal from './ShutdownModal.svelte';
  import { cn } from '$lib/utils/cn';

  export interface PipelineStep {
    id: number;
    label: string;
    description: string;
  }

  const PIPELINE_STEPS: PipelineStep[] = [
    { id: 1, label: 'Load', description: 'Load Playbook' },
    { id: 2, label: 'Inspect', description: 'Inspect & Configure' },
    { id: 3, label: 'Execute', description: 'Live Streaming' },
    { id: 4, label: 'Results', description: 'Scorecard & Reports' },
  ];

  interface Props {
    activeStep?: number;
    playbookName?: string;
    connectionState?: ConnectionState;
    maxAccessibleStep?: number;
    class?: string;
    onstepchange?: (step: number) => void;
    onshutdown?: () => Promise<void> | void;
  }

  let {
    activeStep = 1,
    playbookName,
    connectionState = 'connected',
    maxAccessibleStep = 1,
    class: className = '',
    onstepchange,
    onshutdown,
  }: Props = $props();

  let shutdownModalOpen = $state(false);

  function canNavigateTo(step: number): boolean {
    return step <= maxAccessibleStep && step !== activeStep;
  }

  function handleStepClick(step: number) {
    if (canNavigateTo(step)) {
      onstepchange?.(step);
    }
  }

  function handlePrevStep() {
    if (activeStep > 1 && canNavigateTo(activeStep - 1)) {
      onstepchange?.(activeStep - 1);
    }
  }

  function handleNextStep() {
    if (activeStep < 4 && canNavigateTo(activeStep + 1)) {
      onstepchange?.(activeStep + 1);
    }
  }

  const progressPercentage = $derived((activeStep / 4) * 100);
</script>

<header class={cn('sticky top-0 z-40 w-full border-b border-zinc-800 bg-zinc-950/95 backdrop-blur-none', className)}>
  <!-- Desktop & Tablet Header (>= 768px) -->
  <div class="hidden md:flex h-14 items-center justify-between px-4 max-w-7xl mx-auto">
    <!-- Left Zone: Brand & Context -->
    <div class="flex items-center gap-3 min-w-[200px]">
      <div class="flex items-center gap-2 font-bold tracking-tight text-zinc-100 select-none">
        <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-sky-500/10 text-sky-400 border border-sky-500/20">
          <Shield class="h-4 w-4" />
        </div>
        <span class="text-base font-semibold">crobe</span>
        <span class="text-[10px] font-mono font-normal text-zinc-500 bg-zinc-900 border border-zinc-800 px-1.5 py-0.5 rounded">
          v0.1
        </span>
      </div>

      {#if playbookName}
        <Badge variant="code" size="sm" class="truncate max-w-[160px] font-mono">
          {playbookName}
        </Badge>
      {/if}
    </div>

    <!-- Center Zone: 4-Step Pipeline Breadcrumb -->
    <nav class="flex items-center gap-1.5 select-none" aria-label="Pipeline Steps">
      {#each PIPELINE_STEPS as step, idx (step.id)}
        {#if idx > 0}
          <div class="h-px w-4 bg-zinc-800 shrink-0"></div>
        {/if}

        <button
          type="button"
          disabled={!canNavigateTo(step.id)}
          onclick={() => handleStepClick(step.id)}
          class={cn(
            'flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all',
            activeStep === step.id && 'bg-sky-500/10 text-sky-400 border border-sky-500/30 font-semibold shadow-xs',
            activeStep > step.id && 'text-zinc-300 hover:text-white cursor-pointer hover:bg-zinc-900',
            activeStep < step.id && 'text-zinc-600 cursor-not-allowed opacity-60'
          )}
        >
          {#if activeStep > step.id}
            <span class="flex h-3.5 w-3.5 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400">
              <Check class="h-2.5 w-2.5" />
            </span>
          {:else if activeStep === step.id}
            <span class="flex h-2 w-2 rounded-full bg-sky-400 shadow-[0_0_6px_rgba(56,189,248,0.8)]"></span>
          {:else}
            <span class="flex h-2 w-2 rounded-full bg-zinc-700"></span>
          {/if}
          <span>{step.id}. {step.label}</span>
        </button>
      {/each}
    </nav>

    <!-- Right Zone: Connection Status, Theme Toggle & Server Stop -->
    <div class="flex items-center gap-3 min-w-[200px] justify-end">
      <ConnectionStatus state={connectionState} />
      <div class="h-4 w-px bg-zinc-800"></div>
      <ThemeToggle />
      <Button
        variant="ghost"
        size="xs"
        onclick={() => {
          shutdownModalOpen = true;
        }}
        class="text-zinc-400 hover:text-rose-400 hover:bg-rose-500/10 border border-transparent hover:border-rose-500/20"
      >
        <PowerOff class="h-3.5 w-3.5 mr-1" />
        <span class="text-xs">Stop</span>
      </Button>
    </div>
  </div>

  <!-- Mobile Header (< 768px, 2-Tier Layout) -->
  <div class="md:hidden flex flex-col">
    <!-- Tier 1: Identity & Status -->
    <div class="flex h-11 items-center justify-between px-3 border-b border-zinc-800/80">
      <div class="flex items-center gap-2 font-bold tracking-tight text-zinc-100">
        <div class="flex h-6 w-6 items-center justify-center rounded-md bg-sky-500/10 text-sky-400 border border-sky-500/20">
          <Shield class="h-3.5 w-3.5" />
        </div>
        <span class="text-sm font-semibold">crobe</span>
        <span class="text-[9px] font-mono text-zinc-500 bg-zinc-900 px-1 py-0.2 rounded border border-zinc-800">
          v0.1
        </span>
        {#if playbookName}
          <span class="text-[10px] font-mono text-sky-400 bg-sky-500/10 px-1.5 py-0.5 rounded truncate max-w-[100px]">
            {playbookName}
          </span>
        {/if}
      </div>

      <div class="flex items-center gap-2">
        <ConnectionStatus state={connectionState} />
        <ThemeToggle />
      </div>
    </div>

    <!-- Tier 2: Pager & Stop Action -->
    <div class="flex h-10 items-center justify-between px-3 bg-zinc-900/40">
      <div class="flex items-center gap-1.5">
        <button
          type="button"
          disabled={activeStep <= 1 || !canNavigateTo(activeStep - 1)}
          onclick={handlePrevStep}
          aria-label="Previous step"
          class="p-1 rounded text-zinc-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
        >
          <ChevronLeft class="h-4 w-4" />
        </button>

        <span class="text-xs font-medium text-zinc-200">
          Step {activeStep} of 4: <span class="text-sky-400 font-semibold">{PIPELINE_STEPS[activeStep - 1]?.label}</span>
        </span>

        <button
          type="button"
          disabled={activeStep >= 4 || !canNavigateTo(activeStep + 1)}
          onclick={handleNextStep}
          aria-label="Next step"
          class="p-1 rounded text-zinc-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
        >
          <ChevronRight class="h-4 w-4" />
        </button>
      </div>

      <Button
        variant="ghost"
        size="xs"
        onclick={() => {
          shutdownModalOpen = true;
        }}
        class="text-zinc-400 hover:text-rose-400 hover:bg-rose-500/10 h-7 text-xs"
      >
        <PowerOff class="h-3 w-3 mr-1" />
        Stop
      </Button>
    </div>

    <!-- 1px Progress Track at Bottom of Mobile Nav -->
    <div class="w-full h-[2px] bg-zinc-800">
      <div
        class="h-full bg-sky-500 transition-all duration-300 ease-out"
        style="width: {progressPercentage}%"
      ></div>
    </div>
  </div>

  <ShutdownModal bind:open={shutdownModalOpen} {onshutdown} />
</header>
