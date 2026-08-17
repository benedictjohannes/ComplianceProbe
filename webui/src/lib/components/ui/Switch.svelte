<script lang="ts">
  import { Switch as BitsSwitch } from 'bits-ui';
  import { cn } from '$lib/utils/cn';

  interface Props {
    checked?: boolean;
    disabled?: boolean;
    class?: string;
    onCheckedChange?: (checked: boolean) => void;
  }

  let {
    checked = $bindable(false),
    disabled = false,
    class: className = '',
    onCheckedChange,
  }: Props = $props();
</script>

<BitsSwitch.Root
  bind:checked
  {disabled}
  onCheckedChange={(c) => {
    checked = c;
    onCheckedChange?.(c);
  }}
  class={cn(
    'peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border border-transparent transition-colors',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950',
    'disabled:cursor-not-allowed disabled:opacity-50',
    'data-[state=checked]:bg-emerald-600 data-[state=unchecked]:bg-zinc-800 dark:data-[state=unchecked]:bg-zinc-700',
    className
  )}
>
  <BitsSwitch.Thumb
    class="pointer-events-none block h-3.5 w-3.5 rounded-full bg-white shadow-md ring-0 transition-transform data-[state=checked]:translate-x-4.5 data-[state=unchecked]:translate-x-0.5"
  />
</BitsSwitch.Root>
