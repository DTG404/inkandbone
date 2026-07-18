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

  it('exposes every panel as grouped navigation buttons with one current controller', async () => {
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
    expect(screen.queryAllByRole('tab')).toHaveLength(0)
    expect(screen.queryAllByRole('tablist')).toHaveLength(0)
    for (const label of EXPECTED_PANELS) expect(screen.getByRole('button', { name: label })).toBeVisible()

    const handouts = screen.getByRole('button', { name: 'Handouts' })
    expect(handouts).toHaveAttribute('aria-current', 'page')
    expect(handouts).toHaveAttribute('aria-controls', 'workspace-active-panel')
    expect(handouts).toHaveAttribute('id', 'workspace-panel-control-handouts')
    handouts.focus()
    await user.tab()
    expect(screen.getByRole('button', { name: 'Decks' })).toHaveFocus()
    await user.keyboard('{Enter}')
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
    expect(screen.getByRole('button', { name: 'Handouts' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Calendar' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Objectives' })).not.toBeInTheDocument()
    expect(screen.queryAllByRole('tab')).toHaveLength(0)
  })
})
