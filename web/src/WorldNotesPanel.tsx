import { useState, useEffect, useCallback } from 'react'
import { fetchWorldNotes, draftWorldNote, patchWorldNotePersonality, patchWorldNoteRevealed } from './api'
import type { WorldNote } from './types'

interface Props {
  campaignId: number
  lastEvent: unknown
  aiEnabled: boolean
}

function parseTags(json: string): string[] {
  try { return JSON.parse(json) as string[] }
  catch { return [] }
}

function PersonalityEditor({ note, onSaved }: { note: WorldNote; onSaved: () => void }) {
  const [value, setValue] = useState(note.personality_json || '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    setSaving(true)
    setError(null)
    try {
      await patchWorldNotePersonality(note.id, value)
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="personality-editor">
      <label className="personality-label">Personality JSON</label>
      <textarea
        className="personality-textarea"
        rows={3}
        placeholder='{"traits":["brave"],"motivation":"justice"}'
        value={value}
        onChange={(e) => setValue(e.target.value)}
      />
      {error && <p className="personality-error">{error}</p>}
      <button className="personality-save-btn" disabled={saving} onClick={handleSave}>
        {saving ? 'Saving…' : 'Save Personality'}
      </button>
    </div>
  )
}

export function WorldNotesPanel({ campaignId, lastEvent, aiEnabled }: Props) {
  const [notes, setNotes] = useState<WorldNote[]>([])
  const [query, setQuery] = useState('')
  const [activeTag, setActiveTag] = useState<string | null>(null)
  const [drafting, setDrafting] = useState(false)

  const loadNotes = useCallback(() => {
    fetchWorldNotes(campaignId, query || undefined, activeTag || undefined)
      .then(setNotes)
      .catch(() => setNotes([]))
  }, [campaignId, query, activeTag])

  useEffect(() => {
    loadNotes()
  }, [loadNotes])

  useEffect(() => {
    const ev = lastEvent as { type?: string } | null
    if (ev?.type === 'world_note_revealed' || ev?.type === 'world_note_created' || ev?.type === 'world_note_updated') {
      loadNotes()
    }
  }, [lastEvent, loadNotes])

  async function handleDraftWithAI() {
    const hint = window.prompt('Describe the note:')
    if (!hint) return
    setDrafting(true)
    try {
      await draftWorldNote(campaignId, hint)
    } catch (err) {
      console.error(err)
    } finally {
      setDrafting(false)
    }
  }

  return (
    <>
      <input
        className="notes-search"
        type="search"
        placeholder="Search notes…"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      {notes.length === 0 ? (
        <p className="empty">No notes found.</p>
      ) : (
        notes.map((n) => {
          const tags = parseTags(n.tags_json)
          return (
            <div key={n.id} className="note-card">
              <div className="note-title">
                {n.title}
                <button
                  className="reveal-toggle-btn"
                  title={n.is_revealed ? 'Hide from players' : 'Reveal to players'}
                  onClick={async () => {
                    await patchWorldNoteRevealed(n.id, !n.is_revealed)
                    loadNotes()
                  }}
                >
                  {n.is_revealed ? '✅' : '📤'}
                </button>
                {n.is_revealed && <span className="revealed-badge">Revealed</span>}
              </div>
              {tags.length > 0 && (
                <div className="note-tags">
                  {tags.map((tag) => (
                    <button
                      key={tag}
                      className={`note-tag${activeTag === tag ? ' active' : ''}`}
                      onClick={() => setActiveTag((t) => (t === tag ? null : tag))}
                    >
                      {tag}
                    </button>
                  ))}
                </div>
              )}
              <p className="note-content">{n.content}</p>
              {n.category === 'npc' && (
                <PersonalityEditor note={n} onSaved={loadNotes} />
              )}
            </div>
          )
        })
      )}
      {aiEnabled && (
        <button className="ai-text-btn" disabled={drafting} onClick={handleDraftWithAI}>
          {drafting ? 'Drafting…' : 'Draft with AI'}
        </button>
      )}
    </>
  )
}
