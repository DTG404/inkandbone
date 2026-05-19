import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { waitFor } from '@testing-library/dom'
import userEvent from '@testing-library/user-event'
import { AutomationSettingsPanel } from './AutomationSettingsPanel'

const mockSettings = [
  { key: 'extractNPCs', label: 'Extract NPCs', enabled: true },
  { key: 'autoGenerateMap', label: 'Auto Generate Map', enabled: false },
]

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
    if (url === '/api/settings/automations') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve(mockSettings) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
  }))
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('AutomationSettingsPanel', () => {
  it('renders loading state initially', () => {
    render(<AutomationSettingsPanel />)
    expect(screen.getByText('Loading automation settings…')).toBeInTheDocument()
  })

  it('renders settings after fetch', async () => {
    render(<AutomationSettingsPanel />)
    await screen.findByText('Extract NPCs')
    expect(screen.getByText('Auto Generate Map')).toBeInTheDocument()
  })

  it('renders section title and hint', async () => {
    render(<AutomationSettingsPanel />)
    await screen.findByText('Extract NPCs')
    expect(screen.getByText('Automation')).toBeInTheDocument()
    expect(screen.getByText(/Enable or disable background automation goroutines/)).toBeInTheDocument()
  })

  it('toggles setting on checkbox click', async () => {
    const mockFetch = vi.fn().mockImplementation((url: string, options?: RequestInit) => {
      if (url === '/api/settings/automations' && (!options || options.method !== 'PATCH')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockSettings) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })
    vi.stubGlobal('fetch', mockFetch)

    render(<AutomationSettingsPanel />)
    await screen.findByText('Extract NPCs')

    const toggles = screen.getAllByRole('checkbox')
    expect(toggles[0]).toBeChecked()

        // Click the label wrapping the checkbox (checkbox itself is opacity:0 for custom toggle)
    const label = toggles[0].closest('label') || toggles[0].parentElement
    await userEvent.click(label!)

    await waitFor(() => {
      // First call is GET (initial fetch), second call is PATCH (toggle)
      expect(mockFetch.mock.calls.length).toBeGreaterThanOrEqual(2)
      const patchCall = mockFetch.mock.calls[1]
      expect(patchCall[0]).toBe('/api/settings/automations')
      expect(patchCall[1]).toMatchObject({
        method: 'PATCH',
        body: JSON.stringify({ key: mockSettings[0].key, enabled: false }),
      })
    })
  })

  it('disables checkbox during PATCH', async () => {
    const mockFetch = vi.fn().mockImplementation((url: string, options?: RequestInit) => {
      if (url === '/api/settings/automations' && (!options || options.method !== 'PATCH')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockSettings) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })
    vi.stubGlobal('fetch', mockFetch)
    render(<AutomationSettingsPanel />)
    await screen.findByText('Extract NPCs')

    const toggles = screen.getAllByRole('checkbox')

    let resolvePatch: (value: unknown) => void
    mockFetch.mockImplementationOnce(
      () => new Promise((resolve) => { resolvePatch = resolve }),
    )

    await userEvent.click(toggles[0])

    expect(toggles[0]).toBeDisabled()

    resolvePatch!({ ok: true })
    await waitFor(() => {
      expect(toggles[0]).not.toBeDisabled()
    })
  })
})
