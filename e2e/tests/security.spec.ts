import { expect, test, type APIRequestContext, type Browser } from '@playwright/test'
import { spawn, spawnSync, type ChildProcessWithoutNullStreams } from 'child_process'
import { randomBytes } from 'crypto'
import fs from 'fs'
import http from 'http'
import https from 'https'
import net from 'net'
import os from 'os'
import path from 'path'
import {
  createE2ERunDirectory,
  removeE2ERunDirectory,
  sanitizedE2EEnvironment,
  TTRPG_SERVER_ENVIRONMENT_KEYS,
} from '../environment.mjs'

test.describe.configure({ mode: 'serial' })

const BINARY = process.env.INKANDBONE_E2E_BINARY || path.resolve(__dirname, '..', '..', 'ttrpg')

interface RunningServer {
  child: ChildProcessWithoutNullStreams
  dbPath: string
  origin: string
  output: () => string
}

interface TestState {
  authSecret: string
  certPath: string
  keyPath: string
  loopback: RunningServer
  tls: RunningServer
  root: string
}

let state: TestState

async function unusedLoopbackPort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.unref()
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      if (address === null || typeof address === 'string') {
        server.close(() => reject(new Error('could not allocate a loopback port')))
        return
      }
      server.close((error) => error ? reject(error) : resolve(address.port))
    })
  })
}

function probe(url: URL): Promise<number> {
  const transport = url.protocol === 'https:' ? https : http
  return new Promise((resolve, reject) => {
    const request = transport.get(url, {
      rejectUnauthorized: false,
      timeout: 1_000,
    }, (response) => {
      response.resume()
      resolve(response.statusCode ?? 0)
    })
    request.once('timeout', () => request.destroy(new Error('health check timed out')))
    request.once('error', reject)
  })
}

async function waitUntilHealthy(server: RunningServer): Promise<void> {
  const deadline = Date.now() + 15_000
  const health = new URL('/api/health', server.origin)
  while (Date.now() < deadline) {
    if (server.child.exitCode !== null) {
      throw new Error(`server exited before becoming healthy (${server.child.exitCode}): ${server.output()}`)
    }
    try {
      if (await probe(health) === 200) return
    } catch {
      // The process may still be opening and migrating its disposable database.
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  throw new Error(`server did not become healthy: ${server.output()}`)
}

async function startServer(options: {
  dbName: string
  root: string
  tls?: { certPath: string; keyPath: string }
  public?: boolean
  secret?: string
}): Promise<RunningServer> {
  const port = await unusedLoopbackPort()
  const protocol = options.tls ? 'https' : 'http'
  const origin = `${protocol}://127.0.0.1:${port}`
  const dbPath = path.join(options.root, options.dbName)
  const listenHost = options.public ? '0.0.0.0' : '127.0.0.1'
  const args = ['-db', dbPath, '-listen', `${listenHost}:${port}`]
  if (options.tls) {
    args.push('-tls-cert', options.tls.certPath, '-tls-key', options.tls.keyPath)
  }
  if (options.public) args.push('-allowed-origin', origin)

  const child = spawn(BINARY, args, {
    env: sanitizedE2EEnvironment({ TTRPG_AUTH_SECRET: options.secret }),
    stdio: ['pipe', 'pipe', 'pipe'],
  })
  let output = ''
  const appendOutput = (chunk: Buffer) => {
    output = (output + chunk.toString()).slice(-16_384)
  }
  child.stdout.on('data', appendOutput)
  child.stderr.on('data', appendOutput)
  const server = { child, dbPath, origin, output: () => output }
  try {
    await waitUntilHealthy(server)
    return server
  } catch (error) {
    await stopServer(server)
    throw error
  }
}

async function stopServer(server?: RunningServer): Promise<void> {
  if (!server || server.child.exitCode !== null) return
  await new Promise<void>((resolve) => {
    const timeout = setTimeout(() => {
      server.child.kill('SIGKILL')
    }, 5_000)
    server.child.once('exit', () => {
      clearTimeout(timeout)
      resolve()
    })
    server.child.kill('SIGTERM')
  })
}

function generateCertificate(root: string): { certPath: string; keyPath: string } {
  const certPath = path.join(root, 'server-cert.pem')
  const keyPath = path.join(root, 'server-key.pem')
  const result = spawnSync('openssl', [
    'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
    '-keyout', keyPath,
    '-out', certPath,
    '-days', '1',
    '-subj', '/CN=127.0.0.1',
    '-addext', 'subjectAltName=IP:127.0.0.1',
  ], { cwd: root, encoding: 'utf8' })
  if (result.status !== 0) {
    const detail = result.error?.message || result.stderr?.trim() || `openssl exited ${result.status}`
    throw new Error(`could not generate disposable TLS certificate: ${detail}`)
  }
  return { certPath, keyPath }
}

function rejectedPublicStart(
  root: string,
  name: string,
  args: string[],
  secret?: string,
): { status: number | null; output: string } {
  const result = spawnSync(BINARY, [
    '-db', path.join(root, `${name}.db`),
    '-listen', '0.0.0.0:0',
    ...args,
  ], {
    encoding: 'utf8',
    env: sanitizedE2EEnvironment({ TTRPG_AUTH_SECRET: secret }),
    timeout: 10_000,
  })
  return { status: result.status, output: `${result.stdout}${result.stderr}` }
}

async function loginInBrowser(browser: Browser) {
  const context = await browser.newContext({ ignoreHTTPSErrors: true })
  const page = await context.newPage()
  await page.goto(state.tls.origin)
  await expect(page.getByRole('heading', { name: 'Unlock the table' })).toBeVisible()
  await page.getByLabel('Master secret').fill(state.authSecret)
  await page.getByRole('button', { name: 'Unlock' }).click()
  await expect(page.locator('.grimoire')).toBeVisible()
  return { context, page }
}

async function createCampaign(api: APIRequestContext, name: string): Promise<number> {
  const rulesetResponse = await api.get('/api/rulesets')
  expect(rulesetResponse.ok()).toBe(true)
  const rulesets = await rulesetResponse.json() as Array<{ id: number; name: string }>
  const ruleset = rulesets.find((candidate) => candidate.name === 'ironsworn')
  expect(ruleset, 'fresh database should contain the Ironsworn ruleset').toBeDefined()

  const campaignResponse = await api.post('/api/campaigns', {
    data: { name, description: 'Phase 1 security E2E', ruleset_id: ruleset!.id },
  })
  expect(campaignResponse.status()).toBe(201)
  return (await campaignResponse.json() as { id: number }).id
}

function rejectedWebSocketUpgrade(origin: string, secret: string): Promise<{ body: string; status: number }> {
  const url = new URL('/ws', origin)
  return new Promise((resolve, reject) => {
    const request = https.request({
      hostname: url.hostname,
      port: url.port,
      path: url.pathname,
      method: 'GET',
      rejectUnauthorized: false,
      headers: {
        Authorization: `Bearer ${secret}`,
        Connection: 'Upgrade',
        Origin: 'https://attacker.invalid',
        'Sec-WebSocket-Key': randomBytes(16).toString('base64'),
        'Sec-WebSocket-Version': '13',
        Upgrade: 'websocket',
      },
    })
    request.once('upgrade', (response, socket) => {
      socket.destroy()
      reject(new Error(`untrusted WebSocket origin was upgraded (${response.statusCode})`))
    })
    request.once('response', (response) => {
      let body = ''
      response.setEncoding('utf8')
      response.on('data', (chunk) => { body += chunk })
      response.on('end', () => resolve({ body, status: response.statusCode ?? 0 }))
    })
    request.once('error', reject)
    request.end()
  })
}

function permittedWebSocketUpgrade(origin: string, secret: string): Promise<number> {
  const url = new URL('/ws', origin)
  return new Promise((resolve, reject) => {
    const request = https.request({
      hostname: url.hostname,
      port: url.port,
      path: url.pathname,
      method: 'GET',
      rejectUnauthorized: false,
      headers: {
        Authorization: `Bearer ${secret}`,
        Connection: 'Upgrade',
        Origin: origin,
        'Sec-WebSocket-Key': randomBytes(16).toString('base64'),
        'Sec-WebSocket-Version': '13',
        Upgrade: 'websocket',
      },
    })
    request.once('upgrade', (response, socket) => {
      const status = response.statusCode ?? 0
      socket.destroy()
      resolve(status)
    })
    request.once('response', (response) => {
      response.resume()
      reject(new Error(`permitted WebSocket origin was rejected (${response.statusCode})`))
    })
    request.once('error', reject)
    request.end()
  })
}

function expectSanitizedChildEnvironment(): void {
  const previous = new Map<string, string | undefined>()
  try {
    for (const key of TTRPG_SERVER_ENVIRONMENT_KEYS) {
      previous.set(key, process.env[key])
      process.env[key] = 'hostile-ambient-marker'
    }
    const childEnvironment = sanitizedE2EEnvironment()
    for (const key of TTRPG_SERVER_ENVIRONMENT_KEYS) {
      expect(childEnvironment[key], `${key} should be cleared`).toBe('')
    }
  } finally {
    for (const [key, value] of previous) {
      if (value === undefined) delete process.env[key]
      else process.env[key] = value
    }
  }
}

function expectIsolatedRecursiveCleanup(): void {
  const first = createE2ERunDirectory()
  const second = createE2ERunDirectory()
  try {
    fs.writeFileSync(path.join(first, 'smoke.db-wal'), 'sidecar')
    fs.writeFileSync(path.join(first, 'smoke.db.backup-interrupted'), 'backup')
    const staging = path.join(first, '.smoke.db.backup-stage-interrupted')
    fs.mkdirSync(staging)
    fs.writeFileSync(path.join(staging, 'database.sqlite'), 'staging')
    fs.writeFileSync(path.join(second, 'concurrent-run.db'), 'other run')

    removeE2ERunDirectory(first)
    expect(fs.existsSync(first)).toBe(false)
    expect(fs.existsSync(second)).toBe(true)
    expect(fs.readFileSync(path.join(second, 'concurrent-run.db'), 'utf8')).toBe('other run')
  } finally {
    removeE2ERunDirectory(first)
    removeE2ERunDirectory(second)
  }

}

test.describe('phase one security boundaries', () => {
  test.beforeAll(async () => {
    expect(fs.existsSync(BINARY), 'run `make build` before the E2E suite').toBe(true)
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'inkandbone-security-e2e-'))
    const authSecret = randomBytes(32).toString('base64url')
    let loopback: RunningServer | undefined
    let tls: RunningServer | undefined
    try {
      const { certPath, keyPath } = generateCertificate(root)
      loopback = await startServer({ dbName: 'loopback.db', root })
      tls = await startServer({
        dbName: 'tls.db',
        root,
        public: true,
        secret: authSecret,
        tls: { certPath, keyPath },
      })
      state = { authSecret, certPath, keyPath, loopback, root, tls }
    } catch (error) {
      await stopServer(tls)
      await stopServer(loopback)
      fs.rmSync(root, { force: true, recursive: true })
      throw error
    }
  })

  test.afterAll(async () => {
    await stopServer(state?.tls)
    await stopServer(state?.loopback)
    if (state?.root) fs.rmSync(state.root, { force: true, recursive: true })
  })

  test('loopback startup does not require login', async ({ page }) => {
    expectSanitizedChildEnvironment()
    const sessionResponse = await page.request.get(`${state.loopback.origin}/api/auth/session`)
    expect(sessionResponse.ok()).toBe(true)
    expect(await sessionResponse.json()).toEqual({ authenticated: true })

    await page.goto(state.loopback.origin)
    await expect(page.locator('.grimoire')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Unlock the table' })).toHaveCount(0)
  })

  test('public startup rejects missing secret, TLS, and origin controls', () => {
    expectIsolatedRecursiveCleanup()
    const noSecret = rejectedPublicStart(state.root, 'no-secret', [])
    expect(noSecret.status).not.toBe(0)
    expect(noSecret.output).toContain('TTRPG_AUTH_SECRET')

    const noTLS = rejectedPublicStart(
      state.root,
      'no-tls',
      ['-allowed-origin', 'https://127.0.0.1:7443'],
      state.authSecret,
    )
    expect(noTLS.status).not.toBe(0)
    expect(noTLS.output).toContain('TLS certificate and key')

    const noOrigin = rejectedPublicStart(
      state.root,
      'no-origin',
      ['-tls-cert', state.certPath, '-tls-key', state.keyPath],
      state.authSecret,
    )
    expect(noOrigin.status).not.toBe(0)
    expect(noOrigin.output).toContain('at least one allowed origin')
  })

  test('TLS browser login establishes an authenticated session', async ({ browser }) => {
    const { context, page } = await loginInBrowser(browser)
    try {
      const cookies = await context.cookies(state.tls.origin)
      const cookie = cookies.find((candidate) => candidate.name === 'ttrpg_session')
      expect(cookie).toBeDefined()
      expect(cookie!.value).not.toBe('')
      expect(cookie!.value).not.toBe(state.authSecret)
      expect(cookie!.value).toMatch(/^[A-Za-z0-9_-]+$/)
      expect(cookie).toMatchObject({
        httpOnly: true,
        path: '/',
        sameSite: 'Strict',
        secure: true,
      })

      const session = await page.evaluate(async () => {
        const response = await fetch('/api/auth/session')
        return { body: await response.json(), status: response.status }
      })
      expect(session.status).toBe(200)
      expect(session.body).toMatchObject({ authenticated: true })
      expect(session.body.csrf_token).toEqual(expect.any(String))
      expect(session.body.csrf_token.length).toBeGreaterThan(20)
    } finally {
      await context.close()
    }
  })

  test('authenticated cookies cannot mutate state without CSRF', async ({ browser }) => {
    const { context, page } = await loginInBrowser(browser)
    try {
      const result = await page.evaluate(async () => {
        const sessionResponse = await fetch('/api/auth/session')
        const session = await sessionResponse.json() as { csrf_token?: string }
        const init: RequestInit = {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ campaign_id: null }),
        }
        const rejected = await fetch('/api/settings', init)
        const accepted = await fetch('/api/settings', {
          ...init,
          headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': session.csrf_token ?? '',
          },
        })
        return {
          acceptedStatus: accepted.status,
          rejectedBody: await rejected.text(),
          rejectedStatus: rejected.status,
          tokenPresent: Boolean(session.csrf_token),
        }
      })
      expect(result.tokenPresent).toBe(true)
      expect(result.rejectedStatus).toBe(403)
      expect(result.rejectedBody).toContain('invalid CSRF token')
      expect(result.acceptedStatus).toBe(204)
    } finally {
      await context.close()
    }
  })

  test('WebSocket upgrades reject an untrusted origin', async () => {
    expect(await permittedWebSocketUpgrade(state.tls.origin, state.authSecret)).toBe(101)
    const response = await rejectedWebSocketUpgrade(state.tls.origin, state.authSecret)
    expect(response.status).toBe(403)
    expect(response.body).toContain('Forbidden')
  })

  test('typed map assets serve while the database cannot be downloaded', async ({ playwright }) => {
    const api = await playwright.request.newContext({ baseURL: state.loopback.origin })
    try {
      const campaignID = await createCampaign(api, 'Asset Boundary Campaign')
      const svg = Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><rect width="1" height="1"/></svg>')
      const upload = await api.post(`/api/campaigns/${campaignID}/maps`, {
        multipart: {
          image: { buffer: svg, mimeType: 'image/svg+xml', name: 'boundary.svg' },
          name: 'Boundary Map',
        },
      })
      expect(upload.status()).toBe(201)
      const map = await upload.json() as { id: number; image_path: string }
      expect(map.image_path).toMatch(/^maps\/[a-f0-9]+\.svg$/)

      const asset = await api.get(`/api/assets/maps/${map.id}`)
      expect(asset.status()).toBe(200)
      expect(asset.headers()['content-type']).toContain('image/svg+xml')
      expect(asset.headers()['content-security-policy']).toBe("sandbox; default-src 'none'")
      expect(asset.headers()['x-content-type-options']).toBe('nosniff')
      expect(await asset.body()).toEqual(svg)

      const databaseDownload = await api.get(`/api/files/${path.basename(state.loopback.dbPath)}`)
      expect(databaseDownload.status()).toBe(404)
      expect(await databaseDownload.text()).not.toContain('SQLite format 3')
    } finally {
      await api.dispose()
    }
  })

  test('whisper sentinels remain visible to the player but absent from AI context', async ({ playwright }) => {
    const api = await playwright.request.newContext({ baseURL: state.loopback.origin })
    const whisperSentinel = 'WHISPER_SENTINEL_MUST_NEVER_REACH_AI_CONTEXT'
    const publicSentinel = 'PUBLIC_SENTINEL_MAY_REACH_AI_CONTEXT'
    try {
      const campaignID = await createCampaign(api, 'Whisper Boundary Campaign')
      const sessionResponse = await api.post(`/api/campaigns/${campaignID}/sessions`, {
        data: { date: '2026-07-17', title: 'Private Session' },
      })
      expect(sessionResponse.status()).toBe(201)
      const sessionID = (await sessionResponse.json() as { id: number }).id

      const settings = await api.patch('/api/settings', {
        data: { campaign_id: campaignID, character_id: null, session_id: sessionID },
      })
      expect(settings.ok()).toBe(true)

      for (const message of [
        { content: publicSentinel, role: 'user', whisper: false },
        { content: whisperSentinel, role: 'user', whisper: true },
      ]) {
        const created = await api.post(`/api/sessions/${sessionID}/messages`, { data: message })
        expect(created.status()).toBe(201)
      }

      const contextResponse = await api.get('/api/context')
      expect(contextResponse.ok()).toBe(true)
      const aiContext = JSON.stringify(await contextResponse.json())
      expect(aiContext).toContain(publicSentinel)
      expect(aiContext).not.toContain(whisperSentinel)

      const transcriptResponse = await api.get(`/api/sessions/${sessionID}/messages`)
      expect(transcriptResponse.ok()).toBe(true)
      const transcript = await transcriptResponse.json() as Array<{ content: string; whisper: boolean }>
      expect(transcript).toEqual(expect.arrayContaining([
        expect.objectContaining({ content: whisperSentinel, whisper: true }),
      ]))
    } finally {
      await api.dispose()
    }
  })
})
