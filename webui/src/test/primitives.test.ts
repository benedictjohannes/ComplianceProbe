import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { Button, Badge, Card, Input, Switch } from '$lib/components/ui';
import ConnectionStatus from '$lib/components/common/ConnectionStatus.svelte';
import ErrorBanner from '$lib/components/common/ErrorBanner.svelte';
import Header from '$lib/components/common/Header.svelte';

describe('UI Primitives', () => {
  it('renders Badge with correct text and variant styles', () => {
    const { container } = render(Badge, {
      props: {
        variant: 'passed',
      },
    });
    expect(container.firstChild).toHaveClass('border-emerald-500/20');
  });

  it('renders Card container with precision border', () => {
    const { container } = render(Card, {
      props: {
        hover: true,
      },
    });
    expect(container.firstChild).toHaveClass('rounded-lg');
    expect(container.firstChild).toHaveClass('dark:hover:border-zinc-700');
  });

  it('renders Button with variants, disabled and onclick', async () => {
    const handleClick = vi.fn();
    const { rerender } = render(Button, {
      props: {
        variant: 'primary',
        onclick: handleClick,
      },
    });

    const btn = screen.getByRole('button');
    expect(btn).toHaveClass('bg-emerald-600');
    await fireEvent.click(btn);
    expect(handleClick).toHaveBeenCalledTimes(1);

    rerender({ disabled: true });
    expect(btn).toBeDisabled();
  });

  it('renders Input with monospace and error styling', () => {
    render(Input, {
      props: {
        value: 'test-value',
        mono: true,
        error: true,
        placeholder: 'Enter URL...',
      },
    });

    const input = screen.getByPlaceholderText('Enter URL...');
    expect(input).toHaveClass('font-mono');
    expect(input).toHaveClass('border-rose-500');
  });

  it('renders Switch component with accessible toggle role', () => {
    render(Switch, {
      props: {
        checked: true,
      },
    });
    const switchEl = screen.getByRole('switch');
    expect(switchEl).toBeInTheDocument();
    expect(switchEl).toHaveAttribute('data-state', 'checked');
  });

  it('renders ConnectionStatus in different states', () => {
    const { rerender } = render(ConnectionStatus, {
      props: { state: 'connected' },
    });
    expect(screen.getByText('Connected')).toBeInTheDocument();

    rerender({ state: 'reconnecting' });
    expect(screen.getByText('Reconnecting...')).toBeInTheDocument();

    rerender({ state: 'disconnected' });
    expect(screen.getByText('Disconnected')).toBeInTheDocument();
  });

  it('renders ErrorBanner with error code and messages', () => {
    render(ErrorBanner, {
      props: {
        code: 'PLAYBOOK_PARSE_FAILED',
        message: 'Invalid YAML syntax',
        detail: [{ path: 'root', message: 'Unexpected colon at line 4' }],
      },
    });

    expect(screen.getByText('PLAYBOOK_PARSE_FAILED')).toBeInTheDocument();
    expect(screen.getByText('Invalid YAML syntax')).toBeInTheDocument();
    expect(screen.getByText('root:')).toBeInTheDocument();
    expect(screen.getByText('Unexpected colon at line 4')).toBeInTheDocument();
  });

  it('renders Header with static pipeline steps and active playbook', () => {
    render(Header, {
      props: {
        activeStep: 2,
        playbookName: 'test-playbook.yaml',
      },
    });

    expect(screen.getAllByText('crobe').length).toBeGreaterThan(0);
    expect(screen.getAllByText('test-playbook.yaml').length).toBeGreaterThan(0);
    const activeStepNode = screen.getByText('2. Inspect').closest('li');
    expect(activeStepNode).toHaveAttribute('aria-current', 'step');
    expect(screen.getByText('Step 2 of 4:')).toBeInTheDocument();
  });
});

