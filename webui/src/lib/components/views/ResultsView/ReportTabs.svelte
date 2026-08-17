<script lang="ts">
  import { onMount } from 'svelte';
  import { appState } from '$lib/state/appState.svelte';
  import { apiClient } from '$lib/api/client';
  import type { Assertion, AssertionSnapshot, Section } from '$lib/api/types';
  import MarkdownViewer from './MarkdownViewer.svelte';
  import ListChecks from 'lucide-svelte/icons/list-checks';
  import FileText from 'lucide-svelte/icons/file-text';
  import Terminal from 'lucide-svelte/icons/terminal';
  import Search from 'lucide-svelte/icons/search';
  import ChevronsUpDown from 'lucide-svelte/icons/chevrons-up-down';
  import Check from 'lucide-svelte/icons/check';
  import X from 'lucide-svelte/icons/x';
  import Ban from 'lucide-svelte/icons/ban';
  import Shield from 'lucide-svelte/icons/shield';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';
  import CodeBlock from '$lib/components/ui/CodeBlock.svelte';
  import { cn } from '$lib/utils/cn';

  interface Props {
    onFullscreenMarkdown?: (content: string) => void;
    onFullscreenLogs?: (content: string) => void;
    class?: string;
  }

  let {
    onFullscreenMarkdown,
    onFullscreenLogs,
    class: className = '',
  }: Props = $props();

  type Tab = 'audit' | 'markdown' | 'logs';
  let activeTab = $state<Tab>('audit');

  // Filter toolbar state for Audit
  type OutcomeFilter = 'all' | 'failed' | 'passed';
  let outcomeFilter = $state<OutcomeFilter>('all');
  let searchQuery = $state('');
  let expandedAssertionCodes = $state<Record<string, boolean>>({});

  // Markdown & Logs content
  let markdownContent = $state('');
  let rawLogsContent = $state('');
  let loadingMarkdown = $state(false);
  let loadingLogs = $state(false);

  async function loadMarkdownReport() {
    if (markdownContent) return;
    loadingMarkdown = true;
    try {
      markdownContent = await apiClient.getReportMarkdown();
    } catch {
      markdownContent = '# Compliance Report\n\nNo report generated or report is unavailable.';
    } finally {
      loadingMarkdown = false;
    }
  }

  async function loadLogsReport() {
    if (rawLogsContent) return;
    loadingLogs = true;
    try {
      rawLogsContent = await apiClient.getReportLog();
    } catch {
      if (appState.logs.length > 0) {
        rawLogsContent = appState.logs.join('\n');
      } else {
        rawLogsContent = 'No logs available.';
      }
    } finally {
      loadingLogs = false;
    }
  }

  function handleTabChange(tab: Tab) {
    activeTab = tab;
    if (tab === 'markdown') {
      loadMarkdownReport();
    } else if (tab === 'logs') {
      loadLogsReport();
    }
  }

  function handleToggleAssertion(code: string, currentExpanded: boolean) {
    expandedAssertionCodes[code] = !currentExpanded;
  }

  function handleExpandAll() {
    const updated: Record<string, boolean> = {};
    if (appState.playbook?.sections) {
      appState.playbook.sections.forEach((sec) => {
        sec.assertions?.forEach((a) => {
          updated[a.code] = true;
        });
      });
    } else if (appState.execution?.assertions) {
      appState.execution.assertions.forEach((a) => {
        updated[a.code] = true;
      });
    }
    expandedAssertionCodes = updated;
  }

  function handleCollapseAll() {
    const updated: Record<string, boolean> = {};
    if (appState.playbook?.sections) {
      appState.playbook.sections.forEach((sec) => {
        sec.assertions?.forEach((a) => {
          updated[a.code] = false;
        });
      });
    } else if (appState.execution?.assertions) {
      appState.execution.assertions.forEach((a) => {
        updated[a.code] = false;
      });
    }
    expandedAssertionCodes = updated;
  }

  // Find snapshot for an assertion
  function getSnapshot(code: string): AssertionSnapshot | undefined {
    return appState.execution?.assertions.find((a) => a.code === code);
  }

  // Filter sections & assertions
  const filteredSections = $derived.by(() => {
    const sections: Section[] = appState.playbook?.sections || [
      {
        title: 'Audit Assertions',
        description: [],
        assertions: (appState.execution?.assertions || []).map((snap) => ({
          code: snap.code,
          title: snap.title,
          description: '',
          cmds: [],
          passDescription: '',
          failDescription: '',
        })),
      },
    ];

    return sections
      .map((sec) => {
        const filteredAssertions = (sec.assertions || []).filter((a) => {
          const snap = getSnapshot(a.code);
          const isPassed = snap?.status === 'passed';
          const isFailed = snap?.status === 'failed';

          // Outcome filter
          if (outcomeFilter === 'passed' && !isPassed) return false;
          if (outcomeFilter === 'failed' && !isFailed) return false;

          // Search query
          if (searchQuery.trim()) {
            const q = searchQuery.toLowerCase();
            const matchesCode = a.code.toLowerCase().includes(q);
            const matchesTitle = a.title.toLowerCase().includes(q);
            const matchesDesc = (a.description || '').toLowerCase().includes(q);
            if (!matchesCode && !matchesTitle && !matchesDesc) return false;
          }

          return true;
        });

        return {
          ...sec,
          assertions: filteredAssertions,
        };
      })
      .filter((sec) => sec.assertions.length > 0);
  });
</script>

<div class={cn('space-y-4', className)}>
  <!-- Tab Navigation Bar -->
  <div class="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800">
    <div class="flex items-center gap-1 sm:gap-2">
      <!-- Tab 1: Execution Audit -->
      <button
        type="button"
        onclick={() => handleTabChange('audit')}
        class={cn(
          'flex items-center gap-2 px-3.5 py-2.5 text-xs font-semibold border-b-2 transition cursor-pointer select-none',
          activeTab === 'audit'
            ? 'border-sky-500 text-sky-600 dark:text-sky-400 bg-sky-500/5'
            : 'border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'
        )}
      >
        <ListChecks class="h-4 w-4" />
        <span>Execution Audit</span>
        <span class="px-1.5 py-0.2 rounded-full text-[10px] bg-zinc-200 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-mono">
          {appState.totalAssertions}
        </span>
      </button>

      <!-- Tab 2: Markdown Report -->
      <button
        type="button"
        onclick={() => handleTabChange('markdown')}
        class={cn(
          'flex items-center gap-2 px-3.5 py-2.5 text-xs font-semibold border-b-2 transition cursor-pointer select-none',
          activeTab === 'markdown'
            ? 'border-sky-500 text-sky-600 dark:text-sky-400 bg-sky-500/5'
            : 'border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'
        )}
      >
        <FileText class="h-4 w-4" />
        <span>Markdown Report</span>
      </button>

      <!-- Tab 3: Raw Logs -->
      <button
        type="button"
        onclick={() => handleTabChange('logs')}
        class={cn(
          'flex items-center gap-2 px-3.5 py-2.5 text-xs font-semibold border-b-2 transition cursor-pointer select-none',
          activeTab === 'logs'
            ? 'border-sky-500 text-sky-600 dark:text-sky-400 bg-sky-500/5'
            : 'border-transparent text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'
        )}
      >
        <Terminal class="h-4 w-4" />
        <span>Execution Logs</span>
      </button>
    </div>
  </div>

  <!-- TAB CONTENT 1: EXECUTION AUDIT BREAKDOWN -->
  {#if activeTab === 'audit'}
    <div class="space-y-4 animate-in fade-in-50">
      <!-- Filter Toolbar -->
      <div class="flex flex-wrap items-center justify-between gap-3 p-3 rounded-xl bg-zinc-50 dark:bg-zinc-900/60 border border-zinc-200 dark:border-zinc-800">
        <!-- Left: Status Filter Pills -->
        <div class="flex items-center gap-1.5 bg-zinc-200/70 dark:bg-zinc-950 p-1 rounded-lg text-xs font-medium">
          <button
            type="button"
            onclick={() => (outcomeFilter = 'all')}
            class={cn(
              'px-2.5 py-1 rounded-md transition cursor-pointer',
              outcomeFilter === 'all'
                ? 'bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 shadow-xs font-semibold'
                : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'
            )}
          >
            All ({appState.totalAssertions})
          </button>
          <button
            type="button"
            onclick={() => (outcomeFilter = 'failed')}
            class={cn(
              'px-2.5 py-1 rounded-md transition cursor-pointer flex items-center gap-1',
              outcomeFilter === 'failed'
                ? 'bg-rose-500/15 text-rose-700 dark:text-rose-300 font-semibold shadow-xs'
                : 'text-zinc-600 dark:text-zinc-400 hover:text-rose-600 dark:hover:text-rose-400'
            )}
          >
            <X class="h-3 w-3 stroke-[3]" />
            Failed ({appState.failedAssertions})
          </button>
          <button
            type="button"
            onclick={() => (outcomeFilter = 'passed')}
            class={cn(
              'px-2.5 py-1 rounded-md transition cursor-pointer flex items-center gap-1',
              outcomeFilter === 'passed'
                ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 font-semibold shadow-xs'
                : 'text-zinc-600 dark:text-zinc-400 hover:text-emerald-600 dark:hover:text-emerald-400'
            )}
          >
            <Check class="h-3 w-3 stroke-[3]" />
            Passed ({appState.passedAssertions})
          </button>
        </div>

        <!-- Right: Search Input & Bulk Expand/Collapse -->
        <div class="flex items-center gap-2">
          <!-- Search Input -->
          <div class="relative flex items-center">
            <Search class="absolute left-2.5 h-3.5 w-3.5 text-zinc-400 pointer-events-none" />
            <input
              type="text"
              bind:value={searchQuery}
              placeholder="Filter rules (e.g. SEC-001)..."
              class="h-8 w-44 sm:w-56 pl-8 pr-3 rounded-lg bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-xs text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 focus:outline-none focus:border-sky-500"
            />
          </div>

          <!-- Bulk Toggles -->
          <button
            type="button"
            onclick={handleExpandAll}
            class="h-8 px-2.5 rounded-lg bg-white dark:bg-zinc-950 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 text-xs font-medium border border-zinc-200 dark:border-zinc-800 transition cursor-pointer"
            title="Expand all assertion details"
          >
            Expand All
          </button>
          <button
            type="button"
            onclick={handleCollapseAll}
            class="h-8 px-2.5 rounded-lg bg-white dark:bg-zinc-950 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 text-xs font-medium border border-zinc-200 dark:border-zinc-800 transition cursor-pointer"
            title="Collapse all assertion details"
          >
            Collapse All
          </button>
        </div>
      </div>

      <!-- Assertion Trees Grouped by Section -->
      {#if filteredSections.length === 0}
        <div class="p-8 text-center rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800 text-zinc-500 dark:text-zinc-400 text-xs">
          No assertions matching active filters.
        </div>
      {:else}
        <div class="space-y-6">
          {#each filteredSections as section}
            {@const sectionPassedCount = section.assertions.filter((a) => getSnapshot(a.code)?.status === 'passed').length}
            {@const sectionFailedCount = section.assertions.filter((a) => getSnapshot(a.code)?.status === 'failed').length}

            <div class="space-y-3">
              <!-- Section Header -->
              <div class="flex items-center justify-between pb-1.5 border-b border-zinc-200 dark:border-zinc-800">
                <div class="space-y-0.5">
                  <h3 class="text-sm font-bold text-zinc-900 dark:text-zinc-100">
                    {section.title}
                  </h3>
                  {#if section.description && section.description.length > 0}
                    <p class="text-[11px] text-zinc-500 dark:text-zinc-400">
                      {section.description.join(' ')}
                    </p>
                  {/if}
                </div>

                <!-- Section Pass Ratio Pill -->
                <div class="flex items-center gap-1 text-[11px] font-mono shrink-0">
                  <span class="px-2 py-0.5 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-300 font-semibold border border-zinc-200 dark:border-zinc-700">
                    {sectionPassedCount}/{section.assertions.length} passed
                  </span>
                </div>
              </div>

              <!-- Assertion Cards -->
              <div class="space-y-2.5">
                {#each section.assertions as assertion (assertion.code)}
                  {@const snap = getSnapshot(assertion.code)}
                  {@const isExpanded = expandedAssertionCodes[assertion.code] ?? (snap?.status === 'failed')}
                  {@const isPassed = snap?.status === 'passed'}
                  {@const isFailed = snap?.status === 'failed'}
                  {@const isCancelled = snap?.status === 'cancelled'}
                  {@const requiresElevation = Boolean(
                    assertion.cmds?.some((c) => c.exec?.requireElevation) ||
                    assertion.preCmds?.some((e) => e?.requireElevation) ||
                    assertion.postCmds?.some((e) => e?.requireElevation)
                  )}

                  <div
                    class={cn(
                      'rounded-xl border transition-all duration-200 overflow-hidden',
                      isPassed
                        ? 'border-emerald-500/30 bg-emerald-500/5'
                        : isFailed
                        ? 'border-rose-500/40 bg-rose-500/5'
                        : isCancelled
                        ? 'border-amber-500/30 bg-amber-500/5'
                        : 'border-zinc-200 dark:border-zinc-800 bg-white/70 dark:bg-zinc-900/60'
                    )}
                  >
                    <!-- Header -->
                    <button
                      type="button"
                      onclick={() => handleToggleAssertion(assertion.code, isExpanded)}
                      class="w-full text-left p-3.5 flex items-center justify-between gap-3 hover:bg-zinc-500/5 transition-colors cursor-pointer select-none"
                    >
                      <div class="flex items-center gap-2.5 min-w-0 flex-1">
                        <!-- Icon -->
                        <div class="shrink-0">
                          {#if isPassed}
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
                            <span class="h-4 w-4 rounded-full border border-zinc-400 inline-block"></span>
                          {/if}
                        </div>

                        <!-- Code Pill -->
                        <span class={cn(
                          'font-mono text-xs font-semibold px-2 py-0.5 rounded shrink-0',
                          isPassed ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300' :
                          isFailed ? 'bg-rose-500/15 text-rose-700 dark:text-rose-300' :
                          'bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400'
                        )}>
                          {assertion.code}
                        </span>

                        <!-- Title -->
                        <span class="text-sm font-medium truncate text-zinc-900 dark:text-zinc-100">
                          {assertion.title}
                        </span>
                      </div>

                      <!-- Right Badges -->
                      <div class="flex items-center gap-2 shrink-0 font-mono text-xs">
                        {#if requiresElevation}
                          <span class="hidden sm:inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-sans font-medium bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
                            <Shield class="h-2.5 w-2.5" />
                            <span>sudo</span>
                          </span>
                        {/if}

                        <!-- Assertion Point Score Pill (strictly individual point score) -->
                        {#if isPassed && snap}
                          <span class="px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30 text-[11px] font-semibold">
                            +{snap.score} {snap.score === 1 ? 'pt' : 'pts'}
                          </span>
                        {:else if isFailed && snap}
                          <span class="px-2 py-0.5 rounded-full bg-rose-500/15 text-rose-600 dark:text-rose-400 border border-rose-500/30 text-[11px] font-semibold">
                            {snap.score}/{snap.min_score} pts
                          </span>
                        {/if}

                        <!-- Duration -->
                        {#if snap?.duration_ms && snap.duration_ms > 0}
                          <span class="text-zinc-400 dark:text-zinc-500 text-[11px]">
                            {snap.duration_ms}ms
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

                    <!-- Expanded Body -->
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
                              <span class="font-bold">Compliance Criteria Not Met</span>
                              <p class="text-[11px] text-rose-700 dark:text-rose-400">
                                This assertion did not meet the required threshold. Review execution stdout/stderr below.
                              </p>
                            </div>
                          </div>
                        {/if}

                        <!-- Output CodeBlock -->
                        {#if snap?.output}
                          <div class="space-y-1">
                            <div class="text-[11px] font-semibold text-zinc-700 dark:text-zinc-300">
                              Probe Output & Diagnostic Logs:
                            </div>
                            <CodeBlock
                              code={snap.output}
                              variant="terminal"
                              copyable={true}
                            />
                          </div>
                        {/if}
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

  <!-- TAB CONTENT 2: MARKDOWN REPORT -->
  {:else if activeTab === 'markdown'}
    <div class="space-y-3 animate-in fade-in-50">
      {#if loadingMarkdown}
        <div class="p-12 text-center text-zinc-500 text-xs font-mono">
          Loading markdown report...
        </div>
      {:else}
        <MarkdownViewer
          content={markdownContent}
          filename="report.md"
          downloadUrl="/api/report/md?download=1"
          showFullscreenButton={true}
          onFullscreen={() => onFullscreenMarkdown?.(markdownContent)}
        />
      {/if}
    </div>

  <!-- TAB CONTENT 3: RAW EXECUTION LOGS -->
  {:else if activeTab === 'logs'}
    <div class="space-y-3 animate-in fade-in-50">
      {#if loadingLogs}
        <div class="p-12 text-center text-zinc-500 text-xs font-mono">
          Loading execution logs...
        </div>
      {:else}
        <MarkdownViewer
          content={rawLogsContent || appState.logs.join('\n')}
          filename="report.log"
          downloadUrl="/api/report/log?download=1"
          showFullscreenButton={true}
          onFullscreen={() => onFullscreenLogs?.(rawLogsContent || appState.logs.join('\n'))}
        />
      {/if}
    </div>
  {/if}
</div>
