import { useCallback, useEffect, useRef, useState } from 'react'

export type WebSocketStatus = 'connecting' | 'open' | 'reconnecting' | 'offline'

export interface WebSocketOptions {
  jitter?: () => number
}

interface SequencedEvent {
  type?: string
  sequence?: number
}

export function webSocketURL(page: Pick<Location, 'protocol' | 'host'>): string {
  const protocol = page.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${page.host}/ws`
}

export function useWebSocket(
  url: string,
  onMessage: (data: unknown) => void,
  enabled = true,
  options: WebSocketOptions = {},
): {
  lastEvent: unknown
  status: WebSocketStatus
  needsReconcile: boolean
  acknowledgeReconcile: () => void
} {
  const [lastEvent, setLastEvent] = useState<unknown>(null)
  const [status, setStatus] = useState<WebSocketStatus>(enabled ? 'connecting' : 'offline')
  const [needsReconcile, setNeedsReconcile] = useState(false)
  const onMessageRef = useRef(onMessage)
  const jitterRef = useRef(options.jitter ?? (() => 0.8 + Math.random() * 0.4))
  const lastSequenceRef = useRef<number | null>(null)
  useEffect(() => { onMessageRef.current = onMessage })
  useEffect(() => { jitterRef.current = options.jitter ?? (() => 0.8 + Math.random() * 0.4) }, [options.jitter])

  const acknowledgeReconcile = useCallback(() => {
    lastSequenceRef.current = null
    setNeedsReconcile(false)
  }, [])

  useEffect(() => {
    if (!enabled) {
      setStatus('offline')
      return
    }

    let ws: WebSocket | null = null
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let cancelled = false
    let generation = 0
    let attempt = 0
    let hasOpened = false

    function connect() {
      if (cancelled) return
      if (typeof navigator !== 'undefined' && !navigator.onLine) {
        setStatus('offline')
        return
      }
      const socketGeneration = ++generation
      const socket = new WebSocket(url)
      ws = socket
      setStatus(hasOpened || attempt > 0 ? 'reconnecting' : 'connecting')

      socket.onopen = () => {
        if (cancelled || ws !== socket || generation !== socketGeneration) return
        if (hasOpened) setNeedsReconcile(true)
        hasOpened = true
        attempt = 0
        setStatus('open')
      }

      socket.onmessage = (e) => {
        if (cancelled || ws !== socket || generation !== socketGeneration) return
        try {
          const parsed = JSON.parse(e.data as string) as SequencedEvent
          if (parsed.type === 'resync_required') setNeedsReconcile(true)
          if (typeof parsed.sequence === 'number') {
            if (lastSequenceRef.current !== null && parsed.sequence !== lastSequenceRef.current + 1) setNeedsReconcile(true)
            lastSequenceRef.current = parsed.sequence
          }
          setLastEvent(parsed)
          onMessageRef.current(parsed)
        } catch {
          // Ignore malformed messages without changing reconciliation state.
        }
      }

      socket.onclose = () => {
        if (cancelled || ws !== socket || generation !== socketGeneration) return
        setStatus('reconnecting')
        if (hasOpened) setNeedsReconcile(true)
        const delay = Math.min(1000 * (2 ** attempt), 30000) * jitterRef.current()
        attempt++
        reconnectTimer = setTimeout(connect, delay)
      }
    }

    const handleOffline = () => {
      if (cancelled) return
      generation++
      if (reconnectTimer !== null) clearTimeout(reconnectTimer)
      reconnectTimer = null
      setStatus('offline')
      ws?.close()
    }
    const handleOnline = () => {
      if (cancelled || ws?.readyState === 1) return
      setStatus(hasOpened ? 'reconnecting' : 'connecting')
      connect()
    }

    window.addEventListener('offline', handleOffline)
    window.addEventListener('online', handleOnline)
    connect()

    return () => {
      cancelled = true
      generation++
      if (reconnectTimer !== null) clearTimeout(reconnectTimer)
      ws?.close()
      window.removeEventListener('offline', handleOffline)
      window.removeEventListener('online', handleOnline)
    }
  }, [url, enabled])

  return { lastEvent, status, needsReconcile, acknowledgeReconcile }
}
