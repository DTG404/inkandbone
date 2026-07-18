import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { ToastProvider, useToast } from './ToastProvider'

afterEach(cleanup)

function Harness() {
  const toast = useToast()
  return (
    <>
      <button type="button" onClick={() => toast.info('Saved')}>Info</button>
      <button type="button" onClick={() => toast.error('Could not save', { label: 'Retry', onClick: () => toast.success('Recovered') })}>Error</button>
      <button type="button" onClick={() => {
        for (let index = 1; index <= 6; index++) toast.info(`Message ${index}`)
      }}>Fill</button>
    </>
  )
}

describe('ToastProvider', () => {
  it('exposes stable polite and assertive live regions with retry actions', async () => {
    const user = userEvent.setup()
    render(<ToastProvider><Harness /></ToastProvider>)

    await user.click(screen.getByRole('button', { name: 'Info' }))
    expect(screen.getByRole('status')).toHaveTextContent('Saved')
    await user.click(screen.getByRole('button', { name: 'Error' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Could not save')
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(screen.getByRole('status')).toHaveTextContent('Recovered')
  })

  it('bounds the visible queue to the five newest messages', async () => {
    const user = userEvent.setup()
    render(<ToastProvider><Harness /></ToastProvider>)
    await user.click(screen.getByRole('button', { name: 'Fill' }))

    expect(screen.queryByText('Message 1')).not.toBeInTheDocument()
    for (let index = 2; index <= 6; index++) expect(screen.getByText(`Message ${index}`)).toBeInTheDocument()
  })
})
