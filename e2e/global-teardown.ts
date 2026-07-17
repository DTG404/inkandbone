import type { FullConfig } from '@playwright/test'
import { removeE2ERunDirectory } from './environment'

export default function globalTeardown(config: FullConfig) {
  const runDirectory = config.metadata.e2eRunDirectory
  if (typeof runDirectory !== 'string') {
    throw new Error('Playwright E2E run directory metadata is missing')
  }
  removeE2ERunDirectory(runDirectory)
}
