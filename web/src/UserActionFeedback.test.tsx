import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Character, CombatSnapshot, Objective } from './types'
import { ToastProvider } from './ui/ToastProvider'

const api = vi.hoisted(() => ({
  fetchObjectives: vi.fn(),
  patchObjective: vi.fn(),
  deleteObjective: vi.fn(),
  createObjective: vi.fn(),
  reanalyzeSession: vi.fn(),
  deduplicateObjectives: vi.fn(),
  fetchRuleset: vi.fn(),
  patchCharacter: vi.fn(),
  uploadPortrait: vi.fn(),
  portraitAssetURL: vi.fn(() => '/portrait'),
  patchCombatant: vi.fn(),
  advanceTurn: vi.fn(),
  reorderCombatants: vi.fn(),
}))

vi.mock('./api', () => api)

import { CharacterSheetPanel } from './CharacterSheetPanel'
import { CombatPanel } from './CombatPanel'
import { ObjectivesPanel } from './ObjectivesPanel'

const objective: Objective = {
  id: 7, campaign_id: 1, title: 'Find the archive', description: '', status: 'active', parent_id: null, created_at: '',
}

const character: Character = {
  id: 3, campaign_id: 1, name: 'Mara', data_json: JSON.stringify({ strength: 3 }), portrait_path: '',
  currency_balance: 0, currency_label: 'Gold', created_at: '',
}

const combat: CombatSnapshot = {
  encounter: { id: 9, session_id: 2, name: 'Ambush', active: true, active_turn_index: 0, round_number: 1, created_at: '' },
  combatants: [{
    id: 11, encounter_id: 9, character_id: 3, name: 'Mara', initiative: 12,
    hp_current: 8, hp_max: 10, conditions_json: '[]', is_player: true, sort_order: 0,
  }],
}

describe('user action failure feedback', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.fetchObjectives.mockResolvedValue([objective])
    api.fetchRuleset.mockResolvedValue({
      id: 1, name: 'dnd5e', version: '1',
      schema_json: JSON.stringify([
        { key: 'strength', label: 'Strength', type: 'number', category: 'resource' },
        { key: 'dexterity', label: 'Dexterity', type: 'number', category: 'resource' },
      ]),
    })
  })
  afterEach(() => {
    cleanup()
    vi.useRealTimers()
  })

  it('announces a failed objective status change while preserving the objective state', async () => {
    const user = userEvent.setup()
    api.patchObjective.mockRejectedValue(new Error('offline'))
    render(<ToastProvider><ObjectivesPanel campaignId={1} sessionId={2} lastEvent={null} /></ToastProvider>)
    await screen.findByText('Find the archive')

    await user.click(screen.getByRole('button', { name: 'Mark completed' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not update objective')
    expect(screen.getByText('◈ Active')).toBeInTheDocument()
  })

  it('restores every unsaved character field after a combined debounced save fails and announces the failure', async () => {
    api.patchCharacter.mockRejectedValue(new Error('offline'))
    render(
      <ToastProvider>
        <CharacterSheetPanel character={character} rulesetId={1} lastEvent={null} />
      </ToastProvider>,
    )
    const strength = await screen.findByLabelText('Strength')
    const dexterity = screen.getByLabelText('Dexterity')
    fireEvent.change(strength, { target: { value: '4' } })
    fireEvent.change(dexterity, { target: { value: '5' } })
    expect(strength).toHaveValue('4')
    expect(dexterity).toHaveValue('5')

    await waitFor(() => expect(api.patchCharacter).toHaveBeenCalled(), { timeout: 1500 })
    expect(screen.getByRole('alert')).toHaveTextContent('Could not save Dexterity')
    expect(strength).toHaveValue('3')
    expect(dexterity).toHaveValue('')
  })

  it('rolls back an optimistic combat condition after failure and announces it', async () => {
    const user = userEvent.setup()
    api.patchCombatant.mockRejectedValue(new Error('offline'))
    render(<ToastProvider><CombatPanel combat={combat} /></ToastProvider>)

    await user.click(screen.getByRole('button', { name: '+ Condition' }))
    await user.click(screen.getByRole('button', { name: 'Poisoned' }))

    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Could not add Poisoned'))
    expect(screen.queryByRole('button', { name: /Remove Poisoned/ })).not.toBeInTheDocument()
  })
})
