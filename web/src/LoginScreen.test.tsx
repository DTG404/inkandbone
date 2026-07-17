import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LoginScreen } from './LoginScreen'

describe('LoginScreen', () => {
  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('clears the secret immediately and returns the authenticated session', async () => {
    let finishLogin: ((response: Response) => void) | undefined
    const loginResponse = new Promise<Response>((resolve) => { finishLogin = resolve })
    const fetchMock = vi.fn()
      .mockReturnValueOnce(loginResponse)
      .mockResolvedValueOnce(new Response(JSON.stringify({
        authenticated: true,
        csrf_token: 'csrf-token',
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const onAuthenticated = vi.fn()
    const user = userEvent.setup()
    render(<LoginScreen onAuthenticated={onAuthenticated} />)

    const input = screen.getByLabelText('Master secret')
    await user.type(input, 'not-a-real-secret')
    await user.click(screen.getByRole('button', { name: 'Unlock' }))

    expect(input).toHaveValue('')
    expect(onAuthenticated).not.toHaveBeenCalled()

    finishLogin?.(new Response(null, { status: 204 }))
    await waitFor(() => expect(onAuthenticated).toHaveBeenCalledWith({
      authenticated: true,
      csrf_token: 'csrf-token',
    }))

    const loginInit = fetchMock.mock.calls[0][1] as RequestInit
    expect(JSON.parse(String(loginInit.body))).toEqual({ secret: 'not-a-real-secret' })
  })

  it('shows a generic error after a rejected secret', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 401 })))
    const user = userEvent.setup()
    render(<LoginScreen onAuthenticated={vi.fn()} />)

    await user.type(screen.getByLabelText('Master secret'), 'not-a-real-secret')
    await user.click(screen.getByRole('button', { name: 'Unlock' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Login failed')
    expect(screen.getByLabelText('Master secret')).toHaveValue('')
    expect(screen.queryByText('not-a-real-secret')).not.toBeInTheDocument()
  })
})
