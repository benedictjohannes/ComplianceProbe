<script lang="ts">
  import Dialog from '$lib/components/ui/Dialog.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import AlertTriangle from 'lucide-svelte/icons/alert-triangle';

  interface Props {
    open?: boolean;
    onConfirm: () => void;
    onCancel?: () => void;
  }

  let {
    open = $bindable(false),
    onConfirm,
    onCancel,
  }: Props = $props();

  function handleCancel() {
    open = false;
    onCancel?.();
  }

  function handleConfirm() {
    open = false;
    onConfirm();
  }
</script>

<Dialog
  bind:open
  title="Cancel Playbook Execution?"
  description="Are you sure you want to stop the active audit run?"
  maxWidth="md"
>
  {#snippet children()}
    <div class="flex items-start gap-3 p-3 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-900 dark:text-amber-300 text-xs">
      <AlertTriangle class="h-5 w-5 text-amber-500 shrink-0 mt-0.5" />
      <div class="space-y-1">
        <p class="font-semibold text-amber-900 dark:text-amber-200">
          In-flight processes will be aborted
        </p>
        <p class="text-amber-800/90 dark:text-amber-300/90 leading-relaxed">
          Cancelling will terminate child shell processes and mark remaining assertions as skipped. Logs and completed assertions up to this point will be saved.
        </p>
      </div>
    </div>
  {/snippet}

  {#snippet footer()}
    <Button
      variant="outline"
      size="sm"
      onclick={handleCancel}
    >
      Keep Running
    </Button>
    <Button
      variant="destructive"
      size="sm"
      onclick={handleConfirm}
    >
      Cancel Execution
    </Button>
  {/snippet}
</Dialog>
