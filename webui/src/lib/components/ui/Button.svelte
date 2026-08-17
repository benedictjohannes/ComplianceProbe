<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import { cn } from '$lib/utils/cn';

  type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'accent' | 'indigo';
  type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'icon' | 'icon-sm';

  interface Props extends HTMLButtonAttributes {
    variant?: ButtonVariant;
    size?: ButtonSize;
    loading?: boolean;
    disabled?: boolean;
    class?: string;
    children?: Snippet;
  }

  let {
    variant = 'secondary',
    size = 'md',
    loading = false,
    disabled = false,
    type = 'button',
    class: className = '',
    children,
    onclick,
    ...restProps
  }: Props = $props();

  const variantStyles: Record<ButtonVariant, string> = {
    primary: 'bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white font-semibold shadow-sm focus-visible:ring-emerald-500 border border-emerald-500/20',
    secondary: 'bg-zinc-100 hover:bg-zinc-200 active:bg-zinc-300 text-zinc-900 font-medium border border-zinc-300 shadow-xs focus-visible:ring-zinc-400 dark:bg-zinc-900 dark:hover:bg-zinc-800 dark:active:bg-zinc-800 dark:text-zinc-200 dark:border-zinc-700/80',
    outline: 'border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800/60 active:bg-zinc-200 dark:active:bg-zinc-800 text-zinc-700 dark:text-zinc-300 hover:text-zinc-900 dark:hover:text-zinc-100 font-medium focus-visible:ring-zinc-400 bg-transparent',
    ghost: 'hover:bg-zinc-100 dark:hover:bg-zinc-800/60 active:bg-zinc-200 dark:active:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200 font-medium focus-visible:ring-zinc-400',
    destructive: 'border border-rose-300 dark:border-rose-500/40 text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 active:bg-rose-100 dark:active:bg-rose-500/20 font-semibold focus-visible:ring-rose-500',
    accent: 'bg-sky-600 hover:bg-sky-500 active:bg-sky-700 text-white font-semibold shadow-sm focus-visible:ring-sky-500 border border-sky-500/20',
    indigo: 'bg-indigo-600 hover:bg-indigo-500 active:bg-indigo-700 text-white font-semibold shadow-sm focus-visible:ring-indigo-500 border border-indigo-500/20',
  };

  const sizeStyles: Record<ButtonSize, string> = {
    xs: 'px-2 py-1 text-xs rounded',
    sm: 'px-3 py-1.5 text-xs rounded-md gap-1.5',
    md: 'px-4 py-2 text-sm rounded-md gap-2',
    lg: 'px-5 py-2.5 text-base rounded-lg gap-2.5',
    icon: 'p-2 rounded-md justify-center items-center',
    'icon-sm': 'p-1.5 rounded-md justify-center items-center',
  };

  const baseStyles =
    'inline-flex items-center justify-center font-medium transition-colors cursor-pointer select-none disabled:opacity-50 disabled:cursor-not-allowed disabled:pointer-events-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950';
</script>

<button
  {type}
  disabled={disabled || loading}
  class={cn(baseStyles, variantStyles[variant], sizeStyles[size], className)}
  {onclick}
  {...restProps}
>
  {#if loading}
    <Loader2 class="h-4 w-4 animate-spin shrink-0" />
  {/if}
  {#if children}
    {@render children()}
  {/if}
</button>
