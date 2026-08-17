import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { EventStreamManager } from '../lib/api/events';
import type { AppStateResponse, AssertionProgressEventData, LogEventData } from '../lib/api/types';

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  onopen: ((ev: Event) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  readyState = 0;
  listeners: Record<string, ((ev: MessageEvent) => void)[]> = {};

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: (ev: MessageEvent) => void): void {
    if (!this.listeners[type]) this.listeners[type] = [];
    this.listeners[type].push(listener);
  }

  removeEventListener(type: string, listener: (ev: MessageEvent) => void): void {
    if (this.listeners[type]) {
      this.listeners[type] = this.listeners[type].filter((l) => l !== listener);
    }
  }

  close(): void {
    this.readyState = 2; // CLOSED
  }

  // Helper to simulate incoming SSE event from backend
  emit(type: string, data: unknown, lastEventId = ''): void {
    const event = {
      type,
      data: JSON.stringify(data),
      lastEventId,
    } as MessageEvent;

    if (this.listeners[type]) {
      this.listeners[type].forEach((l) => l(event));
    }
  }

  // Helper to simulate connection established
  open(): void {
    this.readyState = 1; // OPEN
    if (this.onopen) this.onopen(new Event('open'));
  }

  // Helper to simulate connection error / drop
  fail(): void {
    this.readyState = 2; // CLOSED
    if (this.onerror) this.onerror(new Event('error'));
  }
}

describe('EventStreamManager', () => {
  beforeEach(() => {
    MockEventSource.instances = [];
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('should initialize and connect with expected URL and query token', () => {
    const manager = new EventStreamManager({
      baseUrl: 'http://localhost:8080',
      token: 'tok-123',
      eventSourceFactory: (url) => new MockEventSource(url) as unknown as EventSource,
    });

    expect(manager.getStatus()).toBe('disconnected');
    manager.connect();

    expect(MockEventSource.instances.length).toBe(1);
    const es = MockEventSource.instances[0];
    expect(es.url).toBe('http://localhost:8080/api/events?token=tok-123');
    expect(manager.getStatus()).toBe('connecting');

    es.open();
    expect(manager.getStatus()).toBe('connected');

    manager.disconnect();
    expect(manager.getStatus()).toBe('disconnected');
    expect(es.readyState).toBe(2);
  });

  it('should dispatch typed events to specific listeners', () => {
    const manager = new EventStreamManager({
      eventSourceFactory: (url) => new MockEventSource(url) as unknown as EventSource,
    });

    const stateChangeSpy = vi.fn();
    const assertionProgressSpy = vi.fn();
    const logSpy = vi.fn();
    const anySpy = vi.fn();

    manager.on('state_change', stateChangeSpy);
    manager.on('assertion_progress', assertionProgressSpy);
    manager.on('log', logSpy);
    manager.onAny(anySpy);

    manager.connect();
    const es = MockEventSource.instances[0];
    es.open();

    // 1. Emit state_change
    const mockState: AppStateResponse = {
      status: 'running',
      errors: [],
      active_run_id: 'run-1',
      report_destination: { folder_source: 'default', https_source: 'off' },
    };
    es.emit('state_change', mockState, '1');

    expect(stateChangeSpy).toHaveBeenCalledWith(mockState, 1);
    expect(anySpy).toHaveBeenCalledWith({
      id: 1,
      type: 'state_change',
      data: mockState,
    });
    expect(manager.getLastEventId()).toBe(1);

    // 2. Emit assertion_progress
    const mockProgress: AssertionProgressEventData = {
      run_id: 'run-1',
      code: 'SEC_01',
      status: 'passed',
      passed: true,
      score: 1,
      min_score: 1,
      duration_ms: 250,
    };
    es.emit('assertion_progress', mockProgress, '2');
    expect(assertionProgressSpy).toHaveBeenCalledWith(mockProgress, 2);
    expect(manager.getLastEventId()).toBe(2);

    // 3. Emit log
    const mockLog: LogEventData = {
      run_id: 'run-1',
      message: 'Running assertion SEC_01',
    };
    es.emit('log', mockLog, '3');
    expect(logSpy).toHaveBeenCalledWith(mockLog, 3);
    expect(manager.getLastEventId()).toBe(3);

    manager.disconnect();
  });

  it('should enforce monotonic event deduplication', () => {
    const manager = new EventStreamManager({
      eventSourceFactory: (url) => new MockEventSource(url) as unknown as EventSource,
    });

    const logSpy = vi.fn();
    manager.on('log', logSpy);

    manager.connect();
    const es = MockEventSource.instances[0];
    es.open();

    // Send event 10
    es.emit('log', { run_id: 'r1', message: 'msg 10' }, '10');
    expect(logSpy).toHaveBeenCalledTimes(1);

    // Send duplicate event 10 -> must be skipped
    es.emit('log', { run_id: 'r1', message: 'msg 10 again' }, '10');
    expect(logSpy).toHaveBeenCalledTimes(1);

    // Send older event 8 -> must be skipped
    es.emit('log', { run_id: 'r1', message: 'msg 8' }, '8');
    expect(logSpy).toHaveBeenCalledTimes(1);

    // Send newer event 11 -> must be processed
    es.emit('log', { run_id: 'r1', message: 'msg 11' }, '11');
    expect(logSpy).toHaveBeenCalledTimes(2);

    manager.disconnect();
  });

  it('setLastEventId: should set high-water mark and skip older events', () => {
    const manager = new EventStreamManager({
      eventSourceFactory: (url) => new MockEventSource(url) as unknown as EventSource,
    });

    const logSpy = vi.fn();
    manager.on('log', logSpy);

    manager.setLastEventId(100);
    expect(manager.getLastEventId()).toBe(100);

    manager.connect();
    const es = MockEventSource.instances[0];
    es.open();

    // Event id 50 <= 100 -> dropped
    es.emit('log', { run_id: 'r1', message: 'old msg' }, '50');
    expect(logSpy).not.toHaveBeenCalled();

    // Event id 101 > 100 -> accepted
    es.emit('log', { run_id: 'r1', message: 'new msg' }, '101');
    expect(logSpy).toHaveBeenCalledTimes(1);
    expect(manager.getLastEventId()).toBe(101);

    manager.disconnect();
  });

  it('should reconnect on connection drop with backoff and trigger onReconnected', async () => {
    const manager = new EventStreamManager({
      initialBackoffMs: 1000,
      maxBackoffMs: 8000,
      backoffMultiplier: 2,
      eventSourceFactory: (url) => new MockEventSource(url) as unknown as EventSource,
    });

    const statusSpy = vi.fn();
    const reconnectedSpy = vi.fn();

    manager.onStatusChange(statusSpy);
    manager.onReconnected(reconnectedSpy);

    manager.connect();
    expect(MockEventSource.instances.length).toBe(1);
    const es1 = MockEventSource.instances[0];
    es1.open();
    expect(manager.getStatus()).toBe('connected');

    // Simulate connection drop
    es1.fail();
    expect(manager.getStatus()).toBe('reconnecting');
    expect(MockEventSource.instances.length).toBe(1);

    // Fast-forward backoff time (~1000ms + jitter)
    vi.advanceTimersByTime(1200);

    // Reconnection attempt #1
    expect(MockEventSource.instances.length).toBe(2);
    const es2 = MockEventSource.instances[1];
    expect(manager.getStatus()).toBe('reconnecting');

    // Reconnection succeeds!
    es2.open();
    expect(manager.getStatus()).toBe('connected');
    expect(reconnectedSpy).toHaveBeenCalledTimes(1);

    manager.disconnect();
    expect(manager.getStatus()).toBe('disconnected');
  });

  it('disconnect: should cancel scheduled reconnect timers', () => {
    const manager = new EventStreamManager({
      initialBackoffMs: 5000,
      eventSourceFactory: (url) => new MockEventSource(url) as unknown as EventSource,
    });

    manager.connect();
    const es1 = MockEventSource.instances[0];
    es1.open();

    // Drop connection -> status reconnecting
    es1.fail();
    expect(manager.getStatus()).toBe('reconnecting');

    // Disconnect explicitly
    manager.disconnect();
    expect(manager.getStatus()).toBe('disconnected');

    // Advance time past reconnect delay
    vi.advanceTimersByTime(6000);

    // No new instance should have been spawned
    expect(MockEventSource.instances.length).toBe(1);
  });
});
