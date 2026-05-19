import { useState, useEffect, useRef, useCallback } from 'react'
import { patchSessionSummary, generateRecap, patchSessionNotes, fetchXP, createXP, deleteXP, postImprovise, postPreSessionBrief, postDetectThreads, postCampaignAsk } from './api'
import type { XPEntry } from './types'

interface JournalPanelProps {
  session: { id: number; summary: string; notes: string } | null
  campaignId?: number | null
  lastEvent: unknown
  aiEnabled: boolean
}

interface SessionUpdatedPayload {
  session_id: number
  summary: string
}

interface SessionUpdatedEvent {
  type: 'session_updated'
  payload: SessionUpdatedPayload
}

interface XPAddedPayload {
  session_id: number
  id: number
  note: string
  amount: number | null
}

interface XPAddedEvent {
  type: 'xp_added'
  payload: XPAddedPayload
}

function isSessionUpdatedEvent(ev: unknown): ev is SessionUpdatedEvent {
  if (typeof ev !== 'object' || ev === null) return false
  const e = ev as Record<string, unknown>
  if (e['type'] !== 'session_updated') return false
  const payload = e['payload']
  if (typeof payload !== 'object' || payload === null) return false
  const p = payload as Record<string, unknown>
  return typeof p['session_id'] === 'number' && typeof p['summary'] === 'string'
}

function isXPAddedEvent(ev: unknown): ev is XPAddedEvent {
  if (typeof ev !== 'object' || ev === null) return false
  const e = ev as Record<string, unknown>
  if (e['type'] !== 'xp_added') return false
  const payload = e['payload']
  if (typeof payload !== 'object' || payload === null) return false
  const p = payload as Record<string, unknown>
  return typeof p['session_id'] === 'number' && typeof p['id'] === 'number' && typeof p['note'] === 'string'
}

export function JournalPanel({ session, campaignId, lastEvent, aiEnabled }: JournalPanelProps) {
  const [draft, setDraft] = useState(session?.summary ?? '')
  const [notes, setNotes] = useState(session?.notes ?? '')
  const [xpEntries, setXpEntries] = useState<XPEntry[]>([])
  const [milestoneNote, setMilestoneNote] = useState('')
  const [milestoneXP, setMilestoneXP] = useState('')
  const notesDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [gmToolsOpen, setGmToolsOpen] = useState(false)
  const [gmResult, setGmResult] = useState('')
  const [gmLoading, setGmLoading] = useState(false)
  const [askQuestion, setAskQuestion] = useState('')

  useEffect(() => {
    return () => {
      if (notesDebounceRef.current) {
        clearTimeout(notesDebounceRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (session) {
      setDraft(session.summary)
      setNotes(session.notes ?? '')
    }
  // Intentionally omit session.summary and session.notes: reset only when session changes.
  }, [session?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!session) return
    fetchXP(session.id).then(setXpEntries).catch(console.error)
  }, [session?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!isSessionUpdatedEvent(lastEvent)) return
    if (lastEvent.payload.session_id !== session?.id) return
    setDraft(lastEvent.payload.summary)
  }, [lastEvent, session?.id])

  useEffect(() => {
    if (!isXPAddedEvent(lastEvent)) return
    if (lastEvent.payload.session_id !== session?.id) return
    const incoming = lastEvent.payload
    setXpEntries(prev => {
      if (prev.some(e => e.id === incoming.id)) return prev
      return [...prev, {
        id: incoming.id,
        session_id: incoming.session_id,
        note: incoming.note,
        amount: incoming.amount,
        created_at: new Date().toISOString(),
      }]
    })
  }, [lastEvent, session?.id])

  const handleNotesChange = useCallback((value: string) => {
    setNotes(value)
    if (!session) return
    if (notesDebounceRef.current) clearTimeout(notesDebounceRef.current)
    notesDebounceRef.current = setTimeout(() => {
      patchSessionNotes(session.id, value).catch(console.error)
    }, 300)
  }, [session])

  if (session === null) return null

  function handleBlur() {
    patchSessionSummary(session!.id, draft).catch(console.error)
  }

  async function handleGenerateRecap() {
    const result = await generateRecap(session!.id)
    setDraft(result.summary)
  }

  async function handleAddMilestone(e: React.FormEvent) {
    e.preventDefault()
    const note = milestoneNote.trim()
    if (!note) return
    const parsed = parseInt(milestoneXP, 10)
    const amount = milestoneXP.trim() !== '' && !isNaN(parsed) ? parsed : undefined
    try {
      const entry = await createXP(session!.id, note, amount)
      setXpEntries(prev => [...prev, entry])
    } catch (err) {
      console.error('Failed to add milestone:', err)
    } finally {
      setMilestoneNote('')
      setMilestoneXP('')
    }
  }

  async function handleDeleteMilestone(id: number) {
    try {
      await deleteXP(id)
      setXpEntries(prev => prev.filter(e => e.id !== id))
    } catch (err) {
      console.error('Failed to delete milestone:', err)
    }
  }

  async function runTool(fn: () => Promise<string>) {
    setGmLoading(true)
    setGmResult('')
    try {
      const result = await fn()
      setGmResult(result)
    } catch {
      setGmResult('Error: tool failed')
    } finally {
      setGmLoading(false)
    }
  }

  return (
    <>
      <textarea
        className="journal-textarea"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={handleBlur}
        placeholder="Your session journal…"
      />
      {aiEnabled && (
        <button className="ai-text-btn" onClick={handleGenerateRecap}>
          Generate recap
        </button>
      )}

      <div className="scratchpad-section">
        <span className="scratchpad-label">Notes</span>
        <textarea
          className="scratchpad-textarea"
          value={notes}
          onChange={(e) => handleNotesChange(e.target.value)}
          placeholder="Quick notes for this session…"
        />
      </div>

      <div className="milestones-section">
        <div className="milestones-header">Milestones</div>
        {xpEntries.map(entry => (
          <div key={entry.id} className="milestone-entry">
            <span className="milestone-note">{entry.note}</span>
            {entry.amount != null && (
              <span className="milestone-amount">{entry.amount} XP</span>
            )}
            <button
              className="milestone-delete"
              onClick={() => handleDeleteMilestone(entry.id)}
              aria-label="Delete milestone"
            >
              ×
            </button>
          </div>
        ))}
        <form className="milestone-add-form" onSubmit={handleAddMilestone}>
          <input
            className="milestone-note-input"
            type="text"
            value={milestoneNote}
            onChange={(e) => setMilestoneNote(e.target.value)}
            placeholder="＋ Add milestone"
          />
          <input
            className="milestone-xp-input"
            type="text" inputMode="numeric" pattern="[0-9]*"
            value={milestoneXP}
            onChange={(e) => setMilestoneXP(e.target.value)}
            placeholder="XP"
            min={0}
          />
          <button
            className="milestone-submit"
            type="submit"
            disabled={milestoneNote.trim() === ''}
          >
            Add
          </button>
        </form>
      </div>

      {aiEnabled && (
        <div className="gm-tools-section">
          <div
            className="gm-tools-header"
            onClick={() => setGmToolsOpen(o => !o)}
          >
            <span className="gm-tools-toggle">{gmToolsOpen ? '▼' : '▶'}</span>
            GM Tools
          </div>
          {gmToolsOpen && (
            <div className="gm-tools-body">
              <div className="gm-tools-buttons">
                <button
                  className="gm-tool-btn"
                  disabled={gmLoading || !session}
                  onClick={() => runTool(() => postImprovise(session!.id))}
                >
                  {gmLoading ? '…' : 'Improvise'}
                </button>
                <button
                  className="gm-tool-btn"
                  disabled={gmLoading || !campaignId}
                  onClick={() => runTool(() => postPreSessionBrief(campaignId!))}
                >
                  {gmLoading ? '…' : 'Pre-Session Brief'}
                </button>
                <button
                  className="gm-tool-btn"
                  disabled={gmLoading || !session}
                  onClick={() => runTool(() => postDetectThreads(session!.id))}
                >
                  {gmLoading ? '…' : 'Detect Threads'}
                </button>
              </div>
              <div className="gm-tools-ask">
                <input
                  className="gm-tools-ask-input"
                  type="text"
                  value={askQuestion}
                  onChange={e => setAskQuestion(e.target.value)}
                  placeholder="Ask about the campaign..."
                />
                <button
                  className="gm-tool-btn"
                  disabled={gmLoading || !askQuestion.trim() || !campaignId}
                  onClick={() => runTool(() => postCampaignAsk(campaignId!, askQuestion))}
                >
                  {gmLoading ? '…' : 'Ask'}
                </button>
              </div>
              {gmResult && (
                <blockquote className="gm-tools-result">{gmResult}</blockquote>
              )}
            </div>
          )}
        </div>
      )}
    </>
  )
}
