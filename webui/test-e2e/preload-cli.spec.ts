import { test, expect } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { startTestServer, RunningServer } from './helpers/server';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

test.describe('Compliance Probe Preloaded CLI E2E', () => {
  let server: RunningServer;

  test.beforeAll(async () => {
    // Start server preloading the GUI test fixture directly from CLI
    const fixturePath = path.resolve(__dirname, 'fixtures/gui-test.playbook.yaml');
    server = await startTestServer({ playbookPath: fixturePath });
  });

  test.afterAll(async () => {
    if (server) {
      await server.stop();
    }
  });

  test('preloaded playbook lands immediately in Inspection View and can be executed', async ({ page }) => {
    await page.goto(server.url);

    // Should skip LoadView and directly show InspectionView
    await expect(page.getByRole('heading', { name: 'GUI Verification Playbook' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'System Checks' })).toBeVisible();

    // Execute the preloaded playbook
    const executeBtn = page.getByRole('button', { name: /Execute Playbook/i });
    await expect(executeBtn).toBeVisible();
    await executeBtn.click();

    // Verify ResultsView reached
    await expect(page.getByText('Pass Rate').first()).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('100%').first()).toBeVisible();
    await expect(page.getByText(/Status:\s*completed/)).toBeVisible();
  });
});
