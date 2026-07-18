import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, cleanup, act, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'
import type { GameContext } from './types'

const mockCtx: GameContext = {
  campaign: { id: 1, ruleset_id: 1, name: 'Greyhawk', description: '', active: true, chronicle_night: 1, chronicle_night_start_dow: -1, created_at: '' },
  character: { id: 1, campaign_id: 1, name: 'Zara', data_json: '{}', portrait_path: '', created_at: '', currency_balance: 0, currency_label: 'Gold' },
  session: { id: 1, campaign_id: 1, title: 'Session 1', date: '2026-04-03', summary: '', notes: '', scene_tags: '', adventure_id: null, created_at: '' },
  recent_messages: [
    { id: 1, session_id: 1, role: 'assistant', content: 'You enter the tavern.', created_at: '' },
    { id: 2, session_id: 1, role: 'user', content: 'I look for a table.', created_at: '' },
  ],
  active_combat: null,
}

class MockWebSocket {
  static instances: MockWebSocket[] = []
  readyState = 0
  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror = null
  close = vi.fn()

  constructor() {
    MockWebSocket.instances.push(this)
  }

  open() {
    this.readyState = 1
    this.onopen?.()
  }

  drop() {
    this.readyState = 3
    this.onclose?.()
  }
}

function requireSocket(socket: MockWebSocket | undefined): asserts socket is MockWebSocket {
  if (!socket) throw new Error('expected WebSocket instance')
}

describe('App', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (url === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx.recent_messages) })
      }
      // WorldNotesPanel and DiceHistoryPanel sub-fetches return empty arrays
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('renders campaign name in state bar', async () => {
    render(<App />)
    expect(await screen.findByText('Greyhawk')).toBeInTheDocument()
  })

  it('shows login without opening the API or WebSocket before authentication', async () => {
    const webSocket = vi.fn().mockImplementation(() => new MockWebSocket())
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (url === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: false }) })
      }
      return Promise.reject(new Error(`unexpected request: ${url}`))
    })
    vi.stubGlobal('WebSocket', webSocket)
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByRole('button', { name: 'Unlock' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/auth/session')
    expect(webSocket).not.toHaveBeenCalled()
  })

  it('renders character name in state bar', async () => {
    render(<App />)
    expect(await screen.findByText('Zara')).toBeInTheDocument()
  })

  it('renders session title in state bar', async () => {
    render(<App />)
    expect(await screen.findByText('Session 1')).toBeInTheDocument()
  })

  it('renders session log messages', async () => {
    render(<App />)
    expect(await screen.findByText('You enter the tavern.')).toBeInTheDocument()
    expect(await screen.findByText('I look for a table.')).toBeInTheDocument()
  })

  it('does not refetch panel data for an unrelated typing event', async () => {
    const fetchMock = vi.mocked(fetch)
    render(<App />)
    expect(await screen.findByText('Greyhawk')).toBeInTheDocument()
    await act(async () => {})
    fetchMock.mockClear()

    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)
    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({
        type: 'typing',
        sequence: 1,
        payload: { session_id: 1, character_name: 'Zara', status: 'start' },
      }) })
    })

    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('authoritatively reloads context for cross-client combat and session state events', async () => {
    let contextCalls = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx.recent_messages) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('Greyhawk')).toBeInTheDocument()
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)

    for (const type of ['combat_started', 'combatant_updated', 'combat_ended', 'turn_advanced', 'tension_updated']) {
      const expected = contextCalls + 1
      await act(async () => {
        socket.onmessage?.({ data: JSON.stringify({ type, payload: { session_id: 1 } }) })
      })
      await waitFor(() => expect(contextCalls).toBe(expected))
    }

    const beforeTyping = contextCalls
    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', payload: { character_name: 'Zara', status: 'done' } }) })
    })
    expect(contextCalls).toBe(beforeTyping)
  })

  it('reloads the authorized transcript so a submitted whisper remains visibly marked', async () => {
    const user = userEvent.setup()
    const publicMessage = {
      id: 1, session_id: 1, role: 'user', content: 'PUBLIC_SENTINEL', whisper: false, created_at: '',
    }
    let authorizedMessages = [publicMessage]
    const fetchMock = vi.fn().mockImplementation((input: string, init?: RequestInit) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ ...mockCtx, recent_messages: [publicMessage] }),
        })
      }
      if (input === '/api/sessions/1/messages' && init?.method === 'POST') {
        const body = JSON.parse(String(init.body)) as { content: string; whisper?: boolean }
        authorizedMessages = [
          publicMessage,
          { id: 2, session_id: 1, role: 'user', content: body.content, whisper: body.whisper === true, created_at: '' },
        ]
        return Promise.resolve({ ok: true })
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(authorizedMessages) })
      }
      if (input === '/api/health') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ai_enabled: false }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    expect(await screen.findByText('PUBLIC_SENTINEL')).toBeInTheDocument()

    await user.click(screen.getByTitle('Enable whisper mode'))
    await user.type(screen.getByPlaceholderText('Whisper (private, no GM response)…'), 'PRIVATE_SENTINEL')
    await user.click(screen.getByRole('button', { name: '↵' }))

    const privateMessage = await screen.findByText('PRIVATE_SENTINEL')
    expect(privateMessage.closest('.prose-player')).toHaveClass('prose-player--whisper')
    expect(screen.getByText('PUBLIC_SENTINEL')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/1/messages')
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/1/messages', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ role: 'user', content: 'PRIVATE_SENTINEL', whisper: true }),
    }))
  })

  it('excludes displayed whispers from AI map-generation context', async () => {
    const publicMessage = {
      id: 1, session_id: 1, role: 'user', content: 'PUBLIC_SENTINEL', whisper: false, created_at: '',
    }
    const privateMessage = {
      id: 2, session_id: 1, role: 'user', content: 'PRIVATE_SENTINEL', whisper: true, created_at: '',
    }
    let generatedContext = ''
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string, init?: RequestInit) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ...mockCtx, recent_messages: [publicMessage] }) })
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([publicMessage, privateMessage]) })
      }
      if (input === '/api/health') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ai_enabled: true }) })
      }
      if (input === '/api/campaigns/1/maps/generate') {
        generatedContext = (JSON.parse(String(init?.body)) as { context: string }).context
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 1 }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('PRIVATE_SENTINEL')).toBeInTheDocument()
    await userEvent.click(await screen.findByRole('button', { name: '✦ Generate Map' }))

    expect(generatedContext).toContain('PUBLIC_SENTINEL')
    expect(generatedContext).not.toContain('PRIVATE_SENTINEL')
  })

  it('excludes displayed whispers from session export', async () => {
    const publicMessage = {
      id: 1, session_id: 1, role: 'user', content: 'PUBLIC_SENTINEL', whisper: false, created_at: '',
    }
    const privateMessage = {
      id: 2, session_id: 1, role: 'user', content: 'PRIVATE_SENTINEL', whisper: true, created_at: '',
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ...mockCtx, recent_messages: [publicMessage] }) })
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([publicMessage, privateMessage]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    let exportedText = ''
    vi.stubGlobal('Blob', class {
      constructor(parts: BlobPart[]) {
        exportedText = parts.map(String).join('')
      }
    })
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:test-export'),
      revokeObjectURL: vi.fn(),
    })
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    render(<App />)
    expect(await screen.findByText('PRIVATE_SENTINEL')).toBeInTheDocument()
    await userEvent.click(screen.getByTitle('Export session'))

    expect(exportedText).toContain('PUBLIC_SENTINEL')
    expect(exportedText).not.toContain('PRIVATE_SENTINEL')
  })

  it('clears messages when context has no active session', async () => {
    const privateMessage = {
      id: 2, session_id: 1, role: 'user', content: 'PRIVATE_SENTINEL', whisper: true, created_at: '',
    }
    const fetchMock = vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ...mockCtx, session: null, recent_messages: [privateMessage] }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    expect(await screen.findByText('No session')).toBeInTheDocument()
    expect(screen.queryByText('PRIVATE_SENTINEL')).not.toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith('/api/sessions/1/messages')
  })

  it('keeps a same-session transcript usable after a transient refresh failure and recovers on the next message event', async () => {
    const originalMessage = {
      id: 1, session_id: 1, role: 'user', content: 'ORIGINAL_PUBLIC', whisper: false, created_at: '',
    }
    const recoveredWhisper = {
      id: 2, session_id: 1, role: 'user', content: 'RECOVERED_PRIVATE', whisper: true, created_at: '',
    }
    let messageCalls = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ...mockCtx, recent_messages: [originalMessage] }) })
      }
      if (input === '/api/sessions/1/messages') {
        messageCalls++
        if (messageCalls === 2) return Promise.resolve({ ok: false, status: 500 })
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(messageCalls === 1 ? [originalMessage] : [originalMessage, recoveredWhisper]),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('ORIGINAL_PUBLIC')).toBeInTheDocument()
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)
    act(() => socket.open())

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'message_created' }) })
    })
    await waitFor(() => expect(messageCalls).toBe(2))
    expect(screen.getByText('ORIGINAL_PUBLIC')).toBeInTheDocument()
    expect(screen.queryByText('Could not load game state')).not.toBeInTheDocument()

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'message_created' }) })
    })
    const privateMessage = await screen.findByText('RECOVERED_PRIVATE')
    expect(privateMessage.closest('.prose-player')).toHaveClass('prose-player--whisper')
    expect(messageCalls).toBe(3)
  })

  it('clears a fatal context error after a later successful realtime refresh', async () => {
    const recoveredMessage = {
      id: 1, session_id: 1, role: 'user', content: 'RECOVERED_PUBLIC', whisper: false, created_at: '',
    }
    let contextCalls = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        if (contextCalls === 1) return Promise.resolve({ ok: false, status: 500 })
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([recoveredMessage]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('Could not load game state')).toBeInTheDocument()
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)
    act(() => socket.open())
    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'future_event' }) })
    })

    expect(await screen.findByText('RECOVERED_PUBLIC')).toBeInTheDocument()
    expect(screen.queryByText('Could not load game state')).not.toBeInTheDocument()
  })

  it('refreshes the transcript only for message-relevant same-session events', async () => {
    let contextCalls = 0
    let messageCalls = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (input === '/api/sessions/1/messages') {
        messageCalls++
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx.recent_messages) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('You enter the tavern.')).toBeInTheDocument()
    expect(messageCalls).toBe(1)
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', payload: { character_name: 'Zara', status: 'done' } }) })
    })
    expect(contextCalls).toBe(1)
    expect(messageCalls).toBe(1)

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'dice_rolled' }) })
    })
    expect(contextCalls).toBe(1)
    expect(messageCalls).toBe(1)

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'message_created' }) })
    })
    await waitFor(() => expect(messageCalls).toBe(2))
    expect(contextCalls).toBe(1)
  })

  it('does not let a same-session non-message refresh cancel an in-flight transcript refresh', async () => {
    const originalMessage = {
      id: 1, session_id: 1, role: 'user', content: 'ORIGINAL_PUBLIC', whisper: false, created_at: '',
    }
    const realtimeMessage = {
      id: 2, session_id: 1, role: 'user', content: 'REALTIME_PUBLIC', whisper: false, created_at: '',
    }
    let contextCalls = 0
    let messageCalls = 0
    let resolveRealtimeMessages: ((value: { ok: boolean; json: () => Promise<unknown> }) => void) | null = null
    const realtimeMessages = new Promise<{ ok: boolean; json: () => Promise<unknown> }>((resolve) => {
      resolveRealtimeMessages = resolve
    })
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (input === '/api/sessions/1/messages') {
        messageCalls++
        if (messageCalls === 2) return realtimeMessages
        return Promise.resolve({ ok: true, json: () => Promise.resolve([originalMessage]) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('ORIGINAL_PUBLIC')).toBeInTheDocument()
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'message_created' }) })
    })
    await waitFor(() => expect(messageCalls).toBe(2))

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', payload: { character_name: 'Zara', status: 'done' } }) })
    })
    expect(contextCalls).toBe(1)
    expect(messageCalls).toBe(2)

    await act(async () => {
      resolveRealtimeMessages?.({
        ok: true,
        json: () => Promise.resolve([originalMessage, realtimeMessage]),
      })
    })
    expect(await screen.findByText('REALTIME_PUBLIC')).toBeInTheDocument()
  })

  it('performs one authoritative reconciliation for a sequence gap without context-fetch storms', async () => {
    let contextCalls = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx.recent_messages) })
      }
      if (input === '/api/rulesets/1') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 1, name: 'vtm' }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('You enter the tavern.')).toBeInTheDocument()
    expect(contextCalls).toBe(1)
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)
    act(() => socket.open())

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', sequence: 1, payload: { character_name: 'Zara', status: 'done' } }) })
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', sequence: 3, payload: { character_name: 'Zara', status: 'done' } }) })
    })
    await waitFor(() => expect(contextCalls).toBe(2))

    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', sequence: 4, payload: { character_name: 'Zara', status: 'done' } }) })
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', sequence: 5, payload: { character_name: 'Zara', status: 'done' } }) })
    })
    expect(contextCalls).toBe(2)
  })

  it('starts a new reconciliation when a second gap arrives during an in-flight load', async () => {
    const pending: Array<(value: { ok: boolean; json: () => Promise<GameContext> }) => void> = []
    let contextCalls = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        if (contextCalls === 1) return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
        return new Promise((resolve) => pending.push(resolve))
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx.recent_messages) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('You enter the tavern.')).toBeInTheDocument()
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)
    act(() => socket.open())
    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', sequence: 1 }) })
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', sequence: 3 }) })
    })
    await waitFor(() => expect(contextCalls).toBe(2))
    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'typing', sequence: 5 }) })
    })
    await waitFor(() => expect(contextCalls).toBe(3))

    await act(async () => {
      pending[1]?.({ ok: true, json: () => Promise.resolve(mockCtx) })
    })
    await act(async () => {
      pending[0]?.({ ok: true, json: () => Promise.resolve(mockCtx) })
    })
    expect(contextCalls).toBe(3)
  })

  it('waits for a successful reopen and retries failed reconciliation until state is visible', async () => {
    let contextCalls = 0
    const reconciled = {
      ...mockCtx,
      campaign: { ...mockCtx.campaign!, chronicle_night: 2 },
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        if (contextCalls === 1) return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
        if (contextCalls === 2) return Promise.reject(new Error('temporary reconnect failure'))
        return Promise.resolve({ ok: true, json: () => Promise.resolve(reconciled) })
      }
      if (input === '/api/sessions/1/messages') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx.recent_messages) })
      }
      if (input === '/api/rulesets/1') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 1, name: 'vtm' }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByTitle('Chronicle night 1')).toBeInTheDocument()
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)
    act(() => socket.open())
    act(() => socket.drop())
    await act(async () => {})
    expect(contextCalls).toBe(1)

    act(() => socket.open())
    await waitFor(() => expect(contextCalls).toBe(3))
    expect(await screen.findByTitle('Chronicle night 2')).toBeInTheDocument()
  })

  it('ignores a stale transcript response after the active session changes', async () => {
    let resolveFirstMessages: ((value: { ok: boolean; json: () => Promise<unknown> }) => void) | null = null
    const firstMessages = new Promise<{ ok: boolean; json: () => Promise<unknown> }>((resolve) => {
      resolveFirstMessages = resolve
    })
    const secondContext: GameContext = {
      ...mockCtx,
      session: { ...mockCtx.session!, id: 2, title: 'Session 2' },
      recent_messages: [],
    }
    let contextCalls = 0
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: string) => {
      if (input === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (input === '/api/context') {
        contextCalls++
        return Promise.resolve({ ok: true, json: () => Promise.resolve(contextCalls === 1 ? mockCtx : secondContext) })
      }
      if (input === '/api/sessions/1/messages') return firstMessages
      if (input === '/api/sessions/2/messages') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve([{ id: 3, session_id: 2, role: 'user', content: 'SECOND_SESSION', created_at: '' }]),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))

    render(<App />)
    expect(await screen.findByText('Session 1')).toBeInTheDocument()
    const socket = MockWebSocket.instances.at(-1)
    requireSocket(socket)
    await act(async () => {
      socket.onmessage?.({ data: JSON.stringify({ type: 'future_event' }) })
    })
    expect(await screen.findByText('Session 2')).toBeInTheDocument()
    expect(await screen.findByText('SECOND_SESSION')).toBeInTheDocument()

    await act(async () => {
      resolveFirstMessages?.({
        ok: true,
        json: () => Promise.resolve([{ id: 2, session_id: 1, role: 'user', content: 'STALE_PRIVATE', whisper: true, created_at: '' }]),
      })
    })
    expect(screen.queryByText('STALE_PRIVATE')).not.toBeInTheDocument()
    expect(screen.getByText('SECOND_SESSION')).toBeInTheDocument()
  })

  it('renders world notes panel', async () => {
    render(<App />)
    await screen.findByText('Greyhawk')
    // WorldNotesPanel shows "No notes found." when empty and "Notes" tab button
    expect(screen.getByText('No notes found.')).toBeInTheDocument()
  })

  it('renders dice roller buttons when character is present', async () => {
    render(<App />)
    await screen.findByText('Greyhawk')
    expect(screen.getByText('Zara')).toBeInTheDocument()
  })

  it('renders portrait img when portrait_path is set', async () => {
    const ctxWithPortrait = {
      ...mockCtx,
      character: { ...mockCtx.character!, portrait_path: 'portraits/zara.jpg' },
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(ctxWithPortrait) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    // Both the state-bar portrait and the character sheet panel render an img with name "Zara"
    const imgs = await screen.findAllByRole('img', { name: 'Zara' })
    expect(imgs.length).toBeGreaterThanOrEqual(1)
    expect(imgs[0]).toHaveAttribute('src', '/api/assets/portraits/1')
  })

  it('does not render portrait img when portrait_path is empty', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    await screen.findByText('Greyhawk')
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('renders combat panel when active_combat is set', async () => {
    const ctxWithCombat: GameContext = {
      ...mockCtx,
      active_combat: {
        encounter: { id: 1, session_id: 1, name: 'Dragon Fight', active: true, active_turn_index: 0, round_number: 1, created_at: '' },
        combatants: [
          { id: 1, encounter_id: 1, character_id: null, name: 'Zara', initiative: 20, hp_current: 40, hp_max: 40, conditions_json: '[]', is_player: true, sort_order: 0 },
        ],
      },
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(ctxWithCombat) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    // CombatPanel renders <h2>⚔ Dragon Fight</h2>
    expect(await screen.findByText('⚔ Dragon Fight')).toBeInTheDocument()
    expect(screen.getAllByText('Zara').length).toBeGreaterThanOrEqual(1)
  })

  it('renders MapPanel', async () => {
    render(<App />)
    await screen.findByText('Greyhawk')
    // MapPanel fetches maps and gets [] back, so it shows "No map uploaded."
    expect(screen.getByText('No map uploaded.')).toBeInTheDocument()
  })

  it('passes aiEnabled to WorldNotesPanel', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (url === '/api/health') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ ai_enabled: true }) })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    await screen.findByText('Greyhawk')
    // aiEnabled=true means the "Draft with AI" button is visible
    expect(screen.getByRole('button', { name: 'Draft with AI' })).toBeInTheDocument()
  })

  it('renders character sheet panel when character is present', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => {
      if (url === '/api/auth/session') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ authenticated: true, csrf_token: 'test-csrf' }) })
      }
      if (url === '/api/context') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCtx) })
      }
      if (url === '/api/rulesets/1') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            id: 1, name: 'dnd5e',
            schema_json: JSON.stringify([{ key: 'hp', label: 'HP', type: 'number' }]),
          }),
        })
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    }))
    render(<App />)
    await screen.findByText('Greyhawk')
    expect(await screen.findByLabelText('HP')).toBeInTheDocument()
  })
})
