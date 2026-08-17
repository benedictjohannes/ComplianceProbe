import type {
  AppStateResponse,
  PlaybookInspection,
  ExecutionSnapshot,
  DestinationUpdateRequest,
  RemotePlaybookRequest,
  FinalReport,
  ArchiveFormat,
  ShutdownResponse,
  AppError,
} from './types';

export class ApiClientError extends Error {
  readonly status: number;
  readonly code: string;
  readonly detail?: unknown;
  readonly rawResponse?: unknown;

  constructor(message: string, status: number, code = 'INTERNAL_ERROR', detail?: unknown, rawResponse?: unknown) {
    super(message);
    this.name = 'ApiClientError';
    this.status = status;
    this.code = code;
    this.detail = detail;
    this.rawResponse = rawResponse;
    Object.setPrototypeOf(this, ApiClientError.prototype);
  }
}

export interface ApiClientConfig {
  baseUrl?: string;
  token?: string | null;
}

export class ApiClient {
  private baseUrl: string;
  private token: string | null;

  constructor(config: ApiClientConfig = {}) {
    this.baseUrl = config.baseUrl?.replace(/\/+$/, '') ?? '';
    this.token = config.token ?? null;
  }

  setToken(token: string | null): void {
    this.token = token;
  }

  getToken(): string | null {
    return this.token;
  }

  setBaseUrl(baseUrl: string): void {
    this.baseUrl = baseUrl.replace(/\/+$/, '');
  }

  getBaseUrl(): string {
    return this.baseUrl;
  }

  private buildUrl(path: string, queryParams?: Record<string, string | number | boolean | undefined>): string {
    const cleanPath = path.startsWith('/') ? path : `/${path}`;
    const urlStr = `${this.baseUrl}${cleanPath}`;
    const url = new URL(urlStr, typeof window !== 'undefined' ? window.location.origin : 'http://localhost');

    if (queryParams) {
      for (const [key, value] of Object.entries(queryParams)) {
        if (value !== undefined) {
          url.searchParams.set(key, String(value));
        }
      }
    }

    // In browser relative mode, return pathname + search
    if (!this.baseUrl && typeof window !== 'undefined') {
      return `${url.pathname}${url.search}`;
    }

    return url.toString();
  }

  private getAuthHeaders(): Record<string, string> {
    const headers: Record<string, string> = {};
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }
    return headers;
  }

  private async handleResponse<T>(res: Response, returnType: 'json' | 'text' | 'blob' = 'json'): Promise<T> {
    if (!res.ok) {
      let errorCode = 'INTERNAL_ERROR';
      let errorMessage = `HTTP Error ${res.status}: ${res.statusText}`;
      let errorDetail: unknown = undefined;
      let rawJson: unknown = undefined;

      const contentType = res.headers.get('content-type') || '';
      if (contentType.includes('application/json')) {
        try {
          rawJson = await res.json();
          const jsonObj = rawJson as { error?: AppError; code?: string; message?: string; detail?: unknown };
          if (jsonObj.error) {
            errorCode = jsonObj.error.code || errorCode;
            errorMessage = jsonObj.error.message || errorMessage;
            errorDetail = jsonObj.error.detail;
          } else if (jsonObj.code || jsonObj.message) {
            errorCode = jsonObj.code || errorCode;
            errorMessage = jsonObj.message || errorMessage;
            errorDetail = jsonObj.detail;
          }
        } catch {
          // Ignored fallback
        }
      } else {
        try {
          const text = await res.text();
          if (text) errorMessage = text;
        } catch {
          // Ignored fallback
        }
      }

      throw new ApiClientError(errorMessage, res.status, errorCode, errorDetail, rawJson);
    }

    if (returnType === 'text') {
      return (await res.text()) as unknown as T;
    }
    if (returnType === 'blob') {
      return (await res.blob()) as unknown as T;
    }
    return (await res.json()) as T;
  }

  // --- API Endpoints ---

  /** GET /api/state: Get current state and destination config */
  async getState(): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/state');
    const res = await fetch(url, {
      method: 'GET',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** POST /api/playbook/upload: Upload playbook file (YAML/JSON) */
  async uploadPlaybook(file: File | Blob, filename = 'playbook.yaml'): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/playbook/upload');
    const formData = new FormData();
    formData.append('file', file, filename);

    const res = await fetch(url, {
      method: 'POST',
      headers: {
        ...this.getAuthHeaders(),
      },
      body: formData,
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** POST /api/playbook/remote: Load playbook from remote URL */
  async loadRemotePlaybook(req: RemotePlaybookRequest): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/playbook/remote');
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeaders(),
      },
      body: JSON.stringify(req),
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** GET /api/playbook: Get parsed and inspected playbook */
  async getPlaybook(): Promise<PlaybookInspection> {
    const url = this.buildUrl('/api/playbook');
    const res = await fetch(url, {
      method: 'GET',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<PlaybookInspection>(res, 'json');
  }

  /** DELETE /api/playbook: Unload current playbook */
  async deletePlaybook(): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/playbook');
    const res = await fetch(url, {
      method: 'DELETE',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** PUT /api/report/destination: Update reporting folder or HTTPS destinations */
  async updateDestination(req: DestinationUpdateRequest): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/report/destination');
    const res = await fetch(url, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        ...this.getAuthHeaders(),
      },
      body: JSON.stringify(req),
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** POST /api/run: Start executing loaded playbook */
  async startRun(): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/run');
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** POST /api/execution/cancel: Cancel active execution */
  async cancelExecution(): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/execution/cancel');
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** GET /api/execution: Get execution snapshot for state reconstruction */
  async getExecution(): Promise<ExecutionSnapshot> {
    const url = this.buildUrl('/api/execution');
    const res = await fetch(url, {
      method: 'GET',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<ExecutionSnapshot>(res, 'json');
  }

  /** GET /api/report: Get structured JSON report */
  async getReport(): Promise<FinalReport> {
    const url = this.buildUrl('/api/report');
    const res = await fetch(url, {
      method: 'GET',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<FinalReport>(res, 'json');
  }

  /** GET /api/report/md: Get Markdown report preview or trigger download */
  async getReportMarkdown(download = false): Promise<string> {
    const url = this.buildUrl('/api/report/md', download ? { download: '1' } : undefined);
    const res = await fetch(url, {
      method: 'GET',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<string>(res, 'text');
  }

  /** GET /api/report/log: Get raw execution log preview or trigger download */
  async getReportLog(download = false): Promise<string> {
    const url = this.buildUrl('/api/report/log', download ? { download: '1' } : undefined);
    const res = await fetch(url, {
      method: 'GET',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<string>(res, 'text');
  }

  /** Construct download URL for direct anchor download */
  getDownloadUrl(format?: ArchiveFormat): string {
    const params: Record<string, string> = {};
    if (format) {
      params['format'] = format;
    }
    if (this.token) {
      params['token'] = this.token;
    }
    return this.buildUrl('/api/report/download', params);
  }

  /** GET /api/report/download: Download report archive as Blob */
  async downloadReportArchive(format?: ArchiveFormat): Promise<Blob> {
    const url = this.buildUrl('/api/report/download', format ? { format } : undefined);
    const res = await fetch(url, {
      method: 'GET',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<Blob>(res, 'blob');
  }

  /** POST /api/report/remote-submit: Manually trigger remote HTTPS submission */
  async submitRemoteReport(): Promise<AppStateResponse> {
    const url = this.buildUrl('/api/report/remote-submit');
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<AppStateResponse>(res, 'json');
  }

  /** POST /api/shutdown: Request graceful server shutdown */
  async shutdown(): Promise<ShutdownResponse> {
    const url = this.buildUrl('/api/shutdown');
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        ...this.getAuthHeaders(),
      },
    });
    return this.handleResponse<ShutdownResponse>(res, 'json');
  }
}

export const apiClient = new ApiClient();
