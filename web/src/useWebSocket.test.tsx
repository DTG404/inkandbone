import { renderHook, act } from '@testing-library/react'
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
    act(() => instances[0].receive({ type: 'ping' }))
    expect(onMessage).toHaveBeenCalledWith({ type: 'ping' })
  })

  it('reconnects after 2 seconds when connection drops', () => {
    renderHook(() => useWebSocket('/ws', vi.fn()))
    act(() => instances[0].open())
    act(() => instances[0].drop())
    expect(instances).toHaveLength(1)
    act(() => vi.advanceTimersByTime(2000))
    expect(instances).toHaveLength(2)
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
    act(() => instances[0].receive({ type: 'dice_rolled', payload: { total: 15 } }))
    expect(result.current.lastEvent).toEqual({ type: 'dice_rolled', payload: { total: 15 } })
  })
})
