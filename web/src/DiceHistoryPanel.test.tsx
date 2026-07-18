import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { DiceHistoryPanel } from './DiceHistoryPanel'
import type { DiceRoll } from './types'

const rolls: DiceRoll[] = [
  { id: 1, session_id: 1, expression: '1d20+5', result: 18, breakdown_json: '[]', created_at: '' },
  { id: 2, session_id: 1, expression: '2d6', result: 7, breakdown_json: '[]', created_at: '' },
]

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('DiceHistoryPanel', () => {
  it('renders dice rolls', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(rolls) }))
    render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    expect(await screen.findByText('1d20+5')).toBeInTheDocument()
    expect(screen.getByText('18')).toBeInTheDocument()
    expect(screen.getByText('2d6')).toBeInTheDocument()
    expect(screen.getByText('7')).toBeInTheDocument()
  })

  it('fetches rolls on mount', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    await vi.waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/sessions/1/dice-rolls')
    })
  })

  it('renders breakdown badges from breakdown_json', async () => {
    const withBreakdown: DiceRoll[] = [
      { id: 1, session_id: 1, expression: '3d6', result: 12, breakdown_json: '[4,3,5]', created_at: '' },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(withBreakdown) }))
    render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    expect(await screen.findByText('3d6')).toBeInTheDocument()
    expect(screen.getByText('12')).toBeInTheDocument()
  })

  it('does not render badges when breakdown_json is empty array', async () => {
    const noBreakdown: DiceRoll[] = [
      { id: 1, session_id: 1, expression: '1d20', result: 15, breakdown_json: '[]', created_at: '' },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(noBreakdown) }))
    render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    await screen.findByText('1d20')
    expect(screen.queryByText(/^\[/)).toBeNull()
  })

  it('adds live roll to feed on dice_rolled event without refetching', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    const { rerender } = render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledOnce())
    rerender(<DiceHistoryPanel sessionId={1} lastEvent={{
      type: 'dice_rolled',
      payload: { session_id: 1, expression: '1d20', result: 15, character_name: '', hidden: false }
    }} />)
    expect(await screen.findByText('1d20')).toBeInTheDocument()
    expect(mockFetch).toHaveBeenCalledOnce()
  })

  it('shows character_name from dice_rolled event', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    rerender(<DiceHistoryPanel sessionId={1} lastEvent={{
      type: 'dice_rolled',
      payload: { session_id: 1, expression: '1d20', result: 15, character_name: 'Kira', hidden: false }
    }} />)
    expect(await screen.findByText('Kira')).toBeInTheDocument()
    expect(screen.getByText('15')).toBeInTheDocument()
  })

  it('shows hidden roll as GM roll with result masked', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    rerender(<DiceHistoryPanel sessionId={1} lastEvent={{
      type: 'dice_rolled',
      payload: { session_id: 1, expression: '1d20', result: 19, character_name: 'GM', hidden: true }
    }} />)
    expect(await screen.findByText('[GM]')).toBeInTheDocument()
    expect(screen.queryByText('19')).not.toBeInTheDocument()
  })

  it('adds dice-entry-new class to incoming live roll', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    rerender(<DiceHistoryPanel sessionId={1} lastEvent={{
      type: 'dice_rolled',
      payload: { session_id: 1, expression: '2d6', result: 8, character_name: '', hidden: false }
    }} />)
    const row = (await screen.findByText('2d6')).closest('.dice-compact-row')
    expect(row).toHaveClass('dice-entry-new')
  })

  it('ignores rolls from another session', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    const { rerender } = render(<DiceHistoryPanel sessionId={1} lastEvent={null} />)
    await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledOnce())
    rerender(<DiceHistoryPanel sessionId={1} lastEvent={{
      type: 'dice_rolled',
      payload: { session_id: 2, expression: '1d20', result: 15, character_name: '', hidden: false },
    }} />)
    expect(screen.queryByText('1d20')).not.toBeInTheDocument()
  })
})
