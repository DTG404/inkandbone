import { useState, type FormEvent } from 'react'
import { fetchSessionInfo, request, setCSRFToken, type SessionInfo } from './transport'

interface LoginScreenProps {
  onAuthenticated: (session: SessionInfo) => void
}

export function LoginScreen({ onAuthenticated }: LoginScreenProps) {
  const [secret, setSecret] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!secret || submitting) return

    const submittedSecret = secret
    setSecret('')
    setSubmitting(true)
    setError(null)
    setCSRFToken(null)
    try {
      const response = await request('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ secret: submittedSecret }),
      })
      if (!response.ok) throw new Error('login rejected')
      const session = await fetchSessionInfo()
      if (!session.authenticated || !session.csrf_token) throw new Error('session unavailable')
      setCSRFToken(session.csrf_token)
      onAuthenticated(session)
    } catch {
      setError('Login failed')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="login-screen">
      <form className="login-card" onSubmit={handleSubmit}>
        <p className="login-kicker">ink &amp; bone</p>
        <h1>Unlock the table</h1>
        <p>Enter the server master secret. It is sent once and is never stored by this browser.</p>
        <label htmlFor="master-secret">Master secret</label>
        <input
          id="master-secret"
          type="password"
          autoComplete="current-password"
          value={secret}
          onChange={(event) => setSecret(event.target.value)}
          disabled={submitting}
          autoFocus
        />
        {error && <p role="alert" className="login-error">{error}</p>}
        <button type="submit" disabled={!secret || submitting}>
          {submitting ? 'Unlocking…' : 'Unlock'}
        </button>
      </form>
    </main>
  )
}
