import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, waitFor, fireEvent } from '@testing-library/react'
import { WorldNotesPanel } from './WorldNotesPanel'
import type { WorldNote } from './types'

const notes: WorldNote[] = [
  { id: 1, campaign_id: 1, title: 'Tavern', content: 'A seedy place.', category: 'location', tags_json: '["inn"]', personality_json: '', is_revealed: false, created_at: '' },
  { id: 2, campaign_id: 1, title: 'Dragon', content: 'Ancient red dragon.', category: 'npc', tags_json: '[]', personality_json: '', is_revealed: false, created_at: '' },
]

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('WorldNotesPanel', () => {
  it('renders notes returned by fetch', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(notes) }))
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    expect(await screen.findByText('Tavern')).toBeInTheDocument()
    expect(screen.getByText('Dragon')).toBeInTheDocument()
  })

  it('shows empty state when no notes', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    expect(await screen.findByText('No notes found.')).toBeInTheDocument()
  })

  it('calls fetch with q param when search input changes', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    await screen.findByText('No notes found.')
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'tavern' } })
    await waitFor(() => {
      expect(mockFetch).toHaveBeenLastCalledWith('/api/campaigns/1/world-notes?q=tavern')
    })
  })

  it('renders tag pills from tags_json', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(notes) }))
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    expect(await screen.findByText('inn')).toBeInTheDocument()
  })

  it('filters by tag when a tag pill is clicked', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(notes) })
    vi.stubGlobal('fetch', mockFetch)
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    fireEvent.click(await screen.findByText('inn'))
    await waitFor(() => {
      expect(mockFetch).toHaveBeenLastCalledWith('/api/campaigns/1/world-notes?tag=inn')
    })
  })

  it('deselects tag when same pill is clicked again', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(notes) })
    vi.stubGlobal('fetch', mockFetch)
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    const pill = await screen.findByText('inn')
    fireEvent.click(pill)
    await waitFor(() => expect(mockFetch).toHaveBeenLastCalledWith('/api/campaigns/1/world-notes?tag=inn'))
    fireEvent.click(pill)
    await waitFor(() => expect(mockFetch).toHaveBeenLastCalledWith('/api/campaigns/1/world-notes'))
  })

  it('refetches on world_note_updated event', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    const { rerender } = render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    await screen.findByText('No notes found.')
    const callsBefore = mockFetch.mock.calls.length
    rerender(<WorldNotesPanel campaignId={1} lastEvent={{ type: 'world_note_updated', payload: { note_id: 1 } }} aiEnabled={false} />)
    await waitFor(() => {
      expect(mockFetch.mock.calls.length).toBeGreaterThan(callsBefore)
    })
  })

  it('refetches on world_note_created event', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    const { rerender } = render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    await screen.findByText('No notes found.')
    const callsBefore = mockFetch.mock.calls.length
    rerender(<WorldNotesPanel campaignId={1} lastEvent={{ type: 'world_note_created', payload: { note_id: 2 } }} aiEnabled={false} />)
    await waitFor(() => {
      expect(mockFetch.mock.calls.length).toBeGreaterThan(callsBefore)
    })
  })

  it('draft button not shown when aiEnabled=false', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={false} />)
    await screen.findByText('No notes found.')
    expect(screen.queryByRole('button', { name: /draft with ai/i })).not.toBeInTheDocument()
  })

  it('draft button shown when aiEnabled=true', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={true} />)
    await screen.findByText('No notes found.')
    expect(screen.getByRole('button', { name: /draft with ai/i })).toBeInTheDocument()
  })

  it('draft button calls draftWorldNote with hint from window.prompt', async () => {
    vi.stubGlobal('prompt', vi.fn().mockReturnValue('Elven smith'))
    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) }) // initial load
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ id: 10, title: 'Smith', content: 'A skilled elven smith.' }) }) // draftWorldNote
    vi.stubGlobal('fetch', mockFetch)
    render(<WorldNotesPanel campaignId={1} lastEvent={null} aiEnabled={true} />)
    await screen.findByText('No notes found.')
    fireEvent.click(screen.getByRole('button', { name: /draft with ai/i }))
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/campaigns/1/world-notes/draft',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ hint: 'Elven smith' }),
        })
      )
    })
  })
})
