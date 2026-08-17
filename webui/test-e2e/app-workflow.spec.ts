import { test, expect } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { startTestServer, RunningServer } from './helpers/server';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

test.describe('Compliance Probe GUI Workflow E2E', () => {
  let server: RunningServer;

  test.beforeAll(async () => {
    // Start clean server with no preloaded playbook
    server = await startTestServer();
  });

  test.afterAll(async () => {
    if (server) {
      await server.stop();
    }
  });

  test('complete GUI lifecycle: Upload -> Inspect -> Execute -> Scorecard & Reports', async ({ page }) => {
    // 1. Initial Load: Navigate to bootstrap URL
    await page.goto(server.url);

    // Verify LoadView is rendered
    await expect(page.locator('text=Ingest Compliance Playbook')).toBeVisible();
    await expect(page.locator('text=Upload or fetch a YAML or JSON compliance definition')).toBeVisible();

    // 2. Upload Playbook via Dropzone file input
    const fixturePath = path.resolve(__dirname, 'fixtures/gui-test.playbook.yaml');
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles(fixturePath);

    // 3. Inspection View: Verify parsed playbook metadata & assertions
    await expect(page.getByRole('heading', { name: 'GUI Verification Playbook' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'System Checks' })).toBeVisible();
    await expect(page.getByText('Operating System Check').first()).toBeVisible();
    await expect(page.getByText('Security Benchmark').first()).toBeVisible();

    // 4. Start Execution Run
    const executeBtn = page.getByRole('button', { name: /Execute Playbook/i });
    await expect(executeBtn).toBeVisible();
    await executeBtn.click();

    // 5. Execution & Results: Wait for live execution to conclude and navigate to ResultsView
    await expect(page.getByText('Pass Rate').first()).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('100%').first()).toBeVisible();
    await expect(page.getByText(/Status:\s*completed/)).toBeVisible();

    // 6. Test Report Tabs: Markdown Report & Execution Logs
    const markdownTabBtn = page.getByRole('button', { name: /Markdown Report/i });
    await markdownTabBtn.click();
    await expect(page.getByText('Compliance Report').or(page.getByText('Operating System Check').first())).toBeVisible();

    const logsTabBtn = page.getByRole('button', { name: /Execution Logs/i });
    await logsTabBtn.click();
    await expect(page.getByText('OS Check: Linux').first()).toBeVisible();

    const auditTabBtn = page.getByRole('button', { name: /Execution Audit/i });
    await auditTabBtn.click();
    await expect(page.getByText('OSCheck').first()).toBeVisible();
    await expect(page.getByText('SecurityCheck').first()).toBeVisible();

    // 7. Test Lifecycle: "Load Another Playbook" unloads state back to LoadView
    const loadAnotherBtn = page.getByRole('button', { name: /Load Another Playbook/i });
    await loadAnotherBtn.click();
    await expect(page.getByText('Ingest Compliance Playbook')).toBeVisible({ timeout: 10000 });
  });
});
