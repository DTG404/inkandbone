import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { waitFor } from '@testing-library/dom'
import userEvent from '@testing-library/user-event'
import { AutomationSettingsPanel } from './AutomationSettingsPanel'
import { ToastProvider } from './ui/ToastProvider'

const renderPanel = () => render(<ToastProvider><AutomationSettingsPanel /></ToastProvider>)

const mockSettings = [
  { key: 'extractNPCs', label: 'Extract NPCs', enabled: true, status: 'closed', failure_count: 0, cooling_down: false, queued: 1, running: 0, last_success: null, last_error: '' },
  { key: 'autoGenerateMap', label: 'Auto Generate Map', enabled: false, status: 'open', failure_count: 3, cooling_down: true, queued: 0, running: 0, last_success: null, last_error: 'sanitized' },
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
    renderPanel()
    expect(screen.getByText('Loading automation settings…')).toBeInTheDocument()
  })

  it('renders settings after fetch', async () => {
    renderPanel()
    await screen.findByText('Extract NPCs')
    expect(screen.getByText('Auto Generate Map')).toBeInTheDocument()
  })

  it('renders section title and hint', async () => {
    renderPanel()
    await screen.findByText('Extract NPCs')
    expect(screen.getByText('Automation')).toBeInTheDocument()
    expect(screen.getByText(/Enable or disable background automation goroutines/)).toBeInTheDocument()
  })

  it('explains cooling-down health and next probe behavior', async () => {
    renderPanel()
    await screen.findByText('Auto Generate Map')
    expect(screen.getByText(/cooling down/i)).toBeInTheDocument()
    expect(screen.getByText(/probe.*automatically/i)).toBeInTheDocument()
    expect(screen.getByText(/1 queued/i)).toBeInTheDocument()
  })

  it('toggles setting on checkbox click', async () => {
    const mockFetch = vi.fn().mockImplementation((url: string, options?: RequestInit) => {
      if (url === '/api/settings/automations' && (!options || options.method !== 'PATCH')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockSettings) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) })
    })
    vi.stubGlobal('fetch', mockFetch)

    renderPanel()
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
    renderPanel()
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
