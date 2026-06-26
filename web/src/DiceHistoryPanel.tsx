import { useState, useEffect, useRef } from 'react'
import { fetchDiceRolls } from './api'
import type { DiceRoll } from './types'

interface Props {
  sessionId: number
  lastEvent: unknown
}

interface LiveRoll {
  key: number
  expression: string
  result: number
  characterName: string
  hidden: boolean
  isNew: boolean
}

function fromDbRoll(r: DiceRoll, key: number): LiveRoll {
  return { key, expression: r.expression, result: r.result, characterName: '', hidden: false, isNew: false }
}

const isVtMPool = (expr: string) => /\(\d+N\+\d+H\)/.test(expr)

function resultDisplay(roll: LiveRoll): string {
  if (roll.hidden) return '?'
  if (isVtMPool(roll.expression)) {
    return `${roll.result} ${roll.result === 1 ? 'success' : 'successes'}`
  }
  return String(roll.result)
}

export function DiceHistoryPanel({ sessionId, lastEvent }: Props) {
  const [liveRolls, setLiveRolls] = useState<LiveRoll[]>([])
  const keyRef = useRef(0)

  useEffect(() => {
    let ignored = false
    fetchDiceRolls(sessionId)
      .then((data) => {
        if (ignored) return
        // Capture key offset before potential mutation; only apply if no live
        // events have already populated the list.
        const startKey = keyRef.current
        keyRef.current += data.length
        setLiveRolls((current) => {
          if (current.length > 0) return current
          return data.slice(0, 8).map((r, i) => fromDbRoll(r, startKey + i))
        })
      })
      .catch(() => { if (!ignored) setLiveRolls([]) })
    return () => { ignored = true }
  }, [sessionId])

  useEffect(() => {
    const ev = lastEvent as { type?: string; payload?: Record<string, unknown> } | null
    if (ev?.type !== 'dice_rolled' || !ev.payload) return
    const { expression, result, character_name, hidden } = ev.payload as {
      expression: string
      result: number
      character_name: string
      hidden: boolean
    }
    const newRoll: LiveRoll = {
      key: keyRef.current++,
      expression,
      result,
      characterName: character_name ?? '',
      hidden: hidden ?? false,
      isNew: true,
    }
    setLiveRolls((prev) => [newRoll, ...prev].slice(0, 8))
    // Strip isNew after animation duration
    const id = setTimeout(() => {
      setLiveRolls((prev) => prev.map((r) => r.isNew ? { ...r, isNew: false } : r))
    }, 600)
    return () => clearTimeout(id)
  }, [lastEvent])

  if (liveRolls.length === 0) return null

  return (
    <div className="dice-compact">
      <div className="dice-compact-label">Dice</div>
      {liveRolls.map((r) => (
        <div key={r.key} className={`dice-compact-row${r.isNew ? ' dice-entry-new' : ''}`}>
          <span className="dice-compact-expr">
            {r.hidden
              ? '[GM]'
              : r.characterName
              ? <><span>{r.characterName}</span>{`: ${r.expression}`}</>
              : r.expression}
          </span>
          <span className="dice-compact-result">{resultDisplay(r)}</span>
        </div>
      ))}
    </div>
  )
}
