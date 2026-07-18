import { parseRealtimeEvent, type RealtimeEvent } from './realtime.gen'

export type WsEvent = RealtimeEvent

export function wsEvent(value: unknown): WsEvent | null {
  return parseRealtimeEvent(value)
}

export function isScopedEvent<T extends RealtimeEvent['type']>(
  value: unknown,
  type: T,
  scope: string,
  id: number | null,
): value is Extract<RealtimeEvent, { type: T }> {
  if (id === null) return false
  const event = wsEvent(value)
  return event?.type === type && Reflect.get(event.payload, scope) === id
}
