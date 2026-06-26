import { useState, useEffect, useCallback } from 'react'
import ReactMarkdown from 'react-markdown'
import { fetchWorldNotes } from './api'
import type { WorldNote } from './types'

interface Props {
  campaignId: number
  lastEvent: unknown
}

export function HandoutsPanel({ campaignId, lastEvent }: Props) {
  const [notes, setNotes] = useState<WorldNote[]>([])

  const load = useCallback(() => {
    fetchWorldNotes(campaignId, undefined, undefined, true)
      .then(setNotes)
      .catch(console.error)
  }, [campaignId])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    const e = lastEvent as { type?: string } | null
    if (e?.type === 'world_note_revealed') load()
  }, [lastEvent, load])

  if (notes.length === 0) {
    return <p className="panel-empty">No handouts revealed yet.</p>
  }

  return (
    <div className="handouts-panel">
      {notes.map((n) => (
        <div key={n.id} className="handout-card">
          <div className="handout-header">
            <strong>{n.title}</strong>
            {n.category && <span className="category-badge">{n.category}</span>}
          </div>
          <div className="handout-content">
            <ReactMarkdown>{n.content}</ReactMarkdown>
          </div>
        </div>
      ))}
    </div>
  )
}
