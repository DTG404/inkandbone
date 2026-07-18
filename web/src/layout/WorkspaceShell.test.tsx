import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { WorkspaceShell } from './WorkspaceShell'
import { LAYOUT_STORAGE_KEY } from './useWorkspaceLayout'

describe('WorkspaceShell', () => {
  beforeEach(() => localStorage.clear())
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

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

  function setViewportWidth(width: number) {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query.includes('max-width: 599px')
        ? width <= 599
        : query.includes('min-width: 600px') && width >= 600 && width <= 899,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
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

  it('preloads collapsed desktop columns without reserving their grid tracks', () => {
    setViewportWidth(1440)
    localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify({
      leftWidth: 420, rightWidth: 320, leftCollapsed: true, rightCollapsed: true,
    }))
    const { container } = renderShell()
    expect(container.firstChild).toHaveAttribute('data-left-collapsed', 'true')
    expect(container.firstChild).toHaveAttribute('data-right-collapsed', 'true')
    expect(screen.queryByText('Character slot')).not.toBeInTheDocument()
    expect(screen.queryByText('World slot')).not.toBeInTheDocument()
    expect(screen.getByText('Story slot')).toBeVisible()
  })

  it('renders mobile Character and tools content regardless of persisted desktop collapse', () => {
    setViewportWidth(390)
    localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify({
      leftWidth: 420, rightWidth: 320, leftCollapsed: true, rightCollapsed: true,
    }))
    renderShell()
    expect(screen.getByText('Character slot')).toBeInTheDocument()
    expect(screen.getByText('World slot')).toBeInTheDocument()
  })

  it('cleans pointer resize listeners on pointercancel and unmount', () => {
    const add = vi.spyOn(document, 'addEventListener')
    const remove = vi.spyOn(document, 'removeEventListener')
    const { unmount } = renderShell()
    const separator = screen.getByRole('separator', { name: 'Resize character panel' })
    fireEvent.pointerDown(separator, { clientX: 420, pointerId: 11 })
    const firstMove = add.mock.calls.find(([name]) => name === 'pointermove')?.[1]
    fireEvent.pointerCancel(document, { pointerId: 11 })
    expect(remove).toHaveBeenCalledWith('pointermove', firstMove)
    fireEvent.pointerDown(separator, { clientX: 420, pointerId: 12 })
    const secondMove = add.mock.calls.filter(([name]) => name === 'pointermove').at(-1)?.[1]
    unmount()
    expect(remove).toHaveBeenCalledWith('pointermove', secondMove)
    expect(remove.mock.calls.some(([name]) => name === 'pointercancel')).toBe(true)
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
