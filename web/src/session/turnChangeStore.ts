import { request } from '../api/transport'

export interface TurnChange {
  id: number
  message_id: number
  kind: 'stat' | 'currency' | 'objective' | 'npc' | 'item'
  label: string
  operation: 'set' | 'add' | 'remove'
  before?: unknown
  after?: unknown
  can_undo: boolean
  superseded: boolean
  corrected_at?: string
}

const empty: readonly TurnChange[] = []
const sessions = new Map<number, SessionChanges>()

// Every visible reply subscribes to the same session read. A websocket burst
// during that read schedules a final refresh so its latest write is visible.
class SessionChanges {
  byMessage = new Map<number, TurnChange[]>()
  listeners = new Set<() => void>()
  reading = false
  pending = false

  readonly session: number

  constructor(session: number) { this.session = session }

  async refresh() {
    if (this.reading) { this.pending = true; return }
    this.reading = true
    try {
      const res = await request(`/api/sessions/${this.session}/changes`)
      if (res.ok) {
        const rows = await res.json()
        if (Array.isArray(rows)) {
          this.byMessage = new Map()
          for (const row of rows as TurnChange[]) {
            const group = this.byMessage.get(row.message_id)
            if (group) group.push(row)
            else this.byMessage.set(row.message_id, [row])
          }
          this.listeners.forEach(notify => notify())
        }
      }
    } catch {
      // Retain the last confirmed journal until reconnect or the next update.
    } finally {
      this.reading = false
      if (this.pending && this.listeners.size) {
        this.pending = false
        void this.refresh()
      }
    }
  }
}

export function subscribeToChanges(session: number, notify: () => void) {
  let entry = sessions.get(session)
  if (!entry) { entry = new SessionChanges(session); sessions.set(session, entry) }
  entry.listeners.add(notify)
  if (entry.listeners.size === 1) void entry.refresh()
  return () => {
    entry.listeners.delete(notify)
    if (!entry.listeners.size) sessions.delete(session)
  }
}

export function sessionChanges(session: number, message: number) {
  return sessions.get(session)?.byMessage.get(message) ?? empty
}

export function refreshTurnChanges() {
  sessions.forEach(entry => { void entry.refresh() })
}

export async function undoTurnChange(session: number, id: number) {
  const response = await request(`/api/sessions/${session}/changes/${id}/undo`, { method: 'POST' })
  void sessions.get(session)?.refresh()
  if (!response.ok) throw new Error(response.status === 409 ? 'Later edits prevent undoing this change.' : 'Could not undo this change. Try again.')
}
