import { useState, useEffect } from 'react'
import { listRelationships, createRelationship, deleteRelationship } from './api'
import type { Relationship } from './types'
import { isScopedEvent } from './wsEvents'

interface RelationshipsPanelProps {
  campaignId: number
  lastEvent: unknown
}

export function RelationshipsPanel({ campaignId, lastEvent }: RelationshipsPanelProps) {
  const [rels, setRels] = useState<Relationship[]>([])
  const [showForm, setShowForm] = useState(false)
  const [fromName, setFromName] = useState('')
  const [toName, setToName] = useState('')
  const [relType, setRelType] = useState('neutral')
  const [description, setDescription] = useState('')

  useEffect(() => {
    listRelationships(campaignId).then(setRels).catch(() => {})
  }, [campaignId])

  useEffect(() => {
    if (!isScopedEvent(lastEvent, 'relationship_updated', 'campaign_id', campaignId)) return
    listRelationships(campaignId).then(setRels).catch(() => {})
  }, [campaignId, lastEvent])

  async function handleCreate() {
    if (!fromName || !toName) return
    await createRelationship(campaignId, fromName, toName, relType, description)
    const updated = await listRelationships(campaignId)
    setRels(updated)
    setFromName('')
    setToName('')
    setRelType('neutral')
    setDescription('')
    setShowForm(false)
  }

  async function handleDelete(id: number) {
    await deleteRelationship(id)
    setRels(rels.filter(r => r.id !== id))
  }

  return (
    <div className="relationships-panel">
      <h3>Relationships</h3>
      <button onClick={() => setShowForm(f => !f)}>
        {showForm ? 'Cancel' : '+ Add Relationship'}
      </button>

      {showForm && (
        <div className="relationship-form">
          <input
            placeholder="From (character/NPC name)"
            value={fromName}
            onChange={e => setFromName(e.target.value)}
          />
          <input
            placeholder="To (character/NPC name)"
            value={toName}
            onChange={e => setToName(e.target.value)}
          />
          <select value={relType} onChange={e => setRelType(e.target.value)}>
            <option value="neutral">Neutral</option>
            <option value="ally">Ally</option>
            <option value="rival">Rival</option>
            <option value="enemy">Enemy</option>
            <option value="friend">Friend</option>
            <option value="lover">Lover</option>
            <option value="mentor">Mentor</option>
          </select>
          <textarea
            placeholder="Description"
            value={description}
            onChange={e => setDescription(e.target.value)}
          />
          <button onClick={handleCreate} disabled={!fromName || !toName}>Save</button>
        </div>
      )}

      <ul className="relationship-list">
        {rels.map(r => (
          <li key={r.id}>
            <strong>{r.from_name}</strong> &rarr; <em>{r.relationship_type}</em> &rarr; <strong>{r.to_name}</strong>
            {r.description && <p>{r.description}</p>}
            <button onClick={() => handleDelete(r.id)}>Delete</button>
          </li>
        ))}
      </ul>
    </div>
  )
}
