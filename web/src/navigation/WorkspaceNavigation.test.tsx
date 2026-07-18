import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PANEL_DEFINITIONS, createPanelRegistry } from './panelRegistry'
import { WorkspaceNavigation } from './WorkspaceNavigation'

afterEach(cleanup)

const EXPECTED_PANELS = [
  'Handouts', 'Decks', 'Oracle',
  'Compendium', 'Notes', 'Journal', 'NPCs', 'Relationships', 'Factions', 'Calendar',
  'Objectives', 'NPC Stat Blocks', 'Adventures', 'Secrets', 'GM Tools',
]

describe('WorkspaceNavigation', () => {
  it('registers every stable panel with a group and renderer', () => {
    const renderers = Object.fromEntries(PANEL_DEFINITIONS.map(({ id }) => [id, () => <p>{id} content</p>]))
    const registry = createPanelRegistry(renderers)
    expect(registry.map(({ label }) => label)).toEqual(EXPECTED_PANELS)
    expect(registry.every(({ render }) => typeof render === 'function')).toBe(true)
  })

  it('exposes every panel in keyboard-operable desktop groups', async () => {
    const user = userEvent.setup()
    const onPanelChange = vi.fn()
    render(
      <WorkspaceNavigation
        activePanel="handouts"
        onPanelChange={onPanelChange}
        mobileDestination="story"
        onMobileDestinationChange={() => undefined}
        mobile={false}
      />,
    )

    expect(screen.getByRole('heading', { name: 'Play' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'World' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'GM' })).toBeInTheDocument()
    for (const label of EXPECTED_PANELS) expect(screen.getByRole('tab', { name: label })).toBeVisible()

    const handouts = screen.getByRole('tab', { name: 'Handouts' })
    handouts.focus()
    await user.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: 'Decks' })).toHaveFocus()
    expect(onPanelChange).toHaveBeenCalledWith('decks')
  })

  it('renders direct Story, Character, World, and GM mobile destinations', async () => {
    const user = userEvent.setup()
    const onDestinationChange = vi.fn()
    const { rerender } = render(
      <WorkspaceNavigation
        activePanel="notes"
        onPanelChange={() => undefined}
        mobileDestination="story"
        onMobileDestinationChange={onDestinationChange}
        mobile
      />,
    )

    const destinations = screen.getByRole('navigation', { name: 'Workspace destinations' })
    expect(destinations).toHaveTextContent('Story')
    expect(destinations).toHaveTextContent('Character')
    expect(destinations).toHaveTextContent('World')
    expect(destinations).toHaveTextContent('GM')
    expect(screen.getByRole('button', { name: 'Story' })).toHaveAttribute('aria-current', 'page')

    await user.click(screen.getByRole('button', { name: 'World' }))
    expect(onDestinationChange).toHaveBeenCalledWith('world')

    rerender(
      <WorkspaceNavigation
        activePanel="notes"
        onPanelChange={() => undefined}
        mobileDestination="world"
        onMobileDestinationChange={onDestinationChange}
        mobile
      />,
    )
    expect(screen.getByRole('tab', { name: 'Handouts' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Calendar' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Objectives' })).not.toBeInTheDocument()
  })
})
