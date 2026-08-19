import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import ShutdownModal from '$lib/components/common/ShutdownModal.svelte';

describe('ShutdownModal', () => {
  afterEach(() => {
    cleanup();
  });

  it('renders confirmation view with action buttons when open', () => {
    render(ShutdownModal, {
      props: {
        open: true,
      },
    });

    expect(screen.getByText('Stop crobe process?')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Keep Running/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Stop Server/i })).toBeInTheDocument();
  });

  it('triggers onshutdown and closes modal when Stop Server is clicked', async () => {
    const onshutdownMock = vi.fn().mockResolvedValue(undefined);
    render(ShutdownModal, {
      props: {
        open: true,
        onshutdown: onshutdownMock,
      },
    });

    const stopButton = screen.getByRole('button', { name: /Stop Server/i });
    await fireEvent.click(stopButton);

    expect(onshutdownMock).toHaveBeenCalledTimes(1);
  });
});


