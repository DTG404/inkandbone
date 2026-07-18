import { renderHook, act, cleanup } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useWebSocket, webSocketURL } from './useWebSocket'

// Minimal WebSocket mock
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState = MockWebSocket.CONNECTING
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close = vi.fn(() => { this.readyState = MockWebSocket.CLOSED })

  url: string
  constructor(url: string) { this.url = url }

  open() {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.()
  }
  receive(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent)
  }
  drop() {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.()
  }
}

let instances: MockWebSocket[] = []

beforeEach(() => {
  instances = []
  vi.stubGlobal('WebSocket', function (url: string) {
    const ws = new MockWebSocket(url)
    instances.push(ws)
    return ws
  })
  vi.useFakeTimers()
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('useWebSocket', () => {
  it('uses secure WebSockets for HTTPS pages', () => {
    expect(webSocketURL({ protocol: 'https:', host: 'table.example' })).toBe('wss://table.example/ws')
    expect(webSocketURL({ protocol: 'http:', host: '127.0.0.1:7432' })).toBe('ws://127.0.0.1:7432/ws')
  })

  it('does not connect while disabled', () => {
    renderHook(() => useWebSocket('/ws', vi.fn(), false))
    expect(instances).toHaveLength(0)
  })

  it('calls onMessage with parsed JSON when a message arrives', () => {
    const onMessage = vi.fn()
    renderHook(() => useWebSocket('/ws', onMessage))
    act(() => instances[0].open())
    act(() => instances[0].receive({ type: 'ping', sequence: 1 }))
    expect(onMessage).toHaveBeenCalledWith({ type: 'ping', sequence: 1 })
  })

  it('reports statuses, reconnects with capped deterministic backoff, and resets after open', () => {
    const { result } = renderHook(() => useWebSocket('/ws', vi.fn(), true, { jitter: () => 1 }))
    expect(result.current.status).toBe('connecting')
    act(() => instances[0].open())
    expect(result.current.status).toBe('open')
    act(() => instances[0].drop())
    expect(result.current.status).toBe('reconnecting')
    expect(instances).toHaveLength(1)
    act(() => vi.advanceTimersByTime(999))
    expect(instances).toHaveLength(1)
    act(() => vi.advanceTimersByTime(1))
    expect(instances).toHaveLength(2)
    act(() => instances[1].drop())
    act(() => vi.advanceTimersByTime(1999))
    expect(instances).toHaveLength(2)
    act(() => vi.advanceTimersByTime(1))
    act(() => instances[2].open())
    act(() => instances[2].drop())
    act(() => vi.advanceTimersByTime(1000))
    expect(instances).toHaveLength(4)
  })

  it('caps the exponential base delay at 30 seconds', () => {
    const jitter = vi.fn(() => 1)
    renderHook(() => useWebSocket('/ws', vi.fn(), true, { jitter }))
    for (const delay of [1000, 2000, 4000, 8000, 16000, 30000, 30000]) {
      const currentCount = instances.length
      act(() => instances.at(-1)?.drop())
      act(() => vi.advanceTimersByTime(delay - 1))
      expect(instances).toHaveLength(currentCount)
      act(() => vi.advanceTimersByTime(1))
      expect(instances).toHaveLength(currentCount + 1)
    }
    expect(jitter).toHaveBeenCalledTimes(7)
  })

  it('uses the injected jitter multiplier', () => {
    const jitter = vi.fn(() => 0.8)
    renderHook(() => useWebSocket('/ws', vi.fn(), true, { jitter }))
    act(() => instances[0].drop())
    act(() => vi.advanceTimersByTime(799))
    expect(instances).toHaveLength(1)
    act(() => vi.advanceTimersByTime(1))
    expect(instances).toHaveLength(2)
    expect(jitter).toHaveBeenCalledOnce()
  })

  it('closes the WebSocket on unmount', () => {
    const { unmount } = renderHook(() => useWebSocket('/ws', vi.fn()))
    act(() => instances[0].open())
    unmount()
    expect(instances[0].close).toHaveBeenCalled()
  })

  it('returns lastEvent after receiving a message', () => {
    const { result } = renderHook(() => useWebSocket('/ws', vi.fn()))
    act(() => instances[0].open())
    act(() => instances[0].receive({ type: 'dice_rolled', sequence: 1, payload: { total: 15 } }))
    expect(result.current.lastEvent).toEqual({ type: 'dice_rolled', sequence: 1, payload: { total: 15 } })
  })

  it('requires reconciliation after a sequence gap, resync signal, or reconnect', () => {
    const { result } = renderHook(() => useWebSocket('/ws', vi.fn(), true, { jitter: () => 1 }))
    act(() => instances[0].open())
    act(() => instances[0].receive({ type: 'dice_rolled', sequence: 4 }))
    expect(result.current.needsReconcile).toBe(false)
    act(() => instances[0].receive({ type: 'typing', sequence: 6 }))
    expect(result.current.needsReconcile).toBe(true)
    act(() => result.current.acknowledgeReconcile(result.current.reconcileGeneration))
    expect(result.current.needsReconcile).toBe(false)
    act(() => instances[0].receive({ type: 'dice_rolled', sequence: 8 }))
    expect(result.current.needsReconcile).toBe(true)
    act(() => result.current.acknowledgeReconcile(result.current.reconcileGeneration))
    act(() => instances[0].receive({ type: 'resync_required', sequence: 9 }))
    expect(result.current.needsReconcile).toBe(true)
  })

  it('does not let an older reconciliation acknowledge a newer sequence gap', () => {
    const { result } = renderHook(() => useWebSocket('/ws', vi.fn()))
    act(() => instances[0].open())
    act(() => instances[0].receive({ type: 'typing', sequence: 1 }))
    act(() => instances[0].receive({ type: 'typing', sequence: 3 }))
    const firstGeneration = result.current.reconcileGeneration
    expect(result.current.needsReconcile).toBe(true)

    act(() => instances[0].receive({ type: 'typing', sequence: 5 }))
    const secondGeneration = result.current.reconcileGeneration
    expect(secondGeneration).toBeGreaterThan(firstGeneration)
    act(() => result.current.acknowledgeReconcile(firstGeneration))
    expect(result.current.needsReconcile).toBe(true)
    act(() => result.current.acknowledgeReconcile(secondGeneration))
    expect(result.current.needsReconcile).toBe(false)
  })

  it('reports offline and reconnects when the browser returns online', () => {
    let online = false
    vi.spyOn(navigator, 'onLine', 'get').mockImplementation(() => online)
    const { result } = renderHook(() => useWebSocket('/ws', vi.fn(), true, { jitter: () => 1 }))
    expect(result.current.status).toBe('offline')
    expect(instances).toHaveLength(0)
    online = true
    act(() => window.dispatchEvent(new Event('online')))
    expect(result.current.status).toBe('connecting')
    expect(instances).toHaveLength(1)
    online = false
    act(() => window.dispatchEvent(new Event('offline')))
    expect(result.current.status).toBe('offline')
    expect(instances[0].close).toHaveBeenCalled()
  })

  it('cleans timers and ignores stale socket callbacks', () => {
    const onMessage = vi.fn()
    const { unmount } = renderHook(() => useWebSocket('/ws', onMessage, true, { jitter: () => 1 }))
    const stale = instances[0]
    act(() => stale.drop())
    act(() => vi.advanceTimersByTime(1000))
    expect(instances).toHaveLength(2)
    act(() => stale.receive({ type: 'dice_rolled', sequence: 1 }))
    expect(onMessage).not.toHaveBeenCalled()
    act(() => instances[1].drop())
    unmount()
    act(() => vi.runAllTimers())
    expect(instances).toHaveLength(2)
  })
})
