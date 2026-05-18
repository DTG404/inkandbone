import { describe, it, expect, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { CombatPanel } from './CombatPanel'
import type { CombatSnapshot } from './types'

const combat: CombatSnapshot = {
  encounter: { id: 1, session_id: 1, name: 'Bandit Ambush', active: true, active_turn_index: 0, created_at: '' },
  combatants: [
    {
      id: 1, encounter_id: 1, character_id: null,
      name: 'Kael', initiative: 18, hp_current: 30, hp_max: 40,
      conditions_json: '[]', is_player: true,
    },
    {
      id: 2, encounter_id: 1, character_id: null,
      name: 'Bandit', initiative: 12, hp_current: 5, hp_max: 20,
      conditions_json: '["frightened"]', is_player: false,
    },
  ],
}

afterEach(cleanup)

describe('CombatPanel', () => {
  it('renders encounter name and all combatants', () => {
    render(<CombatPanel combat={combat} />)
    expect(screen.getByText('⚔ Bandit Ambush')).toBeInTheDocument()
    expect(screen.getByText('Kael')).toBeInTheDocument()
    expect(screen.getByText('Bandit')).toBeInTheDocument()
  })

  it('marks first combatant (highest initiative) as active turn', () => {
    render(<CombatPanel combat={combat} />)
    const cards = document.querySelectorAll('.combatant-card')
    expect(cards[0]).toHaveClass('active-turn')
    expect(cards[1]).not.toHaveClass('active-turn')
  })

  it('applies player class to player combatant', () => {
    render(<CombatPanel combat={combat} />)
    const cards = document.querySelectorAll('.combatant-card')
    expect(cards[0]).toHaveClass('player')
    expect(cards[1]).toHaveClass('enemy')
  })

  it('renders condition badges', () => {
    render(<CombatPanel combat={combat} />)
    // Condition renders as a badge button with the condition name
    expect(screen.getByTitle('Remove frightened')).toBeInTheDocument()
  })

  it('applies red hp bar class when HP is at or below 25%', () => {
    // Bandit: 5/20 = 25% → red
    render(<CombatPanel combat={combat} />)
    const fills = document.querySelectorAll('.hp-bar-fill')
    expect(fills[1]).toHaveClass('hp-bar-red')
  })

  it('applies green hp bar class when HP is above 50%', () => {
    // Kael: 30/40 = 75% → green
    render(<CombatPanel combat={combat} />)
    const fills = document.querySelectorAll('.hp-bar-fill')
    expect(fills[0]).toHaveClass('hp-bar-green')
  })

  it('renders hp text label', () => {
    render(<CombatPanel combat={combat} />)
    expect(screen.getByText('30 / 40 HP')).toBeInTheDocument()
  })
})
