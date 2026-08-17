<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { appState } from '$lib/state/appState.svelte';
  import type { Section, Assertion, AssertionSnapshot } from '$lib/api/types';
  import AssertionCard from './AssertionCard.svelte';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import ArrowDown from 'lucide-svelte/icons/arrow-down';
  import { cn } from '$lib/utils/cn';

  interface Props {
    class?: string;
  }

  let { class: className = '' }: Props = $props();

  // Track collapsed section indices
  let collapsedSections = $state<Record<number, boolean>>({});
  let isScrolledAway = $state(false);

  function toggleSection(index: number) {
    collapsedSections[index] = !collapsedSections[index];
  }

  // Active running assertion code
  const activeAssertionCode = $derived(appState.execution?.active_assertion_code);

  // Auto-scroll to active assertion when it changes
  $effect(() => {
    const activeCode = activeAssertionCode;
    if (activeCode && !isScrolledAway) {
      tick().then(() => {
        const el = document.getElementById(`assertion-${activeCode}`);
        if (el && typeof el.scrollIntoView === 'function') {
          el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }
      });
    }
  });

  // Calculate section passing statistics
  function getSectionStats(section: Section) {
    const total = section.assertions.length;
    const passed = section.assertions.filter((a) => {
      const snap = appState.execution?.assertions.find((s) => s.code === a.code);
      return snap?.status === 'passed';
    }).length;
    const failed = section.assertions.filter((a) => {
      const snap = appState.execution?.assertions.find((s) => s.code === a.code);
      return snap?.status === 'failed';
    }).length;
    return { total, passed, failed };
  }

  function getAssertionSnapshot(code: string): AssertionSnapshot | undefined {
    return appState.execution?.assertions.find((s) => s.code === code);
  }

  function handleJumpToActive() {
    isScrolledAway = false;
    if (activeAssertionCode) {
      const el = document.getElementById(`assertion-${activeAssertionCode}`);
      if (el && typeof el.scrollIntoView === 'function') {
        el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    }
  }

  // Detect user scroll away
  onMount(() => {
    function onWindowScroll() {
      if (!activeAssertionCode) return;
      const el = document.getElementById(`assertion-${activeAssertionCode}`);
      if (!el) return;

      const rect = el.getBoundingClientRect();
      const isVisible = rect.top >= 0 && rect.bottom <= (window.innerHeight || document.documentElement.clientHeight);
      if (!isVisible && !isScrolledAway) {
        isScrolledAway = true;
      } else if (isVisible && isScrolledAway) {
        isScrolledAway = false;
      }
    }

    window.addEventListener('scroll', onWindowScroll, { passive: true });
    return () => {
      window.removeEventListener('scroll', onWindowScroll);
    };
  });
</script>

<div class={cn('space-y-6', className)}>
  <!-- If Playbook is loaded, render by sections -->
  {#if appState.playbook && appState.playbook.sections && appState.playbook.sections.length > 0}
    {#each appState.playbook.sections as section, sIndex (section.title || sIndex)}
      {@const stats = getSectionStats(section)}
      {@const isCollapsed = Boolean(collapsedSections[sIndex])}

      <section class="space-y-3">
        <!-- Section Header Bar -->
        <button
          type="button"
          onclick={() => toggleSection(sIndex)}
          class="w-full flex items-start justify-between gap-3 text-left group cursor-pointer select-none"
          aria-expanded={!isCollapsed}
        >
          <div class="space-y-0.5 min-w-0">
            <div class="flex items-center gap-2">
              <span class="text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200 transition-colors">
                {#if isCollapsed}
                  <ChevronRight class="h-4 w-4" />
                {:else}
                  <ChevronDown class="h-4 w-4" />
                {/if}
              </span>
              <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100 group-hover:text-sky-600 dark:group-hover:text-sky-400 transition-colors">
                {sIndex + 1}. {section.title}
              </h3>
              <span class="text-xs font-mono text-zinc-500">
                ({section.assertions.length} {section.assertions.length === 1 ? 'assertion' : 'assertions'})
              </span>
            </div>

            {#if section.description && section.description.length > 0 && !isCollapsed}
              <p class="text-xs text-zinc-500 dark:text-zinc-400 pl-6 leading-relaxed">
                {section.description.join(' ')}
              </p>
            {/if}
          </div>

          <!-- Section Completion Pill -->
          <div class="shrink-0 flex items-center gap-1.5 font-mono text-xs pt-0.5">
            {#if stats.passed > 0}
              <span class="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 text-[11px]">
                {stats.passed} passed
              </span>
            {/if}
            {#if stats.failed > 0}
              <span class="px-2 py-0.5 rounded-full bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 text-[11px]">
                {stats.failed} failed
              </span>
            {/if}
          </div>
        </button>

        <!-- Assertions List in Section -->
        {#if !isCollapsed}
          <div class="space-y-2.5 pl-2 sm:pl-4 border-l-2 border-zinc-200 dark:border-zinc-800">
            {#each section.assertions as assertion (assertion.code)}
              <AssertionCard
                {assertion}
                snapshot={getAssertionSnapshot(assertion.code)}
                isActive={activeAssertionCode === assertion.code}
              />
            {/each}
          </div>
        {/if}
      </section>
    {/each}

  <!-- Fallback: When Playbook object is missing, render snapshots directly -->
  {:else if appState.execution?.assertions && appState.execution.assertions.length > 0}
    <div class="space-y-2.5">
      {#each appState.execution.assertions as snap (snap.code)}
        <AssertionCard
          assertion={{ code: snap.code, title: snap.title || snap.code }}
          snapshot={snap}
          isActive={activeAssertionCode === snap.code}
        />
      {/each}
    </div>

  <!-- Empty state while execution begins -->
  {:else}
    <div class="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white/50 dark:bg-zinc-900/50 p-8 text-center text-zinc-500 dark:text-zinc-400 text-sm">
      Initializing execution engine and preparing assertions...
    </div>
  {/if}

  <!-- Floating Jump to Active Assertion Indicator -->
  {#if activeAssertionCode && isScrolledAway}
    <div class="fixed bottom-16 right-6 z-20 animate-bounce">
      <button
        type="button"
        onclick={handleJumpToActive}
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-sky-600 hover:bg-sky-500 text-white text-xs font-semibold shadow-lg border border-sky-400/30 transition-transform active:scale-95 cursor-pointer"
      >
        <ArrowDown class="h-3.5 w-3.5" />
        <span>Jump to active assertion ({activeAssertionCode})</span>
      </button>
    </div>
  {/if}
</div>
