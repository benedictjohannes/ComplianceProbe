import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { AppState } from '../lib/state/appState.svelte';
import { ApiClient } from '../lib/api/client';
import { EventStreamManager } from '../lib/api/events';
import type {
  AppStateResponse,
  PlaybookInspection,
  ExecutionSnapshot,
  AssertionProgressEventData,
} from '../lib/api/types';

describe('AppState Store', () => {
  let mockClient: ApiClient;
  let mockStream: EventStreamManager;
  let state: AppState;

  beforeEach(() => {
    mockClient = new ApiClient({ baseUrl: 'http://localhost:8080' });
    mockStream = new EventStreamManager({
      eventSourceFactory: () => ({
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        close: vi.fn(),
      }) as unknown as EventSource,
    });
    state = new AppState(mockClient, mockStream);
  });

  afterEach(() => {
    state.destroy();
    vi.restoreAllMocks();
  });

  it('should initialize with idle default state and computed derived helpers', () => {
    expect(state.status).toBe('idle');
    expect(state.isIdle).toBe(true);
    expect(state.isLoaded).toBe(false);
    expect(state.isRunning).toBe(false);
    expect(state.isCompleted).toBe(false);
    expect(state.isSubmitted).toBe(false);
    expect(state.isSubmissionError).toBe(false);
    expect(state.hasErrors).toBe(false);
    expect(state.currentPipelineStep).toBe(0);
    expect(state.totalAssertions).toBe(0);
    expect(state.completedAssertions).toBe(0);
    expect(state.progressPercent).toBe(0);
  });

  it('should compute isSubmitted and isSubmissionError accurately', () => {
    state.status = 'completed.submitted';
    expect(state.isCompleted).toBe(true);
    expect(state.isSubmitted).toBe(true);
    expect(state.isSubmissionError).toBe(false);

    state.status = 'completed.submission_error';
    expect(state.isCompleted).toBe(true);
    expect(state.isSubmitted).toBe(false);
    expect(state.isSubmissionError).toBe(true);
  });

  it('reconcileState: should update state and clear execution/playbook when transitioning to idle', () => {
    state.playbook = { title: 'Test', sections: [], requiresElevation: false };
    state.execution = {
      run_id: 'r1',
      status: 'running',
      last_event_id: 1,
      duration_ms: 0,
      assertions: [],
      logs: ['log1'],
    };
    state.logs = ['log1'];

    const idleResp: AppStateResponse = {
      status: 'idle',
      errors: [],
      report_destination: { folder_source: 'default', https_source: 'off' },
    };

    state.reconcileState(idleResp);

    expect(state.status).toBe('idle');
    expect(state.playbook).toBeNull();
    expect(state.execution).toBeNull();
    expect(state.logs).toEqual([]);
  });

  it('reconcileState: should ignore invalid or malformed responses and log a warning', () => {
    state.status = 'loaded';
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    // @ts-expect-error test invalid payload
    state.reconcileState(null);
    expect(state.status).toBe('loaded');
    expect(warnSpy).toHaveBeenCalled();

    // @ts-expect-error test payload without status
    state.reconcileState({ errors: [] });
    expect(state.status).toBe('loaded');

    // @ts-expect-error test invalid state_change
    state.handleStateChange(null, 10);
    expect(state.status).toBe('loaded');

    warnSpy.mockRestore();
  });

  it('reconcileExecution: should reconstruct snapshot and compute metrics', () => {
    const mockSnapshot: ExecutionSnapshot = {
      run_id: 'run-99',
      status: 'running',
      last_event_id: 55,
      duration_ms: 3200,
      assertions: [
        {
          code: 'CHECK_1',
          title: 'Check 1',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 100,
        },
        {
          code: 'CHECK_2',
          title: 'Check 2',
          status: 'failed',
          passed: false,
          score: -1,
          min_score: 1,
          duration_ms: 200,
        },
        {
          code: 'CHECK_3',
          title: 'Check 3',
          status: 'pending',
          passed: false,
          score: 0,
          min_score: 1,
          duration_ms: 0,
        },
      ],
      logs: ['log 1', 'log 2'],
    };

    state.reconcileExecution(mockSnapshot);

    expect(state.execution?.run_id).toBe('run-99');
    expect(state.lastEventId).toBe(55);
    expect(state.logs).toEqual(['log 1', 'log 2']);
    expect(state.totalAssertions).toBe(3);
    expect(state.passedAssertions).toBe(1);
    expect(state.failedAssertions).toBe(1);
    expect(state.completedAssertions).toBe(2);
    expect(state.progressPercent).toBe(67); // 2/3 = 66.66% -> 67%
  });

  it('fullSync: should fetch state, playbook and execution when running', async () => {
    const mockStateResp: AppStateResponse = {
      status: 'running',
      errors: [],
      active_run_id: 'run-101',
      report_destination: { folder_source: 'default', https_source: 'off' },
    };

    const mockPlaybook: PlaybookInspection = {
      title: 'Security Probe',
      sections: [
        {
          title: 'Section 1',
          description: ['Description'],
          assertions: [
            {
              code: 'A_01',
              title: 'Assert 1',
              description: 'Desc',
              cmds: [],
              passDescription: 'Pass',
              failDescription: 'Fail',
            },
          ],
        },
      ],
      requiresElevation: true,
    };

    const mockExec: ExecutionSnapshot = {
      run_id: 'run-101',
      status: 'running',
      last_event_id: 10,
      duration_ms: 500,
      assertions: [],
      logs: ['Init log'],
    };

    vi.spyOn(mockClient, 'getState').mockResolvedValue(mockStateResp);
    vi.spyOn(mockClient, 'getPlaybook').mockResolvedValue(mockPlaybook);
    vi.spyOn(mockClient, 'getExecution').mockResolvedValue(mockExec);

    await state.fullSync();

    expect(state.status).toBe('running');
    expect(state.playbook?.title).toBe('Security Probe');
    expect(state.execution?.run_id).toBe('run-101');
    expect(state.isRunning).toBe(true);
    expect(state.currentPipelineStep).toBe(2);
  });

  it('handleAssertionProgress: should dynamically append and update assertions', () => {
    state.status = 'running';
    state.playbook = {
      title: 'Playbook',
      sections: [
        {
          title: 'Sec',
          description: [],
          assertions: [
            {
              code: 'NET_01',
              title: 'Network Port Check',
              description: '',
              cmds: [],
              passDescription: '',
              failDescription: '',
            },
          ],
        },
      ],
      requiresElevation: false,
    };

    // 1. Assertion starts running
    const progress1: AssertionProgressEventData = {
      run_id: 'run-1',
      code: 'NET_01',
      status: 'running',
      passed: false,
      score: 0,
      min_score: 1,
      duration_ms: 0,
    };
    state.handleAssertionProgress(progress1, 1);

    expect(state.execution?.assertions.length).toBe(1);
    expect(state.execution?.assertions[0].code).toBe('NET_01');
    expect(state.execution?.assertions[0].title).toBe('Network Port Check');
    expect(state.execution?.assertions[0].status).toBe('running');
    expect(state.execution?.active_assertion_code).toBe('NET_01');

    // 2. Assertion completes with pass
    const progress2: AssertionProgressEventData = {
      run_id: 'run-1',
      code: 'NET_01',
      status: 'passed',
      passed: true,
      score: 1,
      min_score: 1,
      duration_ms: 120,
      output: 'Port open',
    };
    state.handleAssertionProgress(progress2, 2);

    expect(state.execution?.assertions.length).toBe(1);
    expect(state.execution?.assertions[0].status).toBe('passed');
    expect(state.execution?.assertions[0].passed).toBe(true);
    expect(state.execution?.assertions[0].duration_ms).toBe(120);
    expect(state.execution?.assertions[0].output).toBe('Port open');
    expect(state.execution?.active_assertion_code).toBeUndefined();
    expect(state.passedAssertions).toBe(1);
  });

  it('handleLog: should append log messages', () => {
    state.status = 'running';
    state.execution = {
      run_id: 'run-1',
      status: 'running',
      last_event_id: 1,
      duration_ms: 0,
      assertions: [],
      logs: [],
    };

    state.handleLog({ run_id: 'run-1', message: 'First log line' }, 2);
    state.handleLog({ run_id: 'run-1', message: 'Second log line' }, 3);

    expect(state.logs).toEqual(['First log line', 'Second log line']);
    expect(state.execution.logs).toEqual(['First log line', 'Second log line']);
  });

  it('handleExecutionCompleted and handleExecutionCancelled: should update execution status', () => {
    state.status = 'running';
    state.execution = {
      run_id: 'run-1',
      status: 'running',
      last_event_id: 1,
      duration_ms: 100,
      assertions: [],
      logs: [],
    };

    state.handleExecutionCompleted({ run_id: 'run-1', status: 'completed', duration_ms: 4500 }, 4);
    expect(state.execution.status).toBe('completed');
    expect(state.execution.duration_ms).toBe(4500);

    state.handleExecutionCancelled({ run_id: 'run-1' }, 5);
    expect(state.execution.status).toBe('cancelled');
  });

  it('user action dispatchers should update state cleanly', async () => {
    const mockFile = new File(['playbook content'], 'my-playbook.yaml');
    const mockLoadedState: AppStateResponse = {
      status: 'loaded',
      errors: [],
      report_destination: { folder_source: 'default', https_source: 'off' },
    };
    const mockPlaybook: PlaybookInspection = {
      title: 'My Playbook',
      sections: [],
      requiresElevation: false,
    };

    vi.spyOn(mockClient, 'uploadPlaybook').mockResolvedValue(mockLoadedState);
    vi.spyOn(mockClient, 'getPlaybook').mockResolvedValue(mockPlaybook);

    await state.uploadPlaybookFile(mockFile);
    expect(state.status).toBe('loaded');
    expect(state.playbook?.title).toBe('My Playbook');
    expect(state.isLoaded).toBe(true);

    // Unload playbook
    vi.spyOn(mockClient, 'deletePlaybook').mockResolvedValue({
      status: 'idle',
      errors: [],
      report_destination: { folder_source: 'default', https_source: 'off' },
    });

    await state.unloadPlaybook();
    expect(state.status).toBe('idle');
    expect(state.playbook).toBeNull();
  });

  it('clearErrors and dismissError: should modify errors array', () => {
    state.errors = [
      { code: 'ERR_1', message: 'First error' },
      { code: 'ERR_2', message: 'Second error' },
    ];
    expect(state.hasErrors).toBe(true);

    state.dismissError(0);
    expect(state.errors.length).toBe(1);
    expect(state.errors[0].code).toBe('ERR_2');

    state.clearErrors();
    expect(state.errors).toEqual([]);
    expect(state.hasErrors).toBe(false);
  });

  it('race condition prevention: startRun must not overwrite assertions populated by early SSE events', async () => {
    state.playbook = {
      title: 'Security Probe',
      sections: [
        {
          title: 'Section 1',
          description: [],
          assertions: [
            { code: 'CHK_1', title: 'Check 1', description: '', cmds: [], passDescription: '', failDescription: '' },
            { code: 'CHK_2', title: 'Check 2', description: '', cmds: [], passDescription: '', failDescription: '' },
          ],
        },
      ],
      requiresElevation: false,
    };

    // Simulate startRun taking some time over HTTP
    let resolveStartRun: (value: AppStateResponse) => void = () => {};
    const startRunPromise = new Promise<AppStateResponse>((resolve) => {
      resolveStartRun = resolve;
    });
    vi.spyOn(mockClient, 'startRun').mockReturnValue(startRunPromise);

    // 1. User starts run
    const runCall = state.startRun();

    // 2. While POST /api/run is in flight, SSE events arrive for run-99
    state.handleStateChange(
      {
        status: 'running',
        active_run_id: 'run-99',
        errors: [],
        report_destination: { folder_source: 'default', https_source: 'off' },
      },
      1
    );

    state.handleAssertionProgress(
      {
        run_id: 'run-99',
        code: 'CHK_1',
        status: 'passed',
        passed: true,
        score: 1,
        min_score: 1,
        duration_ms: 50,
      },
      2
    );

    // Verify assertion is recorded as passed
    expect(state.passedAssertions).toBe(1);
    expect(state.execution?.assertions.find((a) => a.code === 'CHK_1')?.status).toBe('passed');

    // 3. POST /api/run finally resolves
    resolveStartRun({
      status: 'running',
      active_run_id: 'run-99',
      errors: [],
      report_destination: { folder_source: 'default', https_source: 'off' },
    });
    await runCall;

    // 4. Assert that CHK_1 is NOT wiped out
    expect(state.execution?.assertions.find((a) => a.code === 'CHK_1')?.status).toBe('passed');
    expect(state.passedAssertions).toBe(1);
  });

  it('race condition prevention: reconcileExecution must not revert completed assertions to pending', () => {
    state.execution = {
      run_id: 'run-100',
      status: 'running',
      last_event_id: 10,
      duration_ms: 120,
      assertions: [
        {
          code: 'CHK_1',
          title: 'Check 1',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 60,
        },
      ],
      logs: ['Log 1'],
    };
    state.lastEventId = 10;

    // Stale snapshot arrived from initial state where CHK_1 was pending
    const staleSnapshot: ExecutionSnapshot = {
      run_id: 'run-100',
      status: 'running',
      last_event_id: 2,
      duration_ms: 0,
      assertions: [
        {
          code: 'CHK_1',
          title: 'Check 1',
          status: 'pending',
          passed: false,
          score: 0,
          min_score: 1,
          duration_ms: 0,
        },
      ],
      logs: [],
    };

    state.reconcileExecution(staleSnapshot);

    // CHK_1 should remain passed
    expect(state.execution?.assertions.find((a) => a.code === 'CHK_1')?.status).toBe('passed');
    expect(state.passedAssertions).toBe(1);
  });
});

