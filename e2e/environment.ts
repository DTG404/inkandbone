import fs from 'fs'
import os from 'os'
import path from 'path'

export const E2E_RUN_DIRECTORY_PREFIX = 'inkandbone-playwright-'
export const E2E_RUN_DIRECTORY_ENV = 'INKANDBONE_E2E_RUN_DIRECTORY'

// Keep this list synchronized with every environment selector read by
// cmd/ttrpg/main.go. E2E servers must never inherit workstation credentials or
// accidentally select a real AI provider.
export const TTRPG_SERVER_ENVIRONMENT_KEYS = [
  'TTRPG_AUTH_SECRET',
  'ANTHROPIC_API_KEY',
  'DEEPSEEK_API_KEY',
  'DEEPSEEK_AUTO_MODEL',
  'OPENROUTER_API_KEY',
  'OPENROUTER_AUTO_MODEL',
  'OLLAMA_GM_MODEL',
  'OLLAMA_AI_MODEL',
  'OLLAMA_MODEL',
] as const

export function sanitizedE2EEnvironment(
  overrides: Record<string, string | undefined> = {},
): Record<string, string> {
  const environment: Record<string, string> = {}
  for (const [key, value] of Object.entries(process.env)) {
    if (value !== undefined) environment[key] = value
  }
  // Playwright merges webServer.env over the parent process environment, so
  // explicit empty values are required; merely omitting a key would still let
  // an ambient credential or model selector reach the server.
  for (const key of TTRPG_SERVER_ENVIRONMENT_KEYS) environment[key] = ''
  for (const [key, value] of Object.entries(overrides)) {
    if (value === undefined) delete environment[key]
    else environment[key] = value
  }
  return environment
}

export function createE2ERunDirectory(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), E2E_RUN_DIRECTORY_PREFIX))
}

function isSafeE2ERunDirectory(directory: string): boolean {
  const resolved = path.resolve(directory)
  const base = path.basename(resolved)
  return path.dirname(resolved) === path.resolve(os.tmpdir()) &&
    base.startsWith(E2E_RUN_DIRECTORY_PREFIX) &&
    base.length > E2E_RUN_DIRECTORY_PREFIX.length
}

// Playwright may evaluate its configuration in more than one process. Reuse
// one guarded directory across those evaluations so config loading itself
// cannot leak extra temp directories.
export function getOrCreateE2ERunDirectory(): string {
  const existing = process.env[E2E_RUN_DIRECTORY_ENV]
  if (existing && isSafeE2ERunDirectory(existing) && fs.existsSync(existing)) {
    return path.resolve(existing)
  }
  const created = createE2ERunDirectory()
  process.env[E2E_RUN_DIRECTORY_ENV] = created
  return created
}

export function removeE2ERunDirectory(directory: string): void {
  const resolved = path.resolve(directory)
  if (!isSafeE2ERunDirectory(resolved)) {
    throw new Error(`refusing to remove unsafe E2E run directory: ${resolved}`)
  }
  fs.rmSync(resolved, { force: true, recursive: true })
}
