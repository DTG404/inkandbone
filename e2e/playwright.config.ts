import { defineConfig } from '@playwright/test'

const BASE_URL = process.env.INKANDBONE_E2E_BASE_URL
if (!BASE_URL) throw new Error('run Playwright through `npm test` or `make e2e`')

export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
})
