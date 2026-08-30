import { defineConfig, devices } from '@playwright/test';

/**
 * E2Eテスト設定。
 * - テスト対象APIはテスト用APIサーバー（api_test / http://localhost:8081）
 * - フロントエンドは .env.e2e を読み込む dev サーバー（ポート5174）
 * - マシン負荷を抑えるため worker は 1 に固定する
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  outputDir: 'test-results',
  use: {
    baseURL: 'http://localhost:5174',
    testIdAttribute: 'data-testid',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    video: 'off',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'npm run dev:e2e',
    url: 'http://localhost:5174',
    reuseExistingServer: true,
    timeout: 120_000,
  },
});
