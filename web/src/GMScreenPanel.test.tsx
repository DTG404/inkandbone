import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { GMScreenPanel } from './GMScreenPanel'

const config = {
  description: '',
  gm_notes: '',
  system_prompt_override: 'Noir tone',
  content_boundaries: 'No graphic harm',
  narrative_locale: 'fr-CA',
  character_count: 1,
  session_count: 2,
  ruleset_name: 'vtm',
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('GMScreenPanel narrative preferences', () => {
  it('labels preference scope and persists boundaries without override authority', async () => {
    const fetchMock = vi.fn().mockImplementation((_input: string, init?: RequestInit) => {
      if (init?.method === 'PATCH') return Promise.resolve({ ok: true })
      return Promise.resolve({ ok: true, json: () => Promise.resolve(config) })
    })
    vi.stubGlobal('fetch', fetchMock)
    render(<GMScreenPanel campaignId={1} sessionId={1} aiEnabled={false} onClose={vi.fn()} />)

    expect(await screen.findByLabelText('Campaign Narration Guidance')).toHaveValue('Noir tone')
    expect(screen.getByText(/cannot override privacy, provider, role, or streaming protocol constraints/i)).toBeInTheDocument()
    const boundaries = screen.getByLabelText('Content Boundaries')
    expect(boundaries).toHaveAttribute('maxlength', '8192')
    expect(screen.getByText(/topics to avoid, soften, or handle off-screen/i)).toBeInTheDocument()
    const locale = screen.getByLabelText('Narrative Locale')
    expect(locale).toHaveValue('fr-CA')
    expect(locale).toHaveAttribute('maxlength', '32')
    expect(screen.getByText(/protocol labels remain stable/i)).toBeInTheDocument()

    fireEvent.change(boundaries, { target: { value: 'Fade violence to black' } })
    fireEvent.blur(boundaries)
    await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/campaigns/1/config', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content_boundaries: 'Fade violence to black' }),
    }))
  })
})
