import {
  type Status,
  type AppError,
  type PlaybookInspection,
  type ExecutionSnapshot,
  type ReportDestinationState,
  type ConnectionStatus,
  type AppStateResponse,
  type AssertionProgressEventData,
  type LogEventData,
  type ExecutionCompletedEventData,
  type ExecutionCancelledEventData,
  type DestinationUpdateRequest,
  type RemotePlaybookRequest,
  type AssertionSnapshot,
} from '../api/types';
import { ApiClient, apiClient as defaultApiClient } from '../api/client';
import { EventStreamManager, eventStream as defaultEventStream } from '../api/events';

export class AppState {
  // Core reactive state
  status = $state<Status>('idle');
  errors = $state<AppError[]>([]);
  playbook = $state<PlaybookInspection | null>(null);
  execution = $state<ExecutionSnapshot | null>(null);
  lastEventId = $state<number>(0);
  reportDestination = $state<ReportDestinationState>({
    folder_source: 'default',
    https_source: 'off',
  });
  connectionStatus = $state<ConnectionStatus>('disconnected');
  isLoading = $state<boolean>(false);
  activeRunId = $state<string | undefined>(undefined);
  logs = $state<string[]>([]);
  isTerminated = $state<boolean>(false);

  // Dependencies
  private client: ApiClient;
  private stream: EventStreamManager;
  private unsubscribers: Array<() => void> = [];

  constructor(client: ApiClient = defaultApiClient, stream: EventStreamManager = defaultEventStream) {
    this.client = client;
    this.stream = stream;
  }

  // Derived state
  isIdle = $derived(this.status === 'idle');
  isLoaded = $derived(this.status === 'loaded');
  isRunning = $derived(this.status.startsWith('running'));
  isElevating = $derived(this.status === 'running.elevating');
  isCancelling = $derived(this.status === 'running.cancelling');
  isCompleted = $derived(this.status.startsWith('completed'));
  isConfirmingSubmission = $derived(this.status === 'completed.confirming_submission');
  isSubmitting = $derived(this.status === 'completed.submitting');
  isError = $derived(this.status === 'error');
  hasErrors = $derived(this.errors.length > 0);
  isConnected = $derived(this.connectionStatus === 'connected');
  isReconnecting = $derived(this.connectionStatus === 'reconnecting');

  currentPipelineStep = $derived.by(() => {
    if (this.status === 'idle') return 0;
    if (this.status === 'loaded') return 1;
    if (this.status.startsWith('running')) return 2;
    if (this.status.startsWith('completed')) return 3;
    return 0;
  });

  totalAssertions = $derived.by(() => {
    if (this.playbook && this.playbook.sections) {
      return this.playbook.sections.reduce((acc, s) => acc + (s.assertions?.length || 0), 0);
    }
    return this.execution?.assertions.length ?? 0;
  });

  passedAssertions = $derived.by(() => {
    return this.execution?.assertions.filter((a) => a.status === 'passed').length ?? 0;
  });

  failedAssertions = $derived.by(() => {
    return this.execution?.assertions.filter((a) => a.status === 'failed').length ?? 0;
  });

  completedAssertions = $derived.by(() => {
    return (
      this.execution?.assertions.filter(
        (a) => a.status === 'passed' || a.status === 'failed' || a.status === 'cancelled'
      ).length ?? 0
    );
  });

  progressPercent = $derived.by(() => {
    if (this.totalAssertions <= 0) return 0;
    return Math.min(100, Math.round((this.completedAssertions / this.totalAssertions) * 100));
  });

  // --- Lifecycle & Subscriptions ---

  init(token?: string): void {
    if (token) {
      this.client.setToken(token);
      this.stream.setToken(token);
    }

    this.destroy(); // Clear existing subscriptions if any

    // 1. Listen for connection status changes
    this.unsubscribers.push(
      this.stream.onStatusChange((status) => {
        this.connectionStatus = status;
      })
    );

    // 2. Listen for reconnection to trigger snapshot reconciliation
    this.unsubscribers.push(
      this.stream.onReconnected(() => {
        this.fullSync().catch((err) => {
          console.error('[AppState] Error during post-reconnect fullSync:', err);
        });
      })
    );

    // 3. Listen for SSE event streams
    this.unsubscribers.push(
      this.stream.on('state_change', (data, eventId) => {
        this.handleStateChange(data, eventId);
      })
    );

    this.unsubscribers.push(
      this.stream.on('assertion_progress', (data, eventId) => {
        this.handleAssertionProgress(data, eventId);
      })
    );

    this.unsubscribers.push(
      this.stream.on('log', (data, eventId) => {
        this.handleLog(data, eventId);
      })
    );

    this.unsubscribers.push(
      this.stream.on('execution_completed', (data, eventId) => {
        this.handleExecutionCompleted(data, eventId);
      })
    );

    this.unsubscribers.push(
      this.stream.on('execution_cancelled', (data, eventId) => {
        this.handleExecutionCancelled(data, eventId);
      })
    );

    // 4. Start SSE stream
    this.stream.connect();

    // 5. Initial Full Sync
    this.fullSync().catch((err) => {
      console.error('[AppState] Error during initial fullSync:', err);
    });
  }

  destroy(): void {
    this.unsubscribers.forEach((unsub) => unsub());
    this.unsubscribers = [];
    this.stream.disconnect();
  }

  // --- Snapshot Reconciliation ---

  reconcileState(resp: AppStateResponse): void {
    if (!resp || typeof resp.status !== 'string') {
      console.warn('[AppState] Received invalid state response:', resp);
      return;
    }

    this.status = resp.status;
    this.errors = resp.errors ? [...resp.errors] : [];
    if (resp.report_destination) {
      this.reportDestination = resp.report_destination;
    }
    this.activeRunId = resp.active_run_id;

    if (this.status === 'idle') {
      this.playbook = null;
      this.execution = null;
      this.logs = [];
    }
  }

  private createInitialAssertionsFromPlaybook(): AssertionSnapshot[] {
    const list: AssertionSnapshot[] = [];
    if (this.playbook && this.playbook.sections) {
      for (const section of this.playbook.sections) {
        if (section.assertions) {
          for (const a of section.assertions) {
            list.push({
              code: a.code,
              title: a.title,
              status: 'pending',
              passed: false,
              score: 0,
              min_score: 0,
              duration_ms: 0,
            });
          }
        }
      }
    }
    return list;
  }

  reconcileExecution(snapshot: ExecutionSnapshot): void {
    if (!snapshot) return;

    if (!this.execution || this.execution.run_id !== snapshot.run_id) {
      this.execution = {
        ...snapshot,
        assertions: snapshot.assertions ? [...snapshot.assertions] : [],
        logs: snapshot.logs ? [...snapshot.logs] : [],
      };
      this.logs = snapshot.logs ? [...snapshot.logs] : [];
      if (snapshot.last_event_id > this.lastEventId) {
        this.lastEventId = snapshot.last_event_id;
        this.stream.setLastEventId(snapshot.last_event_id);
      }
      return;
    }

    // Reconciling same active run:
    this.execution.status = snapshot.status;
    if (snapshot.duration_ms > 0) {
      this.execution.duration_ms = snapshot.duration_ms;
    }
    if (snapshot.active_assertion_code !== undefined) {
      this.execution.active_assertion_code = snapshot.active_assertion_code;
    }

    // Merge assertions conservatively without overwriting finished results with pending
    const incomingAssertions = snapshot.assertions || [];
    for (const inc of incomingAssertions) {
      const idx = this.execution.assertions.findIndex((a) => a.code === inc.code);
      if (idx >= 0) {
        const current = this.execution.assertions[idx];
        const currentCompleted =
          current.status === 'passed' || current.status === 'failed' || current.status === 'cancelled';
        const incCompleted =
          inc.status === 'passed' || inc.status === 'failed' || inc.status === 'cancelled';
        if (!currentCompleted || incCompleted) {
          this.execution.assertions[idx] = { ...current, ...inc };
        }
      } else {
        this.execution.assertions.push(inc);
      }
    }

    if (snapshot.logs && snapshot.logs.length >= this.logs.length) {
      this.logs = [...snapshot.logs];
      this.execution.logs = [...snapshot.logs];
    }

    if (snapshot.last_event_id > this.lastEventId) {
      this.lastEventId = snapshot.last_event_id;
      this.stream.setLastEventId(snapshot.last_event_id);
    }
  }

  async fullSync(): Promise<void> {
    this.isLoading = true;
    try {
      // 1. Fetch state
      const stateResp = await this.client.getState();
      this.reconcileState(stateResp);

      // 2. Fetch playbook if loaded or in progress
      if (this.status !== 'idle') {
        try {
          this.playbook = await this.client.getPlaybook();
        } catch (e) {
          console.warn('[AppState] Failed to fetch playbook during fullSync:', e);
        }
      } else {
        this.playbook = null;
      }

      // 3. Fetch execution snapshot if running or completed
      if (this.status.startsWith('running') || this.status.startsWith('completed')) {
        try {
          const execSnapshot = await this.client.getExecution();
          this.reconcileExecution(execSnapshot);
        } catch (e) {
          console.warn('[AppState] Failed to fetch execution snapshot during fullSync:', e);
        }
      } else {
        this.execution = null;
        this.logs = [];
      }
    } finally {
      this.isLoading = false;
    }
  }

  // --- SSE Event Handlers ---

  handleStateChange(data: AppStateResponse, eventId: number): void {
    if (!data || typeof data.status !== 'string') {
      console.warn('[AppState] Received invalid state_change event payload:', data);
      return;
    }

    if (eventId > this.lastEventId) {
      this.lastEventId = eventId;
    }

    const previousStatus = this.status;
    this.reconcileState(data);

    // If transitioned to loaded from idle/error, fetch playbook if missing
    if (this.status === 'loaded' && (!this.playbook || previousStatus !== 'loaded')) {
      this.client
        .getPlaybook()
        .then((pb) => {
          this.playbook = pb;
        })
        .catch((err) => {
          console.warn('[AppState] Could not fetch playbook after state_change:', err);
        });
    }

    // If transitioned to running, ensure execution snapshot exists for this run ID
    if (this.status.startsWith('running') && data.active_run_id) {
      if (!this.execution || this.execution.run_id !== data.active_run_id) {
        this.execution = {
          run_id: data.active_run_id,
          status: this.status,
          last_event_id: eventId,
          duration_ms: 0,
          assertions: this.createInitialAssertionsFromPlaybook(),
          logs: [],
        };
      } else {
        this.execution.status = this.status;
      }
    }
  }

  handleAssertionProgress(data: AssertionProgressEventData, eventId: number): void {
    if (eventId > this.lastEventId) {
      this.lastEventId = eventId;
    }

    // If execution snapshot is not yet created, initialize it
    if (!this.execution || (data.run_id && this.execution.run_id !== data.run_id)) {
      this.execution = {
        run_id: data.run_id,
        status: 'running',
        active_assertion_code: data.status === 'running' ? data.code : undefined,
        last_event_id: eventId,
        duration_ms: data.duration_ms || 0,
        assertions: this.createInitialAssertionsFromPlaybook(),
        logs: [],
      };
    }

    // Discard if belonging to an older/mismatched run ID
    if (this.execution.run_id && data.run_id && this.execution.run_id !== data.run_id) {
      return;
    }

    // Update active assertion code pointer
    if (data.status === 'running') {
      this.execution.active_assertion_code = data.code;
    } else if (this.execution.active_assertion_code === data.code) {
      this.execution.active_assertion_code = undefined;
    }

    // Update assertion entry in list
    const existingIndex = this.execution.assertions.findIndex((a) => a.code === data.code);
    const updatedAssertion: AssertionSnapshot = {
      code: data.code,
      title: data.code, // Placeholder title if not found
      status: data.status,
      passed: data.passed,
      score: data.score,
      min_score: data.min_score,
      duration_ms: data.duration_ms,
      output: data.output,
    };

    // If we have the playbook loaded, retain the real assertion title
    if (this.playbook) {
      for (const section of this.playbook.sections) {
        const found = section.assertions?.find((a) => a.code === data.code);
        if (found) {
          updatedAssertion.title = found.title;
          break;
        }
      }
    }

    if (existingIndex >= 0) {
      const existing = this.execution.assertions[existingIndex];
      if (existing.title && existing.title !== existing.code) {
        updatedAssertion.title = existing.title;
      }
      this.execution.assertions[existingIndex] = updatedAssertion;
    } else {
      this.execution.assertions.push(updatedAssertion);
    }
  }

  handleLog(data: LogEventData, eventId: number): void {
    if (eventId > this.lastEventId) {
      this.lastEventId = eventId;
    }

    // Discard if from mismatched run ID
    if (this.execution?.run_id && data.run_id && this.execution.run_id !== data.run_id) {
      return;
    }

    this.logs.push(data.message);
    if (this.execution) {
      this.execution.logs.push(data.message);
    }
  }

  handleExecutionCompleted(data: ExecutionCompletedEventData, eventId: number): void {
    if (eventId > this.lastEventId) {
      this.lastEventId = eventId;
    }

    if (this.execution) {
      this.execution.status = data.status;
      this.execution.duration_ms = data.duration_ms;
      this.execution.active_assertion_code = undefined;
    }
  }

  handleExecutionCancelled(data: ExecutionCancelledEventData, eventId: number): void {
    if (eventId > this.lastEventId) {
      this.lastEventId = eventId;
    }

    if (this.execution) {
      this.execution.status = 'cancelled';
      this.execution.active_assertion_code = undefined;
    }
  }

  // --- High-Level User Actions ---

  async uploadPlaybookFile(file: File): Promise<void> {
    this.isLoading = true;
    try {
      const stateResp = await this.client.uploadPlaybook(file, file.name);
      this.reconcileState(stateResp);
      if (stateResp.status === 'loaded') {
        this.playbook = await this.client.getPlaybook();
      }
    } finally {
      this.isLoading = false;
    }
  }

  async loadRemotePlaybook(req: RemotePlaybookRequest): Promise<void> {
    this.isLoading = true;
    try {
      const stateResp = await this.client.loadRemotePlaybook(req);
      this.reconcileState(stateResp);
      if (stateResp.status === 'loaded') {
        this.playbook = await this.client.getPlaybook();
      }
    } finally {
      this.isLoading = false;
    }
  }

  async unloadPlaybook(): Promise<void> {
    this.isLoading = true;
    try {
      const stateResp = await this.client.deletePlaybook();
      this.reconcileState(stateResp);
      this.playbook = null;
      this.execution = null;
      this.logs = [];
    } finally {
      this.isLoading = false;
    }
  }

  async updateDestination(req: DestinationUpdateRequest): Promise<void> {
    this.isLoading = true;
    try {
      const stateResp = await this.client.updateDestination(req);
      this.reconcileState(stateResp);
    } finally {
      this.isLoading = false;
    }
  }

  async startRun(): Promise<void> {
    this.isLoading = true;
    try {
      this.logs = [];
      const initialAssertions = this.createInitialAssertionsFromPlaybook();
      const stateResp = await this.client.startRun();
      this.reconcileState(stateResp);
      const runId = stateResp.active_run_id;
      if (runId) {
        if (!this.execution || this.execution.run_id !== runId) {
          this.execution = {
            run_id: runId,
            status: stateResp.status || 'running',
            last_event_id: this.lastEventId,
            duration_ms: 0,
            assertions: initialAssertions,
            logs: [],
          };
        } else {
          this.execution.status = stateResp.status || this.execution.status;
        }
      }
    } finally {
      this.isLoading = false;
    }
  }

  async cancelRun(): Promise<void> {
    this.isLoading = true;
    try {
      const stateResp = await this.client.cancelExecution();
      this.reconcileState(stateResp);
    } finally {
      this.isLoading = false;
    }
  }

  async submitRemoteReport(): Promise<void> {
    this.isLoading = true;
    try {
      const stateResp = await this.client.submitRemoteReport();
      this.reconcileState(stateResp);
    } finally {
      this.isLoading = false;
    }
  }

  clearErrors(): void {
    this.errors = [];
  }

  dismissError(index: number): void {
    if (index >= 0 && index < this.errors.length) {
      this.errors = this.errors.filter((_, i) => i !== index);
    }
  }

  async shutdownServer(): Promise<void> {
    this.isTerminated = true;
    try {
      await this.client.shutdown();
    } catch {
      // Server may drop connection immediately
    } finally {
      this.destroy();
    }
  }
}


export const appState = new AppState();
