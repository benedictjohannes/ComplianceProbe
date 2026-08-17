import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ApiClient, ApiClientError } from '../lib/api/client';
import type { AppStateResponse, PlaybookInspection, ExecutionSnapshot } from '../lib/api/types';

describe('ApiClient', () => {
  let client: ApiClient;
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    client = new ApiClient({ baseUrl: 'http://localhost:8080' });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it('should initialize with default or provided configuration', () => {
    expect(client.getBaseUrl()).toBe('http://localhost:8080');
    expect(client.getToken()).toBeNull();

    client.setToken('test-token');
    expect(client.getToken()).toBe('test-token');

    client.setBaseUrl('http://127.0.0.1:3000///');
    expect(client.getBaseUrl()).toBe('http://127.0.0.1:3000');
  });

  it('getState: should fetch /api/state and return response', async () => {
    const mockState: AppStateResponse = {
      status: 'idle',
      errors: [],
      report_destination: {
        folder_source: 'default',
        https_source: 'off',
      },
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockState,
    });

    const result = await client.getState();
    expect(result).toEqual(mockState);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/state', {
      method: 'GET',
      headers: {},
    });
  });

  it('getState: should include Authorization header when token is set', async () => {
    client.setToken('secret-token');

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ status: 'loaded', errors: [], report_destination: { folder_source: 'default', https_source: 'off' } }),
    });

    await client.getState();
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/state', {
      method: 'GET',
      headers: {
        Authorization: 'Bearer secret-token',
      },
    });
  });

  it('should throw ApiClientError with code and details on HTTP error', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 422,
      statusText: 'Unprocessable Entity',
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({
        error: {
          code: 'PLAYBOOK_VALIDATION_FAILED',
          message: 'Validation failed on sections',
          detail: [{ path: 'sections[0]', code: 'REQUIRED', message: 'Title is required' }],
        },
      }),
    });

    try {
      await client.getState();
      expect.fail('Should have thrown ApiClientError');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiClientError);
      const apiErr = err as ApiClientError;
      expect(apiErr.status).toBe(422);
      expect(apiErr.code).toBe('PLAYBOOK_VALIDATION_FAILED');
      expect(apiErr.message).toBe('Validation failed on sections');
      expect(apiErr.detail).toEqual([{ path: 'sections[0]', code: 'REQUIRED', message: 'Title is required' }]);
    }
  });

  it('uploadPlaybook: should send multipart FormData with file', async () => {
    const mockBlob = new Blob(['title: Test Playbook'], { type: 'text/yaml' });
    const mockState: AppStateResponse = {
      status: 'loaded',
      errors: [],
      report_destination: { folder_source: 'default', https_source: 'off' },
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockState,
    });

    const result = await client.uploadPlaybook(mockBlob, 'custom.yaml');
    expect(result).toEqual(mockState);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/playbook/upload', {
      method: 'POST',
      headers: {},
      body: expect.any(FormData),
    });
  });

  it('loadRemotePlaybook: should send JSON payload', async () => {
    const mockState: AppStateResponse = {
      status: 'loaded',
      errors: [],
      report_destination: { folder_source: 'default', https_source: 'off' },
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockState,
    });

    const result = await client.loadRemotePlaybook({
      url: 'https://example.com/playbook.yaml',
      headers: { 'X-Auth': '123' },
    });

    expect(result).toEqual(mockState);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/playbook/remote', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        url: 'https://example.com/playbook.yaml',
        headers: { 'X-Auth': '123' },
      }),
    });
  });

  it('getPlaybook: should fetch and return PlaybookInspection', async () => {
    const mockPlaybook: PlaybookInspection = {
      title: 'Compliance Check',
      sections: [],
      requiresElevation: false,
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockPlaybook,
    });

    const result = await client.getPlaybook();
    expect(result).toEqual(mockPlaybook);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/playbook', {
      method: 'GET',
      headers: {},
    });
  });

  it('deletePlaybook: should dispatch DELETE /api/playbook', async () => {
    const mockState: AppStateResponse = {
      status: 'idle',
      errors: [],
      report_destination: { folder_source: 'default', https_source: 'off' },
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockState,
    });

    const result = await client.deletePlaybook();
    expect(result).toEqual(mockState);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/playbook', {
      method: 'DELETE',
      headers: {},
    });
  });

  it('updateDestination: should send PUT payload to /api/report/destination', async () => {
    const mockState: AppStateResponse = {
      status: 'loaded',
      errors: [],
      report_destination: { folder: '/custom/dir', folder_source: 'custom', https_source: 'off' },
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockState,
    });

    const result = await client.updateDestination({ folder: '/custom/dir', folder_source: 'custom' });
    expect(result).toEqual(mockState);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/report/destination', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ folder: '/custom/dir', folder_source: 'custom' }),
    });
  });

  it('startRun and cancelExecution: should trigger respective endpoints', async () => {
    const mockRunningState: AppStateResponse = {
      status: 'running',
      errors: [],
      active_run_id: 'run-123',
      report_destination: { folder_source: 'default', https_source: 'off' },
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockRunningState,
    });

    const runResult = await client.startRun();
    expect(runResult.status).toBe('running');
    expect(runResult.active_run_id).toBe('run-123');

    const cancelResult = await client.cancelExecution();
    expect(cancelResult.status).toBe('running');
  });

  it('getExecution: should fetch ExecutionSnapshot', async () => {
    const mockSnapshot: ExecutionSnapshot = {
      run_id: 'run-123',
      status: 'running',
      last_event_id: 42,
      duration_ms: 1250,
      assertions: [
        {
          code: 'AUTH_01',
          title: 'Authentication verification',
          status: 'passed',
          passed: true,
          score: 1,
          min_score: 1,
          duration_ms: 150,
        },
      ],
      logs: ['[INFO] starting assertion AUTH_01'],
    };

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockSnapshot,
    });

    const result = await client.getExecution();
    expect(result).toEqual(mockSnapshot);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/execution', {
      method: 'GET',
      headers: {},
    });
  });

  it('getReportMarkdown and getReportLog: should fetch text content', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'text/markdown' }),
      text: async () => '# Markdown Report',
    });

    const md = await client.getReportMarkdown(true);
    expect(md).toBe('# Markdown Report');
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/report/md?download=1', {
      method: 'GET',
      headers: {},
    });

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'text/plain' }),
      text: async () => 'Log content...',
    });

    const log = await client.getReportLog();
    expect(log).toBe('Log content...');
  });

  it('getDownloadUrl: should format archive URLs correctly', () => {
    client.setToken('tok-456');
    const url = client.getDownloadUrl('tar.gz');
    expect(url).toBe('http://localhost:8080/api/report/download?format=tar.gz&token=tok-456');
  });

  it('downloadReportArchive: should fetch blob', async () => {
    const mockBlob = new Blob(['zip data'], { type: 'application/zip' });
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      headers: new Headers({ 'content-type': 'application/zip' }),
      blob: async () => mockBlob,
    });

    const result = await client.downloadReportArchive('zip');
    expect(result).toEqual(mockBlob);
    expect(globalThis.fetch).toHaveBeenCalledWith('http://localhost:8080/api/report/download?format=zip', {
      method: 'GET',
      headers: {},
    });
  });

  it('submitRemoteReport and shutdown: should dispatch POST requests', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ status: 'completed', errors: [], report_destination: { folder_source: 'default', https_source: 'custom' } }),
    }).mockResolvedValueOnce({
      ok: true,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ status: 'shutting_down' }),
    });

    const submitRes = await client.submitRemoteReport();
    expect(submitRes.status).toBe('completed');

    const shutdownRes = await client.shutdown();
    expect(shutdownRes.status).toBe('shutting_down');
  });
});
