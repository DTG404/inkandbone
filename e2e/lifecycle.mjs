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

function running(child) {
  return child && child.exitCode === null && child.signalCode === null
}

function parsePort() {
  const raw = process.env.INKANDBONE_E2E_PORT ?? '7432'
  const port = Number(raw)
  if (!Number.isInteger(port) || port < 1 || port > 65_535) {
    throw new Error(`invalid INKANDBONE_E2E_PORT: ${raw}`)
  }
  return port
}

function waitForExit(child) {
  if (!running(child)) return Promise.resolve({ code: child.exitCode, signal: child.signalCode })
  return new Promise((resolve) => {
    child.once('exit', (code, signal) => resolve({ code, signal }))
  })
}

async function terminateAndReap(child) {
  if (!running(child)) return
  const exited = waitForExit(child)
  child.kill('SIGTERM')
  const graceful = await Promise.race([
    exited.then(() => true),
    new Promise((resolve) => setTimeout(() => resolve(false), 5_000)),
  ])
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

  process.once('SIGINT', handleSignal)
  process.once('SIGTERM', handleSignal)
  try {
    serverChild = spawn(binary, ['-db', databasePath, '-listen', `${HOST}:${port}`], {
      env: sanitizedE2EEnvironment(),
      stdio: ['ignore', 'pipe', 'pipe'],
    })
    const appendServerOutput = (chunk) => {
      const text = chunk.toString()
      serverOutput = (serverOutput + text).slice(-16_384)
      process.stderr.write(`[WebServer] ${text}`)
    }
    serverChild.stdout.on('data', appendServerOutput)
    serverChild.stderr.on('data', appendServerOutput)
    await waitUntilHealthy(serverChild, origin, () => serverOutput)

    testChild = spawn(process.execPath, [playwrightCLI, 'test', ...process.argv.slice(2)], {
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
    process.removeListener('SIGINT', handleSignal)
    process.removeListener('SIGTERM', handleSignal)
  }
}

process.exitCode = await run()
