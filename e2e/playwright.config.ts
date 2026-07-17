import { defineConfig } from '@playwright/test'
import path from 'path'
import { getOrCreateE2ERunDirectory, sanitizedE2EEnvironment } from './environment'

const BINARY = path.resolve(__dirname, '..', 'ttrpg')
const RUN_DIRECTORY = getOrCreateE2ERunDirectory()
const DB_PATH = path.join(RUN_DIRECTORY, 'smoke.db')

export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  workers: 1,
  globalTeardown: './global-teardown.ts',
  metadata: { e2eRunDirectory: RUN_DIRECTORY },
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://localhost:7432',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: `${JSON.stringify(BINARY)} -db ${JSON.stringify(DB_PATH)}`,
    env: sanitizedE2EEnvironment(),
    url: 'http://localhost:7432',
    reuseExistingServer: false,
    timeout: 15_000,
  },
})
