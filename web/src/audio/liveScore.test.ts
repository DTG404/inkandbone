import { afterEach, expect, it, vi } from 'vitest'
import { request } from '../api/transport'

vi.mock('../api/transport', () => ({ request: vi.fn() }))

afterEach(() => { vi.restoreAllMocks(); vi.useRealTimers(); delete window.syrinscape })

it('retries a busy ambience restore without replaying a heard event', async () => {
  vi.useFakeTimers()
  vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
  vi.spyOn(document.head, 'append').mockImplementation((node) => {
    if (node instanceof HTMLScriptElement) queueMicrotask(() => node.dispatchEvent(new Event('load')))
  })
  window.syrinscape = { config: {}, player: {
    async init(callbacks) { callbacks.configure(); callbacks.onActive() },
    audioSystem: { setLocalVolume: vi.fn() },
  } }
  const calls: string[] = []
  vi.mocked(request).mockImplementation(async (path) => {
    calls.push(String(path).split('/').pop()!)
    if (calls.length === 2) return new Response('{}', { status: 502 })
    return new Response(JSON.stringify({ listenerSession: 'test' }))
  })
  const { joinScore, leaveScore } = await import('./liveScore')
  try {
    await joinScore(4)
    await vi.advanceTimersByTimeAsync(3100)
    expect(calls).toEqual(['join', 'resume', 'resume'])
  } finally { leaveScore() }
})
