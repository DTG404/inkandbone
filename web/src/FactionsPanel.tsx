import { useState, useEffect } from 'react'
import { listFactions, createFaction, updateFaction, deleteFaction } from './api'
import type { Faction } from './types'

interface FactionsPanelProps {
  campaignId: number
}

const FACTION_TYPE_OPTIONS = ['faction', 'guild', 'clan', 'cult', 'kingdom', 'order', 'gang', 'corporation', 'tribe', 'other']

export function FactionsPanel({ campaignId }: FactionsPanelProps) {
  const [factions, setFactions] = useState<Faction[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [factionType, setFactionType] = useState('faction')
  const [influence, setInfluence] = useState(5)
  const [color, setColor] = useState('#c9a84c')
  const [expandedId, setExpandedId] = useState<number | null>(null)

  useEffect(() => {
    listFactions(campaignId).then(setFactions).catch(() => {})
  }, [campaignId])

  function resetForm() {
    setName('')
    setDescription('')
    setFactionType('faction')
    setInfluence(5)
    setColor('#c9a84c')
    setEditId(null)
    setShowForm(false)
  }

  function startEdit(f: Faction) {
    setEditId(f.id)
    setName(f.name)
    setDescription(f.description)
    setFactionType(f.faction_type)
    setInfluence(f.influence)
    setColor(f.color)
    setShowForm(true)
  }

  async function handleSave() {
    if (!name.trim()) return
    try {
      if (editId !== null) {
        await updateFaction(editId, name.trim(), description, factionType, influence, '{}', color)
      } else {
        await createFaction(campaignId, name.trim(), description, factionType, influence, '{}', color)
      }
      const updated = await listFactions(campaignId)
      setFactions(updated)
      resetForm()
    } catch (err) {
      console.error('Failed to save faction:', err)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteFaction(id)
      setFactions(factions.filter(f => f.id !== id))
    } catch (err) {
      console.error('Failed to delete faction:', err)
    }
  }

  return (
    <div className="factions-panel">
      <h3>Factions</h3>
      <button onClick={() => { resetForm(); setShowForm(f => !f) }}>
        {showForm ? 'Cancel' : '+ Add Faction'}
      </button>

      {showForm && (
        <div className="faction-form">
          <input
            placeholder="Faction name"
            value={name}
            onChange={e => setName(e.target.value)}
          />
          <select value={factionType} onChange={e => setFactionType(e.target.value)}>
            {FACTION_TYPE_OPTIONS.map(opt => (
              <option key={opt} value={opt}>{opt.charAt(0).toUpperCase() + opt.slice(1)}</option>
            ))}
          </select>
          <textarea
            placeholder="Description"
            value={description}
            onChange={e => setDescription(e.target.value)}
            rows={3}
          />
          <div className="faction-form-row">
            <label>
              Influence: {influence}
              <input
                type="range"
                min={1}
                max={10}
                value={influence}
                onChange={e => setInfluence(Number(e.target.value))}
              />
            </label>
          </div>
          <div className="faction-form-row">
            <label>
              Color:
              <input
                type="color"
                value={color}
                onChange={e => setColor(e.target.value)}
              />
            </label>
          </div>
          <button onClick={handleSave} disabled={!name.trim()}>
            {editId !== null ? 'Update' : 'Save'}
          </button>
        </div>
      )}

      <ul className="faction-list">
        {factions.map(f => (
          <li key={f.id} className="faction-item">
            <div className="faction-header" onClick={() => setExpandedId(expandedId === f.id ? null : f.id)}>
              <span className="faction-color-dot" style={{ backgroundColor: f.color }} />
              <strong className="faction-name">{f.name}</strong>
              <span className="faction-type-badge">{f.faction_type}</span>
              <div className="faction-influence-bar">
                <div
                  className="faction-influence-fill"
                  style={{ width: `${(f.influence / 10) * 100}%`, backgroundColor: f.color }}
                />
              </div>
              <span className="faction-expand-icon">{expandedId === f.id ? '▴' : '▾'}</span>
            </div>
            {expandedId === f.id && (
              <div className="faction-body">
                {f.description && <p className="faction-description">{f.description}</p>}
                <div className="faction-actions">
                  <button onClick={(e) => { e.stopPropagation(); startEdit(f) }}>Edit</button>
                  <button onClick={(e) => { e.stopPropagation(); handleDelete(f.id) }}>Delete</button>
                </div>
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
