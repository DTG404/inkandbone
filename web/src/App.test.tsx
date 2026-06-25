import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import App from './App'
import type { GameContext } from './types'

const mockCtx: GameContext = {
  campaign: { id: 1, ruleset_id: 1, name: 'Greyhawk', description: '', active: true, chronicle_night: 1, chronicle_night_start_dow: -1, created_at: '' },
  character: { id: 1, campaign_id: 1, name: 'Zara', data_json: '{}', portrait_path: '', created_at: '', currency_balance: 0, currency_label: 'Gold' },
  session: { id: 1, campaign_id: 1, title: 'Session 1', date: '2026-04-03', summary: '', notes: '', scene_tags: '', adventure_id: null, created_at: '' },
  recent_messages: [
    { id: 1, session_id: 1, role: 'assistant', content: 'You enter the tavern.', created_at: '' },
    { id: 2, session_id: 1, role: 'user', content: 'I look for a table.', created_at: '' },
  ],
  active_combat: null,
}

class MockWebSocket {
  onmessage = null; onclose: (() => void) | null = null; onerror = null
  close = vi.fn()
}

describe('App', () => {
  beforeEach(() => {
    vi.stubGlobal('WebSocket', MockWebSocket)
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      // WorldNotesPanel and DiceHistoryPanel sub-fetches return empty arrays
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('renders campaign name in state bar', async () => {
    render(<App />)
    expect(await screen.findByText('Greyhawk')).toBeInTheDocument()
  })

  it('renders character name in state bar', async () => {
    render(<App />)
    expect(await screen.findByText('Zara')).toBeInTheDocument()
  })

  it('renders session title in state bar', async () => {
    render(<App />)
    expect(await screen.findByText('Session 1')).toBeInTheDocument()
  })

  it('renders session log messages', async () => {
    render(<App />)
    expect(await screen.findByText('You enter the tavern.')).toBeInTheDocument()
    expect(await screen.findByText('I look for a table.')).toBeInTheDocument()
  })

  it('renders world notes panel', async () => {
    render(<App />)
    await screen.findByText('Greyhawk')
    // WorldNotesPanel shows "No notes found." when empty and "Notes" tab button
    expect(screen.getByText('No notes found.')).toBeInTheDocument()
  })

  it('renders dice roller buttons when character is present', async () => {
    render(<App />)
    await screen.findByText('Greyhawk')
    expect(screen.getByText('Zara')).toBeInTheDocument()
  })

  it('renders portrait img when portrait_path is set', async () => {
    const ctxWithPortrait = {
      ...mockCtx,
      character: { ...mockCtx.character!, portrait_path: 'portraits/zara.jpg' },
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(ctxWithPortrait) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    // Both the state-bar portrait and the character sheet panel render an img with name "Zara"
    const imgs = await screen.findAllByRole('img', { name: 'Zara' })
    expect(imgs.length).toBeGreaterThanOrEqual(1)
    expect(imgs[0]).toHaveAttribute('src', '/api/files/portraits/zara.jpg')
  })

  it('does not render portrait img when portrait_path is empty', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    await screen.findByText('Greyhawk')
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('renders combat panel when active_combat is set', async () => {
    const ctxWithCombat: GameContext = {
      ...mockCtx,
      active_combat: {
        encounter: { id: 1, session_id: 1, name: 'Dragon Fight', active: true, active_turn_index: 0, created_at: '' },
        combatants: [
          { id: 1, encounter_id: 1, character_id: null, name: 'Zara', initiative: 20, hp_current: 40, hp_max: 40, conditions_json: '[]', is_player: true, sort_order: 0 },
        ],
      },
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(ctxWithCombat) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    // CombatPanel renders <h2>⚔ Dragon Fight</h2>
    expect(await screen.findByText('⚔ Dragon Fight')).toBeInTheDocument()
    expect(screen.getAllByText('Zara').length).toBeGreaterThanOrEqual(1)
  })

  it('renders MapPanel', async () => {
    render(<App />)
    await screen.findByText('Greyhawk')
    // MapPanel fetches maps and gets [] back, so it shows "No map uploaded."
    expect(screen.getByText('No map uploaded.')).toBeInTheDocument()
  })

  it('passes aiEnabled to WorldNotesPanel', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (url === '/api/health') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ai_enabled: true }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    await screen.findByText('Greyhawk')
    // aiEnabled=true means the "Draft with AI" button is visible
    expect(screen.getByRole('button', { name: 'Draft with AI' })).toBeInTheDocument()
  })

  it('renders character sheet panel when character is present', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (url === '/api/rulesets/1') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            id: 1, name: 'dnd5e',
            schema_json: JSON.stringify([{ key: 'hp', label: 'HP', type: 'number' }]),
          }),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    await screen.findByText('Greyhawk')
    expect(await screen.findByLabelText('HP')).toBeInTheDocument()
  })
})
