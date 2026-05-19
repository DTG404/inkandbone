import { useState, useEffect } from 'react'
import { listAdventures, createAdventure, updateAdventure, deleteAdventure, fetchSessions } from './api'
import type { Adventure, Session } from './types'

interface AdventuresPanelProps {
  campaignId: number
  onSessionClick: (sessionId: number) => void
  lastEvent: unknown
}

const STATUS_OPTIONS = ['upcoming', 'active', 'completed', 'abandoned']
const STATUS_LABELS: Record<string, string> = {
  upcoming: 'Upcoming',
  active: 'Active',
  completed: 'Completed',
  abandoned: 'Abandoned',
}

export function AdventuresPanel({ campaignId, onSessionClick, lastEvent }: AdventuresPanelProps) {
  const [adventures, setAdventures] = useState<Adventure[]>([])
  const [sessions, setSessions] = useState<Session[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState('upcoming')
  const [expandedId, setExpandedId] = useState<number | null>(null)

  useEffect(() => {
    listAdventures(campaignId).then(setAdventures).catch(() => {})
    fetchSessions(campaignId).then(setSessions).catch(() => {})
  }, [campaignId, lastEvent])

  function resetForm() {
    setTitle('')
    setDescription('')
    setStatus('upcoming')
    setEditId(null)
    setShowForm(false)
  }

  function startEdit(a: Adventure) {
    setEditId(a.id)
    setTitle(a.title)
    setDescription(a.description)
    setStatus(a.status)
    setShowForm(true)
  }

  async function handleSave() {
    if (!title.trim()) return
    try {
      if (editId !== null) {
        await updateAdventure(editId, title.trim(), description, status)
      } else {
        await createAdventure(campaignId, title.trim(), description, status)
      }
      const updated = await listAdventures(campaignId)
      setAdventures(updated)
      resetForm()
    } catch (err) {
      console.error('Failed to save adventure:', err)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteAdventure(id)
      setAdventures(adventures.filter(a => a.id !== id))
    } catch (err) {
      console.error('Failed to delete adventure:', err)
    }
  }

  async function quickToggleStatus(a: Adventure) {
    const nextStatus = a.status === 'active' ? 'completed' : a.status === 'completed' ? 'upcoming' : 'active'
    try {
      await updateAdventure(a.id, a.title, a.description, nextStatus)
      const updated = await listAdventures(campaignId)
      setAdventures(updated)
    } catch (err) {
      console.error('Failed to update adventure status:', err)
    }
  }

  function sessionsForAdventure(adventureId: number): Session[] {
    return sessions.filter(s => s.adventure_id === adventureId)
  }

  return (
    <div className="adventures-panel">
      <h3>Adventures</h3>
      <button onClick={() => { resetForm(); setShowForm(f => !f) }}>
        {showForm ? 'Cancel' : '+ New Adventure'}
      </button>

      {showForm && (
        <div className="adventure-form">
          <input
            placeholder="Adventure title"
            value={title}
            onChange={e => setTitle(e.target.value)}
          />
          <select value={status} onChange={e => setStatus(e.target.value)}>
            {STATUS_OPTIONS.map(opt => (
              <option key={opt} value={opt}>{STATUS_LABELS[opt]}</option>
            ))}
          </select>
          <textarea
            placeholder="Description"
            value={description}
            onChange={e => setDescription(e.target.value)}
            rows={3}
          />
          <button onClick={handleSave} disabled={!title.trim()}>
            {editId !== null ? 'Update' : 'Save'}
          </button>
        </div>
      )}

      <ul className="adventure-list">
        {adventures.map(a => (
          <li key={a.id} className="adventure-item">
            <div className="adventure-header" onClick={() => setExpandedId(expandedId === a.id ? null : a.id)}>
              <strong className="adventure-name">{a.title}</strong>
              <span className={`adventure-status-badge status-${a.status}`}>{STATUS_LABELS[a.status] ?? a.status}</span>
              <button
                className="adventure-toggle-status"
                onClick={(e) => { e.stopPropagation(); quickToggleStatus(a) }}
                title="Toggle status"
              >
                ↻
              </button>
              <span className="adventure-expand-icon">{expandedId === a.id ? '▴' : '▾'}</span>
            </div>
            {expandedId === a.id && (
              <div className="adventure-body">
                {a.description && <p className="adventure-description">{a.description}</p>}
                <div className="adventure-sessions-section">
                  <div className="adventure-sessions-label">Sessions ({sessionsForAdventure(a.id).length})</div>
                  {sessionsForAdventure(a.id).length > 0 ? (
                    <ul className="adventure-sessions-list">
                      {sessionsForAdventure(a.id).map(s => (
                        <li
                          key={s.id}
                          className="adventure-session-item"
                          onClick={() => onSessionClick(s.id)}
                        >
                          {s.title}
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="adventure-empty-sessions">No sessions assigned</p>
                  )}
                </div>
                <div className="adventure-actions">
                  <button onClick={(e) => { e.stopPropagation(); startEdit(a) }}>Edit</button>
                  <button onClick={(e) => { e.stopPropagation(); handleDelete(a.id) }}>Delete</button>
                </div>
              </div>
            )}
          </li>
        ))}
      </ul>


    </div>
  )
}
