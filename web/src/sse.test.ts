import { describe, expect, it } from 'vitest'
import { parseSSE, type SSEEvent } from './sse'

function responseFrom(parts: Uint8Array[]): Response {
  return new Response(new ReadableStream({
    start(controller) {
      for (const part of parts) controller.enqueue(part)
      controller.close()
    },
  }))
}

describe('parseSSE', () => {
  it('preserves split UTF-8, embedded newlines, CRLF, and multiple frames', async () => {
    const bytes = new TextEncoder().encode(
      'data: {"type":"delta","delta":"first\\nsecond ☃"}\r\n\r\n' +
      'data:{"type":"delta",\r\n' +
      'data: "delta":"!"}\r\n\r\n' +
      'data: {"type":"done"}\n\n',
    )
    const events: SSEEvent[] = []
    await parseSSE(responseFrom([
      bytes.slice(0, 8), bytes.slice(8, 51), bytes.slice(51, 52), bytes.slice(52, 75), bytes.slice(75),
    ]), (event) => events.push(event))
    expect(events).toEqual([
      { type: 'delta', delta: 'first\nsecond ☃' },
      { type: 'delta', delta: '!' },
      { type: 'done' },
    ])
  })

  it('flushes a final frame at EOF and dispatches errors', async () => {
    const events: SSEEvent[] = []
    await parseSSE(responseFrom([
      new TextEncoder().encode('data: {"type":"error","code":"gm_failed","request_id":"req-1"}'),
    ]), (event) => events.push(event))
    expect(events).toEqual([{ type: 'error', code: 'gm_failed', request_id: 'req-1' }])
  })
})
