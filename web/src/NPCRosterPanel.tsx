import { useState, useEffect } from 'react'
import { fetchNPCs, createNPC, patchNPC, deleteNPC, reanalyzeSession } from './api'
import type { SessionNPC } from './types'
import { isScopedEvent } from './wsEvents'

interface Props {
  sessionId: number | null
  lastEvent: unknown
}

export function NPCRosterPanel({ sessionId, lastEvent }: Props) {
  const [npcs, setNpcs] = useState<SessionNPC[]>([])
  const [showAddForm, setShowAddForm] = useState(false)
  const [addName, setAddName] = useState('')
  const [addNote, setAddNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [reanalyzing, setReanalyzing] = useState(false)

  useEffect(() => {
    if (sessionId === null) return
    fetchNPCs(sessionId).then(setNpcs).catch(() => setNpcs([]))
  }, [sessionId])

  useEffect(() => {
    if (sessionId !== null && isScopedEvent(lastEvent, 'npc_updated', 'session_id', sessionId)) {
      fetchNPCs(sessionId).then(setNpcs).catch(() => {})
    }
  }, [lastEvent, sessionId])

  async function handleAdd() {
    if (!addName.trim() || sessionId === null) return
    setSaving(true)
    try {
      const npc = await createNPC(sessionId, addName.trim(), addNote.trim())
      setNpcs((prev) => [...prev, npc])
      setAddName('')
      setAddNote('')
      setShowAddForm(false)
    } catch (err) {
      console.error(err)
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteNPC(id)
      setNpcs((prev) => prev.filter((n) => n.id !== id))
    } catch (err) {
      console.error(err)
    }
  }

  function handleNoteBlur(npc: SessionNPC, note: string) {
    if (note === npc.note) return
    patchNPC(npc.id, note).catch(console.error)
  }

  async function handleReanalyze() {
    if (sessionId === null || reanalyzing) return
    setReanalyzing(true)
    try {
      await reanalyzeSession(sessionId)
    } catch (err) {
      console.error(err)
    } finally {
      setReanalyzing(false)
    }
  }

  if (sessionId === null) return <p className="empty">No active session.</p>

  return (
    <div className="npc-roster">
      {npcs.length === 0 && !showAddForm && (
        <p className="empty">No NPCs yet.</p>
      )}
      {npcs.map((npc) => (
        <NPCCard
          key={npc.id}
          npc={npc}
          onDelete={() => handleDelete(npc.id)}
          onNoteBlur={(note) => handleNoteBlur(npc, note)}
        />
      ))}

      <button
        className="reanalyze-btn"
        onClick={handleReanalyze}
        disabled={reanalyzing}
        title="Re-run AI analysis on full session history to add/remove NPCs"
      >
        {reanalyzing ? 'Analyzing…' : '↻ Reanalyze'}
      </button>

      {showAddForm ? (
        <div className="npc-add-form">
          <input
            className="npc-input"
            placeholder="NPC name…"
            value={addName}
            onChange={(e) => setAddName(e.target.value)}
            autoFocus
          />
          <textarea
            className="npc-note-input"
            placeholder="Notes…"
            value={addNote}
            onChange={(e) => setAddNote(e.target.value)}
            rows={3}
          />
          <div className="npc-add-actions">
            <button className="npc-save-btn" onClick={handleAdd} disabled={saving || !addName.trim()}>
              {saving ? 'Saving…' : 'Save'}
            </button>
            <button className="npc-cancel-btn" onClick={() => { setShowAddForm(false); setAddName(''); setAddNote('') }}>
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button className="npc-add-btn" onClick={() => setShowAddForm(true)}>
          + Add NPC
        </button>
      )}
    </div>
  )
}

function NPCCard({
  npc,
  onDelete,
  onNoteBlur,
}: {
  npc: SessionNPC
  onDelete: () => void
  onNoteBlur: (note: string) => void
}) {
  const [note, setNote] = useState(npc.note)

  useEffect(() => {
    setNote(npc.note)
  }, [npc.note])

  return (
    <div className="npc-card">
      <div className="npc-card-header">
        <span className="npc-name">{npc.name}</span>
        <button className="npc-delete-btn" onClick={onDelete} title="Remove NPC">×</button>
      </div>
      <textarea
        className="npc-note-input"
        value={note}
        onChange={(e) => setNote(e.target.value)}
        onBlur={() => onNoteBlur(note)}
        placeholder="Notes…"
        rows={2}
      />
    </div>
  )
}
