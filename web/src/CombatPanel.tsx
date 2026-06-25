import { useState } from 'react'
import type { CombatSnapshot, Combatant } from './types'
import { patchCombatant, advanceTurn, reorderCombatants } from './api'

interface Props {
  combat: CombatSnapshot
}

const STANDARD_CONDITIONS = [
  'Poisoned', 'Prone', 'Stunned', 'Blinded',
  'Exhausted', 'Frightened', 'Paralyzed', 'Invisible',
]

function hpBarClass(current: number, max: number): string {
  if (max === 0) return 'hp-bar-green'
  const ratio = current / max
  if (ratio > 0.5) return 'hp-bar-green'
  if (ratio > 0.25) return 'hp-bar-yellow'
  return 'hp-bar-red'
}

function parseConditions(json: string): string[] {
  try {
    return JSON.parse(json) as string[]
  } catch {
    return []
  }
}

function CombatantRow({
  c,
  isActive,
  onMoveUp,
  onMoveDown,
  isFirst,
  isLast,
}: {
  c: Combatant
  isActive: boolean
  onMoveUp: () => void
  onMoveDown: () => void
  isFirst: boolean
  isLast: boolean
}) {
  const [conditions, setConditions] = useState<string[]>(() => parseConditions(c.conditions_json))
  const [showDropdown, setShowDropdown] = useState(false)
  const [editingInit, setEditingInit] = useState(false)
  const [initInput, setInitInput] = useState(String(c.initiative))

  const pct = c.hp_max > 0 ? Math.max(0, Math.round((c.hp_current / c.hp_max) * 100)) : 0
  const colorClass = hpBarClass(c.hp_current, c.hp_max)

  function removeCondition(cond: string) {
    const next = conditions.filter((x) => x !== cond)
    setConditions(next)
    patchCombatant(c.id, { conditions_json: JSON.stringify(next) }).catch(console.error)
  }

  function addCondition(cond: string) {
    if (conditions.includes(cond)) return
    const next = [...conditions, cond]
    setConditions(next)
    patchCombatant(c.id, { conditions_json: JSON.stringify(next) }).catch(console.error)
    setShowDropdown(false)
  }

  function saveInitiative() {
    const val = parseInt(initInput, 10)
    if (!isNaN(val) && val !== c.initiative) {
      patchCombatant(c.id, { initiative: val }).catch(console.error)
    }
    setEditingInit(false)
  }

  const available = STANDARD_CONDITIONS.filter((s) => !conditions.includes(s))

  return (
    <div className={`combatant-card ${c.is_player ? 'player' : 'enemy'} ${isActive ? 'active-turn' : ''}`}>
      <div className="combatant-header">
        <span className="combatant-name">{c.name}</span>
        <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          {editingInit ? (
            <input
              type="number"
              value={initInput}
              onChange={(e) => setInitInput(e.target.value)}
              onBlur={saveInitiative}
              onKeyDown={(e) => { if (e.key === 'Enter') saveInitiative() }}
              autoFocus
              style={{ width: '48px', fontSize: '11px', padding: '1px 3px',
                       background: 'var(--surface)', border: '1px solid var(--gold)',
                       color: 'var(--text)' }}
            />
          ) : (
            <span
              className="combatant-init"
              onClick={() => { setEditingInit(true); setInitInput(String(c.initiative)) }}
              title="Click to edit initiative"
              style={{ cursor: 'pointer' }}
            >
              Init {c.initiative}
            </span>
          )}
          <button
            onClick={onMoveUp}
            disabled={isFirst}
            title="Move up"
            style={{ padding: '0 3px', fontSize: '10px', lineHeight: 1,
                     opacity: isFirst ? 0.3 : 1, cursor: isFirst ? 'default' : 'pointer',
                     background: 'none', border: 'none', color: 'var(--gold-dim)' }}
          >↑</button>
          <button
            onClick={onMoveDown}
            disabled={isLast}
            title="Move down"
            style={{ padding: '0 3px', fontSize: '10px', lineHeight: 1,
                     opacity: isLast ? 0.3 : 1, cursor: isLast ? 'default' : 'pointer',
                     background: 'none', border: 'none', color: 'var(--gold-dim)' }}
          >↓</button>
        </div>
      </div>
      <div className="hp-bar-track">
        <div className={`hp-bar-fill ${colorClass}`} style={{ width: `${pct}%` }} />
      </div>
      <div className="hp-label">{c.hp_current} / {c.hp_max} HP</div>
      {(conditions.length > 0 || available.length > 0) && (
        <div className="conditions">
          {conditions.map((cond) => (
            <button
              key={cond}
              className="condition-badge condition-badge-btn"
              onClick={() => removeCondition(cond)}
              title={`Remove ${cond}`}
            >
              {cond} ×
            </button>
          ))}
          <div className="condition-add-wrap">
            <button
              className="condition-add-btn"
              onClick={() => setShowDropdown((v) => !v)}
            >+ Condition</button>
            {showDropdown && available.length > 0 && (
              <div className="condition-dropdown">
                {available.map((s) => (
                  <button key={s} className="condition-option" onClick={() => addCondition(s)}>
                    {s}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export function CombatPanel({ combat }: Props) {
  const { encounter, combatants } = combat

  function move(index: number, direction: -1 | 1) {
    const swapIndex = index + direction
    if (swapIndex < 0 || swapIndex >= combatants.length) return
    const newOrder = combatants.map((c) => c.id)
    ;[newOrder[index], newOrder[swapIndex]] = [newOrder[swapIndex], newOrder[index]]
    reorderCombatants(encounter.id, newOrder).catch(console.error)
  }

  return (
    <div className="combat-grimoire">
      <h2>⚔ {encounter.name}</h2>
      {combatants.map((c, idx) => (
        <CombatantRow
          key={c.id}
          c={c}
          isActive={idx === encounter.active_turn_index}
          onMoveUp={() => move(idx, -1)}
          onMoveDown={() => move(idx, 1)}
          isFirst={idx === 0}
          isLast={idx === combatants.length - 1}
        />
      ))}
      <button className="next-turn-btn" onClick={() => advanceTurn(encounter.id).catch(console.error)}>
        Next Turn →
      </button>
    </div>
  )
}
