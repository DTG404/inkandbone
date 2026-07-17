#!/usr/bin/env node

import { spawn } from 'node:child_process'
import http from 'node:http'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import {
  createE2ERunDirectory,
  removeE2ERunDirectory,
  sanitizedE2EEnvironment,
} from './environment.mjs'

const E2E_ROOT = path.dirname(fileURLToPath(import.meta.url))
const DEFAULT_BINARY = path.resolve(E2E_ROOT, '..', 'ttrpg')
const DEFAULT_PLAYWRIGHT_CLI = path.join(E2E_ROOT, 'node_modules', '@playwright', 'test', 'cli.js')
const HOST = '127.0.0.1'

let serverChild
let testChild
let receivedSignal
const childStates = new WeakMap()

function running(child) {
  return Boolean(child?.pid) && child.exitCode === null && child.signalCode === null
}

function spawnTracked(command, args, options) {
  const child = spawn(command, args, options)
  const state = { error: undefined, result: undefined }
  state.spawned = new Promise((resolve) => {
    child.once('spawn', () => resolve({}))
    child.once('error', (error) => {
      state.error = error
      resolve({ error })
    })
  })
  state.exited = new Promise((resolve) => {
    child.once('error', (error) => {
      state.error = error
      resolve({ error })
    })
    child.once('exit', (code, signal) => {
      state.result = { code, signal }
      resolve(state.result)
    })
  })
  childStates.set(child, state)
  return child
}

async function waitForSpawn(child) {
  const state = childStates.get(child)
  if (!state) throw new Error('untracked child process')
  const result = await state.spawned
  if (result.error) throw result.error
}

function parsePort() {
  const raw = process.env.INKANDBONE_E2E_PORT ?? '7432'
  const port = Number(raw)
  if (!Number.isInteger(port) || port < 1 || port > 65_535) {
    throw new Error(`invalid INKANDBONE_E2E_PORT: ${raw}`)
  }
  return port
}

async function waitForExit(child) {
  if (!child) return { code: null, signal: null }
  const state = childStates.get(child)
  if (!state) throw new Error('untracked child process')
  const result = await state.exited
  if (result.error) throw result.error
  return result
}

async function terminateAndReap(child) {
  if (!child) return
  const exited = waitForExit(child).catch(() => ({ code: child.exitCode, signal: child.signalCode }))
  if (!running(child)) {
    await exited
    return
  }
  child.kill('SIGTERM')
  let timeout
  const graceful = await Promise.race([
    exited.then(() => true),
    new Promise((resolve) => {
      timeout = setTimeout(() => resolve(false), 1_000)
    }),
  ]).finally(() => clearTimeout(timeout))
  if (!graceful && running(child)) {
    child.kill('SIGKILL')
    await exited
  }
}

function probe(url) {
  return new Promise((resolve, reject) => {
    const request = http.get(url, { timeout: 1_000 }, (response) => {
      response.resume()
      resolve(response.statusCode ?? 0)
    })
    request.once('timeout', () => request.destroy(new Error('health check timed out')))
    request.once('error', reject)
  })
}

async function waitUntilHealthy(child, origin, output) {
  const deadline = Date.now() + 15_000
  while (Date.now() < deadline) {
    const state = childStates.get(child)
    if (state?.error) throw state.error
    if (!running(child)) {
      throw new Error(`server exited before readiness: ${output()}`)
    }
    try {
      if (await probe(new URL('/api/health', origin)) === 200) return
    } catch {
      // The child may still be migrating its disposable database.
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  throw new Error(`server readiness timed out: ${output()}`)
}

function signalExitCode(signal) {
  return signal === 'SIGINT' ? 130 : 143
}

function handleSignal(signal) {
  if (receivedSignal) return
  receivedSignal = signal
  if (running(testChild)) testChild.kill(signal)
  else if (running(serverChild)) serverChild.kill(signal)
}

async function run() {
  const port = parsePort()
  const origin = `http://${HOST}:${port}`
  const binary = process.env.INKANDBONE_E2E_BINARY || DEFAULT_BINARY
  const playwrightCLI = process.env.INKANDBONE_E2E_PLAYWRIGHT_CLI || DEFAULT_PLAYWRIGHT_CLI
  const runDirectory = createE2ERunDirectory()
  const databasePath = path.join(runDirectory, 'smoke.db')
  let serverOutput = ''

  const handleSIGINT = () => handleSignal('SIGINT')
  const handleSIGTERM = () => handleSignal('SIGTERM')
  process.once('SIGINT', handleSIGINT)
  process.once('SIGTERM', handleSIGTERM)
  try {
    serverChild = spawnTracked(binary, ['-db', databasePath, '-listen', `${HOST}:${port}`], {
      env: sanitizedE2EEnvironment(),
      stdio: ['ignore', 'pipe', 'pipe'],
    })
    const appendServerOutput = (chunk) => {
      const text = chunk.toString()
      serverOutput = (serverOutput + text).slice(-16_384)
      process.stderr.write(`[WebServer] ${text}`)
    }
    serverChild.stdout?.on('data', appendServerOutput)
    serverChild.stderr?.on('data', appendServerOutput)
    await waitForSpawn(serverChild)
    await waitUntilHealthy(serverChild, origin, () => serverOutput)

    testChild = spawnTracked(process.execPath, [playwrightCLI, 'test', ...process.argv.slice(2)], {
      env: sanitizedE2EEnvironment({ INKANDBONE_E2E_BASE_URL: origin }),
      stdio: 'inherit',
    })
    const result = await waitForExit(testChild)
    if (receivedSignal) return signalExitCode(receivedSignal)
    return result.code ?? 1
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error)
    process.stderr.write(`E2E lifecycle failed: ${detail}\n`)
    return receivedSignal ? signalExitCode(receivedSignal) : 1
  } finally {
    await terminateAndReap(testChild)
    await terminateAndReap(serverChild)
    removeE2ERunDirectory(runDirectory)
    process.removeListener('SIGINT', handleSIGINT)
    process.removeListener('SIGTERM', handleSIGTERM)
  }
}

process.exitCode = await run()
