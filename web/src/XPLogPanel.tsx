import { useState, useEffect } from 'react'
import { fetchXP, createXP, deleteXP } from './api'
import type { XPEntry } from './types'

interface XPLogPanelProps {
  sessionId: number | null
  lastEvent: unknown
}

export function XPLogPanel({ sessionId, lastEvent }: XPLogPanelProps) {
  const [entries, setEntries] = useState<XPEntry[]>([])
  const [note, setNote] = useState('')
  const [amount, setAmount] = useState('')
  const [adding, setAdding] = useState(false)

  useEffect(() => {
    if (sessionId === null) return
    fetchXP(sessionId).then(setEntries).catch(console.error)
  }, [sessionId])

  useEffect(() => {
    if (lastEvent && sessionId !== null) {
      fetchXP(sessionId).then(setEntries).catch(console.error)
    }
  }, [lastEvent, sessionId])

  async function handleAdd() {
    if (!note.trim() || sessionId === null) return
    setAdding(true)
    try {
      const amt = amount ? parseInt(amount, 10) : undefined
      await createXP(sessionId, note.trim(), isNaN(amt as number) ? undefined : amt)
      setNote('')
      setAmount('')
      const updated = await fetchXP(sessionId)
      setEntries(updated)
    } catch (e) {
      console.error(e)
    } finally {
      setAdding(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteXP(id)
      if (sessionId !== null) {
        const updated = await fetchXP(sessionId)
        setEntries(updated)
      }
    } catch (e) {
      console.error(e)
    }
  }

  if (sessionId === null) return null

  return (
    <div className="xp-log-panel">
      <h4 className="xp-log-title">XP Log</h4>
      {entries.length === 0 && <p className="xp-log-empty">No XP entries yet.</p>}
      {entries.map(e => (
        <div key={e.id} className="xp-log-entry">
          <span className="xp-log-note">{e.note}</span>
          {e.amount != null && <span className="xp-log-amount">+{e.amount}</span>}
          <button className="xp-log-delete" onClick={() => handleDelete(e.id)}>×</button>
        </div>
      ))}
      <div className="xp-log-add">
        <input className="xp-log-input" placeholder="Note…" value={note} onChange={e => setNote(e.target.value)} />
        <input className="xp-log-amount-input" placeholder="XP" type="number" value={amount} onChange={e => setAmount(e.target.value)} />
        <button className="xp-log-btn" onClick={handleAdd} disabled={adding || !note.trim()}>+</button>
      </div>
    </div>
  )
}
