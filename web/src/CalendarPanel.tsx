import { useState, useEffect, useCallback } from 'react'
import { getCampaignCalendar, patchCampaignCalendar, listCalendarEvents, createCalendarEvent, deleteCalendarEvent } from './api'
import type { CampaignCalendarInfo, CalendarEvent } from './types'

const EVENT_TYPES = ['note', 'battle', 'festival', 'ceremony', 'death', 'birth', 'discovery', 'disaster', 'meeting', 'travel']

interface CalendarPanelProps {
  campaignId: number
  sessionId: number | null
  lastEvent: unknown
}

export function CalendarPanel({ campaignId, sessionId, lastEvent }: CalendarPanelProps) {
  const [info, setInfo] = useState<CampaignCalendarInfo | null>(null)
  const [events, setEvents] = useState<CalendarEvent[]>([])
  const [loading, setLoading] = useState(true)

  const [showForm, setShowForm] = useState(false)
  const [formTitle, setFormTitle] = useState('')
  const [formDesc, setFormDesc] = useState('')
  const [formYear, setFormYear] = useState(1)
  const [formMonth, setFormMonth] = useState(1)
  const [formDay, setFormDay] = useState(1)
  const [formType, setFormType] = useState('note')
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    try {
      const [calInfo, calEvents] = await Promise.all([
        getCampaignCalendar(campaignId),
        listCalendarEvents(campaignId),
      ])
      setInfo(calInfo)
      setEvents(calEvents)
      setFormYear(calInfo.in_game_year)
      setFormMonth(calInfo.in_game_month)
      setFormDay(calInfo.in_game_day)
    } catch (err) {
      console.error('Failed to load calendar:', err)
    } finally {
      setLoading(false)
    }
  }, [campaignId])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    if (lastEvent) load()
  }, [lastEvent, load])

  async function handleAdvance(days: number) {
    try {
      const updated = await patchCampaignCalendar(campaignId, { advance_days: days })
      setInfo(updated)
      setFormYear(updated.in_game_year)
      setFormMonth(updated.in_game_month)
      setFormDay(updated.in_game_day)
    } catch (err) {
      console.error('Failed to advance date:', err)
    }
  }

  async function handleCreateEvent(e: React.FormEvent) {
    e.preventDefault()
    if (!formTitle.trim()) return
    setSaving(true)
    try {
      await createCalendarEvent(campaignId, {
        in_game_year: formYear,
        in_game_month: formMonth,
        in_game_day: formDay,
        title: formTitle.trim(),
        description: formDesc.trim(),
        event_type: formType,
        session_id: sessionId,
      })
      setFormTitle('')
      setFormDesc('')
      setShowForm(false)
      await load()
    } catch (err) {
      console.error('Failed to create event:', err)
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteCalendarEvent(id)
      await load()
    } catch (err) {
      console.error('Failed to delete event:', err)
    }
  }

  if (loading) {
    return <div className="panel-loading">Loading calendar…</div>
  }

  return (
    <div className="calendar-panel">
      <div className="calendar-date-display">
        <span className="calendar-date-label">Current Date</span>
        <span className="calendar-date-value">
          Year {info?.in_game_year ?? 1}, Month {info?.in_game_month ?? 1}, Day {info?.in_game_day ?? 1}
        </span>
      </div>

      <div className="calendar-advance-row">
        <button className="calendar-advance-btn" onClick={() => handleAdvance(1)} title="Advance 1 day">+1 Day</button>
        <button className="calendar-advance-btn" onClick={() => handleAdvance(7)} title="Advance 7 days">+1 Week</button>
        <button className="calendar-advance-btn" onClick={() => handleAdvance(30)} title="Advance 30 days">+1 Month</button>
      </div>

      <div className="calendar-section-header">
        <span>Events</span>
        <button className="calendar-add-btn" onClick={() => setShowForm(v => !v)}>
          {showForm ? '−' : '+'}
        </button>
      </div>

      {showForm && (
        <form className="calendar-event-form" onSubmit={handleCreateEvent}>
          <div className="calendar-form-row">
            <label>Year</label>
            <input type="text" inputMode="numeric" pattern="[0-9]*" value={formYear} onChange={e => setFormYear(Number(e.target.value))} min={1} />
          </div>
          <div className="calendar-form-row">
            <label>Month</label>
            <input type="text" inputMode="numeric" pattern="[0-9]*" value={formMonth} onChange={e => setFormMonth(Number(e.target.value))} min={1} max={12} />
          </div>
          <div className="calendar-form-row">
            <label>Day</label>
            <input type="text" inputMode="numeric" pattern="[0-9]*" value={formDay} onChange={e => setFormDay(Number(e.target.value))} min={1} max={30} />
          </div>
          <div className="calendar-form-row">
            <label>Type</label>
            <select value={formType} onChange={e => setFormType(e.target.value)}>
              {EVENT_TYPES.map(t => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>
          <div className="calendar-form-row">
            <label>Title</label>
            <input value={formTitle} onChange={e => setFormTitle(e.target.value)} placeholder="Event title…" required />
          </div>
          <div className="calendar-form-row">
            <label>Description</label>
            <textarea value={formDesc} onChange={e => setFormDesc(e.target.value)} placeholder="Optional description…" rows={2} />
          </div>
          <button type="submit" className="calendar-form-submit" disabled={saving || !formTitle.trim()}>
            {saving ? '…' : 'Create Event'}
          </button>
        </form>
      )}

      <div className="calendar-events-list">
        {events.length === 0 ? (
          <p className="calendar-empty">No events recorded.</p>
        ) : (
          events.map(ev => (
            <div key={ev.id} className={`calendar-event-card calendar-event-type-${ev.event_type}`}>
              <div className="calendar-event-meta">
                <span className="calendar-event-date">Y{ev.in_game_year} M{ev.in_game_month} D{ev.in_game_day}</span>
                <span className="calendar-event-type-badge">{ev.event_type}</span>
                {ev.session_id && (
                  <span className="calendar-event-session-link" title="Linked to session">🔗</span>
                )}
                <button
                  className="calendar-event-delete"
                  onClick={() => handleDelete(ev.id)}
                  title="Delete event"
                >×</button>
              </div>
              <div className="calendar-event-title">{ev.title}</div>
              {ev.description && <div className="calendar-event-desc">{ev.description}</div>}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
