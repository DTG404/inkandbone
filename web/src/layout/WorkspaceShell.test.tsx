import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { WorkspaceShell } from './WorkspaceShell'

describe('WorkspaceShell', () => {
  beforeEach(() => localStorage.clear())
  afterEach(cleanup)

  function renderShell() {
    return render(
      <WorkspaceShell
        header={<div>Header slot</div>}
        left={<div>Character slot</div>}
        story={<div>Story slot</div>}
        right={<div>World slot</div>}
        mobileNav={<div>Mobile slot</div>}
      />,
    )
  }

  it('renders all named slots and default width variables', () => {
    const { container } = renderShell()
    expect(screen.getByText('Header slot')).toBeInTheDocument()
    expect(screen.getByText('Character slot')).toBeInTheDocument()
    expect(screen.getByText('Story slot')).toBeInTheDocument()
    expect(screen.getByText('World slot')).toBeInTheDocument()
    expect(screen.getByText('Mobile slot')).toBeInTheDocument()
    expect(container.firstChild).toHaveStyle({ '--left-panel-width': '420px', '--right-panel-width': '320px' })
  })

  it('resizes with pointer and keyboard increments', async () => {
    const user = userEvent.setup()
    const { container } = renderShell()
    const leftSeparator = screen.getByRole('separator', { name: 'Resize character panel' })

    fireEvent.pointerDown(leftSeparator, { clientX: 420, pointerId: 1 })
    fireEvent.pointerMove(document, { clientX: 470, pointerId: 1 })
    fireEvent.pointerUp(document, { pointerId: 1 })
    expect(container.firstChild).toHaveStyle({ '--left-panel-width': '470px' })

    leftSeparator.focus()
    await user.keyboard('{ArrowRight}')
    expect(container.firstChild).toHaveStyle({ '--left-panel-width': '486px' })
    await user.keyboard('{Shift>}{ArrowLeft}{/Shift}')
    expect(container.firstChild).toHaveStyle({ '--left-panel-width': '438px' })
  })

  it('keeps restore controls visible after panels collapse', async () => {
    const user = userEvent.setup()
    renderShell()

    await user.click(screen.getByRole('button', { name: 'Collapse character panel' }))
    expect(screen.queryByText('Character slot')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Show character panel' }))
    expect(screen.getByText('Character slot')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Collapse tools panel' }))
    expect(screen.getByRole('button', { name: 'Show tools panel' })).toBeVisible()
  })

  it('offers a visible reset command', async () => {
    const user = userEvent.setup()
    const { container } = renderShell()
    const separator = screen.getByRole('separator', { name: 'Resize tools panel' })
    separator.focus()
    await user.keyboard('{ArrowLeft}')
    expect(container.firstChild).toHaveStyle({ '--right-panel-width': '336px' })
    await user.click(screen.getByRole('button', { name: 'Reset workspace layout' }))
    expect(container.firstChild).toHaveStyle({ '--right-panel-width': '320px' })
  })
})
