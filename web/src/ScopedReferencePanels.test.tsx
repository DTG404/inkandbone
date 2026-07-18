import { cleanup, render, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { FactionsPanel } from './FactionsPanel'
import { NPCStatBlockPanel } from './NPCStatBlockPanel'
import { RelationshipsPanel } from './RelationshipsPanel'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('scoped reference panels', () => {
  it.each([
    ['relationship_updated', '/api/campaigns/1/relationships', RelationshipsPanel],
    ['faction_updated', '/api/campaigns/1/factions', FactionsPanel],
    ['npc_stat_updated', '/api/campaigns/1/npc-stats', NPCStatBlockPanel],
  ] as const)('refetches %s only for the owning campaign', async (type, url, Panel) => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', fetchMock)
    const { rerender } = render(<Panel campaignId={1} lastEvent={null} />)
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(url))
    fetchMock.mockClear()

    rerender(<Panel campaignId={1} lastEvent={{ type, payload: { campaign_id: 2 } }} />)
    await Promise.resolve()
    expect(fetchMock).not.toHaveBeenCalled()

    rerender(<Panel campaignId={1} lastEvent={{ type, payload: { campaign_id: 1 } }} />)
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(url))
  })
})
