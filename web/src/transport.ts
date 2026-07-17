let csrfToken: string | null = null

export interface SessionInfo {
  authenticated: boolean
  csrf_token?: string
}

export function setCSRFToken(token: string | null): void {
  csrfToken = token
}

export function request(input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> {
  const method = (init.method ?? 'GET').toUpperCase()
  if (!csrfToken || ['GET', 'HEAD', 'OPTIONS'].includes(method)) {
    if (Object.keys(init).length === 0) return fetch(input)
    return fetch(input, init)
  }
  const headers = new Headers(init.headers)
  headers.set('X-CSRF-Token', csrfToken)
  return fetch(input, {
    ...init,
    headers,
    credentials: 'same-origin',
  })
}

export async function fetchSessionInfo(): Promise<SessionInfo> {
  const response = await request('/api/auth/session')
  if (!response.ok) throw new Error(`session request failed: ${response.status}`)
  return response.json()
}
