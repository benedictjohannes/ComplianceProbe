<script lang="ts">
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Lock from 'lucide-svelte/icons/lock';
  import Terminal from 'lucide-svelte/icons/terminal';
  import Eye from 'lucide-svelte/icons/eye';
  import EyeOff from 'lucide-svelte/icons/eye-off';
  import ShieldAlert from 'lucide-svelte/icons/shield-alert';
  import { Badge, Button } from '$lib/components/ui';
  import { cn } from '$lib/utils/cn';
  import type { Assertion, Cmd, Exec } from '$lib/api/types';

  interface Props {
    assertion: Assertion;
    defaultExpanded?: boolean;
    class?: string;
  }

  let {
    assertion,
    defaultExpanded = false,
    class: className = '',
  }: Props = $props();

  let userExpanded = $state<boolean | null>(null);
  let isExpanded = $derived(userExpanded !== null ? userExpanded : defaultExpanded);
  let showCommands = $state<boolean>(false);

  function toggleExpanded() {
    userExpanded = !isExpanded;
  }

  const totalCmdsCount = $derived.by(() => {
    const pre = assertion.preCmds?.length || 0;
    const main = assertion.cmds?.length || 0;
    const post = assertion.postCmds?.length || 0;
    return pre + main + post;
  });

  const requiresElevation = $derived.by(() => {
    if (assertion.preCmds?.some((e) => e.requireElevation)) return true;
    if (assertion.cmds?.some((c) => c.exec?.requireElevation)) return true;
    if (assertion.postCmds?.some((e) => e.requireElevation)) return true;
    return false;
  });
</script>

<div
  class={cn(
    'rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-3 transition-colors hover:border-zinc-300 dark:hover:border-zinc-700 shadow-xs',
    className
  )}
>
  <!-- Level 2: Collapsed / Header Row -->
  <div class="flex items-start justify-between gap-3">
    <button
      type="button"
      onclick={toggleExpanded}
      class="flex flex-1 items-start gap-2.5 text-left cursor-pointer group focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-500 rounded"
    >
      <span class="mt-0.5 text-zinc-400 dark:text-zinc-500 group-hover:text-zinc-700 dark:group-hover:text-zinc-300 transition-colors">
        {#if isExpanded}
          <ChevronDown class="h-4 w-4 shrink-0" />
        {:else}
          <ChevronRight class="h-4 w-4 shrink-0" />
        {/if}
      </span>

      <div class="space-y-1 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-mono text-xs text-sky-700 dark:text-sky-400 bg-sky-500/10 border border-sky-500/20 px-2 py-0.5 rounded font-semibold select-all">
            {assertion.code}
          </span>
          <span class="text-sm font-medium text-zinc-800 dark:text-zinc-200 group-hover:text-zinc-950 dark:group-hover:text-white transition-colors">
            {assertion.title}
          </span>
        </div>
      </div>
    </button>

    <div class="flex items-center gap-2 shrink-0">
      {#if requiresElevation}
        <Badge variant="warning" size="sm" class="font-mono text-[10px]">
          <Lock class="h-3 w-3 mr-0.5" />
          sudo
        </Badge>
      {/if}

      <span class="rounded-full bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700/60 px-2 py-0.5 text-xs font-mono text-zinc-600 dark:text-zinc-400">
        {totalCmdsCount} {totalCmdsCount === 1 ? 'cmd' : 'cmds'}
      </span>
    </div>
  </div>

  <!-- Expanded Details Body -->
  {#if isExpanded}
    <div class="mt-3 pt-3 border-t border-zinc-200 dark:border-zinc-800/80 space-y-3 pl-6 text-sm text-zinc-700 dark:text-zinc-300">
      {#if assertion.description}
        <p class="text-zinc-700 dark:text-zinc-300 text-xs leading-relaxed">
          <strong class="text-zinc-900 dark:text-zinc-400 font-semibold">Description:</strong> {assertion.description}
        </p>
      {/if}

      <!-- Pass / Fail Descriptions -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
        {#if assertion.passDescription}
          <div class="rounded bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-500/20 p-2 text-emerald-800 dark:text-emerald-300">
            <span class="font-semibold text-emerald-700 dark:text-emerald-400 block mb-0.5">✓ Pass Criteria:</span>
            {assertion.passDescription}
          </div>
        {/if}

        {#if assertion.failDescription}
          <div class="rounded bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-500/20 p-2 text-rose-800 dark:text-rose-300">
            <span class="font-semibold text-rose-700 dark:text-rose-400 block mb-0.5">✕ Fail Criteria:</span>
            {assertion.failDescription}
          </div>
        {/if}
      </div>

      {#if assertion.minPassingScore !== undefined}
        <div class="text-xs font-mono text-zinc-600 dark:text-zinc-400">
          Min Passing Score: <span class="text-zinc-900 dark:text-zinc-200 font-semibold">≥ {assertion.minPassingScore} points</span>
        </div>
      {/if}

      <!-- Toggle Level 3 Commands -->
      {#if totalCmdsCount > 0}
        <div class="pt-1">
          <Button
            variant="ghost"
            size="xs"
            onclick={() => (showCommands = !showCommands)}
            class="text-xs text-sky-400 hover:text-sky-300 gap-1.5 font-mono p-0 h-auto"
          >
            {#if showCommands}
              <ChevronDown class="h-3.5 w-3.5" />
              Hide Commands ({totalCmdsCount})
            {:else}
              <ChevronRight class="h-3.5 w-3.5" />
              View Commands ({totalCmdsCount})
            {/if}
          </Button>
        </div>
      {/if}

      <!-- Level 3: All Command Blocks -->
      {#if showCommands}
        <div class="space-y-3 pt-2">
          <!-- Pre-Commands -->
          {#if assertion.preCmds && assertion.preCmds.length > 0}
            {#each assertion.preCmds as exec, i (i)}
              <div class="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/70 p-2.5 space-y-2 text-xs font-mono">
                <div class="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-200 dark:border-zinc-800/80 pb-1.5 text-zinc-600 dark:text-zinc-400">
                  <span class="font-semibold text-sky-700 dark:text-sky-400">
                    [Pre-Command {i + 1}/{assertion.preCmds.length}]
                  </span>
                  <div class="flex items-center gap-1.5 text-[11px]">
                    {#if exec.shell}
                      <span class="bg-zinc-200 dark:bg-zinc-800 px-1.5 py-0.5 rounded text-zinc-800 dark:text-zinc-300">Shell: {exec.shell}</span>
                    {/if}
                    {#if exec.requireElevation}
                      <Badge variant="warning" size="sm" class="text-[10px]">sudo</Badge>
                    {/if}
                    {#if exec.excludeFromReport}
                      <Badge variant="outline" size="sm" class="text-[10px] text-zinc-500">Excluded from Report</Badge>
                    {/if}
                  </div>
                </div>

                {#if exec.script}
                  <div class="rounded bg-zinc-900 border border-zinc-800 p-2 text-zinc-100 overflow-x-auto select-text whitespace-pre-wrap">
                    {exec.script}
                  </div>
                {/if}

                {#if exec.gather && exec.gather.length > 0}
                  <div class="space-y-1 text-zinc-600 dark:text-zinc-400 text-[11px] pt-1">
                    {#each exec.gather as g}
                      <div>
                        Gather: var <span class="text-sky-700 dark:text-sky-300 font-semibold">"{g.key}"</span>
                        {#if g.regex}
                          &lt;= regex <code class="text-amber-700 dark:text-amber-300">/{g.regex}/</code>
                        {/if}
                        {#if g.excludeFromReport}
                          <span class="text-zinc-500 ml-1">[🔒 Excluded]</span>
                        {/if}
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}
          {/if}

          <!-- Main Commands -->
          {#if assertion.cmds && assertion.cmds.length > 0}
            {#each assertion.cmds as cmd, i (i)}
              <div class="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/70 p-2.5 space-y-2 text-xs font-mono">
                <div class="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-200 dark:border-zinc-800/80 pb-1.5 text-zinc-600 dark:text-zinc-400">
                  <span class="font-semibold text-emerald-700 dark:text-emerald-400">
                    [Command {i + 1}/{assertion.cmds.length}]
                  </span>
                  <div class="flex items-center gap-1.5 text-[11px]">
                    {#if cmd.passScore !== undefined || cmd.failScore !== undefined}
                      <span class="bg-zinc-200 dark:bg-zinc-800 px-1.5 py-0.5 rounded text-zinc-800 dark:text-zinc-300">
                        Score: +{cmd.passScore ?? 1} / -{cmd.failScore ?? 1}
                      </span>
                    {/if}
                    {#if cmd.exec?.shell}
                      <span class="bg-zinc-200 dark:bg-zinc-800 px-1.5 py-0.5 rounded text-zinc-800 dark:text-zinc-300">Shell: {cmd.exec.shell}</span>
                    {/if}
                    {#if cmd.exec?.requireElevation}
                      <Badge variant="warning" size="sm" class="text-[10px]">sudo</Badge>
                    {/if}
                    {#if cmd.exec?.excludeFromReport}
                      <Badge variant="outline" size="sm" class="text-[10px] text-zinc-500">Excluded</Badge>
                    {/if}
                  </div>
                </div>

                {#if cmd.exec?.script}
                  <div class="rounded bg-zinc-900 border border-zinc-800 p-2 text-zinc-100 overflow-x-auto select-text whitespace-pre-wrap">
                    {cmd.exec.script}
                  </div>
                {/if}

                <!-- Evaluation Rules -->
                <div class="space-y-1 text-zinc-600 dark:text-zinc-400 text-[11px] pt-1">
                  {#if cmd.stdOutRule}
                    <div class="flex items-center gap-1">
                      <span class="text-zinc-500">StdOut Rule:</span>
                      {#if cmd.stdOutRule.regex}
                        <span class="text-amber-700 dark:text-amber-300">regex /{cmd.stdOutRule.regex}/</span>
                      {:else if cmd.stdOutRule.func}
                        <span class="text-sky-700 dark:text-sky-300">func()</span>
                      {/if}
                    </div>
                  {/if}

                  {#if cmd.stdErrRule}
                    <div class="flex items-center gap-1">
                      <span class="text-zinc-500">StdErr Rule:</span>
                      {#if cmd.stdErrRule.regex}
                        <span class="text-rose-700 dark:text-rose-300">regex /{cmd.stdErrRule.regex}/</span>
                      {/if}
                    </div>
                  {/if}

                  {#if cmd.exitCodeRules && cmd.exitCodeRules.length > 0}
                    <div class="flex items-center gap-1">
                      <span class="text-zinc-500">Exit Code Rules:</span>
                      <span class="text-zinc-800 dark:text-zinc-300">
                        {cmd.exitCodeRules.map((r) => `[${r.min ?? 0}-${r.max ?? 0}: result ${r.result}]`).join(', ')}
                      </span>
                    </div>
                  {/if}

                  {#if cmd.exec?.gather && cmd.exec.gather.length > 0}
                    {#each cmd.exec.gather as g}
                      <div>
                        Gather: var <span class="text-sky-700 dark:text-sky-300 font-semibold">"{g.key}"</span>
                        {#if g.regex}
                          &lt;= regex <code class="text-amber-700 dark:text-amber-300">/{g.regex}/</code>
                        {/if}
                      </div>
                    {/each}
                  {/if}
                </div>
              </div>
            {/each}
          {/if}

          <!-- Post-Commands -->
          {#if assertion.postCmds && assertion.postCmds.length > 0}
            {#each assertion.postCmds as exec, i (i)}
              <div class="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/70 p-2.5 space-y-2 text-xs font-mono">
                <div class="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-200 dark:border-zinc-800/80 pb-1.5 text-zinc-600 dark:text-zinc-400">
                  <span class="font-semibold text-purple-700 dark:text-purple-400">
                    [Post-Command {i + 1}/{assertion.postCmds.length}]
                  </span>
                  <div class="flex items-center gap-1.5 text-[11px]">
                    {#if exec.shell}
                      <span class="bg-zinc-200 dark:bg-zinc-800 px-1.5 py-0.5 rounded text-zinc-800 dark:text-zinc-300">Shell: {exec.shell}</span>
                    {/if}
                    {#if exec.requireElevation}
                      <Badge variant="warning" size="sm" class="text-[10px]">sudo</Badge>
                    {/if}
                  </div>
                </div>

                {#if exec.script}
                  <div class="rounded bg-zinc-900 border border-zinc-800 p-2 text-zinc-100 overflow-x-auto select-text whitespace-pre-wrap">
                    {exec.script}
                  </div>
                {/if}
              </div>
            {/each}
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</div>
