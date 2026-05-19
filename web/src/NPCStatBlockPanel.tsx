import { useState, useEffect } from 'react'
import { listNpcStats, createNpcStat, updateNpcStat, deleteNpcStat } from './api'
import type { NpcStat } from './types'

interface NPCStatBlockPanelProps {
  campaignId: number
}

const ROLE_OPTIONS = ['brute', 'scout', 'caster', 'leader', 'support', 'minion', 'elite', 'solo', 'other']

export function NPCStatBlockPanel({ campaignId }: NPCStatBlockPanelProps) {
  const [stats, setStats] = useState<NpcStat[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [name, setName] = useState('')
  const [role, setRole] = useState('')
  const [dataJSON, setDataJSON] = useState('{}')
  const [hpMax, setHpMax] = useState(10)
  const [armorClass, setArmorClass] = useState<number | null>(null)
  const [initiativeMod, setInitiativeMod] = useState(0)
  const [skills, setSkills] = useState('')
  const [abilities, setAbilities] = useState('')
  const [loot, setLoot] = useState('')
  const [notes, setNotes] = useState('')
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [search, setSearch] = useState('')

  useEffect(() => {
    listNpcStats(campaignId).then(setStats).catch(() => {})
  }, [campaignId])

  function resetForm() {
    setName('')
    setRole('')
    setDataJSON('{}')
    setHpMax(10)
    setArmorClass(null)
    setInitiativeMod(0)
    setSkills('')
    setAbilities('')
    setLoot('')
    setNotes('')
    setEditId(null)
    setShowForm(false)
  }

  function startEdit(n: NpcStat) {
    setEditId(n.id)
    setName(n.name)
    setRole(n.role)
    setDataJSON(n.data_json)
    setHpMax(n.hp_max)
    setArmorClass(n.armor_class)
    setInitiativeMod(n.initiative_mod)
    setSkills(n.skills)
    setAbilities(n.abilities)
    setLoot(n.loot)
    setNotes(n.notes)
    setShowForm(true)
  }

  async function handleSave() {
    if (!name.trim()) return
    try {
      if (editId !== null) {
        await updateNpcStat(editId, name.trim(), role, dataJSON, hpMax, armorClass, initiativeMod, skills, abilities, loot, notes)
      } else {
        await createNpcStat(campaignId, name.trim(), role, dataJSON, hpMax, armorClass, initiativeMod, skills, abilities, loot, notes)
      }
      const updated = await listNpcStats(campaignId)
      setStats(updated)
      resetForm()
    } catch (err) {
      console.error('Failed to save NPC stat:', err)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteNpcStat(id)
      setStats(stats.filter(s => s.id !== id))
    } catch (err) {
      console.error('Failed to delete NPC stat:', err)
    }
  }

  function parseJSONArray(val: string): string[] {
    try {
      const parsed = JSON.parse(val)
      if (Array.isArray(parsed)) return parsed
      return []
    } catch {
      return []
    }
  }

  function hpPercent(hpMax: number): number {
    return Math.min(100, Math.max(5, (hpMax / 200) * 100))
  }

  const filtered = stats.filter(s =>
    !search || s.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="npc-stats-panel">
      <h3>NPC Stat Blocks</h3>
      <input
        className="npc-stats-search"
        placeholder="Search by name…"
        value={search}
        onChange={e => setSearch(e.target.value)}
      />
      <button onClick={() => { resetForm(); setShowForm(f => !f) }}>
        {showForm ? 'Cancel' : '+ Add Stat Block'}
      </button>

      {showForm && (
        <div className="npc-stats-form">
          <input placeholder="Name *" value={name} onChange={e => setName(e.target.value)} />
          <select value={role} onChange={e => setRole(e.target.value)}>
            <option value="">— Role —</option>
            {ROLE_OPTIONS.map(opt => (
              <option key={opt} value={opt}>{opt.charAt(0).toUpperCase() + opt.slice(1)}</option>
            ))}
          </select>
          <div className="npc-stats-form-row">
            <label>HP: {hpMax}</label>
            <input type="range" min={1} max={500} value={hpMax} onChange={e => setHpMax(Number(e.target.value))} />
          </div>
          <div className="npc-stats-form-row">
            <label>AC:
              <input type="text" inputMode="numeric" pattern="[0-9]*" min={0} max={50} value={armorClass ?? ''} placeholder="—"
                onChange={e => setArmorClass(e.target.value ? Number(e.target.value) : null)} />
            </label>
            <label>Init: {initiativeMod}
              <input type="range" min={-10} max={20} value={initiativeMod} onChange={e => setInitiativeMod(Number(e.target.value))} />
            </label>
          </div>
          <textarea placeholder={'Skills (JSON array, e.g. ["Stealth +4"])'} value={skills} onChange={e => setSkills(e.target.value)} rows={2} />
          <textarea placeholder={'Abilities (JSON array, e.g. ["Nimble Escape"])'} value={abilities} onChange={e => setAbilities(e.target.value)} rows={2} />
          <textarea placeholder={'Loot (JSON array, e.g. ["Shortbow"])'} value={loot} onChange={e => setLoot(e.target.value)} rows={2} />
          <textarea placeholder="Notes" value={notes} onChange={e => setNotes(e.target.value)} rows={3} />
          <button onClick={handleSave} disabled={!name.trim()}>
            {editId !== null ? 'Update' : 'Save'}
          </button>
        </div>
      )}

      <ul className="npc-stats-list">
        {filtered.map(n => (
          <li key={n.id} className="npc-stat-item">
            <div className="npc-stat-header" onClick={() => setExpandedId(expandedId === n.id ? null : n.id)}>
              <strong className="npc-stat-name">{n.name}</strong>
              {n.role && <span className="npc-stat-role-badge">{n.role}</span>}
              <div className="npc-stat-hp-bar">
                <div className="npc-stat-hp-fill" style={{ width: `${hpPercent(n.hp_max)}%` }} />
              </div>
              <span className="npc-stat-ac">{n.armor_class !== null ? `AC ${n.armor_class}` : '—'}</span>
              <span className="npc-stat-expand-icon">{expandedId === n.id ? '▴' : '▾'}</span>
            </div>
            {expandedId === n.id && (
              <div className="npc-stat-body">
                <div className="npc-stat-detail-row"><span className="npc-stat-label">HP:</span> {n.hp_max}</div>
                <div className="npc-stat-detail-row"><span className="npc-stat-label">AC:</span> {n.armor_class !== null ? n.armor_class : '—'}</div>
                <div className="npc-stat-detail-row"><span className="npc-stat-label">Init:</span> {n.initiative_mod >= 0 ? `+${n.initiative_mod}` : n.initiative_mod}</div>
                {n.skills && n.skills !== '[]' && (
                  <div className="npc-stat-section">
                    <span className="npc-stat-label">Skills:</span>
                    <ul className="npc-stat-tag-list">{parseJSONArray(n.skills).map((s, i) => <li key={i} className="npc-stat-tag">{s}</li>)}</ul>
                  </div>
                )}
                {n.abilities && n.abilities !== '[]' && (
                  <div className="npc-stat-section">
                    <span className="npc-stat-label">Abilities:</span>
                    <ul className="npc-stat-tag-list">{parseJSONArray(n.abilities).map((a, i) => <li key={i} className="npc-stat-tag">{a}</li>)}</ul>
                  </div>
                )}
                {n.loot && n.loot !== '[]' && (
                  <div className="npc-stat-section">
                    <span className="npc-stat-label">Loot:</span>
                    <ul className="npc-stat-tag-list">{parseJSONArray(n.loot).map((l, i) => <li key={i} className="npc-stat-tag">{l}</li>)}</ul>
                  </div>
                )}
                {n.notes && <p className="npc-stat-notes">{n.notes}</p>}
                <div className="npc-stat-actions">
                  <button onClick={(e) => { e.stopPropagation(); startEdit(n) }}>Edit</button>
                  <button onClick={(e) => { e.stopPropagation(); handleDelete(n.id) }}>Delete</button>
                </div>
              </div>
            )}
          </li>
        ))}
        {filtered.length === 0 && <p className="npc-stats-empty">No stat blocks yet.</p>}
      </ul>
    </div>
  )
}
