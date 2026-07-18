import { useState, useEffect, useCallback } from 'react'
import ReactMarkdown from 'react-markdown'
import { fetchWorldNotes } from './api'
import type { WorldNote } from './types'
import { isScopedEvent } from './wsEvents'
import { useToast } from './ui/ToastProvider'

interface Props {
  campaignId: number
  lastEvent: unknown
}

export function HandoutsPanel({ campaignId, lastEvent }: Props) {
  const [notes, setNotes] = useState<WorldNote[]>([])
  const toast = useToast()

  const load = useCallback(() => {
    fetchWorldNotes(campaignId, undefined, undefined, true)
      .then(setNotes)
      .catch((error) => {
        console.error(error)
        toast.error('Handouts could not be refreshed. Try again.')
      })
  }, [campaignId, toast])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    if (isScopedEvent(lastEvent, 'world_note_revealed', 'campaign_id', campaignId)) load()
  }, [lastEvent, campaignId, load])

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
          {(() => {
            try { return JSON.parse(n.tags_json) as string[] } catch { return [] }
          })().length > 0 && (
            <div className="handout-tags">
              {(() => {
                try { return JSON.parse(n.tags_json) as string[] } catch { return [] }
              })().map(tag => (
                <span key={tag} className="handout-tag">{tag}</span>
              ))}
            </div>
          )}
          <div className="handout-content">
            <ReactMarkdown>{n.content}</ReactMarkdown>
          </div>
        </div>
      ))}
    </div>
  )
}
