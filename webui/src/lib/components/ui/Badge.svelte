<script lang="ts">
  import type { Snippet } from 'svelte';
  import { cn } from '$lib/utils/cn';

  type BadgeVariant =
    | 'default'
    | 'outline'
    | 'passed'
    | 'failed'
    | 'warning'
    | 'elevating'
    | 'running'
    | 'info'
    | 'neutral'
    | 'code';

  type BadgeSize = 'sm' | 'md';

  interface Props {
    variant?: BadgeVariant;
    size?: BadgeSize;
    class?: string;
    children?: Snippet;
  }

  let {
    variant = 'default',
    size = 'md',
    class: className = '',
    children,
  }: Props = $props();

  const variantStyles: Record<BadgeVariant, string> = {
    default: 'bg-zinc-100 text-zinc-800 border-zinc-300 dark:bg-zinc-800 dark:text-zinc-200 dark:border-zinc-700',
    outline: 'border border-zinc-300 text-zinc-600 dark:border-zinc-700 dark:text-zinc-400 bg-transparent',
    passed: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-500/20 dark:border-emerald-500/30',
    failed: 'bg-rose-500/10 text-rose-700 dark:text-rose-400 border-rose-500/20 dark:border-rose-500/30',
    warning: 'bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-500/30 dark:border-amber-500/30',
    elevating: 'bg-amber-500/20 text-amber-800 dark:text-amber-300 border-amber-500/40 dark:border-amber-500/40 font-semibold',
    running: 'bg-sky-500/10 text-sky-700 dark:text-sky-400 border-sky-500/20 dark:border-sky-500/30',
    info: 'bg-sky-500/10 text-sky-700 dark:text-sky-400 border-sky-500/20 dark:border-sky-500/30',
    neutral: 'bg-zinc-100 text-zinc-600 border-zinc-300 dark:bg-zinc-900/80 dark:text-zinc-400 dark:border-zinc-800',
    code: 'font-mono bg-sky-500/10 text-sky-700 dark:text-sky-400 border-sky-500/20 dark:border-sky-500/30 font-semibold',
  };

  const sizeStyles: Record<BadgeSize, string> = {
    sm: 'text-[10px] px-1.5 py-0.2 rounded font-medium',
    md: 'text-xs px-2 py-0.5 rounded font-medium',
  };

  const baseStyles = 'inline-flex items-center gap-1 border select-none transition-colors';
</script>

<span class={cn(baseStyles, variantStyles[variant], sizeStyles[size], className)}>
  {#if children}
    {@render children()}
  {/if}
</span>
