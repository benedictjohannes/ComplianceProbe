import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import Scorecard from '$lib/components/views/ResultsView/Scorecard.svelte';
import MarkdownViewer from '$lib/components/views/ResultsView/MarkdownViewer.svelte';
import SubmissionPrompt from '$lib/components/views/ResultsView/SubmissionPrompt.svelte';
import ExportDropdown from '$lib/components/views/ResultsView/ExportDropdown.svelte';
import ReportTabs from '$lib/components/views/ResultsView/ReportTabs.svelte';
import ResultsView from '$lib/components/views/ResultsView/ResultsView.svelte';
import { appState } from '$lib/state/appState.svelte';
import { apiClient } from '$lib/api/client';
import type { PlaybookInspection } from '$lib/api/types';

const mockPlaybook: PlaybookInspection = {
  title: 'CIS Linux Security Benchmark',
  requiresElevation: true,
  sections: [
    {
      title: 'Storage & Partition Isolation',
      description: ['Ensure temporary spaces have strict mount flags.'],
      assertions: [
        {
          code: 'SEC-001',
          title: 'Separate /tmp partition is mounted',
          description: 'Verifies dedicated mount points for /tmp.',
          passDescription: '/tmp is mounted as a dedicated filesystem.',
          failDescription: '/tmp is not on a dedicated partition.',
          cmds: [
            {
              exec: { shell: 'bash', script: 'findmnt /tmp', requireElevation: true },
              passScore: 1,
            },
          ],
        },
        {
          code: 'SEC-002',
          title: 'nodev & noexec options set on /dev/shm',
          description: 'Ensure shared memory space has execution prevention flags.',
          passDescription: '/dev/shm mounted with nodev, nosuid, and noexec.',
          failDescription: 'Missing required mount options on /dev/shm.',
          cmds: [
            {
              exec: { shell: 'bash', script: 'findmnt -n /dev/shm' },
              passScore: 1,
              failScore: -1,
            },
          ],
        },
      ],
    },
  ],
};

describe('Scorecard Component', () => {
  beforeEach(() => {
    appState.status = 'completed';
    appState.playbook = mockPlaybook;
    appState.execution = {
      run_id: 'run-101',
      status: 'completed',
      last_event_id: 2,
      duration_ms: 1420,
      assertions: [
        {
          code: 'SEC-001',
          title: 'Separate /tmp partition is mounted',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 18,
        },
        {
          code: 'SEC-002',
          title: 'nodev & noexec options set on /dev/shm',
          status: 'failed',
          passed: false,
          score: -1,
          min_score: 1,
          duration_ms: 22,
          output: 'findmnt: /dev/shm mount flags missing noexec',
        },
      ],
      logs: [],
    };
  });

  it('renders compliance pass rate gauge and strict assertion counts', () => {
    const { getByText, getAllByText, getByLabelText } = render(Scorecard);

    expect(getByLabelText('Compliance Pass Rate Gauge')).toBeInTheDocument();
    expect(getByText('50%')).toBeInTheDocument();
    expect(getByText('FAILED')).toBeInTheDocument();
    expect(getByText(/1.42s/)).toBeInTheDocument();
    expect(getAllByText('Total Assertions').length).toBeGreaterThanOrEqual(1);
    expect(getByText('Passed')).toBeInTheDocument();
    expect(getByText('Failed')).toBeInTheDocument();
  });

  it('renders PASSED badge when 100% assertions are satisfied', () => {
    if (appState.execution) {
      appState.execution.assertions = [
        {
          code: 'SEC-001',
          title: 'Separate /tmp partition is mounted',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 18,
        },
        {
          code: 'SEC-002',
          title: 'nodev & noexec options set on /dev/shm',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 22,
        },
      ];
    }

    const { getByText } = render(Scorecard);
    expect(getByText('100%')).toBeInTheDocument();
    expect(getByText('PASSED')).toBeInTheDocument();
  });

  it('renders ABORTED badge when run was cancelled', () => {
    if (appState.execution) {
      appState.execution.status = 'cancelled';
    }

    const { getByText } = render(Scorecard);
    expect(getByText('ABORTED')).toBeInTheDocument();
  });
});

describe('MarkdownViewer Component', () => {
  const sampleMarkdown = `# Security Audit Report\nDate: 2026-08-18\n\n## Section 1\nSEC-001: PASSED\nSEC-002: FAILED`;

  it('renders line numbers and document content', () => {
    const { getByText } = render(MarkdownViewer, {
      props: { content: sampleMarkdown, filename: 'report.md' },
    });

    expect(getByText('report.md')).toBeInTheDocument();
    expect(getByText('1')).toBeInTheDocument();
    expect(getByText('6 lines • 0.1 KB')).toBeInTheDocument();
  });

  it('supports keyword search with highlight matching', async () => {
    const { getByPlaceholderText, getByText } = render(MarkdownViewer, {
      props: { content: sampleMarkdown, filename: 'report.md' },
    });

    const searchInput = getByPlaceholderText('Search document...');
    await fireEvent.input(searchInput, { target: { value: 'PASSED' } });

    expect(getByText('1/1')).toBeInTheDocument();
    expect(getByText('PASSED').tagName).toBe('MARK');
  });

  it('copies content to clipboard on button click', async () => {
    const writeTextSpy = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: { writeText: writeTextSpy },
    });

    const { getByRole, getByText } = render(MarkdownViewer, {
      props: { content: sampleMarkdown, filename: 'report.md' },
    });

    const copyBtn = getByRole('button', { name: /Copy/i });
    await fireEvent.click(copyBtn);

    expect(writeTextSpy).toHaveBeenCalledWith(sampleMarkdown);
    expect(getByText('Copied!')).toBeInTheDocument();
  });
});

describe('SubmissionPrompt Component', () => {
  beforeEach(() => {
    appState.status = 'completed';
    appState.reportDestination = {
      folder_source: 'default',
      https_source: 'custom',
      https: {
        url: 'https://compliance.internal.net/api/v1',
        format: 'json',
        secret: 'test-secret',
        headers: { 'X-Audit-ID': 'audit-99' },
      },
    };
    appState.errors = [];
  });

  it('displays configured destination attributes and dispatches submission', async () => {
    const submitSpy = vi.spyOn(appState, 'submitRemoteReport').mockResolvedValue(undefined);

    const { getByText, getByRole } = render(SubmissionPrompt, {
      props: { open: true },
    });

    expect(getByText('Remote Report Submission')).toBeInTheDocument();
    expect(getByText('https://compliance.internal.net/api/v1')).toBeInTheDocument();
    expect(getByText('JSON')).toBeInTheDocument();
    expect(getByText('Yes (Secret Configured)')).toBeInTheDocument();
    expect(getByText('X-Audit-ID')).toBeInTheDocument();

    const submitBtn = getByRole('button', { name: /Submit Report/i });
    await fireEvent.click(submitBtn);

    expect(submitSpy).toHaveBeenCalledOnce();
  });

  it('renders retry button when submission encountered an error', async () => {
    appState.errors = [
      {
        code: 'REMOTE_SUBMISSION_FAILED',
        message: 'HTTP 502 Bad Gateway from reporting server',
      },
    ];

    const { getByText, getByRole } = render(SubmissionPrompt, {
      props: { open: true },
    });

    expect(getByText('Submission Failed')).toBeInTheDocument();
    expect(getByText(/HTTP 502 Bad Gateway/)).toBeInTheDocument();
    expect(getByRole('button', { name: /Retry Submission/i })).toBeInTheDocument();
  });
});

describe('ExportDropdown Component', () => {
  beforeEach(() => {
    appState.status = 'completed';
    appState.execution = {
      run_id: 'run-101',
      status: 'completed',
      last_event_id: 2,
      duration_ms: 1000,
      assertions: [],
      logs: [],
    };
  });

  it('renders download trigger button and emits preview callbacks', async () => {
    const onPreviewMarkdown = vi.fn();
    const onPreviewLogs = vi.fn();

    const { getByRole } = render(ExportDropdown, {
      props: { onPreviewMarkdown, onPreviewLogs },
    });

    const trigger = getByRole('button', { name: /Reports & Bundles/i });
    expect(trigger).toBeInTheDocument();
  });
});

describe('ReportTabs Component', () => {
  beforeEach(() => {
    appState.status = 'completed';
    appState.playbook = mockPlaybook;
    appState.execution = {
      run_id: 'run-101',
      status: 'completed',
      last_event_id: 2,
      duration_ms: 1420,
      assertions: [
        {
          code: 'SEC-001',
          title: 'Separate /tmp partition is mounted',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 18,
        },
        {
          code: 'SEC-002',
          title: 'nodev & noexec options set on /dev/shm',
          status: 'failed',
          passed: false,
          score: -1,
          min_score: 1,
          duration_ms: 22,
          output: 'Diagnostic error for SEC-002',
        },
      ],
      logs: ['[INFO] Starting audit', '[ERROR] Failed SEC-002'],
    };
  });

  it('renders section group and filter toolbar in Audit tab', () => {
    const { getByText, getByPlaceholderText } = render(ReportTabs);

    expect(getByText('Execution Audit')).toBeInTheDocument();
    expect(getByText('Storage & Partition Isolation')).toBeInTheDocument();
    expect(getByText('SEC-001')).toBeInTheDocument();
    expect(getByText('SEC-002')).toBeInTheDocument();
    expect(getByPlaceholderText(/Filter rules/i)).toBeInTheDocument();
  });

  it('filters assertions by status filter pill', async () => {
    const { getByRole, queryByText, getByText } = render(ReportTabs);

    const failedFilterBtn = getByRole('button', { name: /Failed \(1\)/i });
    await fireEvent.click(failedFilterBtn);

    expect(getByText('SEC-002')).toBeInTheDocument();
    expect(queryByText('SEC-001')).not.toBeInTheDocument();

    const passedFilterBtn = getByRole('button', { name: /Passed \(1\)/i });
    await fireEvent.click(passedFilterBtn);

    expect(getByText('SEC-001')).toBeInTheDocument();
    expect(queryByText('SEC-002')).not.toBeInTheDocument();
  });

  it('switches to Markdown and Execution Logs tabs and loads content', async () => {
    vi.spyOn(apiClient, 'getReportMarkdown').mockResolvedValue('# Mock Markdown Report');
    vi.spyOn(apiClient, 'getReportLog').mockResolvedValue('[INFO] Mock Raw Log Output');

    const { getByRole, getByText } = render(ReportTabs);

    const markdownTab = getByRole('button', { name: /Markdown Report/i });
    await fireEvent.click(markdownTab);

    await waitFor(() => {
      expect(getByText('report.md')).toBeInTheDocument();
    });

    const logsTab = getByRole('button', { name: /Execution Logs/i });
    await fireEvent.click(logsTab);

    await waitFor(() => {
      expect(getByText('report.log')).toBeInTheDocument();
    });
  });
});

describe('ResultsView Orchestrator Component', () => {
  beforeEach(() => {
    appState.status = 'completed';
    appState.playbook = mockPlaybook;
    appState.execution = {
      run_id: 'run-101',
      status: 'completed',
      last_event_id: 2,
      duration_ms: 1420,
      assertions: [
        {
          code: 'SEC-001',
          title: 'Separate /tmp partition is mounted',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 18,
        },
      ],
      logs: [],
    };
    appState.reportDestination = {
      folder_source: 'default',
      https_source: 'custom',
      https: {
        url: 'https://compliance.internal.net',
        format: 'json',
      },
    };
  });

  it('triggers startRun when Re-run Playbook button is clicked', async () => {
    const startRunSpy = vi.spyOn(appState, 'startRun').mockResolvedValue(undefined);

    const { getByRole } = render(ResultsView);

    const rerunBtn = getByRole('button', { name: /Re-run Playbook/i });
    await fireEvent.click(rerunBtn);

    expect(startRunSpy).toHaveBeenCalledOnce();
  });

  it('triggers unloadPlaybook when Load Another Playbook button is clicked', async () => {
    const unloadSpy = vi.spyOn(appState, 'unloadPlaybook').mockResolvedValue(undefined);

    const { getByRole } = render(ResultsView);

    const unloadBtn = getByRole('button', { name: /Load Another Playbook/i });
    await fireEvent.click(unloadBtn);

    expect(unloadSpy).toHaveBeenCalledOnce();
  });

  it('renders Submit Report to Server button when HTTPS destination is configured', () => {
    const { getByRole } = render(ResultsView);
    expect(getByRole('button', { name: /Submit Report to Server/i })).toBeInTheDocument();
  });

  it('hides Submit Report to Server button when https_source is off', () => {
    appState.reportDestination = {
      folder_source: 'default',
      https_source: 'off',
    };
    const { queryByRole } = render(ResultsView);
    expect(queryByRole('button', { name: /Submit Report to Server/i })).not.toBeInTheDocument();
  });

  it('renders Report Submitted button when status is completed.submitted', () => {
    appState.status = 'completed.submitted';
    const { getByRole } = render(ResultsView);
    expect(getByRole('button', { name: /Report Submitted/i })).toBeInTheDocument();
  });

  it('renders Retry Server Submission button when status is completed.submission_error', () => {
    appState.status = 'completed.submission_error';
    const { getByRole } = render(ResultsView);
    expect(getByRole('button', { name: /Retry Server Submission/i })).toBeInTheDocument();
  });
});
