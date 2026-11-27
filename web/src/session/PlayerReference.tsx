import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { request } from '../transport'
import { Dialog } from '../ui/Dialog'

type Mode = 'clarify' | 'player-brief'
type Reading = { answer?: string; location?: string; decision?: string; leads?: string }

function isReturning(key: string) {
  try {
    const last = Number(localStorage.getItem(key))
    return last > 0 && Date.now() - last > 30 * 60 * 1000
  } catch { return false }
}

export function PlayerReference({ sessionId, characterId }: { sessionId: number; characterId: number }) {
  const key = `inkandbone.lastVisit.${sessionId}.${characterId}`
  const [mode, setMode] = useState<Mode | null>(null)
  const [offerBrief, setOfferBrief] = useState(() => isReturning(key))
  const [question, setQuestion] = useState('')
  const [result, setResult] = useState<Reading | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const current = useRef<AbortController | null>(null)

  function close() {
    current.current?.abort()
    setMode(null)
  }
  async function consult(selected: Mode, query = '') {
    current.current?.abort()
    const controller = new AbortController()
    current.current = controller
    setBusy(true)
    setError('')
    setResult(null)
    try {
      const response = await request(`/api/sessions/${sessionId}/${selected}`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, signal: controller.signal,
        body: JSON.stringify({ character_id: characterId, question: query }),
      })
      if (!response.ok) throw new Error('Scene unavailable')
      const reading: Reading = await response.json()
      if (!controller.signal.aborted) setResult(reading)
    } catch {
      if (!controller.signal.aborted) setError('The GM could not consult the scene. Try again.')
    } finally {
      if (!controller.signal.aborted) setBusy(false)
    }
  }
  function open(selected: Mode) {
    setResult(null)
    setBusy(false)
    setError('')
    setMode(selected)
    if (selected === 'player-brief') void consult(selected)
  }
  const welcomeBack = useEffectEvent(() => {
    setOfferBrief(false)
    if (!mode) open('player-brief')
  })
  useEffect(() => {
    if (!offerBrief) return
    const timer = window.setTimeout(welcomeBack, 0)
    return () => window.clearTimeout(timer)
  }, [offerBrief])
  useEffect(() => {
    // Offer the return card after a break, including a closed or backgrounded phone.
    // Store only a visit timestamp. Narration and questions stay in component state.
    function stamp() {
      try { localStorage.setItem(key, String(Date.now())) } catch { /* optional hint */ }
    }
    stamp()
    const onVisibility = () => {
      if (document.hidden) { stamp(); return }
      if (isReturning(key)) setOfferBrief(true)
      stamp()
    }
    document.addEventListener('visibilitychange', onVisibility)
    window.addEventListener('pagehide', stamp)
    return () => {
      current.current?.abort()
      document.removeEventListener('visibilitychange', onVisibility)
      window.removeEventListener('pagehide', stamp)
      stamp()
    }
  }, [key])
  return (
    <div className="player-reference">
      <button type="button" onClick={() => open('player-brief')}>Where were we?</button>
      <button type="button" onClick={() => open('clarify')}>Ask the GM</button>
      {mode && <Dialog open title={mode === 'clarify' ? 'Ask the GM' : 'Where were we?'} onClose={close}>
        <section className="player-reference-card">
          <div className="player-reference-close"><button type="button" onClick={close}>Close</button></div>
          {mode === 'clarify' && <form onSubmit={e => { e.preventDefault(); if (question.trim() && !busy) void consult(mode, question.trim()) }}>
            <label htmlFor="scene-question">What can your character tell from the scene?</label>
            <textarea id="scene-question" maxLength={1000} value={question} onChange={e => setQuestion(e.target.value)} />
            <button disabled={busy || !question.trim()}>Ask without taking a turn</button>
          </form>}
          {busy && <p role="status">Consulting the scene…</p>}
          {error && <p role="alert">{error} {mode === 'player-brief' && <button onClick={() => void consult(mode)}>Retry</button>}</p>}
          {result && (mode === 'clarify'
            ? <p role="status">{result.answer || 'Your character hasn’t established that yet.'}</p>
            : <dl>{([['Location', result.location], ['Pending decision', result.decision], ['Known people or leads', result.leads]] as const).map(([label, value]) =>
              <div key={label}><dt>{label}</dt><dd>{value || 'Not yet established.'}</dd></div>)}</dl>)}
        </section>
      </Dialog>}
    </div>
  )
}
