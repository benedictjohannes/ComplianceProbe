import type {
  SSEEventType,
  SSEEvent,
  AppStateResponse,
  AssertionProgressEventData,
  LogEventData,
  ExecutionCompletedEventData,
  ExecutionCancelledEventData,
  ConnectionStatus,
} from './types';

export interface EventStreamConfig {
  baseUrl?: string;
  token?: string | null;
  initialBackoffMs?: number;
  maxBackoffMs?: number;
  backoffMultiplier?: number;
  eventSourceFactory?: (url: string) => EventSource;
}

export type TypedSSEEventMap = {
  state_change: AppStateResponse;
  assertion_progress: AssertionProgressEventData;
  log: LogEventData;
  execution_completed: ExecutionCompletedEventData;
  execution_cancelled: ExecutionCancelledEventData;
};

type EventCallback<K extends keyof TypedSSEEventMap> = (data: TypedSSEEventMap[K], eventId: number) => void;
type AnyEventCallback = (event: SSEEvent) => void;
type StatusCallback = (status: ConnectionStatus) => void;
type ReconnectedCallback = () => void;

export class EventStreamManager {
  private baseUrl: string;
  private token: string | null;
  private initialBackoffMs: number;
  private maxBackoffMs: number;
  private backoffMultiplier: number;
  private eventSourceFactory: (url: string) => EventSource;

  private eventSource: EventSource | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private currentBackoffMs: number;
  private isManuallyClosed = false;
  private lastEventId = 0;

  private status: ConnectionStatus = 'disconnected';
  private listeners: { [K in keyof TypedSSEEventMap]?: Set<EventCallback<K>> } = {};
  private anyListeners = new Set<AnyEventCallback>();
  private statusListeners = new Set<StatusCallback>();
  private reconnectedListeners = new Set<ReconnectedCallback>();

  constructor(config: EventStreamConfig = {}) {
    this.baseUrl = config.baseUrl?.replace(/\/+$/, '') ?? '';
    this.token = config.token ?? null;
    this.initialBackoffMs = config.initialBackoffMs ?? 1000;
    this.maxBackoffMs = config.maxBackoffMs ?? 15000;
    this.backoffMultiplier = config.backoffMultiplier ?? 1.5;
    this.currentBackoffMs = this.initialBackoffMs;

    this.eventSourceFactory =
      config.eventSourceFactory ??
      ((url: string) => {
        if (typeof EventSource === 'undefined') {
          throw new Error('EventSource is not supported in this environment');
        }
        return new EventSource(url);
      });
  }

  setToken(token: string | null): void {
    this.token = token;
  }

  setBaseUrl(baseUrl: string): void {
    this.baseUrl = baseUrl.replace(/\/+$/, '');
  }

  getLastEventId(): number {
    return this.lastEventId;
  }

  setLastEventId(id: number): void {
    if (id > this.lastEventId) {
      this.lastEventId = id;
    }
  }

  getStatus(): ConnectionStatus {
    return this.status;
  }

  private setStatus(newStatus: ConnectionStatus): void {
    if (this.status !== newStatus) {
      this.status = newStatus;
      this.statusListeners.forEach((cb) => {
        try {
          cb(newStatus);
        } catch (e) {
          console.error('[SSE] Error in status listener:', e);
        }
      });
    }
  }

  private buildUrl(): string {
    const cleanPath = '/api/events';
    const urlStr = `${this.baseUrl}${cleanPath}`;
    const url = new URL(urlStr, typeof window !== 'undefined' ? window.location.origin : 'http://localhost');

    if (this.token) {
      url.searchParams.set('token', this.token);
    }

    if (!this.baseUrl && typeof window !== 'undefined') {
      return `${url.pathname}${url.search}`;
    }

    return url.toString();
  }

  connect(): void {
    this.isManuallyClosed = false;
    this.clearReconnectTimer();

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    const wasReconnecting = this.status === 'reconnecting';
    this.setStatus(wasReconnecting ? 'reconnecting' : 'connecting');

    const url = this.buildUrl();

    try {
      this.eventSource = this.eventSourceFactory(url);
    } catch (err) {
      console.error('[SSE] Failed to initialize EventSource:', err);
      this.handleConnectionFailure();
      return;
    }

    this.eventSource.onopen = () => {
      const wasReconnectingOnOpen = this.status === 'reconnecting';
      this.currentBackoffMs = this.initialBackoffMs;
      this.setStatus('connected');

      if (wasReconnectingOnOpen) {
        this.reconnectedListeners.forEach((cb) => {
          try {
            cb();
          } catch (e) {
            console.error('[SSE] Error in reconnected listener:', e);
          }
        });
      }
    };

    this.eventSource.onerror = () => {
      if (this.isManuallyClosed) return;
      this.handleConnectionFailure();
    };

    // Register handlers for typed SSE events
    const eventTypes: SSEEventType[] = [
      'state_change',
      'assertion_progress',
      'log',
      'execution_completed',
      'execution_cancelled',
    ];

    for (const type of eventTypes) {
      this.eventSource.addEventListener(type, (event: MessageEvent) => {
        this.handleMessage(type, event);
      });
    }
  }

  private handleMessage(type: SSEEventType, event: MessageEvent): void {
    let eventId = 0;
    if (event.lastEventId) {
      const parsed = parseInt(event.lastEventId, 10);
      if (!isNaN(parsed)) eventId = parsed;
    }

    let parsedData: unknown;
    try {
      parsedData = JSON.parse(event.data);
    } catch (err) {
      console.warn(`[SSE] Failed to parse JSON for event type "${type}":`, err, event.data);
      return;
    }

    // Monotonic Event ID Deduplication check:
    // If event has an ID that is <= lastEventId and lastEventId is non-zero, discard duplicate.
    if (eventId > 0 && eventId <= this.lastEventId) {
      return;
    }

    if (eventId > this.lastEventId) {
      this.lastEventId = eventId;
    }

    const sseEvent: SSEEvent = {
      id: eventId,
      type,
      data: parsedData,
    };

    // Dispatch to typed listeners
    const specificListeners = this.listeners[type as keyof TypedSSEEventMap];
    if (specificListeners) {
      specificListeners.forEach((cb) => {
        try {
          (cb as EventCallback<keyof TypedSSEEventMap>)(parsedData as never, eventId);
        } catch (e) {
          console.error(`[SSE] Error in listener for "${type}":`, e);
        }
      });
    }

    // Dispatch to wildcard anyListeners
    this.anyListeners.forEach((cb) => {
      try {
        cb(sseEvent);
      } catch (e) {
        console.error('[SSE] Error in any listener:', e);
      }
    });
  }

  private handleConnectionFailure(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    if (this.isManuallyClosed) {
      this.setStatus('disconnected');
      return;
    }

    this.setStatus('reconnecting');
    this.scheduleReconnect();
  }

  private scheduleReconnect(): void {
    this.clearReconnectTimer();

    // Add jitter: +/- 10%
    const jitter = (Math.random() * 0.2 - 0.1) * this.currentBackoffMs;
    const delay = Math.max(100, Math.floor(this.currentBackoffMs + jitter));

    this.reconnectTimer = setTimeout(() => {
      if (!this.isManuallyClosed) {
        this.connect();
      }
    }, delay);

    // Increase backoff for next attempt
    this.currentBackoffMs = Math.min(
      this.maxBackoffMs,
      Math.floor(this.currentBackoffMs * this.backoffMultiplier)
    );
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  disconnect(): void {
    this.isManuallyClosed = true;
    this.clearReconnectTimer();
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.setStatus('disconnected');
  }

  // --- Listener Subscriptions ---

  on<K extends keyof TypedSSEEventMap>(type: K, callback: EventCallback<K>): () => void {
    if (!this.listeners[type]) {
      this.listeners[type] = new Set() as never;
    }
    const set = this.listeners[type] as Set<EventCallback<K>>;
    set.add(callback);

    return () => {
      set.delete(callback);
    };
  }

  onAny(callback: AnyEventCallback): () => void {
    this.anyListeners.add(callback);
    return () => {
      this.anyListeners.delete(callback);
    };
  }

  onStatusChange(callback: StatusCallback): () => void {
    this.statusListeners.add(callback);
    // Immediately emit current status
    callback(this.status);
    return () => {
      this.statusListeners.delete(callback);
    };
  }

  onReconnected(callback: ReconnectedCallback): () => void {
    this.reconnectedListeners.add(callback);
    return () => {
      this.reconnectedListeners.delete(callback);
    };
  }
}

export const eventStream = new EventStreamManager();
