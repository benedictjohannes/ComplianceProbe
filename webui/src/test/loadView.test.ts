import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import Dropzone from '$lib/components/views/LoadView/Dropzone.svelte';
import RemoteUrlDialog from '$lib/components/views/LoadView/RemoteUrlDialog.svelte';
import LoadView from '$lib/components/views/LoadView/LoadView.svelte';
import { AppState } from '$lib/state/appState.svelte';
import { ApiClient } from '$lib/api/client';
import { EventStreamManager } from '$lib/api/events';

describe('Dropzone Component', () => {
  it('renders default idle state with upload text and browse button', () => {
    render(Dropzone);

    expect(screen.getByText('Drag & drop your compliance playbook')).toBeInTheDocument();
    expect(screen.getByText('Browse Files')).toBeInTheDocument();
  });

  it('triggers onfile callback on valid file input selection', async () => {
    const handleFile = vi.fn();
    const { container } = render(Dropzone, {
      props: { onfile: handleFile },
    });

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    expect(fileInput).toBeInTheDocument();

    const file = new File(['title: Test Playbook'], 'test.yaml', { type: 'application/x-yaml' });
    await fireEvent.change(fileInput, { target: { files: [file] } });

    expect(handleFile).toHaveBeenCalledWith(file);
  });

  it('shows error on invalid file extension', async () => {
    const handleError = vi.fn();
    const { container } = render(Dropzone, {
      props: { onerror: handleError },
    });

    const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
    const invalidFile = new File(['text'], 'test.txt', { type: 'text/plain' });
    await fireEvent.change(fileInput, { target: { files: [invalidFile] } });

    expect(handleError).toHaveBeenCalled();
    expect(screen.getByText(/Unsupported file type "test.txt"/)).toBeInTheDocument();
  });

  it('handles drag enter, drag over, and drag leave states', async () => {
    const { container } = render(Dropzone);
    const dropRegion = screen.getByRole('region', { name: /playbook dropzone/i });

    await fireEvent.dragEnter(dropRegion);
    expect(screen.getByText('Drop file to load playbook...')).toBeInTheDocument();

    await fireEvent.dragLeave(dropRegion);
    expect(screen.getByText('Drag & drop your compliance playbook')).toBeInTheDocument();
  });

  it('handles drop event with valid file', async () => {
    const handleFile = vi.fn();
    render(Dropzone, { props: { onfile: handleFile } });

    const dropRegion = screen.getByRole('region', { name: /playbook dropzone/i });
    const file = new File(['title: Test'], 'playbook.json', { type: 'application/json' });

    await fireEvent.drop(dropRegion, {
      dataTransfer: {
        files: [file],
      },
    });

    expect(handleFile).toHaveBeenCalledWith(file);
  });

  it('renders loading spinner and disabled state when loading', () => {
    render(Dropzone, { props: { loading: true } });

    expect(screen.getByText(/Parsing and validating playbook schema.../)).toBeInTheDocument();
    const dropRegion = screen.getByRole('region', { name: /playbook dropzone/i });
    expect(dropRegion).toHaveClass('pointer-events-none');
  });
});

describe('RemoteUrlDialog Component', () => {
  it('validates HTTPS URL input', async () => {
    const handleSubmit = vi.fn();
    render(RemoteUrlDialog, {
      props: { open: true, onsubmit: handleSubmit },
    });

    const input = screen.getByPlaceholderText('https://example.com/playbook.yaml');
    const submitBtn = screen.getByText('Fetch & Load ➜');

    // Enter invalid URL
    await fireEvent.input(input, { target: { value: 'ftp://bad-url' } });
    await fireEvent.click(submitBtn);

    expect(handleSubmit).not.toHaveBeenCalled();
    expect(screen.getByText(/Only HTTPS URLs/)).toBeInTheDocument();

    // Enter valid URL
    await fireEvent.input(input, { target: { value: 'https://sec.internal/playbook.yaml' } });
    await fireEvent.click(submitBtn);

    expect(handleSubmit).toHaveBeenCalledWith({
      url: 'https://sec.internal/playbook.yaml',
    });
  });

  it('supports adding and deleting custom request headers', async () => {
    const handleSubmit = vi.fn();
    render(RemoteUrlDialog, {
      props: { open: true, onsubmit: handleSubmit },
    });

    const input = screen.getByPlaceholderText('https://example.com/playbook.yaml');
    await fireEvent.input(input, { target: { value: 'https://sec.internal/playbook.yaml' } });

    // Open advanced headers
    const advancedToggle = screen.getByText(/Advanced: Request Headers/);
    await fireEvent.click(advancedToggle);

    // Add header
    const addHeaderBtn = screen.getByText('Add Custom Header');
    await fireEvent.click(addHeaderBtn);

    const nameInput = screen.getByPlaceholderText('Header Name (e.g. Authorization)');
    const valInput = screen.getByPlaceholderText('Value (e.g. Bearer token_...)');

    await fireEvent.input(nameInput, { target: { value: 'Authorization' } });
    await fireEvent.input(valInput, { target: { value: 'Bearer test-token' } });

    const submitBtn = screen.getByText('Fetch & Load ➜');
    await fireEvent.click(submitBtn);

    expect(handleSubmit).toHaveBeenCalledWith({
      url: 'https://sec.internal/playbook.yaml',
      headers: {
        Authorization: 'Bearer test-token',
      },
    });
  });
});

describe('LoadView Orchestrator Component', () => {
  let mockClient: ApiClient;
  let mockStream: EventStreamManager;
  let state: AppState;

  beforeEach(() => {
    mockClient = {
      uploadPlaybook: vi.fn().mockResolvedValue({
        status: 'loaded',
        errors: [],
        report_destination: { folder_source: 'default', https_source: 'off' },
      }),
      loadRemotePlaybook: vi.fn().mockResolvedValue({
        status: 'loaded',
        errors: [],
        report_destination: { folder_source: 'default', https_source: 'off' },
      }),
      getPlaybook: vi.fn().mockResolvedValue({
        title: 'Remote Loaded Playbook',
        sections: [],
        requiresElevation: false,
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
  });

  it('renders heading, dropzone, and fetch button', () => {
    render(LoadView, { props: { appStateInstance: state } });

    expect(screen.getByText('Ingest Compliance Playbook')).toBeInTheDocument();
    expect(screen.getByText('Drag & drop your compliance playbook')).toBeInTheDocument();
    expect(screen.getByText('Fetch from HTTPS URL...')).toBeInTheDocument();
  });

  it('displays errors when state has errors', () => {
    state.errors = [
      { code: 'PLAYBOOK_PARSE_FAILED', message: 'Syntax error in YAML on line 12' },
    ];

    render(LoadView, { props: { appStateInstance: state } });

    expect(screen.getByText('PLAYBOOK_PARSE_FAILED')).toBeInTheDocument();
    expect(screen.getByText('Syntax error in YAML on line 12')).toBeInTheDocument();
  });
});
