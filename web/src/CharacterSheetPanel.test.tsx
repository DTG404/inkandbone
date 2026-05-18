import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import { CharacterSheetPanel } from './CharacterSheetPanel'

beforeEach(() => {
  vi.restoreAllMocks()
})

afterEach(() => {
  cleanup()
})

const mockCharacter = {
  id: 1,
  name: 'Aria',
  data_json: JSON.stringify({ hp: 10, level: 1, notes: 'brave' }),
  campaign_id: 1,
  portrait_path: '',
  created_at: '',
  currency_balance: 0,
  currency_label: 'Gold',
}

const mockRuleset = {
  id: 1,
  name: 'dnd5e',
  schema_json: JSON.stringify([
    { key: 'hp', label: 'HP', type: 'number' },
    { key: 'level', label: 'Level', type: 'number' },
    { key: 'notes', label: 'Notes', type: 'textarea' },
  ]),
}

describe('CharacterSheetPanel', () => {
  it('renders nothing when character is null', () => {
    const { container } = render(
      <CharacterSheetPanel character={null} rulesetId={null} lastEvent={null} />
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders fields from structured schema format', async () => {
    const structuredRuleset = {
      id: 2,
      name: 'dnd5e',
      schema_json: JSON.stringify([
        { key: 'level', label: 'Level', type: 'number', category: 'resource' },
        { key: 'proficiency_bonus', label: 'Proficiency Bonus', type: 'number', category: 'resource' },
      ]),
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(structuredRuleset),
    }))
    render(<CharacterSheetPanel character={mockCharacter} rulesetId={2} lastEvent={null} />)
    await waitFor(() => expect(screen.getByLabelText('Level')).toBeTruthy())
    expect(screen.getByLabelText('Proficiency Bonus')).toBeTruthy()
  })

  it('fetches ruleset and renders fields', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockRuleset),
    }))
    render(<CharacterSheetPanel character={mockCharacter} rulesetId={1} lastEvent={null} />)
    await waitFor(() => expect(screen.getByLabelText('HP')).toBeTruthy())
    expect((screen.getByLabelText('HP') as HTMLInputElement).value).toBe('10')
    expect((screen.getByLabelText('Level') as HTMLInputElement).value).toBe('1')
    expect((screen.getByLabelText('Notes') as HTMLTextAreaElement).value).toBe('brave')
  })

  it('debounces PATCH on field change', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(mockRuleset) })
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ ...mockCharacter, data_json: JSON.stringify({ hp: 15, level: 1, notes: 'brave' }) }) })
    vi.stubGlobal('fetch', mockFetch)

    render(<CharacterSheetPanel character={mockCharacter} rulesetId={1} lastEvent={null} />)
    await waitFor(() => expect(screen.getByLabelText('HP')).toBeTruthy())

    fireEvent.change(screen.getByLabelText('HP'), { target: { value: '15' } })
    // PATCH should NOT have been called yet (debounce pending)
    expect(mockFetch).toHaveBeenCalledTimes(1) // only the ruleset fetch

    vi.advanceTimersByTime(600)
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(2))

    const patchCall = mockFetch.mock.calls[1]
    expect(patchCall[0]).toBe('/api/characters/1')
    expect(patchCall[1].method).toBe('PATCH')

    vi.useRealTimers()
  })

  it('updates fields on character_updated WS event', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockRuleset),
    }))

    const event = {
      type: 'character_updated',
      payload: { id: 1, data_json: JSON.stringify({ hp: 20, level: 2, notes: 'veteran' }) },
    }

    const { rerender } = render(
      <CharacterSheetPanel character={mockCharacter} rulesetId={1} lastEvent={null} />
    )
    await waitFor(() => expect(screen.getByLabelText('HP')).toBeTruthy())

    rerender(<CharacterSheetPanel character={mockCharacter} rulesetId={1} lastEvent={event} />)
    await waitFor(() =>
      expect((screen.getByLabelText('HP') as HTMLInputElement).value).toBe('20')
    )
  })
})
