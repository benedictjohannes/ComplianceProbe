<script lang="ts">
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import AssertionItem from './AssertionItem.svelte';
  import { cn } from '$lib/utils/cn';
  import type { Section } from '$lib/api/types';

  interface Props {
    sections: Section[];
    class?: string;
  }

  let { sections, class: className = '' }: Props = $props();

  // Track collapsed sections (sections are expanded by default)
  let collapsedSections = $state<Record<number, boolean>>({});

  function toggleSection(index: number) {
    collapsedSections = {
      ...collapsedSections,
      [index]: !collapsedSections[index],
    };
  }
</script>

<div class="space-y-6 {className}">
  {#if sections && sections.length > 0}
    {#each sections as section, sIndex (sIndex)}
      {@const isExpanded = !collapsedSections[sIndex]}
      <div class="space-y-3">
        <!-- Level 1: Section Header (Unboxed, clean) -->
        <div class="border-b border-zinc-200 dark:border-zinc-800 pb-2">
          <button
            type="button"
            onclick={() => toggleSection(sIndex)}
            class="flex w-full items-start justify-between text-left cursor-pointer group focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-500 rounded p-1"
          >
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <span class="text-zinc-400 dark:text-zinc-500 group-hover:text-zinc-700 dark:group-hover:text-zinc-300 transition-colors">
                  {#if isExpanded}
                    <ChevronDown class="h-4 w-4 shrink-0" />
                  {:else}
                    <ChevronRight class="h-4 w-4 shrink-0" />
                  {/if}
                </span>

                <h3 class="font-semibold text-base sm:text-lg text-zinc-900 dark:text-zinc-100 group-hover:text-zinc-950 dark:group-hover:text-white transition-colors">
                  {sIndex + 1}. {section.title}
                </h3>

                <span class="rounded-full bg-zinc-100 dark:bg-zinc-800/90 border border-zinc-200 dark:border-zinc-700/60 px-2 py-0.5 text-xs font-mono text-zinc-600 dark:text-zinc-400">
                  {section.assertions?.length || 0} {(section.assertions?.length || 0) === 1 ? 'assertion' : 'assertions'}
                </span>
              </div>

              {#if section.description && section.description.length > 0}
                <div class="text-xs sm:text-sm text-zinc-600 dark:text-zinc-400 pl-6 leading-relaxed">
                  {#each section.description as desc}
                    <p>{desc}</p>
                  {/each}
                </div>
              {/if}
            </div>
          </button>
        </div>

        <!-- Level 2 Assertions List -->
        {#if isExpanded}
          <div class="space-y-2.5 pl-2 sm:pl-4">
            {#if section.assertions && section.assertions.length > 0}
              {#each section.assertions as assertion (assertion.code)}
                <AssertionItem {assertion} />
              {/each}
            {:else}
              <div class="rounded-lg border border-dashed border-zinc-800 p-4 text-center text-xs text-zinc-500 font-mono">
                No assertions defined in this section.
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
  {:else}
    <div class="rounded-lg border border-dashed border-zinc-800 p-8 text-center text-sm text-zinc-500">
      No sections defined in this playbook.
    </div>
  {/if}
</div>
