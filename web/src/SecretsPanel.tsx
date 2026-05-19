import { useState, useEffect } from 'react'
import { listSecrets, createSecret, revealSecret, updateSecret, deleteSecret } from './api'
import type { Secret } from './types'

interface SecretsPanelProps {
  campaignId: number
  sessionId: number | null
  lastEvent?: unknown
}

const CATEGORY_OPTIONS = ['secret', 'handout', 'clue']

export function SecretsPanel({ campaignId, sessionId, lastEvent }: SecretsPanelProps) {
  const [secrets, setSecrets] = useState<Secret[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [category, setCategory] = useState('secret')
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [filter, setFilter] = useState<'all' | 'hidden' | 'revealed'>('all')

  useEffect(() => {
    listSecrets(campaignId).then(setSecrets).catch(() => {})
  }, [campaignId, lastEvent])

  function resetForm() {
    setTitle('')
    setContent('')
    setCategory('secret')
    setEditId(null)
    setShowForm(false)
  }

  function startEdit(s: Secret) {
    setEditId(s.id)
    setTitle(s.title)
    setContent(s.content)
    setCategory(s.category)
    setShowForm(true)
  }

  async function handleSave() {
    if (!title.trim()) return
    try {
      if (editId !== null) {
        await updateSecret(editId, title.trim(), content, category)
      } else {
        await createSecret(campaignId, title.trim(), content, category)
      }
      const updated = await listSecrets(campaignId)
      setSecrets(updated)
      resetForm()
    } catch (err) {
      console.error('Failed to save secret:', err)
    }
  }

  async function handleReveal(id: number) {
    if (sessionId === null) return
    try {
      await revealSecret(id, sessionId)
      const updated = await listSecrets(campaignId)
      setSecrets(updated)
    } catch (err) {
      console.error('Failed to reveal secret:', err)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteSecret(id)
      setSecrets(secrets.filter(s => s.id !== id))
    } catch (err) {
      console.error('Failed to delete secret:', err)
    }
  }

  const filtered = secrets.filter(s => {
    if (filter === 'hidden') return !s.revealed
    if (filter === 'revealed') return s.revealed
    return true
  })

  const revealed = filtered.filter(s => s.revealed)
  const hidden = filtered.filter(s => !s.revealed)

  return (
    <div className="secrets-panel">
      <h3>Secrets &amp; Handouts</h3>

      <div className="secrets-filter-bar">
        <button
          className={`secrets-filter-btn${filter === 'all' ? ' active' : ''}`}
          onClick={() => setFilter('all')}
        >
          All ({secrets.length})
        </button>
        <button
          className={`secrets-filter-btn${filter === 'hidden' ? ' active' : ''}`}
          onClick={() => setFilter('hidden')}
        >
          Hidden ({secrets.filter(s => !s.revealed).length})
        </button>
        <button
          className={`secrets-filter-btn${filter === 'revealed' ? ' active' : ''}`}
          onClick={() => setFilter('revealed')}
        >
          Revealed ({secrets.filter(s => s.revealed).length})
        </button>
      </div>

      <button onClick={() => { resetForm(); setShowForm(f => !f) }}>
        {showForm ? 'Cancel' : '+ Add Secret'}
      </button>

      {showForm && (
        <div className="secret-form">
          <input
            placeholder="Title"
            value={title}
            onChange={e => setTitle(e.target.value)}
          />
          <select value={category} onChange={e => setCategory(e.target.value)}>
            {CATEGORY_OPTIONS.map(opt => (
              <option key={opt} value={opt}>{opt.charAt(0).toUpperCase() + opt.slice(1)}</option>
            ))}
          </select>
          <textarea
            placeholder="Content"
            value={content}
            onChange={e => setContent(e.target.value)}
            rows={4}
          />
          <button onClick={handleSave} disabled={!title.trim()}>
            {editId !== null ? 'Update' : 'Save'}
          </button>
        </div>
      )}

      {hidden.length > 0 && (
        <div className="secrets-section">
          <h4 className="secrets-section-title">Hidden</h4>
          <ul className="secret-list">
            {hidden.map(s => (
              <li key={s.id} className={`secret-item secret-item--hidden`}>
                <div className="secret-header" onClick={() => setExpandedId(expandedId === s.id ? null : s.id)}>
                  <span className="secret-category-badge secret-category--{s.category}">{s.category}</span>
                  <span className="secret-title secret-title--dimmed">{s.title}</span>
                  <span className="secret-expand-icon">{expandedId === s.id ? '▴' : '▾'}</span>
                </div>
                {expandedId === s.id && (
                  <div className="secret-body">
                    <p className="secret-content-hidden">Content hidden — reveal to view</p>
                    <div className="secret-actions">
                      <button onClick={(e) => { e.stopPropagation(); handleReveal(s.id) }} disabled={sessionId === null}>
                        Reveal
                      </button>
                      <button onClick={(e) => { e.stopPropagation(); startEdit(s) }}>Edit</button>
                      <button onClick={(e) => { e.stopPropagation(); handleDelete(s.id) }}>Delete</button>
                    </div>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      {revealed.length > 0 && (
        <div className="secrets-section">
          <h4 className="secrets-section-title">Revealed</h4>
          <ul className="secret-list">
            {revealed.map(s => (
              <li key={s.id} className="secret-item secret-item--revealed">
                <div className="secret-header" onClick={() => setExpandedId(expandedId === s.id ? null : s.id)}>
                  <span className="secret-category-badge">{s.category}</span>
                  <span className="secret-title">{s.title}</span>
                  <span className="secret-expand-icon">{expandedId === s.id ? '▴' : '▾'}</span>
                </div>
                {expandedId === s.id && (
                  <div className="secret-body">
                    <p className="secret-content">{s.content}</p>
                    <div className="secret-actions">
                      <button onClick={(e) => { e.stopPropagation(); startEdit(s) }}>Edit</button>
                      <button onClick={(e) => { e.stopPropagation(); handleDelete(s.id) }}>Delete</button>
                    </div>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      {filtered.length === 0 && (
        <p className="secrets-empty">No secrets yet.</p>
      )}
    </div>
  )
}
