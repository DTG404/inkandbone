import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { AdventuresPanel } from './AdventuresPanel'
import { FactionsPanel } from './FactionsPanel'
import { NPCStatBlockPanel } from './NPCStatBlockPanel'
import { SecretsPanel } from './SecretsPanel'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

function response(body: unknown) {
  return Promise.resolve({ ok: true, json: () => Promise.resolve(body) })
}

it('uses native primary actions for expandable reference rows', async () => {
  const faction = { id: 1, campaign_id: 1, name: 'Lantern Guild', description: 'Traders', faction_type: 'guild', influence: 5, resources_json: '{}', color: '#d9b85a', created_at: '' }
  const npc = { id: 2, campaign_id: 1, name: 'Watch Captain', role: 'leader', data_json: '{}', hp_max: 20, armor_class: 14, initiative_mod: 2, skills: '[]', abilities: '[]', loot: '[]', notes: 'Alert', created_at: '' }
  const adventure = { id: 3, campaign_id: 1, title: 'The Long Road', description: 'Travel', status: 'active', sort_order: 0, created_at: '' }
  const secret = { id: 4, campaign_id: 1, title: 'Hidden Door', content: 'Behind the tapestry', category: 'clue', revealed: false, revealed_at_session_id: null, created_at: '' }
  vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
    if (input.endsWith('/factions')) return response([faction])
    if (input.endsWith('/npc-stats')) return response([npc])
    if (input.endsWith('/adventures')) return response([adventure])
    if (input.endsWith('/sessions')) return response([])
    if (input.endsWith('/secrets')) return response([secret])
    return response([])
  }))
  const user = userEvent.setup()

  const factionView = render(<FactionsPanel campaignId={1} lastEvent={null} />)
  const factionToggle = await screen.findByRole('button', { name: 'Toggle faction Lantern Guild' })
  factionToggle.focus()
  await user.keyboard('{Enter}')
  expect(screen.getByText('Traders')).toBeInTheDocument()
  factionView.unmount()

  const npcView = render(<NPCStatBlockPanel campaignId={1} lastEvent={null} />)
  await user.click(await screen.findByRole('button', { name: 'Toggle NPC stat block Watch Captain' }))
  expect(screen.getByText('Alert')).toBeInTheDocument()
  npcView.unmount()

  const adventureView = render(<AdventuresPanel campaignId={1} lastEvent={null} onSessionClick={() => undefined} />)
  await user.click(await screen.findByRole('button', { name: 'Toggle adventure The Long Road' }))
  expect(screen.getByText('Travel')).toBeInTheDocument()
  adventureView.unmount()

  render(<SecretsPanel campaignId={1} sessionId={1} />)
  await user.click(await screen.findByRole('button', { name: 'Toggle secret Hidden Door' }))
  expect(screen.getByText(/content hidden/i)).toBeInTheDocument()
})
