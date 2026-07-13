import { defineConfig } from '@playwright/test';

const frontendPort = process.env.SAMBA_ADMIN_E2E_FRONTEND_PORT || '4174';
const externalEnvironment = process.env.SAMBA_ADMIN_E2E_EXTERNAL_ENV === 'true';
const externalFrontend = process.env.SAMBA_ADMIN_E2E_EXTERNAL_FRONTEND === 'true';
const browserExecutable = process.env.SAMBA_ADMIN_E2E_BROWSER_EXECUTABLE;
const externalApiURL = process.env.SAMBA_ADMIN_E2E_API_URL || 'http://127.0.0.1:18080';
if (externalEnvironment && !/^http:\/\/127\.0\.0\.1:\d{2,5}$/.test(externalApiURL)) {
  throw new Error('SAMBA_ADMIN_E2E_API_URL externo deve usar loopback HTTP e porta explícita.');
}

export default defineConfig({
  testDir: './e2e-real',
  outputDir: './test-results-real',
  timeout: 45_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report-real', open: 'never' }]
  ],
  use: {
    baseURL: `http://localhost:${frontendPort}`,
    trace: 'off',
    screenshot: 'off',
    video: 'off',
    launchOptions: browserExecutable ? { executablePath: browserExecutable } : undefined
  },
  webServer: externalFrontend ? undefined : {
    command: externalEnvironment
      ? `VITE_USE_MSW=false SAMBA_ADMIN_E2E_API_URL=${externalApiURL} npm run dev -- --config vite.e2e-real.config.ts --host 127.0.0.1 --port ${frontendPort}`
      : '../scripts/run-e2e-real-env.sh',
    url: `http://localhost:${frontendPort}`,
    timeout: 180_000,
    reuseExistingServer: false
  }
});
