import { useEffect, useRef, useState } from 'react'

export function webSocketURL(page: Pick<Location, 'protocol' | 'host'>): string {
  const protocol = page.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${page.host}/ws`
}

export function useWebSocket(url: string, onMessage: (data: unknown) => void, enabled = true): { lastEvent: unknown } {
  const [lastEvent, setLastEvent] = useState<unknown>(null)
  const onMessageRef = useRef(onMessage)
  useEffect(() => { onMessageRef.current = onMessage })

  useEffect(() => {
    if (!enabled) return

    let ws: WebSocket
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null
    let cancelled = false

    function connect() {
      ws = new WebSocket(url)

      ws.onmessage = (e) => {
        try {
          const parsed = JSON.parse(e.data as string)
          setLastEvent(parsed)
          onMessageRef.current(parsed)
        } catch {
          // ignore malformed messages
        }
      }

      ws.onclose = () => {
        if (!cancelled) {
          reconnectTimer = setTimeout(connect, 2000)
        }
      }
    }

    connect()

    return () => {
      cancelled = true
      if (reconnectTimer !== null) clearTimeout(reconnectTimer)
      ws.close()
    }
  }, [url, enabled])

  return { lastEvent }
}
