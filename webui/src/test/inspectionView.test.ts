import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import PlaybookHeader from '$lib/components/views/InspectionView/PlaybookHeader.svelte';
import AssertionItem from '$lib/components/views/InspectionView/AssertionItem.svelte';
import SectionsTree from '$lib/components/views/InspectionView/SectionsTree.svelte';
import DestinationSummary from '$lib/components/views/InspectionView/DestinationSummary.svelte';
import DestinationDrawer from '$lib/components/views/InspectionView/DestinationDrawer.svelte';
import InspectionView from '$lib/components/views/InspectionView/InspectionView.svelte';
import { AppState } from '$lib/state/appState.svelte';
import { ApiClient } from '$lib/api/client';
import { EventStreamManager } from '$lib/api/events';
import type { PlaybookInspection, Section, Assertion, ReportDestinationState } from '$lib/api/types';

const mockAssertion1: Assertion = {
  code: 'SEC-001',
  title: 'Separate /tmp partition is mounted',
  description: 'Ensure /tmp is on a dedicated filesystem.',
  passDescription: '/tmp is mounted as a dedicated partition.',
  failDescription: '/tmp is part of the root partition.',
  minPassingScore: 1,
  cmds: [
    {
      exec: {
        shell: 'bash',
        script: 'findmnt /tmp',
        requireElevation: true,
      },
      passScore: 1,
      failScore: -1,
      stdOutRule: { regex: '^/tmp' },
    },
  ],
};

const mockAssertion2: Assertion = {
  code: 'SEC-002',
  title: 'nodev & noexec options set on /dev/shm',
  description: 'Ensure shared memory space has restrictive mount flags.',
  passDescription: '/dev/shm has nodev, nosuid, and noexec.',
  failDescription: 'Missing required mount options on /dev/shm.',
  minPassingScore: 2,
  preCmds: [
    {
      shell: 'bash',
      script: 'mount | grep /dev/shm',
      gather: [{ key: 'shm_mount', regex: '^tmpfs on (/dev/shm)' }],
    },
  ],
  cmds: [
    {
      exec: {
        shell: 'bash',
        script: 'findmnt -n /dev/shm',
        requireElevation: true,
      },
      passScore: 1,
      failScore: -1,
      stdOutRule: { regex: '/nodev/' },
    },
    {
      exec: {
        shell: 'bash',
        script: 'grep -E "\\s/dev/shm\\s" /etc/fstab',
      },
      passScore: 1,
      failScore: -1,
      stdOutRule: { regex: '/noexec/' },
    },
  ],
};

const mockSections: Section[] = [
  {
    title: 'System Storage & Partition Isolation',
    description: ['Verifies dedicated mount points and restrictive options.'],
    assertions: [mockAssertion1, mockAssertion2],
  },
];

const mockPlaybook: PlaybookInspection = {
  title: 'CIS Ubuntu 22.04 Benchmark',
  requiresElevation: true,
  reportFrontmatter: {
    version: '1.2.0',
    description: 'Comprehensive security configuration benchmark for Ubuntu LTS servers.',
    author: 'SecOps Team',
  },
  sections: mockSections,
};

const mockDestination: ReportDestinationState = {
  folder_source: 'default',
  https_source: 'playbook',
  playbook_defaults: {
    has_folder: true,
    has_https: true,
    https: {
      url: 'https://sec-ops.corp.internal/ingest',
      format: 'json',
      hasSignatureSecret: true,
      configuredHeaders: ['Authorization'],
    },
  },
};

describe('PlaybookHeader Component', () => {
  it('renders playbook title, description, elevation badge, and metadata chips', () => {
    render(PlaybookHeader, {
      props: {
        playbook: mockPlaybook,
        destination: mockDestination,
      },
    });

    expect(screen.getByText('CIS Ubuntu 22.04 Benchmark')).toBeInTheDocument();
    expect(screen.getByText(/Comprehensive security configuration benchmark/)).toBeInTheDocument();
    expect(screen.getByText('Requires Sudo')).toBeInTheDocument();
    expect(screen.getByText('2 Assertions')).toBeInTheDocument();
    expect(screen.getByText('1 Section')).toBeInTheDocument();
    expect(screen.getByText('SecOps Team')).toBeInTheDocument();
  });
});

describe('AssertionItem Component', () => {
  it('renders collapsed state with code pill, title, sudo badge, and cmd count', () => {
    render(AssertionItem, {
      props: { assertion: mockAssertion1 },
    });

    expect(screen.getByText('SEC-001')).toBeInTheDocument();
    expect(screen.getByText('Separate /tmp partition is mounted')).toBeInTheDocument();
    expect(screen.getByText('sudo')).toBeInTheDocument();
    expect(screen.getByText('1 cmd')).toBeInTheDocument();
  });

  it('expands Level 2 criteria and Level 3 command details on click', async () => {
    render(AssertionItem, {
      props: { assertion: mockAssertion2 },
    });

    // Initially collapsed
    expect(screen.queryByText(/Ensure shared memory space/)).not.toBeInTheDocument();

    // Click header to expand Level 2
    const trigger = screen.getByText('nodev & noexec options set on /dev/shm');
    await fireEvent.click(trigger);

    expect(screen.getByText(/Ensure shared memory space/)).toBeInTheDocument();
    expect(screen.getByText(/✓ Pass Criteria:/)).toBeInTheDocument();
    expect(screen.getByText(/✕ Fail Criteria:/)).toBeInTheDocument();
    expect(screen.getByText(/Min Passing Score:/)).toBeInTheDocument();

    // Click View Commands to expand Level 3
    const viewCommandsBtn = screen.getByText(/View Commands/);
    await fireEvent.click(viewCommandsBtn);

    expect(screen.getByText('[Pre-Command 1/1]')).toBeInTheDocument();
    expect(screen.getByText('[Command 1/2]')).toBeInTheDocument();
    expect(screen.getByText('[Command 2/2]')).toBeInTheDocument();
    expect(screen.getByText('mount | grep /dev/shm')).toBeInTheDocument();
    expect(screen.getByText('findmnt -n /dev/shm')).toBeInTheDocument();
  });
});

describe('SectionsTree Component', () => {
  it('renders section headers and list of child assertions', () => {
    render(SectionsTree, {
      props: { sections: mockSections },
    });

    expect(screen.getByText('1. System Storage & Partition Isolation')).toBeInTheDocument();
    expect(screen.getByText('2 assertions')).toBeInTheDocument();
    expect(screen.getByText('SEC-001')).toBeInTheDocument();
    expect(screen.getByText('SEC-002')).toBeInTheDocument();
  });

  it('allows collapsing and expanding sections', async () => {
    render(SectionsTree, {
      props: { sections: mockSections },
    });

    expect(screen.getByText('SEC-001')).toBeInTheDocument();

    const sectionHeader = screen.getByText('1. System Storage & Partition Isolation');
    await fireEvent.click(sectionHeader);

    // Section collapsed -> assertions hidden
    expect(screen.queryByText('SEC-001')).not.toBeInTheDocument();

    // Click again to re-expand
    await fireEvent.click(sectionHeader);
    expect(screen.getByText('SEC-001')).toBeInTheDocument();
  });
});

describe('DestinationSummary Component', () => {
  it('renders folder and HTTPS status cards', () => {
    const handleOpen = vi.fn();
    render(DestinationSummary, {
      props: {
        destination: mockDestination,
        onOpenDrawer: handleOpen,
      },
    });

    expect(screen.getByText('Local Folder Storage')).toBeInTheDocument();
    expect(screen.getByText('Remote HTTPS Submission')).toBeInTheDocument();
    expect(screen.getByText('https://sec-ops.corp.internal/ingest')).toBeInTheDocument();
    expect(screen.getByText('HMAC Signed')).toBeInTheDocument();
  });
});

describe('DestinationDrawer Component', () => {
  it('allows changing folder source to custom and HTTPS to custom with headers and secret', async () => {
    const handleSave = vi.fn();
    render(DestinationDrawer, {
      props: {
        open: true,
        destination: mockDestination,
        onSave: handleSave,
      },
    });

    // Select custom folder
    const customFolderRadio = screen.getByLabelText(/Custom Path/);
    await fireEvent.click(customFolderRadio);

    const folderInput = screen.getByPlaceholderText('/var/log/compliance-reports/');
    await fireEvent.input(folderInput, { target: { value: '/tmp/audit-reports' } });

    // Select custom HTTPS
    const customHttpsRadio = screen.getByLabelText(/Configure URL/);
    await fireEvent.click(customHttpsRadio);

    const urlInput = screen.getByPlaceholderText('https://sec-ops.corp.internal/ingest');
    await fireEvent.input(urlInput, { target: { value: 'https://my-server.com/api/reports' } });

    // Save
    const saveBtn = screen.getByText('Save Destination Settings');
    await fireEvent.click(saveBtn);

    expect(handleSave).toHaveBeenCalledWith({
      folder_source: 'custom',
      folder: '/tmp/audit-reports',
      https_source: 'custom',
      https: {
        url: 'https://my-server.com/api/reports',
        format: 'json',
        secret: undefined,
        headers: undefined,
      },
    });
  });
});

describe('InspectionView Orchestrator Component', () => {
  let mockClient: ApiClient;
  let mockStream: EventStreamManager;
  let state: AppState;

  beforeEach(() => {
    mockClient = {
      getPlaybook: vi.fn().mockResolvedValue(mockPlaybook),
      deletePlaybook: vi.fn().mockResolvedValue({
        status: 'idle',
        errors: [],
        report_destination: { folder_source: 'default', https_source: 'off' },
      }),
      startRun: vi.fn().mockResolvedValue({
        status: 'running',
        active_run_id: 'run-123',
        errors: [],
        report_destination: { folder_source: 'default', https_source: 'off' },
      }),
      updateDestination: vi.fn().mockResolvedValue({
        status: 'loaded',
        errors: [],
        report_destination: { folder_source: 'custom', https_source: 'off' },
      }),
    } as unknown as ApiClient;

    mockStream = {
      onStatusChange: vi.fn().mockReturnValue(() => {}),
      onReconnected: vi.fn().mockReturnValue(() => {}),
      on: vi.fn().mockReturnValue(() => {}),
      connect: vi.fn(),
      disconnect: vi.fn(),
      setToken: vi.fn(),
      setLastEventId: vi.fn(),
    } as unknown as EventStreamManager;

    state = new AppState(mockClient, mockStream);
    state.status = 'loaded';
    state.playbook = mockPlaybook;
    state.reportDestination = mockDestination;
  });

  it('renders all inspection sections, headers, and execute action', async () => {
    render(InspectionView, { props: { appStateInstance: state } });

    expect(screen.getByText('CIS Ubuntu 22.04 Benchmark')).toBeInTheDocument();
    expect(screen.getByText('1. System Storage & Partition Isolation')).toBeInTheDocument();
    expect(screen.getByText('Unload Playbook')).toBeInTheDocument();

    const executeBtn = screen.getByText('Execute Playbook ➜');
    expect(executeBtn).toBeInTheDocument();
    expect(executeBtn).not.toBeDisabled();

    await fireEvent.click(executeBtn);
    expect(mockClient.startRun).toHaveBeenCalledTimes(1);
  });

  it('disables execute action when validation errors exist', () => {
    state.errors = [
      {
        code: 'PLAYBOOK_VALIDATION_FAILED',
        message: 'Playbook validation failed',
        detail: [{ path: 'sections[0].assertions[1].code', message: 'Duplicate code' }],
      },
    ];

    render(InspectionView, { props: { appStateInstance: state } });

    expect(screen.getByText('PLAYBOOK_VALIDATION_FAILED')).toBeInTheDocument();
    expect(screen.getByText('Duplicate code')).toBeInTheDocument();

    const executeBtn = screen.getByText('Execute Playbook ➜');
    expect(executeBtn).toBeDisabled();
  });

  it('unloads playbook on Unload button click', async () => {
    render(InspectionView, { props: { appStateInstance: state } });

    const unloadBtn = screen.getByText('Unload Playbook');
    await fireEvent.click(unloadBtn);

    expect(mockClient.deletePlaybook).toHaveBeenCalledTimes(1);
  });
});
