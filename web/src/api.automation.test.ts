import { describe, it, expect, vi, afterEach } from 'vitest'
import { fetchAutomationSettings, patchAutomationSetting } from './api'

afterEach(() => vi.restoreAllMocks())

describe('fetchAutomationSettings', () => {
  it('returns parsed AutomationSetting array on success', async () => {
    const settings = [
      { key: 'extractNPCs', label: 'Extract NPCs', enabled: true },
      { key: 'autoGenerateMap', label: 'Auto Generate Map', enabled: false },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(settings),
    }))

    const result = await fetchAutomationSettings()
    expect(result).toHaveLength(2)
    expect(result[0].key).toBe('extractNPCs')
    expect(result[0].enabled).toBe(true)
    expect(result[1].key).toBe('autoGenerateMap')
    expect(result[1].enabled).toBe(false)
  })

  it('calls the correct URL', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)

    await fetchAutomationSettings()
    expect(mockFetch).toHaveBeenCalledWith('/api/settings/automations')
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(fetchAutomationSettings()).rejects.toThrow('fetchAutomationSettings failed')
  })
})

describe('patchAutomationSetting', () => {
  it('sends PATCH request with correct body', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', mockFetch)

    await patchAutomationSetting('extractNPCs', false)
    expect(mockFetch).toHaveBeenCalledWith('/api/settings/automations', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key: 'extractNPCs', enabled: false }),
    })
  })

  it('throws on non-ok response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    await expect(patchAutomationSetting('extractNPCs', true)).rejects.toThrow('patchAutomationSetting failed')
  })
})
