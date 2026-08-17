<script lang="ts">
  import { Dropdown } from '$lib/components/ui';
  import { appState } from '$lib/state/appState.svelte';
  import { apiClient } from '$lib/api/client';
  import FileText from 'lucide-svelte/icons/file-text';
  import Terminal from 'lucide-svelte/icons/terminal';
  import Archive from 'lucide-svelte/icons/archive';
  import FileJson from 'lucide-svelte/icons/file-json';
  import ChevronUp from 'lucide-svelte/icons/chevron-up';
  import Download from 'lucide-svelte/icons/download';

  interface Props {
    onPreviewMarkdown?: () => void;
    onPreviewLogs?: () => void;
    side?: 'top' | 'bottom';
    class?: string;
  }

  let {
    onPreviewMarkdown,
    onPreviewLogs,
    side = 'top',
    class: className = '',
  }: Props = $props();

  const isCancelled = $derived(
    appState.execution?.status === 'cancelled' ||
    appState.status === 'running.cancelling'
  );

  function handleDownload(format?: 'zip' | 'tar.gz' | 'tar.zst' | 'tar' | 'json' | 'md' | 'log') {
    if (typeof window === 'undefined') return;

    let url = '';
    let filename = 'report';

    if (format === 'json') {
      url = '/api/report';
      filename = 'report.json';
    } else if (format === 'md') {
      url = '/api/report/md?download=1';
      filename = 'report.md';
    } else if (format === 'log') {
      url = '/api/report/log?download=1';
      filename = 'report.log';
    } else if (format) {
      url = apiClient.getDownloadUrl(format);
      filename = `report.${format}`;
    }

    const token = apiClient.getToken();
    if (token && !url.includes('token=')) {
      const separator = url.includes('?') ? '&' : '?';
      url = `${url}${separator}token=${encodeURIComponent(token)}`;
    }

    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }

  const dropdownItems = $derived([
    {
      id: 'view-md',
      label: 'Markdown Report (.md)',
      icon: FileText,
      onclick: () => onPreviewMarkdown?.(),
    },
    {
      id: 'view-log',
      label: 'Execution Logs (.log)',
      icon: Terminal,
      onclick: () => onPreviewLogs?.(),
    },
    {
      id: 'sep-1',
      label: '',
      separator: true,
    },
    {
      id: 'dl-zip',
      label: isCancelled ? 'Download Archive (.zip) [Aborted]' : 'Download Archive (.zip)',
      icon: Archive,
      disabled: isCancelled,
      onclick: () => handleDownload('zip'),
    },
    {
      id: 'dl-targz',
      label: isCancelled ? 'Download Tarball (.tar.gz) [Aborted]' : 'Download Tarball (.tar.gz)',
      icon: Archive,
      disabled: isCancelled,
      onclick: () => handleDownload('tar.gz'),
    },
    {
      id: 'dl-tarzst',
      label: isCancelled ? 'Download Zstandard (.tar.zst) [Aborted]' : 'Download Zstandard (.tar.zst)',
      icon: Archive,
      disabled: isCancelled,
      onclick: () => handleDownload('tar.zst'),
    },
    {
      id: 'dl-tar',
      label: isCancelled ? 'Download POSIX Tar (.tar) [Aborted]' : 'Download POSIX Tar (.tar)',
      icon: Archive,
      disabled: isCancelled,
      onclick: () => handleDownload('tar'),
    },
    {
      id: 'dl-json',
      label: isCancelled ? 'Download JSON (.json) [Aborted]' : 'Download JSON (.json)',
      icon: FileJson,
      disabled: isCancelled,
      onclick: () => handleDownload('json'),
    },
  ]);
</script>

<Dropdown
  items={dropdownItems}
  {side}
  align="start"
  class={className}
>
  {#snippet trigger()}
    <span
      class="inline-flex items-center gap-2 px-3.5 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-100 text-xs font-semibold border border-zinc-700 shadow-sm transition cursor-pointer select-none"
    >
      <Download class="h-3.5 w-3.5 text-zinc-400" />
      <span>Reports & Bundles</span>
      <ChevronUp class="h-3.5 w-3.5 text-zinc-400" />
    </span>
  {/snippet}
</Dropdown>
