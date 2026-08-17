import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import LogEntry from '$lib/components/common/LogStream/LogEntry.svelte';
import LogControls from '$lib/components/common/LogStream/LogControls.svelte';
import LogStream from '$lib/components/common/LogStream/LogStream.svelte';
import { appState } from '$lib/state/appState.svelte';

describe('LogEntry Component', () => {
  it('renders raw message content and line number', () => {
    const { getByText } = render(LogEntry, {
      props: {
        message: 'Executing assertion [SEC-001] Separate /tmp partition',
        lineNumber: 42,
      },
    });

    expect(getByText('42')).toBeInTheDocument();
    expect(getByText('Executing assertion [SEC-001] Separate /tmp partition')).toBeInTheDocument();
  });

  it('highlights search query matches with mark elements', () => {
    const { container } = render(LogEntry, {
      props: {
        message: 'Running command findmnt -n /dev/shm',
        lineNumber: 1,
        searchQuery: 'findmnt',
      },
    });

    const mark = container.querySelector('mark');
    expect(mark).toBeInTheDocument();
    expect(mark?.textContent).toBe('findmnt');
    expect(mark?.className).toContain('bg-amber-400');
  });

  it('applies semantic styling for error, warn, and pass keywords', () => {
    const { container: errorContainer } = render(LogEntry, {
      props: { message: '[ERROR] assertion failed' },
    });
    expect(errorContainer.innerHTML).toContain('text-rose-400');

    const { container: passContainer } = render(LogEntry, {
      props: { message: '[INFO] assertion PASSED (duration: 12ms)' },
    });
    expect(passContainer.innerHTML).toContain('text-emerald-400');
  });
});

describe('LogControls Component', () => {
  it('renders line counter and triggers auto-scroll and expand actions', async () => {
    const onToggleAutoScroll = vi.fn();
    const onToggleExpand = vi.fn();
    const onCopy = vi.fn();

    const { getByText, getByRole, getByLabelText } = render(LogControls, {
      props: {
        lineCount: 150,
        autoScroll: true,
        onToggleAutoScroll,
        searchQuery: '',
        onSearchChange: vi.fn(),
        matchCount: 0,
        currentMatchIndex: 0,
        onPrevMatch: vi.fn(),
        onNextMatch: vi.fn(),
        onCopy,
        isCopied: false,
        isMaximized: false,
        onToggleMaximize: vi.fn(),
        isExpanded: false,
        onToggleExpand,
      },
    });

    expect(getByText('Console Logs')).toBeInTheDocument();
    expect(getByText('(150 lines)')).toBeInTheDocument();

    const autoScrollBtn = getByLabelText('Toggle auto scroll');
    await fireEvent.click(autoScrollBtn);
    expect(onToggleAutoScroll).toHaveBeenCalledOnce();

    const copyBtn = getByLabelText('Copy logs');
    await fireEvent.click(copyBtn);
    expect(onCopy).toHaveBeenCalledOnce();

    const expandBtn = getByLabelText('Expand console drawer');
    await fireEvent.click(expandBtn);
    expect(onToggleExpand).toHaveBeenCalledOnce();
  });

  it('renders search bar and match steppers when expanded', async () => {
    const onSearchChange = vi.fn();
    const onPrevMatch = vi.fn();
    const onNextMatch = vi.fn();

    const { getByPlaceholderText, getByText, getByLabelText } = render(LogControls, {
      props: {
        lineCount: 50,
        autoScroll: false,
        onToggleAutoScroll: vi.fn(),
        searchQuery: 'nodev',
        onSearchChange,
        matchCount: 3,
        currentMatchIndex: 1,
        onPrevMatch,
        onNextMatch,
        onCopy: vi.fn(),
        isCopied: false,
        isMaximized: false,
        onToggleMaximize: vi.fn(),
        isExpanded: true,
        onToggleExpand: vi.fn(),
      },
    });

    const searchInput = getByPlaceholderText('Search logs...');
    expect(searchInput).toBeInTheDocument();
    expect(getByText('2/3')).toBeInTheDocument();

    await fireEvent.input(searchInput, { target: { value: 'nosuid' } });
    expect(onSearchChange).toHaveBeenCalledWith('nosuid');

    const prevBtn = getByLabelText('Previous match');
    await fireEvent.click(prevBtn);
    expect(onPrevMatch).toHaveBeenCalledOnce();

    const nextBtn = getByLabelText('Next match');
    await fireEvent.click(nextBtn);
    expect(onNextMatch).toHaveBeenCalledOnce();
  });
});

describe('LogStream Component', () => {
  beforeEach(() => {
    appState.logs = [];
  });

  it('renders collapsed drawer by default with line count', () => {
    const testLogs = ['Line 1', 'Line 2', 'Line 3'];
    const { getByText, queryByText } = render(LogStream, {
      props: {
        logs: testLogs,
        initialExpanded: false,
      },
    });

    expect(getByText('(3 lines)')).toBeInTheDocument();
    expect(queryByText('Line 1')).not.toBeInTheDocument();
  });

  it('expands to show terminal logs and handles copy', async () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: {
        writeText: writeTextMock,
      },
    });

    const testLogs = ['Step 1: Ingesting playbook', 'Step 2: Probing disk storage'];
    const { getByText, getByLabelText } = render(LogStream, {
      props: {
        logs: testLogs,
        initialExpanded: true,
      },
    });

    expect(getByText('Step 1: Ingesting playbook')).toBeInTheDocument();
    expect(getByText('Step 2: Probing disk storage')).toBeInTheDocument();

    const copyBtn = getByLabelText('Copy logs');
    await fireEvent.click(copyBtn);
    expect(writeTextMock).toHaveBeenCalledWith('Step 1: Ingesting playbook\nStep 2: Probing disk storage');
  });

  it('renders empty message when no logs exist in expanded mode', () => {
    const { getByText } = render(LogStream, {
      props: {
        logs: [],
        initialExpanded: true,
      },
    });

    expect(getByText(/No logs available yet/i)).toBeInTheDocument();
  });
});
