import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

export const E2E_RUN_DIRECTORY_PREFIX = 'inkandbone-playwright-'

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
  'OLLAMA_HOST',
]

export function sanitizedE2EEnvironment(overrides = {}) {
  const environment = {}
  for (const [key, value] of Object.entries(process.env)) {
    if (value !== undefined) environment[key] = value
  }
  for (const key of TTRPG_SERVER_ENVIRONMENT_KEYS) environment[key] = ''
  for (const [key, value] of Object.entries(overrides)) {
    if (value === undefined) delete environment[key]
    else environment[key] = value
  }
  return environment
}

export function createE2ERunDirectory() {
  return fs.mkdtempSync(path.join(os.tmpdir(), E2E_RUN_DIRECTORY_PREFIX))
}

function isSafeE2ERunDirectory(directory) {
  const resolved = path.resolve(directory)
  const base = path.basename(resolved)
  return path.dirname(resolved) === path.resolve(os.tmpdir()) &&
    base.startsWith(E2E_RUN_DIRECTORY_PREFIX) &&
    base.length > E2E_RUN_DIRECTORY_PREFIX.length
}

export function removeE2ERunDirectory(directory) {
  const resolved = path.resolve(directory)
  if (!isSafeE2ERunDirectory(resolved)) {
    throw new Error(`refusing to remove unsafe E2E run directory: ${resolved}`)
  }
  fs.rmSync(resolved, { force: true, recursive: true })
}
