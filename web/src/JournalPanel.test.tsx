import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, waitFor, fireEvent } from '@testing-library/react'
import { JournalPanel } from './JournalPanel'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

const makeSession = (id: number, summary: string, notes = '') => ({ id, summary, notes })

describe('JournalPanel', () => {
  it('renders nothing when session is null', () => {
    const { container } = render(
      <JournalPanel session={null} lastEvent={null} aiEnabled={false} />
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders textarea with session summary', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    render(
      <JournalPanel
        session={makeSession(1, 'We fought a dragon.')}
        lastEvent={null}
        aiEnabled={false}
      />
    )
    const textareas = screen.getAllByRole('textbox') as HTMLTextAreaElement[]
    expect(textareas[0].value).toBe('We fought a dragon.')
  })

  it('calls patchSessionSummary on blur', async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) })
    vi.stubGlobal('fetch', mockFetch)
    render(
      <JournalPanel
        session={makeSession(42, 'Initial summary.')}
        lastEvent={null}
        aiEnabled={false}
      />
    )
    const textareas = screen.getAllByRole('textbox')
    fireEvent.change(textareas[0], { target: { value: 'Updated summary.' } })
    fireEvent.blur(textareas[0])
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/sessions/42',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ summary: 'Updated summary.' }),
        })
      )
    })
  })

  it('generate recap button is not rendered when aiEnabled=false', () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    render(
      <JournalPanel
        session={makeSession(1, 'Some summary.')}
        lastEvent={null}
        aiEnabled={false}
      />
    )
    expect(screen.queryByRole('button', { name: /generate recap/i })).toBeNull()
  })

  it('generate recap button calls generateRecap and updates textarea when aiEnabled=true', async () => {
    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) }) // fetchXP
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ summary: 'AI generated recap.' }) }) // generateRecap
    vi.stubGlobal('fetch', mockFetch)
    render(
      <JournalPanel
        session={makeSession(5, 'Old summary.')}
        lastEvent={null}
        aiEnabled={true}
      />
    )
    const button = screen.getByRole('button', { name: /generate recap/i })
    expect(button).not.toBeDisabled()
    fireEvent.click(button)
    await waitFor(() => {
      const textareas = screen.getAllByRole('textbox') as HTMLTextAreaElement[]
      expect(textareas[0].value).toBe('AI generated recap.')
    })
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/sessions/5/recap',
      expect.objectContaining({ method: 'POST' })
    )
  })

  it('announces a safe recoverable error when recap generation fails', async () => {
    const mockFetch = vi.fn()
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) })
      .mockRejectedValueOnce(new Error('provider database secret'))
    vi.stubGlobal('fetch', mockFetch)
    render(
      <JournalPanel session={makeSession(5, 'Old summary.')} lastEvent={null} aiEnabled />
    )

    fireEvent.click(screen.getByRole('button', { name: /generate recap/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/recap could not be generated/i)
    expect(screen.queryByText(/provider database secret/i)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /generate recap/i })).not.toBeDisabled()
  })

  it('session_updated WS event updates draft', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(
      <JournalPanel
        session={makeSession(3, 'Initial.')}
        lastEvent={null}
        aiEnabled={false}
      />
    )
    rerender(
      <JournalPanel
        session={makeSession(3, 'Initial.')}
        lastEvent={{ type: 'session_updated', sequence: 1, payload: { session_id: 3, summary: 'WS updated summary.' } }}
        aiEnabled={false}
      />
    )
    await waitFor(() => {
      const textareas = screen.getAllByRole('textbox') as HTMLTextAreaElement[]
      expect(textareas[0].value).toBe('WS updated summary.')
    })
  })

  it('session_updated event with wrong session_id does not update draft', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    const { rerender } = render(
      <JournalPanel
        session={makeSession(3, 'Initial.')}
        lastEvent={null}
        aiEnabled={false}
      />
    )
    rerender(
      <JournalPanel
        session={makeSession(3, 'Initial.')}
        lastEvent={{ type: 'session_updated', sequence: 1, payload: { session_id: 99, summary: 'Other session.' } }}
        aiEnabled={false}
      />
    )
    const textareas = screen.getAllByRole('textbox') as HTMLTextAreaElement[]
    expect(textareas[0].value).toBe('Initial.')
  })
})
