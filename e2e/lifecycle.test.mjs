import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import net from 'node:net'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { randomBytes } from 'node:crypto'
import { fileURLToPath } from 'node:url'
import { E2E_RUN_DIRECTORY_PREFIX } from './environment.mjs'

const E2E_ROOT = path.dirname(fileURLToPath(import.meta.url))
const LIFECYCLE = path.join(E2E_ROOT, 'lifecycle.mjs')
const BINARY = path.resolve(E2E_ROOT, '..', 'ttrpg')

function runRoots() {
  return fs.readdirSync(os.tmpdir())
    .filter((name) => name.startsWith(E2E_RUN_DIRECTORY_PREFIX))
    .sort()
}

function unusedPort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      if (!address || typeof address === 'string') {
        server.close(() => reject(new Error('could not allocate test port')))
        return
      }
      server.close((error) => error ? reject(error) : resolve(address.port))
    })
  })
}

function runLifecycle(environment) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [LIFECYCLE], {
      env: { ...process.env, ...environment },
      stdio: ['ignore', 'pipe', 'pipe'],
    })
    let output = ''
    child.stdout.on('data', (chunk) => { output += chunk })
    child.stderr.on('data', (chunk) => { output += chunk })
    const timeout = setTimeout(() => {
      child.kill('SIGKILL')
      reject(new Error(`lifecycle test timed out: ${output}`))
    }, 30_000)
    child.once('error', reject)
    child.once('exit', (code, signal) => {
      clearTimeout(timeout)
      resolve({ code, output, signal })
    })
  })
}

function listen(port) {
  return new Promise((resolve, reject) => {
    const server = net.createServer((socket) => socket.destroy())
    server.once('error', reject)
    server.listen(port, '127.0.0.1', () => resolve(server))
  })
}

function close(server) {
  return new Promise((resolve, reject) => {
    server.close((error) => error ? reject(error) : resolve())
  })
}

async function expectPortClosed(port) {
  await assert.rejects(new Promise((resolve, reject) => {
    const socket = net.connect(port, '127.0.0.1')
    socket.once('connect', () => {
      socket.destroy()
      resolve()
    })
    socket.once('error', reject)
  }))
}

test('external E2E lifecycle cleans startup, pass, and test-failure paths', async () => {
  assert.equal(fs.existsSync(BINARY), true, 'run `make build` before lifecycle tests')
  const fixtureRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'inkandbone-lifecycle-fixture-'))
  const unrelated = path.join(
    os.tmpdir(),
    `${E2E_RUN_DIRECTORY_PREFIX}unrelated-${randomBytes(6).toString('hex')}`,
  )
  fs.mkdirSync(unrelated)
  fs.writeFileSync(path.join(unrelated, 'keep.txt'), 'do not remove')
  const runner = path.join(fixtureRoot, 'fake-runner.mjs')
  fs.writeFileSync(runner, 'process.exit(Number(process.env.INKANDBONE_E2E_FAKE_EXIT || 0))\n')

  try {
    const baseline = runRoots()

    const blockedPort = await unusedPort()
    const blocker = await listen(blockedPort)
    try {
      const startupFailure = await runLifecycle({
        INKANDBONE_E2E_BINARY: BINARY,
        INKANDBONE_E2E_PLAYWRIGHT_CLI: runner,
        INKANDBONE_E2E_PORT: String(blockedPort),
      })
      assert.notEqual(startupFailure.code, 0)
      assert.deepEqual(runRoots(), baseline)
      assert.equal(fs.readFileSync(path.join(unrelated, 'keep.txt'), 'utf8'), 'do not remove')
    } finally {
      await close(blocker)
    }

    for (const exitCode of [0, 7]) {
      const port = await unusedPort()
      const result = await runLifecycle({
        INKANDBONE_E2E_BINARY: BINARY,
        INKANDBONE_E2E_FAKE_EXIT: String(exitCode),
        INKANDBONE_E2E_PLAYWRIGHT_CLI: runner,
        INKANDBONE_E2E_PORT: String(port),
        OLLAMA_MODEL: 'hostile-lifecycle-marker',
      })
      assert.equal(result.code, exitCode)
      assert.match(result.output, /AI: disabled/)
      assert.deepEqual(runRoots(), baseline)
      assert.equal(fs.readFileSync(path.join(unrelated, 'keep.txt'), 'utf8'), 'do not remove')
      await expectPortClosed(port)
    }
  } finally {
    fs.rmSync(fixtureRoot, { force: true, recursive: true })
    fs.rmSync(unrelated, { force: true, recursive: true })
  }
})
