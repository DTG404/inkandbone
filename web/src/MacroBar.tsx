import { useState, useEffect } from 'react'
import type { Macro } from './types'
import { fetchMacros, createMacro, updateMacro, deleteMacro, reorderMacros } from './api'

const PRESET_COLORS = [
  { label: 'Gold',   value: 'var(--gold)' },
  { label: 'Red',    value: '#c0392b' },
  { label: 'Blue',   value: '#2980b9' },
  { label: 'Green',  value: '#27ae60' },
  { label: 'Purple', value: '#8e44ad' },
  { label: 'Gray',   value: '#7f8c8d' },
]

interface MacroBarProps {
  characterId: number | null
  onFire: (actionText: string) => void
  disabled?: boolean
}

interface MacroFormState {
  label: string
  action_text: string
  color: string
}

const EMPTY_FORM: MacroFormState = { label: '', action_text: '', color: 'var(--gold)' }

export function MacroBar({ characterId, onFire, disabled }: MacroBarProps) {
  const [macros, setMacros] = useState<Macro[]>([])
  const [editMode, setEditMode] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<MacroFormState>(EMPTY_FORM)
  const [editingId, setEditingId] = useState<number | null>(null)

  useEffect(() => {
    if (!characterId) { setMacros([]); return }
    fetchMacros(characterId).then(setMacros).catch(console.error)
  }, [characterId])

  if (!characterId) return null

  async function handleCreate() {
    if (!form.label.trim() || !form.action_text.trim() || !characterId) return
    try {
      await createMacro(characterId, form)
      const updated = await fetchMacros(characterId)
      setMacros(updated)
      setShowForm(false)
      setForm(EMPTY_FORM)
    } catch (err) {
      console.error(err)
    }
  }

  async function handleUpdate() {
    if (!form.label.trim() || !form.action_text.trim() || editingId === null || !characterId) return
    try {
      await updateMacro(editingId, form)
      const updated = await fetchMacros(characterId)
      setMacros(updated)
      setEditingId(null)
      setForm(EMPTY_FORM)
      setShowForm(false)
    } catch (err) {
      console.error(err)
    }
  }

  async function handleDelete(id: number) {
    if (!characterId) return
    try {
      await deleteMacro(id)
      setMacros((prev) => prev.filter((m) => m.id !== id))
    } catch (err) {
      console.error(err)
    }
  }

  async function handleMove(index: number, direction: -1 | 1) {
    const swapIdx = index + direction
    if (swapIdx < 0 || swapIdx >= macros.length || !characterId) return
    const newOrder = macros.map((m) => m.id)
    ;[newOrder[index], newOrder[swapIdx]] = [newOrder[swapIdx], newOrder[index]]
    try {
      await reorderMacros(characterId, newOrder)
      const updated = await fetchMacros(characterId)
      setMacros(updated)
    } catch (err) {
      console.error(err)
    }
  }

  function openEdit(m: Macro) {
    setEditingId(m.id)
    setForm({ label: m.label, action_text: m.action_text, color: m.color })
    setShowForm(true)
  }

  const atCap = macros.length >= 10

  return (
    <div className="macro-bar">
      <div className="macro-btn-row">
        {macros.map((m, idx) => (
          <div key={m.id} className="macro-slot">
            {editMode && (
              <div className="macro-edit-controls">
                <button
                  onClick={() => handleMove(idx, -1)}
                  disabled={idx === 0}
                  title="Move left"
                  style={{ opacity: idx === 0 ? 0.3 : 1 }}
                >↑</button>
                <button
                  onClick={() => handleMove(idx, 1)}
                  disabled={idx === macros.length - 1}
                  title="Move right"
                  style={{ opacity: idx === macros.length - 1 ? 0.3 : 1 }}
                >↓</button>
                <button onClick={() => openEdit(m)} title="Edit">✎</button>
                <button onClick={() => handleDelete(m.id)} title="Delete" className="macro-delete-btn">×</button>
              </div>
            )}
            <button
              className="macro-btn"
              style={{ borderColor: m.color, color: m.color }}
              onClick={() => !editMode && !disabled && onFire(m.action_text)}
              disabled={disabled || editMode}
              title={m.action_text}
            >
              {m.label}
            </button>
          </div>
        ))}
        <button
          className="macro-add-btn"
          onClick={() => { setEditingId(null); setForm(EMPTY_FORM); setShowForm((v) => !v) }}
          disabled={atCap}
          title={atCap ? 'Maximum 10 macros' : 'Add macro'}
        >+</button>
        <button
          className={`macro-gear-btn${editMode ? ' active' : ''}`}
          onClick={() => setEditMode((v) => !v)}
          title="Edit macros"
        >⚙</button>
      </div>

      {showForm && (
        <div className="macro-form">
          <input
            className="macro-form-input"
            placeholder="Label (e.g. Attack)"
            value={form.label}
            onChange={(e) => setForm((f) => ({ ...f, label: e.target.value }))}
            maxLength={20}
          />
          <input
            className="macro-form-input macro-form-action"
            placeholder="Action (e.g. I swing my sword at the nearest enemy)"
            value={form.action_text}
            onChange={(e) => setForm((f) => ({ ...f, action_text: e.target.value }))}
          />
          <div className="macro-color-row">
            {PRESET_COLORS.map((c) => (
              <button
                key={c.value}
                className={`macro-color-swatch${form.color === c.value ? ' selected' : ''}`}
                style={{ background: c.value === 'var(--gold)' ? '#c9a84c' : c.value }}
                onClick={() => setForm((f) => ({ ...f, color: c.value }))}
                title={c.label}
              />
            ))}
          </div>
          <div className="macro-form-actions">
            <button
              className="macro-form-save"
              onClick={editingId !== null ? handleUpdate : handleCreate}
              disabled={!form.label.trim() || !form.action_text.trim()}
            >
              {editingId !== null ? 'Save' : 'Add'}
            </button>
            <button
              className="macro-form-cancel"
              onClick={() => { setShowForm(false); setEditingId(null); setForm(EMPTY_FORM) }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
