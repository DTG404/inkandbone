import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/react'
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

    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes[0]).toBeChecked()

    fireEvent.click(checkboxes[0])

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/settings/automations',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ key: 'extractNPCs', enabled: false }),
        }),
      )
    })
  })

  it('disables checkbox while toggling', async () => {
    let resolvePatch: (value: unknown) => void
    const patchPromise = new Promise(resolve => { resolvePatch = resolve })

    const mockFetch = vi.fn().mockImplementation((url: string, options?: RequestInit) => {
      if (url === '/api/settings/automations' && (!options || options.method !== 'PATCH')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockSettings) })
      }
      if (options?.method === 'PATCH') {
        return patchPromise.then(() => ({ ok: true }))
      }
      return Promise.resolve({ ok: true })
    })
    vi.stubGlobal('fetch', mockFetch)

    render(<AutomationSettingsPanel />)
    await screen.findByText('Extract NPCs')

    const checkboxes = screen.getAllByRole('checkbox')
    fireEvent.click(checkboxes[0])

    expect(checkboxes[0]).toBeDisabled()

    resolvePatch!({ ok: true })
    await waitFor(() => {
      expect(checkboxes[0]).not.toBeDisabled()
    })
  })
})
