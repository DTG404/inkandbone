import { describe, it, expect, vi, afterEach } from 'vitest'
import { fetchContext, fetchMessages, fetchWorldNotes, fetchDiceRolls, fetchTimeline, fetchMaps, fetchMapPins, patchSessionSummary, generateRecap, draftWorldNote, uploadMap } from './api'

afterEach(() => vi.restoreAllMocks())

describe('fetchContext', () => {
  it('returns parsed GameContext on success', async () => {
    const payload = {
      campaign: null,
      character: null,
      session: null,
      recent_messages: [],
      active_combat: null,
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(payload),
    }))

    const ctx = await fetchContext()
    expect(ctx.campaign).toBeNull()
    expect(ctx.recent_messages).toEqual([])
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
    }))

    await expect(fetchContext()).rejects.toThrow('GET /api/context failed: 500')
  })
})

describe('fetchMessages', () => {
  it('loads the authorized session transcript including whisper metadata', async () => {
    const messages = [
      { id: 1, session_id: 7, role: 'user', content: 'PRIVATE_SENTINEL', whisper: true, created_at: '' },
    ]
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(messages) })
    vi.stubGlobal('fetch', mockFetch)

    await expect(fetchMessages(7)).resolves.toEqual(messages)
    expect(mockFetch).toHaveBeenCalledWith('/api/sessions/7/messages')
  })

  it('throws when the authorized transcript cannot be loaded', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(fetchMessages(7)).rejects.toThrow('GET /api/sessions/7/messages failed: 500')
  })
})

describe('fetchWorldNotes', () => {
  it('returns parsed WorldNote array on success', async () => {
    const notes = [
      { id: 1, campaign_id: 1, title: 'Tavern', content: 'A seedy place.', category: 'location', tags_json: '[]', created_at: '' },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(notes),
    }))

    const result = await fetchWorldNotes(1)
    expect(result).toHaveLength(1)
    expect(result[0].title).toBe('Tavern')
  })

  it('appends q param when query is provided', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)

    await fetchWorldNotes(1, 'dragon')
    expect(mockFetch).toHaveBeenCalledWith('/api/campaigns/1/world-notes?q=dragon')
  })

  it('omits q param when query is empty string', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)

    await fetchWorldNotes(1, '')
    expect(mockFetch).toHaveBeenCalledWith('/api/campaigns/1/world-notes')
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 404 }))
    await expect(fetchWorldNotes(1)).rejects.toThrow('failed: 404')
  })

  it('appends tag param when tag is provided', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    await fetchWorldNotes(1, undefined, 'npc')
    expect(mockFetch).toHaveBeenCalledWith('/api/campaigns/1/world-notes?tag=npc')
  })

  it('appends both q and tag when both provided', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    await fetchWorldNotes(1, 'tavern', 'location')
    expect(mockFetch).toHaveBeenCalledWith('/api/campaigns/1/world-notes?q=tavern&tag=location')
  })
})

describe('fetchDiceRolls', () => {
  it('returns parsed DiceRoll array on success', async () => {
    const rolls = [
      { id: 1, session_id: 1, expression: '1d20+5', result: 18, breakdown_json: '[]', created_at: '' },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(rolls),
    }))

    const result = await fetchDiceRolls(1)
    expect(result).toHaveLength(1)
    expect(result[0].expression).toBe('1d20+5')
    expect(result[0].result).toBe(18)
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(fetchDiceRolls(1)).rejects.toThrow('failed: 500')
  })
})

describe('fetchTimeline', () => {
  it('returns parsed TimelineEntry array on success', async () => {
    const entries = [
      { type: 'message', timestamp: '2026-04-03T10:00:00Z', data: { id: 1, role: 'user', content: 'Hi' } },
      { type: 'dice_roll', timestamp: '2026-04-03T10:01:00Z', data: { id: 1, expression: '1d20', result: 15 } },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(entries) }))
    const result = await fetchTimeline(1)
    expect(result).toHaveLength(2)
    expect(result[0].type).toBe('message')
    expect(result[1].type).toBe('dice_roll')
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 404 }))
    await expect(fetchTimeline(1)).rejects.toThrow('failed: 404')
  })
})

describe('fetchMaps', () => {
  it('returns parsed CampaignMap array on success', async () => {
    const maps = [
      { id: 1, campaign_id: 2, image_path: '/uploads/map1.png', created_at: '2026-04-03T10:00:00Z' },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(maps),
    }))

    const result = await fetchMaps(2)
    expect(result).toHaveLength(1)
    expect(result[0].image_path).toBe('/uploads/map1.png')
  })

  it('calls the correct URL', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)

    await fetchMaps(5)
    expect(mockFetch).toHaveBeenCalledWith('/api/campaigns/5/maps')
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(fetchMaps(2)).rejects.toThrow('failed: 500')
  })
})

describe('fetchMapPins', () => {
  it('returns parsed MapPin array on success', async () => {
    const pins = [
      { id: 1, map_id: 3, x: 100, y: 200, label: 'Tavern', note: 'Meeting point', color: '#ff0000', created_at: '2026-04-03T10:00:00Z' },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(pins),
    }))

    const result = await fetchMapPins(3)
    expect(result).toHaveLength(1)
    expect(result[0].label).toBe('Tavern')
    expect(result[0].x).toBe(100)
  })

  it('calls the correct URL', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)

    await fetchMapPins(7)
    expect(mockFetch).toHaveBeenCalledWith('/api/maps/7/pins')
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(fetchMapPins(3)).rejects.toThrow('failed: 500')
  })
})

describe('patchSessionSummary', () => {
  it('sends PATCH request with correct body', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', mockFetch)

    await patchSessionSummary(4, 'The party defeated the dragon.')
    expect(mockFetch).toHaveBeenCalledWith('/api/sessions/4', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ summary: 'The party defeated the dragon.' }),
    })
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(patchSessionSummary(4, 'summary')).rejects.toThrow('failed: 500')
  })
})

describe('generateRecap', () => {
  it('returns parsed summary on success', async () => {
    const payload = { summary: 'An epic battle ensued.' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(payload),
    }))

    const result = await generateRecap(4)
    expect(result.summary).toBe('An epic battle ensued.')
  })

  it('sends POST request to correct URL', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ summary: '' }) })
    vi.stubGlobal('fetch', mockFetch)

    await generateRecap(9)
    expect(mockFetch).toHaveBeenCalledWith('/api/sessions/9/recap', { method: 'POST' })
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(generateRecap(4)).rejects.toThrow('failed: 500')
  })
})

describe('draftWorldNote', () => {
  it('returns parsed draft on success', async () => {
    const payload = { id: 10, title: 'The Lost Temple', content: 'Ancient ruins deep in the forest.' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(payload),
    }))

    const result = await draftWorldNote(2, 'ancient ruins')
    expect(result.id).toBe(10)
    expect(result.title).toBe('The Lost Temple')
  })

  it('sends POST request with correct body', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 1, title: '', content: '' }) })
    vi.stubGlobal('fetch', mockFetch)

    await draftWorldNote(3, 'dark forest')
    expect(mockFetch).toHaveBeenCalledWith('/api/campaigns/3/world-notes/draft', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ hint: 'dark forest' }),
    })
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(draftWorldNote(2, 'hint')).rejects.toThrow('failed: 500')
  })
})

describe('uploadMap', () => {
  it('returns parsed CampaignMap on success', async () => {
    const payload = { id: 5, campaign_id: 2, image_path: '/uploads/new-map.png', created_at: '2026-04-03T10:00:00Z' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(payload),
    }))

    const file = new File(['binary'], 'map.png', { type: 'image/png' })
    const result = await uploadMap(2, file)
    expect(result.id).toBe(5)
    expect(result.image_path).toBe('/uploads/new-map.png')
  })

  it('sends POST with FormData to correct URL', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ id: 1, campaign_id: 2, image_path: '', created_at: '' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const file = new File(['binary'], 'map.png', { type: 'image/png' })
    await uploadMap(2, file)

    expect(mockFetch).toHaveBeenCalledOnce()
    const [url, options] = mockFetch.mock.calls[0]
    expect(url).toBe('/api/campaigns/2/maps')
    expect(options.method).toBe('POST')
    expect(options.body).toBeInstanceOf(FormData)
    expect((options.body as FormData).get('image')).toBe(file)
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    const file = new File(['binary'], 'map.png', { type: 'image/png' })
    await expect(uploadMap(2, file)).rejects.toThrow('failed: 500')
  })
})
