import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  resolve: {
    alias: {
      $lib: path.resolve(import.meta.dirname, './src/lib'),
    },
    conditions: ['browser'],
  },
  build: {
    outDir: '../internal/webui/dist',
    emptyOutDir: true,
    sourcemap: false,
    minify: 'esbuild',
  },
  // @ts-expect-error vitest config
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
});
