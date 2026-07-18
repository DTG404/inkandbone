export interface WsEvent {
  type?: string
  payload?: Record<string, unknown>
}

export function wsEvent(value: unknown): WsEvent | null {
  if (typeof value !== 'object' || value === null) return null
  return value as WsEvent
}

export function isScopedEvent(
  value: unknown,
  type: string,
  scope: string,
  id: number | null,
): value is WsEvent & { payload: Record<string, unknown> } {
  if (id === null) return false
  const event = wsEvent(value)
  return event?.type === type && event.payload?.[scope] === id
}
