import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { waitFor, fireEvent } from '@testing-library/dom'
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

  it('renders computed fields with evaluated formula', async () => {
    const rulesetWithComputed = {
      id: 3,
      name: 'dnd5e',
      schema_json: JSON.stringify([
        { key: 'level', label: 'Level', type: 'number', category: 'resource' },
        { key: 'proficiency_bonus', label: 'Proficiency Bonus', type: 'number', category: 'resource', computed: 'floor((level-1)/4)+2' },
      ]),
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(rulesetWithComputed),
    }))
    render(<CharacterSheetPanel character={mockCharacter} rulesetId={3} lastEvent={null} />)
    await waitFor(() => expect(screen.getByText('Proficiency Bonus')).toBeTruthy())
    expect(screen.getByText('2')).toBeTruthy()
  })

  it('hides fields when condition is not met', async () => {
    const rulesetWithCondition = {
      id: 4,
      name: 'custom',
      schema_json: JSON.stringify([
        { key: 'character_type', label: 'Type', type: 'text', category: 'identity' },
        { key: 'vampire_power', label: 'Vampire Power', type: 'number', category: 'resource', condition: { field: 'character_type', equals: 'vampire' } },
      ]),
    }
    const characterMortal = {
      ...mockCharacter,
      data_json: JSON.stringify({ character_type: 'mortal' }),
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(rulesetWithCondition),
    }))
    render(<CharacterSheetPanel character={characterMortal} rulesetId={4} lastEvent={null} />)
    await waitFor(() => expect(screen.getByLabelText('Type')).toBeTruthy())
    expect(screen.queryByLabelText('Vampire Power')).toBeNull()
  })

  it('shows conditional fields when condition is met', async () => {
    const rulesetWithCondition = {
      id: 5,
      name: 'custom',
      schema_json: JSON.stringify([
        { key: 'character_type', label: 'Type', type: 'text', category: 'identity' },
        { key: 'vampire_power', label: 'Vampire Power', type: 'number', category: 'resource', condition: { field: 'character_type', equals: 'vampire' } },
      ]),
    }
    const characterVampire = {
      ...mockCharacter,
      data_json: JSON.stringify({ character_type: 'vampire' }),
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(rulesetWithCondition),
    }))
    render(<CharacterSheetPanel character={characterVampire} rulesetId={5} lastEvent={null} />)
    await waitFor(() => expect(screen.getByLabelText('Type')).toBeTruthy())
    expect(screen.getByLabelText('Vampire Power')).toBeTruthy()
  })

  it('renders backstory field at the bottom', async () => {
    const rulesetWithBackstory = {
      id: 6,
      name: 'custom',
      schema_json: JSON.stringify([
        { key: 'hp', label: 'HP', type: 'number', category: 'resource' },
        { key: 'backstory', label: 'Backstory', type: 'textarea', category: 'notes' },
      ]),
    }
    const characterWithBackstory = {
      ...mockCharacter,
      data_json: JSON.stringify({ hp: 10, backstory: 'Born in a small village...' }),
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(rulesetWithBackstory),
    }))
    render(<CharacterSheetPanel character={characterWithBackstory} rulesetId={6} lastEvent={null} />)
    await waitFor(() => expect(screen.getByLabelText('HP')).toBeTruthy())
    expect(screen.getByText('Backstory')).toBeTruthy()
    expect((screen.getByLabelText('Backstory') as HTMLTextAreaElement).value).toBe('Born in a small village...')
  })

  it('does not send computed fields in PATCH payload', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const rulesetWithComputed = {
      id: 7,
      name: 'dnd5e',
      schema_json: JSON.stringify([
        { key: 'level', label: 'Level', type: 'number', category: 'resource' },
        { key: 'proficiency_bonus', label: 'Proficiency Bonus', type: 'number', category: 'resource', computed: 'floor((level-1)/4)+2' },
      ]),
    }
    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(rulesetWithComputed) })
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve(mockCharacter) })
    vi.stubGlobal('fetch', mockFetch)

    render(<CharacterSheetPanel character={mockCharacter} rulesetId={7} lastEvent={null} />)
    await waitFor(() => expect(screen.getByLabelText('Level')).toBeTruthy())

    fireEvent.change(screen.getByLabelText('Level'), { target: { value: '5' } })
    vi.advanceTimersByTime(600)
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(2))

    const patchCall = mockFetch.mock.calls[1]
    const patchBody = JSON.parse(patchCall[1].body)
    const data = JSON.parse(patchBody.data_json)
    expect(data).toHaveProperty('level')
    expect(data).not.toHaveProperty('proficiency_bonus')

    vi.useRealTimers()
  })
})
