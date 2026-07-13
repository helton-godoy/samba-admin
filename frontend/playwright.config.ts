import { defineConfig } from '@playwright/test';

const browserExecutable = process.env.SAMBA_ADMIN_E2E_BROWSER_EXECUTABLE;

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  retries: process.env.CI ? 1 : 0,
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'off',
    screenshot: 'off',
    video: 'off',
    launchOptions: browserExecutable ? { executablePath: browserExecutable } : undefined
  },
  webServer: {
    command: 'npm run dev -- --host 127.0.0.1 --port 4173',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    env: { VITE_USE_MSW: 'true' }
  }
});
