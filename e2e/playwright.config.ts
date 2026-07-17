import { defineConfig } from '@playwright/test'
import path from 'path'
import { DB_PATH } from './global-setup'

const BINARY = path.resolve(__dirname, '..', 'ttrpg')

export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  workers: 1,
  globalSetup: './global-setup.ts',
  globalTeardown: './global-teardown.ts',
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://localhost:7432',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: `${BINARY} -db ${DB_PATH}`,
    url: 'http://localhost:7432',
    reuseExistingServer: false,
    timeout: 15_000,
  },
})
