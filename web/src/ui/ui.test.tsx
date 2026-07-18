import { useState } from 'react'
import { cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { Dialog } from './Dialog'
import { Drawer } from './Drawer'
import { IconButton } from './IconButton'
import { StatusRegion } from './StatusRegion'
import { Tab, TabList, TabPanel } from './Tabs'

function DialogHarness() {
  const [open, setOpen] = useState(false)
  const [refreshes, setRefreshes] = useState(0)
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>Open dialog</button>
      <Dialog open={open} title="Campaign details" onClose={() => setOpen(false)}>
        <button type="button">First action</button>
        <button type="button" onClick={() => setRefreshes((count) => count + 1)}>Refresh data {refreshes}</button>
        <button type="button">Last action</button>
      </Dialog>
    </>
  )
}

afterEach(cleanup)

describe('accessible UI primitives', () => {
  it('labels a modal dialog and moves focus into it', async () => {
    const user = userEvent.setup()
    render(<DialogHarness />)

    await user.click(screen.getByRole('button', { name: 'Open dialog' }))

    const dialog = screen.getByRole('dialog', { name: 'Campaign details' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(within(dialog).getByRole('button', { name: 'First action' })).toHaveFocus()
  })

  it('wraps focus, closes on Escape, and restores the opener', async () => {
    const user = userEvent.setup()
    render(<DialogHarness />)
    const opener = screen.getByRole('button', { name: 'Open dialog' })
    await user.click(opener)

    await user.keyboard('{Shift>}{Tab}{/Shift}')
    expect(screen.getByRole('button', { name: 'Last action' })).toHaveFocus()
    await user.keyboard('{Tab}')
    expect(screen.getByRole('button', { name: 'First action' })).toHaveFocus()
    await user.keyboard('{Escape}')

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })

  it('closes when its backdrop is activated without closing from dialog content', async () => {
    const user = userEvent.setup()
    render(<DialogHarness />)
    await user.click(screen.getByRole('button', { name: 'Open dialog' }))

    await user.click(screen.getByRole('dialog'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await user.click(screen.getByTestId('dialog-backdrop'))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('does not reset focus when its parent supplies a new close callback', async () => {
    const user = userEvent.setup()
    render(<DialogHarness />)
    await user.click(screen.getByRole('button', { name: 'Open dialog' }))
    const refresh = screen.getByRole('button', { name: 'Refresh data 0' })

    await user.click(refresh)

    expect(screen.getByRole('button', { name: 'Refresh data 1' })).toHaveFocus()
  })

  it('gives drawers dialog semantics by default', () => {
    render(<Drawer open title="Character" onClose={() => undefined}><p>Sheet</p></Drawer>)
    expect(screen.getByRole('dialog', { name: 'Character' })).toHaveAttribute('aria-modal', 'true')
  })

  it('connects tabs to panels and supports roving keyboard navigation', async () => {
    const user = userEvent.setup()
    function TabsHarness() {
      const [selected, setSelected] = useState('story')
      return (
        <>
          <TabList label="Workspace">
            <Tab id="story" selected={selected === 'story'} onSelect={setSelected}>Story</Tab>
            <Tab id="world" selected={selected === 'world'} onSelect={setSelected}>World</Tab>
            <Tab id="gm" selected={selected === 'gm'} onSelect={setSelected}>GM</Tab>
          </TabList>
          <TabPanel tabId="story" active={selected === 'story'}>Story panel</TabPanel>
          <TabPanel tabId="world" active={selected === 'world'}>World panel</TabPanel>
          <TabPanel tabId="gm" active={selected === 'gm'}>GM panel</TabPanel>
        </>
      )
    }
    render(<TabsHarness />)

    const story = screen.getByRole('tab', { name: 'Story' })
    expect(story).toHaveAttribute('aria-selected', 'true')
    expect(story).toHaveAttribute('aria-controls', 'panel-story')
    expect(screen.getByRole('tabpanel')).toHaveAttribute('aria-labelledby', 'tab-story')

    story.focus()
    await user.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: 'World' })).toHaveFocus()
    expect(screen.getByRole('tab', { name: 'World' })).toHaveAttribute('aria-selected', 'true')
    await user.keyboard('{End}')
    expect(screen.getByRole('tab', { name: 'GM' })).toHaveFocus()
    await user.keyboard('{Home}')
    expect(story).toHaveFocus()
    await user.keyboard('{ArrowLeft}')
    expect(screen.getByRole('tab', { name: 'GM' })).toHaveFocus()
  })

  it('requires an accessible icon-button label and exposes live status priority', () => {
    render(
      <>
        <IconButton label="Delete campaign" icon="×" onClick={() => undefined} />
        <StatusRegion message="Saved" />
        <StatusRegion message="Connection lost" priority="assertive" />
      </>,
    )

    expect(screen.getByRole('button', { name: 'Delete campaign' })).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Saved')
    expect(screen.getByRole('alert')).toHaveTextContent('Connection lost')
  })
})
