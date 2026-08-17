<script lang="ts">
  import Code from 'lucide-svelte/icons/code';
  import Lock from 'lucide-svelte/icons/lock';
  import EyeOff from 'lucide-svelte/icons/eye-off';
  import Terminal from 'lucide-svelte/icons/terminal';
  import { Badge } from '$lib/components/ui';
  import { cn } from '$lib/utils/cn';
  import type { Cmd, Exec } from '$lib/api/types';

  interface Props {
    type: 'pre' | 'main' | 'post';
    index: number;
    total: number;
    exec: Exec;
    cmd?: Cmd;
    class?: string;
  }

  let {
    type,
    index,
    total,
    exec,
    cmd,
    class: className = '',
  }: Props = $props();

  const typeConfig = $derived.by(() => {
    switch (type) {
      case 'pre':
        return {
          label: 'Pre-Command',
          color: 'text-sky-700 dark:text-sky-400',
        };
      case 'post':
        return {
          label: 'Post-Command',
          color: 'text-purple-700 dark:text-purple-400',
        };
      case 'main':
      default:
        return {
          label: 'Command',
          color: 'text-emerald-700 dark:text-emerald-400',
        };
    }
  });

  function formatExitCodeRule(min?: number, max?: number, result?: number): string {
    const resText = result === 1 ? '+1' : result === -1 ? '-1' : '0';
    if (min !== undefined && max !== undefined) {
      if (min === max) return `[${min} ➜ ${resText}]`;
      return `[${min}..${max} ➜ ${resText}]`;
    }
    if (min !== undefined) return `[≥ ${min} ➜ ${resText}]`;
    if (max !== undefined) return `[≤ ${max} ➜ ${resText}]`;
    return `[any ➜ ${resText}]`;
  }
</script>

<div
  class={cn(
    'rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/70 p-2.5 space-y-2 text-xs font-mono',
    className
  )}
>
  <!-- Command Header -->
  <div class="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-200 dark:border-zinc-800/80 pb-1.5 text-zinc-600 dark:text-zinc-400">
    <div class="flex items-center gap-1.5">
      <span class={cn('font-semibold', typeConfig.color)}>
        [{typeConfig.label} {index + 1}/{total}]
      </span>
      {#if cmd && (cmd.passScore !== undefined || cmd.failScore !== undefined)}
        <span class="bg-zinc-200/80 dark:bg-zinc-800 px-1.5 py-0.5 rounded text-[11px] text-zinc-800 dark:text-zinc-300">
          Score: <strong class="text-emerald-600 dark:text-emerald-400">+{cmd.passScore ?? 1}</strong> / <strong class="text-rose-600 dark:text-rose-400">{cmd.failScore ?? -1}</strong>
        </span>
      {/if}
    </div>

    <div class="flex flex-wrap items-center gap-1.5 text-[11px]">
      {#if exec.shell}
        <span class="bg-zinc-200 dark:bg-zinc-800 px-1.5 py-0.5 rounded text-zinc-800 dark:text-zinc-300">
          Shell: {exec.shell}
        </span>
      {:else if exec.shellFunc}
        <span class="bg-purple-100 dark:bg-purple-950/60 border border-purple-300 dark:border-purple-800/50 px-1.5 py-0.5 rounded text-purple-800 dark:text-purple-300">
          ⚡ Shell: JS func
        </span>
      {/if}

      {#if exec.scriptFileExtension}
        <span class="bg-zinc-200/60 dark:bg-zinc-800/60 px-1.5 py-0.5 rounded text-zinc-700 dark:text-zinc-400">
          .{exec.scriptFileExtension}
        </span>
      {/if}

      {#if exec.requireElevation}
        <Badge variant="warning" size="sm" class="text-[10px]">
          <Lock class="h-2.5 w-2.5 mr-0.5" />
          sudo
        </Badge>
      {/if}

      {#if exec.excludeFromReport}
        <Badge variant="outline" size="sm" class="text-[10px] text-zinc-500 gap-0.5">
          <EyeOff class="h-2.5 w-2.5" />
          Excluded
        </Badge>
      {/if}
    </div>
  </div>

  <!-- Dynamic Shell Function Preview (if present) -->
  {#if exec.shellFunc}
    <div class="space-y-1">
      <div class="text-[10px] text-purple-700 dark:text-purple-400 font-semibold flex items-center gap-1">
        <Code class="h-3 w-3" />
        Dynamic Shell Function:
      </div>
      <div class="rounded bg-zinc-900 border border-zinc-800 p-2 text-purple-200 text-[11px] overflow-x-auto select-text whitespace-pre-wrap">
        {exec.shellFunc}
      </div>
    </div>
  {/if}

  <!-- Script or Dynamic Script Function -->
  {#if exec.func}
    <div class="space-y-1">
      <div class="text-[10px] text-purple-700 dark:text-purple-400 font-semibold flex items-center gap-1">
        <Code class="h-3 w-3" />
        Dynamic Script Function ({'({ assertionContext, env, os, arch, user, cwd })'}):
      </div>
      <div class="rounded bg-zinc-900 border border-zinc-800 p-2 text-purple-200 text-[11px] overflow-x-auto select-text whitespace-pre-wrap">
        {exec.func}
      </div>
    </div>
  {:else if exec.script}
    <div class="rounded bg-zinc-900 border border-zinc-800 p-2 text-zinc-100 overflow-x-auto select-text whitespace-pre-wrap">
      {exec.script}
    </div>
  {/if}

  <!-- Evaluation Rules (Main Commands) -->
  {#if cmd && ((cmd.stdOutRule && (cmd.stdOutRule.regex || cmd.stdOutRule.func)) || (cmd.stdErrRule && (cmd.stdErrRule.regex || cmd.stdErrRule.func)) || (cmd.exitCodeRules && cmd.exitCodeRules.length > 0))}
    <div class="space-y-1.5 text-zinc-600 dark:text-zinc-400 text-[11px] pt-1 border-t border-zinc-200/80 dark:border-zinc-800/60">
      {#if cmd.stdOutRule && (cmd.stdOutRule.regex || cmd.stdOutRule.func)}
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-zinc-500 font-medium">StdOut Rule:</span>
          {#if cmd.stdOutRule.regex}
            <span class="text-amber-700 dark:text-amber-300">
              regex <code class="bg-amber-500/10 px-1 py-0.5 rounded text-amber-800 dark:text-amber-200">/{cmd.stdOutRule.regex}/</code>
            </span>
          {:else if cmd.stdOutRule.func}
            <span class="text-purple-700 dark:text-purple-300">
              ⚡ JS Func <code class="bg-purple-500/10 px-1 py-0.5 rounded text-purple-800 dark:text-purple-200">{cmd.stdOutRule.func}</code>
            </span>
          {/if}
          {#if cmd.stdOutRule.includeStdErr}
            <span class="text-[10px] text-zinc-500 bg-zinc-200 dark:bg-zinc-800 px-1 rounded">[+stderr]</span>
          {/if}
        </div>
      {/if}

      {#if cmd.stdErrRule && (cmd.stdErrRule.regex || cmd.stdErrRule.func)}
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-rose-600 dark:text-rose-400 font-medium">StdErr Rule:</span>
          {#if cmd.stdErrRule.regex}
            <span class="text-rose-700 dark:text-rose-300">
              regex <code class="bg-rose-500/10 px-1 py-0.5 rounded text-rose-800 dark:text-rose-200">/{cmd.stdErrRule.regex}/</code>
            </span>
          {:else if cmd.stdErrRule.func}
            <span class="text-purple-700 dark:text-purple-300">
              ⚡ JS Func <code class="bg-purple-500/10 px-1 py-0.5 rounded text-purple-800 dark:text-purple-200">{cmd.stdErrRule.func}</code>
            </span>
          {/if}
          {#if cmd.stdErrRule.includeStdErr}
            <span class="text-[10px] text-zinc-500 bg-zinc-200 dark:bg-zinc-800 px-1 rounded">[+stderr]</span>
          {/if}
        </div>
      {/if}

      {#if cmd.exitCodeRules && cmd.exitCodeRules.length > 0}
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-zinc-500 font-medium">Exit Code Rules:</span>
          <div class="flex flex-wrap gap-1 text-zinc-800 dark:text-zinc-300">
            {#each cmd.exitCodeRules as rule}
              <span class="bg-zinc-200/80 dark:bg-zinc-800/80 px-1.5 py-0.5 rounded text-[10px]">
                {formatExitCodeRule(rule.min, rule.max, rule.result)}
              </span>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/if}

  <!-- Variable Gather Specifications -->
  {#if exec.gather && exec.gather.length > 0}
    <div class="space-y-1 text-zinc-600 dark:text-zinc-400 text-[11px] pt-1 border-t border-zinc-200/80 dark:border-zinc-800/60">
      <div class="text-[10px] text-zinc-500 uppercase tracking-wider font-semibold">Gathered Context:</div>
      {#each exec.gather as g}
        <div class="flex flex-wrap items-center gap-1.5 pl-2">
          <span>•</span>
          <span>var <code class="text-sky-700 dark:text-sky-300 font-semibold bg-sky-500/10 px-1 py-0.5 rounded">"{g.key}"</code></span>
          <span>⇐</span>
          {#if g.regex}
            <span>regex <code class="text-amber-700 dark:text-amber-300 bg-amber-500/10 px-1 py-0.5 rounded">/{g.regex}/</code></span>
          {:else if g.func}
            <span class="text-purple-700 dark:text-purple-300">⚡ JS Func <code class="bg-purple-500/10 px-1 py-0.5 rounded">{g.func}</code></span>
          {/if}
          {#if g.includeStdErr}
            <span class="text-[10px] text-zinc-500 bg-zinc-200 dark:bg-zinc-800 px-1 rounded">[+stderr]</span>
          {/if}
          {#if g.excludeFromReport}
            <span class="text-[10px] text-zinc-500 bg-zinc-200 dark:bg-zinc-800 px-1 rounded flex items-center gap-0.5">
              <EyeOff class="h-2.5 w-2.5" />
              Excluded
            </span>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
