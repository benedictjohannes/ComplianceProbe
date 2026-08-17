import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ProgressBar from '$lib/components/views/ExecutionView/ProgressBar.svelte';
import AssertionCard from '$lib/components/views/ExecutionView/AssertionCard.svelte';
import AssertionList from '$lib/components/views/ExecutionView/AssertionList.svelte';
import CancelConfirmModal from '$lib/components/views/ExecutionView/CancelConfirmModal.svelte';
import ElevationPrompt from '$lib/components/views/ExecutionView/ElevationPrompt.svelte';
import ExecutionView from '$lib/components/views/ExecutionView/ExecutionView.svelte';
import { appState } from '$lib/state/appState.svelte';
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

describe('ProgressBar Component', () => {
  beforeEach(() => {
    appState.status = 'running';
    appState.playbook = mockPlaybook;
    appState.execution = {
      run_id: 'run-99',
      status: 'running',
      last_event_id: 1,
      duration_ms: 0,
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
  });

  it('renders elapsed timer, counts, and progress track', () => {
    const { getByText, getByRole, getByLabelText } = render(ProgressBar, {
      props: {
        onCancelRequest: vi.fn(),
      },
    });

    expect(getByLabelText('Elapsed time')).toBeInTheDocument();
    expect(getByText(/Run: run-99/i)).toBeInTheDocument();
    expect(getByText('50%')).toBeInTheDocument();
    expect(getByText('(1/2)')).toBeInTheDocument();

    const progressbar = getByRole('progressbar');
    expect(progressbar).toHaveAttribute('aria-valuenow', '50');
  });

  it('triggers onCancelRequest when Cancel Run button is clicked', async () => {
    const onCancelRequest = vi.fn();
    const { getByText } = render(ProgressBar, {
      props: { onCancelRequest },
    });

    const cancelBtn = getByText('Cancel Run');
    await fireEvent.click(cancelBtn);
    expect(onCancelRequest).toHaveBeenCalledOnce();
  });

  it('displays cancelling spinner when state is running.cancelling', () => {
    appState.status = 'running.cancelling';
    const { getByText } = render(ProgressBar);

    expect(getByText('Cancelling...')).toBeInTheDocument();
  });
});

describe('AssertionCard Component', () => {
  it('renders pending state with hollow circle and pending badge', () => {
    const assertion = mockPlaybook.sections[0].assertions[0];
    const { getByText } = render(AssertionCard, {
      props: {
        assertion,
        snapshot: undefined,
      },
    });

    expect(getByText('SEC-001')).toBeInTheDocument();
    expect(getByText('Separate /tmp partition is mounted')).toBeInTheDocument();
    expect(getByText('Pending')).toBeInTheDocument();
    expect(getByText('sudo')).toBeInTheDocument();
  });

  it('renders running state with active indicator', () => {
    const assertion = mockPlaybook.sections[0].assertions[0];
    const { getByText } = render(AssertionCard, {
      props: {
        assertion,
        snapshot: {
          code: 'SEC-001',
          title: assertion.title,
          status: 'running',
          passed: false,
          score: 0,
          min_score: 1,
          duration_ms: 0,
        },
        isActive: true,
      },
    });

    expect(getByText('Running')).toBeInTheDocument();
  });

  it('renders passed state with score and duration', () => {
    const assertion = mockPlaybook.sections[0].assertions[0];
    const { getByText } = render(AssertionCard, {
      props: {
        assertion,
        snapshot: {
          code: 'SEC-001',
          title: assertion.title,
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 24,
        },
      },
    });

    expect(getByText('+1 pt')).toBeInTheDocument();
    expect(getByText('24ms')).toBeInTheDocument();
  });

  it('renders failed state auto-expanded with failure diagnostics and output', () => {
    const assertion = mockPlaybook.sections[0].assertions[1];
    const { getByText } = render(AssertionCard, {
      props: {
        assertion,
        snapshot: {
          code: 'SEC-002',
          title: assertion.title,
          status: 'failed',
          passed: false,
          score: -1,
          min_score: 1,
          duration_ms: 45,
          output: 'StdOutRule failed: nodev not found in mount options',
        },
      },
    });

    expect(getByText('-1/1 pts')).toBeInTheDocument();
    expect(getByText('Execution Failed')).toBeInTheDocument();
    expect(getByText(/nodev not found in mount options/)).toBeInTheDocument();
  });

  it('allows manually toggling card accordion', async () => {
    const assertion = mockPlaybook.sections[0].assertions[0];
    const { getByText, queryByText } = render(AssertionCard, {
      props: {
        assertion,
        snapshot: {
          code: 'SEC-001',
          title: assertion.title,
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 10,
        },
      },
    });

    // Initially collapsed for passed assertion
    expect(queryByText('Verifies dedicated mount points for /tmp.')).not.toBeInTheDocument();

    const trigger = getByText('Separate /tmp partition is mounted');
    await fireEvent.click(trigger);

    expect(getByText('Verifies dedicated mount points for /tmp.')).toBeInTheDocument();
  });
});

describe('AssertionList Component', () => {
  beforeEach(() => {
    appState.playbook = mockPlaybook;
    appState.execution = {
      run_id: 'run-1',
      status: 'running',
      active_assertion_code: 'SEC-002',
      last_event_id: 1,
      duration_ms: 100,
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
          status: 'running',
          passed: false,
          score: 0,
          min_score: 1,
          duration_ms: 0,
        },
      ],
      logs: [],
    };
  });

  it('renders section headers and contained assertion cards', () => {
    const { getByText } = render(AssertionList);

    expect(getByText('1. Storage & Partition Isolation')).toBeInTheDocument();
    expect(getByText('(2 assertions)')).toBeInTheDocument();
    expect(getByText('1 passed')).toBeInTheDocument();
    expect(getByText('SEC-001')).toBeInTheDocument();
    expect(getByText('SEC-002')).toBeInTheDocument();
  });

  it('allows collapsing and expanding sections', async () => {
    const { getByText, queryByText } = render(AssertionList);

    const sectionHeader = getByText('1. Storage & Partition Isolation');
    await fireEvent.click(sectionHeader);

    // Section collapsed: descriptions & child assertions hidden
    expect(queryByText('Ensure temporary spaces have strict mount flags.')).not.toBeInTheDocument();

    await fireEvent.click(sectionHeader);
    expect(getByText('Ensure temporary spaces have strict mount flags.')).toBeInTheDocument();
  });
});

describe('CancelConfirmModal Component', () => {
  it('invokes onConfirm when Cancel Execution is clicked', async () => {
    const onConfirm = vi.fn();
    const { getByText } = render(CancelConfirmModal, {
      props: {
        open: true,
        onConfirm,
      },
    });

    expect(getByText('Cancel Playbook Execution?')).toBeInTheDocument();
    expect(getByText(/In-flight processes will be aborted/i)).toBeInTheDocument();

    const cancelBtn = getByText('Cancel Execution');
    await fireEvent.click(cancelBtn);
    expect(onConfirm).toHaveBeenCalledOnce();
  });
});

describe('ElevationPrompt Component', () => {
  it('renders administrator authorization dialog when open is true', () => {
    const { getByText } = render(ElevationPrompt, {
      props: { open: true },
    });

    expect(getByText('Administrator Privileges Required')).toBeInTheDocument();
    expect(getByText(/Waiting for system authorization/i)).toBeInTheDocument();
  });
});

describe('ExecutionView Orchestrator Component', () => {
  beforeEach(() => {
    appState.status = 'running';
    appState.playbook = mockPlaybook;
    appState.errors = [];
    appState.execution = {
      run_id: 'run-99',
      status: 'running',
      last_event_id: 1,
      duration_ms: 100,
      assertions: [],
      logs: ['Log line 1', 'Log line 2'],
    };
  });

  it('renders ProgressBar, AssertionList, and LogStream drawer', () => {
    const { getByText, getByRole } = render(ExecutionView);

    expect(getByRole('progressbar')).toBeInTheDocument();
    expect(getByText('1. Storage & Partition Isolation')).toBeInTheDocument();
    expect(getByText('Console Logs')).toBeInTheDocument();
  });

  it('renders error banner when execution state has errors', () => {
    appState.errors = [
      {
        code: 'EXECUTION_FAILED',
        message: 'Command execution timed out after 30 seconds',
      },
    ];

    const { getByText } = render(ExecutionView);

    expect(getByText('EXECUTION_FAILED')).toBeInTheDocument();
    expect(getByText('Command execution timed out after 30 seconds')).toBeInTheDocument();
  });
});
