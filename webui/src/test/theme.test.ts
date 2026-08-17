import { describe, it, expect, beforeEach } from 'vitest';
import { themeStore } from '$lib/state/theme.svelte';

describe('ThemeStore', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.className = '';
  });

  it('initializes with default system mode and applies theme', () => {
    themeStore.init();
    expect(themeStore.mode).toBe('system');
  });

  it('switches to dark mode and sets class', () => {
    themeStore.setTheme('dark');
    expect(themeStore.mode).toBe('dark');
    expect(themeStore.resolved).toBe('dark');
    expect(themeStore.isDark).toBe(true);
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(localStorage.getItem('crobe_theme')).toBe('dark');
  });

  it('switches to light mode and removes class', () => {
    themeStore.setTheme('light');
    expect(themeStore.mode).toBe('light');
    expect(themeStore.resolved).toBe('light');
    expect(themeStore.isDark).toBe(false);
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(localStorage.getItem('crobe_theme')).toBe('light');
  });

  it('toggles correctly between dark and light', () => {
    themeStore.setTheme('dark');
    expect(themeStore.resolved).toBe('dark');
    themeStore.toggle();
    expect(themeStore.resolved).toBe('light');
    themeStore.toggle();
    expect(themeStore.resolved).toBe('dark');
  });
});
