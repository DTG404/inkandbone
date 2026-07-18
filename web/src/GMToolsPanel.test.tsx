import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { GMToolsPanel } from './GMToolsPanel'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

it('announces a safe retryable error for a failed GM tool', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('provider database secret')))
  render(<GMToolsPanel sessionId={1} campaignId={1} aiEnabled />)

  fireEvent.click(screen.getByRole('button', { name: 'Generate Complication' }))
  expect(await screen.findByRole('alert')).toHaveTextContent(/GM tool could not complete/i)
  expect(screen.queryByText(/provider database secret/i)).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Generate Complication' })).not.toBeDisabled()
})
