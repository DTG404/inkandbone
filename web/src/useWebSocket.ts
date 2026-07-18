import { useCallback, useEffect, useRef, useState } from 'react'
import { parseRealtimeEvent, type RealtimeEvent } from './realtime.gen'

export type WebSocketStatus = 'connecting' | 'open' | 'reconnecting' | 'offline'

export interface WebSocketOptions {
  jitter?: () => number
}

export function webSocketURL(page: Pick<Location, 'protocol' | 'host'>): string {
  const protocol = page.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${page.host}/ws`
}

export function useWebSocket(
  url: string,
  onMessage: (data: RealtimeEvent) => void,
  enabled = true,
  options: WebSocketOptions = {},
): {
  lastEvent: RealtimeEvent | null
  status: WebSocketStatus
  needsReconcile: boolean
  reconcileGeneration: number
  acknowledgeReconcile: (generation: number) => void
} {
  const [lastEvent, setLastEvent] = useState<RealtimeEvent | null>(null)
  const [status, setStatus] = useState<WebSocketStatus>(enabled ? 'connecting' : 'offline')
  const [reconcileGeneration, setReconcileGeneration] = useState(0)
  const [acknowledgedGeneration, setAcknowledgedGeneration] = useState(0)
  const onMessageRef = useRef(onMessage)
  const jitterRef = useRef(options.jitter ?? (() => 0.8 + Math.random() * 0.4))
  const lastSequenceRef = useRef<number | null>(null)
  const reconcileGenerationRef = useRef(0)
  useEffect(() => { onMessageRef.current = onMessage })
  useEffect(() => { jitterRef.current = options.jitter ?? (() => 0.8 + Math.random() * 0.4) }, [options.jitter])

  const acknowledgeReconcile = useCallback((generation: number) => {
    if (generation !== reconcileGenerationRef.current) return
    setAcknowledgedGeneration(generation)
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
    const requireReconciliation = () => {
      const next = reconcileGenerationRef.current + 1
      reconcileGenerationRef.current = next
      setReconcileGeneration(next)
    }

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
        hasOpened = true
        attempt = 0
        setStatus('open')
      }

      socket.onmessage = (e) => {
        if (cancelled || ws !== socket || generation !== socketGeneration) return
        try {
          const parsed = parseRealtimeEvent(JSON.parse(String(e.data)))
          if (!parsed) return
          let reconcile = parsed.type === 'resync_required'
          if (typeof parsed.sequence === 'number') {
            if (lastSequenceRef.current !== null && parsed.sequence !== lastSequenceRef.current + 1) reconcile = true
            lastSequenceRef.current = parsed.sequence
          }
          if (reconcile) requireReconciliation()
          setLastEvent(parsed)
          onMessageRef.current(parsed)
        } catch {
          // Ignore malformed messages without changing reconciliation state.
        }
      }

      socket.onclose = () => {
        if (cancelled || ws !== socket || generation !== socketGeneration) return
        setStatus('reconnecting')
        if (hasOpened) requireReconciliation()
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
      if (hasOpened) requireReconciliation()
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

  return {
    lastEvent,
    status,
    needsReconcile: reconcileGeneration !== acknowledgedGeneration,
    reconcileGeneration,
    acknowledgeReconcile,
  }
}
