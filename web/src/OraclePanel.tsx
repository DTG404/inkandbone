import { useState, useEffect } from 'react'
import { getTension, patchTension, postOracleRoll } from './api'

interface OraclePanelProps {
  sessionId: number
}

export function OraclePanel({ sessionId }: OraclePanelProps) {
  const [tension, setTension] = useState(5)
  const [table, setTable] = useState<'action' | 'theme'>('action')
  const [roll, setRoll] = useState(1)
  const [result, setResult] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    getTension(sessionId).then(setTension).catch(() => {})
  }, [sessionId])

  async function handleRoll() {
    setLoading(true)
    try {
      const r = await postOracleRoll(table, roll)
      setResult(r.result)
    } finally {
      setLoading(false)
    }
  }

  async function handleTensionChange(newLevel: number) {
    await patchTension(sessionId, newLevel)
    setTension(newLevel)
  }

  return (
    <div className="oracle-panel">
      <h3>Oracle</h3>

      <div className="oracle-roll">
        <select value={table} onChange={e => setTable(e.target.value as 'action' | 'theme')}>
          <option value="action">Action</option>
          <option value="theme">Theme</option>
        </select>
        <input
          type="text" inputMode="numeric" pattern="[0-9]*"
          min={1}
          max={50}
          value={roll}
          onChange={e => setRoll(Number(e.target.value))}
        />
        <button onClick={handleRoll} disabled={loading}>Roll</button>
      </div>
      {result && <blockquote className="oracle-result">{result}</blockquote>}

      <div className="tension-track">
        <h4>Tension: {tension}</h4>
        <div className="tension-pips">
          {Array.from({ length: 10 }, (_, i) => i + 1).map(n => (
            <button
              key={n}
              className={`pip${n <= tension ? ' active' : ''}`}
              onClick={() => handleTensionChange(n)}
              title={`Set tension to ${n}`}
            >
              {n}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
