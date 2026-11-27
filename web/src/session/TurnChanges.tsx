import { useCallback, useState, useSyncExternalStore } from 'react'
import { sessionChanges, subscribeToChanges, undoTurnChange, type TurnChange } from './turnChangeStore'
import './turnChanges.css'

function valueText(value: unknown) {
  if (value === undefined) return 'unset'
  if (typeof value === 'object' && value !== null) return JSON.stringify(value)
  return String(value)
}

function description(change: TurnChange) {
  if (change.operation === 'set') return `${valueText(change.before)} → ${valueText(change.after)}`
  const row = change.operation === 'add' ? change.after : change.before
  const verb = change.operation === 'add' ? 'Added' : 'Removed'
  if (!row || typeof row !== 'object') return verb
  const detail = 'description' in row ? row.description : 'note' in row ? row.note : ''
  const qty = 'quantity' in row ? ` × ${row.quantity}` : ''
  return `${verb}${qty}${detail ? ' · ' + detail : ''}`
}

export function TurnChanges({ sessionId, messageId }: { sessionId: number; messageId: number }) {
  const subscribe = useCallback((notify: () => void) => subscribeToChanges(sessionId, notify), [sessionId])
  const changes = useSyncExternalStore(subscribe, () => sessionChanges(sessionId, messageId))
  const [busy, setBusy] = useState<number | null>(null)
  const [error, setError] = useState('')
  if (changes.length === 0) return null

  async function undo(id: number) {
    setBusy(id)
    setError('')
    try { await undoTurnChange(sessionId, id) }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not undo this change.') }
    finally { setBusy(null) }
  }
  return <details className="turn-changes">
    <summary>Changes ({changes.length})</summary>
    <ul>{changes.map(change => <li key={change.id}>
      <div><small>{change.kind === 'npc' ? 'NPC' : change.kind}</small> <strong>{change.label}</strong>
        <span>{description(change)}</span></div>
      {change.can_undo
        ? <button type="button" aria-label={`Undo ${change.label}`} disabled={busy !== null} onClick={() => void undo(change.id)}>Undo</button>
        : <small>{change.corrected_at ? 'Undone' : change.superseded ? 'Changed later' : 'Unavailable'}</small>}
    </li>)}</ul>
    {error && <p role="alert">{error}</p>}
  </details>
}
