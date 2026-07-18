import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, waitFor } from '@testing-library/react'
import { SessionTimeline } from './SessionTimeline'
import type { TimelineEntry } from './types'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

const msgEntry: TimelineEntry = {
  type: 'message',
  timestamp: '2026-04-03T10:00:00Z',
  data: { id: 1, session_id: 1, role: 'user', content: 'We enter the dungeon.', created_at: '' },
}

const diceEntry: TimelineEntry = {
  type: 'dice_roll',
  timestamp: '2026-04-03T10:01:00Z',
  data: { id: 2, session_id: 1, expression: '1d20+3', result: 17, breakdown_json: '[14]', created_at: '' },
}

describe('SessionTimeline', () => {
  it('renders timeline entries fetched from API', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([msgEntry, diceEntry]),
    }))
    render(<SessionTimeline sessionId={1} lastEvent={null} />)
    expect(await screen.findByText('We enter the dungeon.')).toBeInTheDocument()
    expect(screen.getByText('1d20+3')).toBeInTheDocument()
  })

  it('shows empty state when timeline is empty', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    }))
    render(<SessionTimeline sessionId={1} lastEvent={null} />)
    expect(await screen.findByText('No events yet.')).toBeInTheDocument()
  })

  it('calls fetch with correct session URL', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    render(<SessionTimeline sessionId={42} lastEvent={null} />)
    await screen.findByText('No events yet.')
    expect(mockFetch).toHaveBeenCalledWith('/api/sessions/42/timeline')
  })

  it('appends a dice_rolled WS event as a new timeline entry', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(<SessionTimeline sessionId={1} lastEvent={null} />)
    await screen.findByText('No events yet.')

    rerender(
      <SessionTimeline
        sessionId={1}
        lastEvent={{ type: 'dice_rolled', payload: { session_id: 1, expression: '2d6', total: 8, breakdown: [3, 5] } }}
      />,
    )

    await waitFor(() => {
      expect(screen.getByText('2d6')).toBeInTheDocument()
    })
  })

  it('appends a combat_started WS event as a new timeline entry', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(<SessionTimeline sessionId={1} lastEvent={null} />)
    await screen.findByText('No events yet.')

    rerender(
      <SessionTimeline
        sessionId={1}
        lastEvent={{ type: 'combat_started', payload: { session_id: 1, encounter_id: 1, name: 'Goblin Raid' } }}
      />,
    )

    await waitFor(() => {
      expect(screen.getByText('combat started')).toBeInTheDocument()
      expect(screen.getByText('Goblin Raid')).toBeInTheDocument()
    })
  })

  it('ignores events from another session', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(<SessionTimeline sessionId={1} lastEvent={null} />)
    await screen.findByText('No events yet.')

    rerender(
      <SessionTimeline
        sessionId={1}
        lastEvent={{ type: 'dice_rolled', payload: { session_id: 2, expression: '2d6', total: 8 } }}
      />,
    )
    expect(screen.queryByText('2d6')).not.toBeInTheDocument()
  })
})
