export type SSEEvent =
  | { type: 'delta'; delta: string }
  | { type: 'done' }
  | { type: 'error'; code: string; request_id?: string }

function dispatchFrame(frame: string, onEvent: (event: SSEEvent) => void): void {
  const data = frame
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).replace(/^ /, ''))
    .join('\n')
  if (!data) return
  const value: unknown = JSON.parse(data)
  if (!value || typeof value !== 'object' || !('type' in value)) throw new Error('Invalid SSE event')
  const event = value as Record<string, unknown>
  if (event.type === 'delta' && typeof event.delta === 'string') {
    onEvent({ type: 'delta', delta: event.delta })
    return
  }
  if (event.type === 'done') {
    onEvent({ type: 'done' })
    return
  }
  if (event.type === 'error' && typeof event.code === 'string') {
    onEvent({
      type: 'error',
      code: event.code,
      ...(typeof event.request_id === 'string' ? { request_id: event.request_id } : {}),
    })
    return
  }
  throw new Error('Invalid SSE event')
}

export async function parseSSE(response: Response, onEvent: (event: SSEEvent) => void): Promise<void> {
  const reader = response.body?.getReader()
  if (!reader) return
  const decoder = new TextDecoder()
  let buffer = ''

  const consume = (atEOF = false) => {
    const pendingCR = !atEOF && buffer.endsWith('\r')
    const complete = pendingCR ? buffer.slice(0, -1) : buffer
    buffer = complete.replace(/\r\n/g, '\n').replace(/\r/g, '\n') + (pendingCR ? '\r' : '')
    let boundary = buffer.indexOf('\n\n')
    while (boundary >= 0) {
      dispatchFrame(buffer.slice(0, boundary), onEvent)
      buffer = buffer.slice(boundary + 2)
      boundary = buffer.indexOf('\n\n')
    }
    if (atEOF && buffer.trim()) {
      dispatchFrame(buffer, onEvent)
      buffer = ''
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    consume()
  }
  buffer += decoder.decode()
  consume(true)
}
