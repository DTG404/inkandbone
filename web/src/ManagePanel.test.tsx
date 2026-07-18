import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { ManagePanel } from './ManagePanel'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

it('is a labelled dialog and exposes campaign selection as a keyboard action', async () => {
  const campaign = {
    id: 1, ruleset_id: 1, name: 'Ashes', description: 'A chronicle', active: true,
    chronicle_night: 1, chronicle_night_start_dow: -1, created_at: '',
  }
  vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
    let body: unknown = []
    if (input === '/api/rulesets') body = [{ id: 1, name: 'vtm', display_name: 'Vampire', schema_json: '[]', gm_context: '' }]
    if (input === '/api/campaigns') body = [campaign]
    if (input.includes('/character-options')) body = {}
    return Promise.resolve({ ok: true, json: () => Promise.resolve(body) })
  }))
  const user = userEvent.setup()
  const onClose = vi.fn()
  render(
    <ManagePanel
      activeCampaignId={1}
      activeCharacterId={null}
      activeSessionId={null}
      onClose={onClose}
      onContextChanged={() => undefined}
    />,
  )

  expect(await screen.findByRole('dialog', { name: 'Manage Campaign' })).toBeInTheDocument()
  const selectCampaign = await screen.findByRole('button', { name: 'Select campaign Ashes' })
  selectCampaign.focus()
  await user.keyboard('{Enter}')
  expect(selectCampaign).toHaveFocus()
  expect(screen.getByRole('button', { name: 'Close Manage Campaign' })).toBeInTheDocument()
})

it('announces safe management errors without exposing provider details', async () => {
  vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string, init?: RequestInit) => {
    if (input === '/api/rulesets') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 1, name: 'vtm', display_name: 'Vampire', schema_json: '[]', gm_context: '' }]) })
    }
    if (input === '/api/campaigns' && init?.method === 'POST') {
      return Promise.resolve({ ok: false, status: 503, json: () => Promise.resolve({}) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
  }))
  const user = userEvent.setup()
  render(
    <ManagePanel
      activeCampaignId={null}
      activeCharacterId={null}
      activeSessionId={null}
      onClose={() => undefined}
      onContextChanged={() => undefined}
    />,
  )

  await user.type(screen.getByPlaceholderText('Campaign name'), 'Ashes')
  await user.click(screen.getByRole('button', { name: 'Create Campaign' }))

  expect(await screen.findByRole('alert')).toHaveTextContent('The campaign could not be created. Try again.')
  expect(screen.queryByText(/createCampaign failed|503/)).not.toBeInTheDocument()
})

it('announces a destructive management action after it succeeds', async () => {
  const campaign = {
    id: 1, ruleset_id: 1, name: 'Ashes', description: '', active: false,
    chronicle_night: 1, chronicle_night_start_dow: -1, created_at: '',
  }
  let deleted = false
  vi.stubGlobal('confirm', vi.fn(() => true))
  vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string, init?: RequestInit) => {
    if (input === '/api/rulesets') return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    if (input === '/api/campaigns' && !init?.method) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve(deleted ? [] : [campaign]) })
    }
    if (input === '/api/campaigns/1' && init?.method === 'DELETE') {
      deleted = true
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
  }))
  const user = userEvent.setup()
  render(
    <ManagePanel
      activeCampaignId={null}
      activeCharacterId={null}
      activeSessionId={null}
      onClose={() => undefined}
      onContextChanged={() => undefined}
    />,
  )

  await user.click(await screen.findByRole('button', { name: 'Delete' }))

  expect(await screen.findByRole('status')).toHaveTextContent('Campaign deleted.')
})
