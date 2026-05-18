import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, waitFor } from '@testing-library/react'
import { MapPanel } from './MapPanel'
import type { CampaignMap, MapPin } from './api'

const map: CampaignMap = {
  id: 42,
  campaign_id: 1,
  name: 'Dungeon',
  image_path: 'maps/dungeon.png',
  created_at: '',
}

const pins: MapPin[] = [
  { id: 1, map_id: 42, x: 0.25, y: 0.5, label: 'Entrance', note: 'Main door', color: 'red', created_at: '' },
  { id: 2, map_id: 42, x: 0.75, y: 0.25, label: 'Boss Room', note: '', color: 'blue', created_at: '' },
]

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('MapPanel', () => {
  it('TestMapPanel_nullCampaign: renders nothing when campaignId is null', () => {
    vi.stubGlobal('fetch', vi.fn())
    const { container } = render(<MapPanel campaignId={null} lastEvent={null} />)
    expect(container.firstChild).toBeNull()
  })

  it('TestMapPanel_noMap: shows "No map uploaded." when fetchMaps returns empty', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }),
    )
    render(<MapPanel campaignId={1} lastEvent={null} />)
    expect(await screen.findByText('No map uploaded.')).toBeInTheDocument()
  })

  it('TestMapPanel_showsPins: renders map image and two pin buttons at correct positions', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn()
        .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([map]) })
        .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(pins) }),
    )
    render(<MapPanel campaignId={1} lastEvent={null} />)

    const img = await screen.findByRole('img', { name: 'Dungeon' })
    expect(img).toHaveAttribute('src', '/api/files/maps/dungeon.png')

    const buttons = await screen.findAllByRole('button')
    expect(buttons).toHaveLength(2)

    expect(buttons[0]).toHaveAttribute('title', 'Main door')
    expect(buttons[1]).toHaveAttribute('title', 'Boss Room')

    expect(buttons[0]).toHaveStyle({ left: '25%', top: '50%' })
    expect(buttons[1]).toHaveStyle({ left: '75%', top: '25%' })
  })

  it('TestMapPanel_wsEvent: refetches pins when map_pin_added event fires with matching map_id', async () => {
    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([map]) })
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(pins) })
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(pins) })

    vi.stubGlobal('fetch', mockFetch)

    const { rerender } = render(<MapPanel campaignId={1} lastEvent={null} />)
    await screen.findAllByRole('button')

    rerender(
      <MapPanel
        campaignId={1}
        lastEvent={{ type: 'map_pin_added', payload: { map_id: 42 } }}
      />,
    )

    await waitFor(() => {
      const pinCalls = mockFetch.mock.calls.filter((args: unknown[]) => args[0] === `/api/maps/${map.id}/pins`)
      expect(pinCalls.length).toBeGreaterThanOrEqual(2)
    })
  })
})
