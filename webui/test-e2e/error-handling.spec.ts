import { test, expect } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { startTestServer, RunningServer } from './helpers/server';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

test.describe('Compliance Probe Error Handling E2E', () => {
  let server: RunningServer;

  test.beforeAll(async () => {
    server = await startTestServer();
  });

  test.afterAll(async () => {
    if (server) {
      await server.stop();
    }
  });

  test('uploading invalid YAML shows diagnostic error banner and allows recovery', async ({ page }) => {
    await page.goto(server.url);

    await expect(page.getByText('Ingest Compliance Playbook')).toBeVisible();

    // Upload invalid playbook
    const invalidFixture = path.resolve(__dirname, 'fixtures/invalid.playbook.yaml');
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(invalidFixture);

    // Verify error banner is presented and execution is blocked
    await expect(page.getByText('PLAYBOOK_VALIDATION_FAILED').first()).toBeVisible({ timeout: 10000 });
    const executeBtn = page.getByRole('button', { name: /Execute Playbook/i });
    if (await executeBtn.count() > 0) {
      await expect(executeBtn).toBeDisabled();
    }

    // Unload the invalid playbook to return to LoadView
    const unloadBtn = page.getByRole('button', { name: /Unload Playbook/i }).first();
    await unloadBtn.click();
    await expect(page.getByText('Ingest Compliance Playbook')).toBeVisible({ timeout: 10000 });

    // Recover by uploading a valid playbook
    const validFixture = path.resolve(__dirname, 'fixtures/gui-test.playbook.yaml');
    await fileInput.setInputFiles(validFixture);

    // Verify successful load of valid playbook
    await expect(page.getByRole('heading', { name: 'GUI Verification Playbook' })).toBeVisible({ timeout: 10000 });
  });
});
