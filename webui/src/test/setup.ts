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

// Polyfill global document for bits-ui timer cleanup callbacks in jsdom worker threads
if (typeof globalThis.document === 'undefined' && typeof window !== 'undefined' && window.document) {
  globalThis.document = window.document;
}


