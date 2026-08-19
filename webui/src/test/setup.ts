import '@testing-library/jest-dom';

// Polyfill window.matchMedia and HTMLElement helpers if not available in jsdom
if (typeof window !== 'undefined') {
  if (!window.HTMLElement.prototype.scrollIntoView) {
    window.HTMLElement.prototype.scrollIntoView = () => {};
  }
  if (!window.Element.prototype.scrollIntoView) {
    window.Element.prototype.scrollIntoView = () => {};
  }

  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  });
}

// Ensure document and window are available on globalThis for asynchronous cleanup timers (bits-ui scroll-lock)
if (typeof window !== 'undefined') {
  (globalThis as any).document = window.document;
  (globalThis as any).window = window;
}


