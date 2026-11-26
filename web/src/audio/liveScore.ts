import { request } from '../api/transport'

type Player = {
  config: { sessionId?: string }
  player: {
    init(options: { configure(): void; onActive(): void; onInactive(): void }): Promise<void>
    audioSystem: { setLocalVolume(volume: number): void }
  }
}
declare global { interface Window { syrinscape?: Player } }

let activeSession: number | null = null
let narrating: number | null = null
let engineActive = false
let init: Promise<void> | undefined
let localLevel = 0.7
const root = '/sound-test/live/'
type Recovery = { session: number; delay: number; needed: boolean; busy: boolean; timer?: number }
let recovery: Recovery | undefined

class SoundRequestError extends Error {
  readonly status: number
  constructor(status: number) {
    super('Sound service unavailable')
    this.status = status
  }
}

async function call(path: string, session: number, extra: object = {}, timeout = 15000) {
  const res = await request(root + path, { method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({session, ...extra}), signal: AbortSignal.timeout(timeout) })
  if (!res.ok) throw new SoundRequestError(res.status)
  return res.json()
}

function script(name: string, digest: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const el = document.createElement('script')
    el.src = 'https://syrinscape.com/static/dist/13f6592389d6bd2903c55a45ed525780c625bff2/' + name
    el.integrity = 'sha384-' + digest
    el.crossOrigin = 'anonymous'
    el.onload = () => resolve()
    el.onerror = () => reject(new Error('Could not load the sound player'))
    document.head.append(el)
  })
}

export function setScoreLevel(value: number) {
  localLevel = Number.isFinite(value) ? Math.max(0, Math.min(1, value)) : 0.7
  if (engineActive) window.syrinscape?.player.audioSystem.setLocalVolume(activeSession ? localLevel : 0)
}

export function scoringTurn(session: number | null) {
  narrating = session
  if (session === null) void restoreIfNeeded()
}

export function leaveScore() {
  if (recovery?.timer !== undefined) window.clearTimeout(recovery.timer)
  recovery = undefined
  document.removeEventListener('visibilitychange', requestRestore)
  activeSession = null
  setScoreLevel(localLevel)
}

function requestRestore() {
  if (recovery) recovery.needed = true
  void restoreIfNeeded()
}

async function restoreIfNeeded() {
  const state = recovery
  if (!state?.needed || state.busy || state.timer !== undefined || !engineActive ||
      document.hidden || narrating === state.session) return
  state.busy = true
  state.needed = false
  try {
    await call('resume', state.session, {}, 24000)
    state.delay = 2000
  } catch (error) {
    if (recovery !== state) return
    if (error instanceof SoundRequestError && error.status < 500 && error.status !== 429) return
    state.needed = true
    state.timer = window.setTimeout(() => {
      state.timer = undefined
      void restoreIfNeeded()
    }, state.delay)
    state.delay = Math.min(state.delay * 2, 30000)
  } finally {
    state.busy = false
  }
}

export async function joinScore(session: number) {
  if (activeSession !== session) leaveScore()
  activeSession = session
  recovery ??= { session, delay: 2000, needed: true, busy: false }
  document.addEventListener('visibilitychange', requestRestore)
  const { listenerSession } = await call('join', session)
  if (activeSession !== session) return
  if (!init) {
    init = (async () => {
      await script('integration.5f52f359c6411630754e.js','VzpmchxZ+QqsbQSD8BPwmdjUDE4drHq/AMl68hIDI/ArpZvU31uBiHJ7cI8BAjH2')
      await script('player.3d021545a255e950555d.js','v5YeEsUY9GX0ckicEaZ+LQ9uLfQqAYG+KkqvSp3WbKjO4gZTikSGiFZIUZ5N3gBj')
      const engine = window.syrinscape!
      await engine.player.init({
        configure() { engine.config.sessionId = listenerSession },
        onActive() {
          engineActive = true
          setScoreLevel(localLevel)
          requestRestore()
        },
        onInactive() { engineActive = false },
      })
    })().catch(err => { init = undefined; throw err })
    await init
  } else {
    await init
    if (activeSession === session && engineActive) { setScoreLevel(localLevel); requestRestore() }
  }
}

function painted() {
  // Background tabs cannot acknowledge a visible narrative beat.
  return Promise.race([
    new Promise<boolean>(ok => requestAnimationFrame(() => requestAnimationFrame(() => ok(!document.hidden)))),
    new Promise<boolean>(ok => setTimeout(() => ok(false), 300)),
  ])
}

export async function revealScoredTurn(session: number, text: string, reveal: (text: string) => void) {
  if (activeSession !== session) return
  try {
    const bytes = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text))
    const digest = Array.from(new Uint8Array(bytes), b => b.toString(16).padStart(2, '0')).join('')
    const plan: {message?: number; cues: {anchor: string; slot: number}[]} = await call('plan', session, {digest}, 17000)
    for (const cue of plan.cues) {
      if (activeSession !== session) return
      const at = text.indexOf(cue.anchor)
      if (at < 0) continue
      reveal(text.slice(0, at + cue.anchor.length))
      if (await painted() && activeSession === session) {
        await call('heard', session, {message: plan.message, slot: cue.slot}, 3500)
      } else if (recovery?.session === session) {
        recovery.needed = true
      }
    }
  } catch {
    if (recovery?.session === session) recovery.needed = true
    console.warn('Live score unavailable for this turn')
  } finally {
    if (activeSession === session) reveal(text)
  }
}
