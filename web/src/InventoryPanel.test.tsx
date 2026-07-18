import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { InventoryPanel } from './InventoryPanel'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('InventoryPanel currency events', () => {
  it('applies a complete currency delta atomically', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    render(
      <InventoryPanel
        characterId={7}
        characterCurrencyBalance={200}
        characterCurrencyLabel="Gold"
        lastEvent={{
          type: 'character_updated',
          sequence: 1,
          payload: { character_id: 7, currency_delta: 50, currency_balance: 250, currency_label: 'Credits' },
        }}
      />,
    )

    expect(await screen.findByText('250')).toBeInTheDocument()
    expect(screen.getByText('Credits')).toBeInTheDocument()
    expect(screen.getByText('AI: +50 Credits')).toBeInTheDocument()
  })

  it('ignores a delta missing its required balance or label', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    render(
      <InventoryPanel
        characterId={7}
        characterCurrencyBalance={200}
        characterCurrencyLabel="Gold"
        lastEvent={{
          type: 'character_updated',
          sequence: 2,
          payload: { character_id: 7, currency_delta: 50, currency_balance: 250 },
        }}
      />,
    )

    expect(await screen.findByText('200')).toBeInTheDocument()
    expect(screen.getByText('Gold')).toBeInTheDocument()
    expect(screen.queryByText(/AI: \+50/)).not.toBeInTheDocument()
  })
})
