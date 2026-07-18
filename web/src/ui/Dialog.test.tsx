import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { Dialog } from './Dialog'

afterEach(cleanup)

it('does not render dialog content while closed', () => {
  render(<Dialog open={false} title="Hidden" onClose={() => undefined}>Content</Dialog>)
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
})
