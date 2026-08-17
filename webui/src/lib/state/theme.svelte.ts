export type ThemeMode = 'dark' | 'light' | 'system';

class ThemeStore {
  mode = $state<ThemeMode>('system');
  systemDark = $state<boolean>(true);

  resolved = $derived.by<'dark' | 'light'>(() => {
    if (this.mode === 'system') {
      return this.systemDark ? 'dark' : 'light';
    }
    return this.mode;
  });

  isDark = $derived(this.resolved === 'dark');

  init() {
    if (typeof window === 'undefined') return;

    // Read stored preference
    const stored = localStorage.getItem('crobe_theme') as ThemeMode | null;
    if (stored && (stored === 'dark' || stored === 'light' || stored === 'system')) {
      this.mode = stored;
    }

    // Media query listener
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    this.systemDark = mediaQuery.matches;

    mediaQuery.addEventListener('change', (e) => {
      this.systemDark = e.matches;
      this.applyTheme();
    });

    this.applyTheme();
  }

  setTheme(newMode: ThemeMode) {
    this.mode = newMode;
    if (typeof window !== 'undefined') {
      localStorage.setItem('crobe_theme', newMode);
      this.applyTheme();
    }
  }

  toggle() {
    if (this.resolved === 'dark') {
      this.setTheme('light');
    } else {
      this.setTheme('dark');
    }
  }

  private applyTheme() {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    if (this.resolved === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
  }
}

export const themeStore = new ThemeStore();
