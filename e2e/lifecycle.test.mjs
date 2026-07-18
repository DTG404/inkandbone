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
const BINARY = process.env.INKANDBONE_E2E_BINARY || path.resolve(E2E_ROOT, '..', 'ttrpg')

test('server-owning specs use the lifecycle-selected E2E binary', () => {
  for (const spec of ['reliability.spec.ts', 'security.spec.ts']) {
    const source = fs.readFileSync(path.join(E2E_ROOT, 'tests', spec), 'utf8')
    assert.match(source, /process\.env\.INKANDBONE_E2E_BINARY\s*\|\|/)
  }
})

test('visual snapshots permit only bounded cross-runner rasterization noise', () => {
  const source = fs.readFileSync(path.join(E2E_ROOT, 'tests', 'responsive.spec.ts'), 'utf8')
  assert.match(source, /maxDiffPixelRatio:\s*0\.0001/)
})

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

function startLifecycle(environment) {
  const child = spawn(process.execPath, [LIFECYCLE], {
    env: { ...process.env, ...environment },
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  const startedAt = Date.now()
  let output = ''
  child.stdout.on('data', (chunk) => { output += chunk })
  child.stderr.on('data', (chunk) => { output += chunk })
  const completed = new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      child.kill('SIGKILL')
      reject(new Error(`lifecycle test timed out: ${output}`))
    }, 30_000)
    child.once('error', reject)
    child.once('exit', (code, signal) => {
      clearTimeout(timeout)
      resolve({ code, durationMs: Date.now() - startedAt, output, signal })
    })
  })
  return { child, completed, output: () => output }
}

function runLifecycle(environment) {
  return startLifecycle(environment).completed
}

async function waitForOutput(lifecycle, pattern) {
  const deadline = Date.now() + 15_000
  while (Date.now() < deadline) {
    if (pattern.test(lifecycle.output())) return
    if (lifecycle.child.exitCode !== null) {
      throw new Error(`lifecycle exited before output ${pattern}: ${lifecycle.output()}`)
    }
    await new Promise((resolve) => setTimeout(resolve, 25))
  }
  throw new Error(`lifecycle output timed out waiting for ${pattern}: ${lifecycle.output()}`)
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
  const hangingRunner = path.join(fixtureRoot, 'hanging-runner.mjs')
  fs.writeFileSync(hangingRunner, "process.stdout.write('fake runner ready\\n'); setInterval(() => {}, 1_000)\n")

  try {
    const baseline = runRoots()

    for (const [missing, expectedError] of [
      [{ INKANDBONE_E2E_BINARY: path.join(fixtureRoot, 'missing-server') }, /ENOENT/],
      [{ INKANDBONE_E2E_BINARY: BINARY, INKANDBONE_E2E_PLAYWRIGHT_CLI: path.join(fixtureRoot, 'missing-runner') }, /MODULE_NOT_FOUND/],
    ]) {
      const port = await unusedPort()
      const result = await runLifecycle({ ...missing, INKANDBONE_E2E_PORT: String(port) })
      assert.notEqual(result.code, 0)
      assert.match(result.output, expectedError)
      assert.ok(result.durationMs < 4_000, `spawn failure cleanup took ${result.durationMs}ms: ${result.output}`)
      assert.deepEqual(runRoots(), baseline)
      await expectPortClosed(port)
    }

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
        OLLAMA_HOST: 'http://127.0.0.1:1/hostile-lifecycle-marker',
      })
      assert.equal(result.code, exitCode)
      assert.ok(result.durationMs < 4_000, `graceful lifecycle took ${result.durationMs}ms`)
      assert.match(result.output, /AI: disabled/)
      assert.deepEqual(runRoots(), baseline)
      assert.equal(fs.readFileSync(path.join(unrelated, 'keep.txt'), 'utf8'), 'do not remove')
      await expectPortClosed(port)
    }

    for (const [signal, expectedCode] of [['SIGINT', 130], ['SIGTERM', 143]]) {
      const port = await unusedPort()
      const lifecycle = startLifecycle({
        INKANDBONE_E2E_BINARY: BINARY,
        INKANDBONE_E2E_PLAYWRIGHT_CLI: hangingRunner,
        INKANDBONE_E2E_PORT: String(port),
      })
      await waitForOutput(lifecycle, /fake runner ready/)
      assert.equal(lifecycle.child.kill(signal), true)
      const result = await lifecycle.completed
      assert.equal(result.code, expectedCode)
      assert.equal(result.signal, null)
      assert.deepEqual(runRoots(), baseline)
      await expectPortClosed(port)
    }
  } finally {
    fs.rmSync(fixtureRoot, { force: true, recursive: true })
    fs.rmSync(unrelated, { force: true, recursive: true })
  }
})
