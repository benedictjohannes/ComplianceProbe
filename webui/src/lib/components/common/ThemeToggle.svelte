<script lang="ts">
  import Sun from 'lucide-svelte/icons/sun';
  import Moon from 'lucide-svelte/icons/moon';
  import Monitor from 'lucide-svelte/icons/monitor';
  import { themeStore, type ThemeMode } from '$lib/state/theme.svelte';
  import { Dropdown } from '$lib/components/ui';
  import { cn } from '$lib/utils/cn';

  interface Props {
    class?: string;
  }

  let { class: className = '' }: Props = $props();

  const themeItems = [
    { id: 'light', label: 'Light', icon: Sun, onclick: () => themeStore.setTheme('light') },
    { id: 'dark', label: 'Dark', icon: Moon, onclick: () => themeStore.setTheme('dark') },
    { id: 'system', label: 'System', icon: Monitor, onclick: () => themeStore.setTheme('system') },
  ];
</script>

<Dropdown items={themeItems} side="bottom" align="end">
  {#snippet trigger()}
    <button
      type="button"
      aria-label="Toggle theme"
      class={cn(
        'p-1.5 rounded-md text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer select-none',
        'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-500',
        className
      )}
    >
      {#if themeStore.mode === 'system'}
        {#if themeStore.resolved === 'dark'}
          <Moon class="h-4 w-4" />
        {:else}
          <Sun class="h-4 w-4" />
        {/if}
      {:else if themeStore.mode === 'dark'}
        <Moon class="h-4 w-4" />
      {:else}
        <Sun class="h-4 w-4" />
      {/if}
    </button>
  {/snippet}
</Dropdown>
