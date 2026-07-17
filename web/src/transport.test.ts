import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { request, setCSRFToken } from './transport'

describe('request', () => {
  beforeEach(() => {
    setCSRFToken('csrf-test-token')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
  })

  afterEach(() => {
    setCSRFToken(null)
    vi.unstubAllGlobals()
  })

  for (const method of ['POST', 'PATCH', 'DELETE']) {
    it(`adds the CSRF token to ${method} requests`, async () => {
      await request('/api/example', { method })

      const [, init] = vi.mocked(fetch).mock.calls[0]
      const headers = new Headers(init?.headers)
      expect(headers.get('X-CSRF-Token')).toBe('csrf-test-token')
      expect(init?.credentials).toBe('same-origin')
    })
  }

  it('does not add the CSRF token to safe requests', async () => {
    await request('/api/example')

    const [, init] = vi.mocked(fetch).mock.calls[0]
    expect(new Headers(init?.headers).has('X-CSRF-Token')).toBe(false)
  })
})
