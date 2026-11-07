import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: 1,
  timeout: 45_000,
  use: {
    baseURL: 'http://127.0.0.1:18080',
    browserName: 'chromium',
    channel: 'chromium',
    viewport: { width: 1440, height: 1000 },
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'go run ./cmd/server',
    url: 'http://127.0.0.1:18080/api/health',
    reuseExistingServer: false,
    env: {
      ADMIN_TOKEN: 'e2e-only-admin-token',
      LISTEN_ADDR: '127.0.0.1:18080',
      DATA_DIR: 'data/e2e',
    },
    gracefulShutdown: { signal: 'SIGTERM', timeout: 5000 },
  },
})
