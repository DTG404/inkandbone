import { useState, useEffect, useCallback, useRef } from 'react'
import { fetchTimeline } from './api'
import type { TimelineEntry } from './types'

interface Props {
  sessionId: number
  lastEvent: unknown
}

type WsPayload = Record<string, unknown>
type WsEvent = { type?: string; payload?: WsPayload }

// Build a TimelineEntry from a WS event payload. Returns null if the event
// type is not one the timeline cares about.
function wsToEntry(ev: WsEvent): TimelineEntry | null {
  const now = new Date().toISOString()
  const p = ev.payload ?? {}

  switch (ev.type) {
    case 'dice_rolled':
      return {
        type: 'dice_roll',
        timestamp: now,
        data: {
          expression: p.expression as string,
          result: p.result as number,
          breakdown_json: JSON.stringify(p.breakdown ?? []),
        },
      }
    case 'world_note_created':
      return {
        type: 'world_note_event',
        timestamp: now,
        data: { note_id: p.note_id as number, title: p.title as string, action: 'created' },
      }
    case 'combat_started':
      return {
        type: 'combat_event',
        timestamp: now,
        data: { encounter_id: p.encounter_id as number, name: p.name as string, ended: false },
      }
    case 'combat_ended':
      return {
        type: 'combat_event',
        timestamp: now,
        data: { encounter_id: p.encounter_id as number, ended: true },
      }
    default:
      return null
  }
}

export function SessionTimeline({ sessionId, lastEvent }: Props) {
  const [entries, setEntries] = useState<TimelineEntry[]>([])
  const [newCount, setNewCount] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const newCountTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const loadTimeline = useCallback(() => {
    let ignored = false
    fetchTimeline(sessionId)
      .then((data) => { if (!ignored) setEntries(data) })
      .catch(() => { if (!ignored) setError('Could not load timeline') })
    return () => { ignored = true }
  }, [sessionId])

  useEffect(() => loadTimeline(), [loadTimeline])

  useEffect(() => {
    const ev = lastEvent as WsEvent | null
    if (!ev?.type) return
    if (ev.payload?.session_id !== sessionId) return
    const entry = wsToEntry(ev)
    if (!entry) return

    setEntries((prev) => [...prev, entry])
    setNewCount((c) => c + 1)

    if (newCountTimerRef.current) clearTimeout(newCountTimerRef.current)
    newCountTimerRef.current = setTimeout(() => {
      setNewCount((c) => Math.max(0, c - 1))
    }, 600)

    return () => {
      if (newCountTimerRef.current) clearTimeout(newCountTimerRef.current)
    }
  }, [lastEvent, sessionId])

  if (error) return <p className="error">{error}</p>

  return (
    <section className="panel timeline-panel">
      <h2>Session Timeline</h2>
      {entries.length === 0 ? (
        <p className="empty">No events yet.</p>
      ) : (
        <div className="timeline-feed">
          {entries.map((e, idx) => {
            const isNew = idx >= entries.length - newCount
            return (
              <div
                key={`${e.type}-${idx}`}
                className={`timeline-entry timeline-${e.type}${isNew ? ' entry-new' : ''}`}
              >
                {renderEntry(e)}
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}

function renderEntry(e: TimelineEntry) {
  const d = e.data

  if (e.type === 'message') {
    return (
      <>
        <span className="tl-role">{String(d.role ?? '')}</span>
        <span className="tl-content">{String(d.content ?? '')}</span>
      </>
    )
  }

  if (e.type === 'dice_roll') {
    const breakdown = (() => {
      try {
        return JSON.parse(String(d.breakdown_json ?? '[]')) as number[]
      } catch {
        return []
      }
    })()
    return (
      <>
        <span className="tl-expr">{String(d.expression ?? '')}</span>
        <span className="tl-result">{String(d.result ?? '')}</span>
        {breakdown.length > 0 && (
          <span className="tl-breakdown">
            {breakdown.map((v, i) => (
              <span key={i} className="die-badge">
                [{v}]
              </span>
            ))}
          </span>
        )}
      </>
    )
  }

  if (e.type === 'world_note_event') {
    return (
      <>
        <span className="tl-badge note">note</span>
        <span className="tl-content">{String(d.title ?? '')}</span>
      </>
    )
  }

  if (e.type === 'combat_event') {
    const ended = Boolean(d.ended)
    return (
      <>
        <span className="tl-badge combat">{ended ? 'combat ended' : 'combat started'}</span>
        {!ended && <span className="tl-content">{String(d.name ?? '')}</span>}
      </>
    )
  }

  return null
}
