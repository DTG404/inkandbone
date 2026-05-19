import { useState, useEffect, useRef } from 'react'
import { fetchItems, createItem, patchItem, deleteItem, patchCurrency } from './api'
import type { Item } from './types'

interface InventoryPanelProps {
  characterId: number | null
  characterCurrencyBalance: number
  characterCurrencyLabel: string
  lastEvent: unknown
  characterIdOverride?: number | null
}

interface CurrencyToast {
  delta: number
  label: string
  prevBalance: number
  newBalance: number
}

export function InventoryPanel({
  characterId,
  characterCurrencyBalance,
  characterCurrencyLabel,
  lastEvent,
  characterIdOverride,
}: InventoryPanelProps) {
  const effectiveCharacterId = characterIdOverride ?? characterId
  const [items, setItems] = useState<Item[]>([])
  const [addName, setAddName] = useState('')
  const [saving, setSaving] = useState(false)

  // Currency state (mirrors props but allows inline edit)
  const [balance, setBalance] = useState(characterCurrencyBalance)
  const [label, setLabel] = useState(characterCurrencyLabel)
  const [editingBalance, setEditingBalance] = useState(false)
  const [editingLabel, setEditingLabel] = useState(false)
  const [balanceDraft, setBalanceDraft] = useState('')
  const [labelDraft, setLabelDraft] = useState('')

  // Toast
  const [toast, setToast] = useState<CurrencyToast | null>(null)
  const toastTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Clear timer on unmount to prevent setState on unmounted component
  useEffect(() => {
    return () => {
      if (toastTimer.current) clearTimeout(toastTimer.current)
    }
  }, [])

  // Sync currency from parent when not editing
  useEffect(() => {
    if (!editingBalance) setBalance(characterCurrencyBalance)
  }, [characterCurrencyBalance, editingBalance])

  useEffect(() => {
    if (!editingLabel) setLabel(characterCurrencyLabel)
  }, [characterCurrencyLabel, editingLabel])

  useEffect(() => {
    if (effectiveCharacterId === null) return
    fetchItems(effectiveCharacterId).then(setItems).catch(() => setItems([]))
  }, [effectiveCharacterId])

  useEffect(() => {
    const ev = lastEvent as { type?: string; payload?: Record<string, unknown> } | null
    if (!ev) return

    if (ev.type === 'item_updated' && effectiveCharacterId !== null) {
      fetchItems(effectiveCharacterId).then(setItems).catch(() => {})
    }

    if (ev.type === 'character_updated') {
      const p = ev.payload
      if (p && typeof p.currency_delta === 'number' && p.currency_delta !== 0) {
        const delta = p.currency_delta as number
        const newBal = p.currency_balance as number
        const lbl = (p.currency_label as string) ?? label

        // Clear any existing timer
        if (toastTimer.current) clearTimeout(toastTimer.current)

        setToast({
          delta,
          label: lbl,
          prevBalance: newBal - delta,
          newBalance: newBal,
        })
        setBalance(newBal)
        setLabel(lbl)

        toastTimer.current = setTimeout(() => setToast(null), 5000)
      } else if (p && typeof p.currency_balance === 'number') {
        // Manual patch (no delta) — just sync silently
        setBalance(p.currency_balance as number)
        if (typeof p.currency_label === 'string') setLabel(p.currency_label as string)
      }
    }
  }, [lastEvent, effectiveCharacterId, label])

  async function handleUndoToast() {
    if (!toast || effectiveCharacterId === null) return
    if (toastTimer.current) clearTimeout(toastTimer.current)
    setToast(null)
    setBalance(toast.prevBalance)
    try {
      await patchCurrency(effectiveCharacterId, { currency_balance: toast.prevBalance })
    } catch (err) {
      console.error(err)
    }
  }

  async function handleBalanceSave() {
    const parsed = parseInt(balanceDraft, 10)
    if (isNaN(parsed) || effectiveCharacterId === null) {
      setEditingBalance(false)
      return
    }
    const clamped = Math.max(0, parsed)
    setBalance(clamped)
    setEditingBalance(false)
    try {
      await patchCurrency(effectiveCharacterId, { currency_balance: clamped })
    } catch (err) {
      console.error(err)
    }
  }

  async function handleLabelSave() {
    const trimmed = labelDraft.trim()
    if (!trimmed || effectiveCharacterId === null) {
      setEditingLabel(false)
      return
    }
    setLabel(trimmed)
    setEditingLabel(false)
    try {
      await patchCurrency(effectiveCharacterId, { currency_label: trimmed })
    } catch (err) {
      console.error(err)
    }
  }

  async function handleAdd() {
    if (!addName.trim() || effectiveCharacterId === null) return
    setSaving(true)
    try {
      const item = await createItem(effectiveCharacterId, addName.trim(), '', 1)
      setItems((prev) => [...prev, item])
      setAddName('')
    } catch (err) {
      console.error(err)
    } finally {
      setSaving(false)
    }
  }

  async function handleEquipToggle(item: Item) {
    try {
      await patchItem(item.id, { equipped: !item.equipped })
      setItems((prev) =>
        prev.map((i) => (i.id === item.id ? { ...i, equipped: !item.equipped } : i))
      )
    } catch (err) {
      console.error(err)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteItem(id)
      setItems((prev) => prev.filter((i) => i.id !== id))
    } catch (err) {
      console.error(err)
    }
  }

  if (effectiveCharacterId === null) return null

  const sorted = [
    ...items.filter((i) => i.equipped),
    ...items.filter((i) => !i.equipped),
  ]

  const deltaSign = toast && toast.delta > 0 ? '+' : ''

  return (
    <div className="inventory-panel">
      <div className="inventory-panel-label">Inventory</div>

      {/* Currency row */}
      <div className="currency-row">
        <span className="currency-icon">◈</span>

        {editingLabel ? (
          <input
            className="currency-label-input"
            value={labelDraft}
            onChange={(e) => setLabelDraft(e.target.value)}
            onBlur={handleLabelSave}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleLabelSave()
              if (e.key === 'Escape') setEditingLabel(false)
            }}
            autoFocus
          />
        ) : (
          <span
            className="currency-label-text"
            onClick={() => { setLabelDraft(label); setEditingLabel(true) }}
            title="Click to rename"
          >
            {label}
          </span>
        )}

        {editingBalance ? (
          <input
            className="currency-input"
            value={balanceDraft}
            onChange={(e) => setBalanceDraft(e.target.value)}
            onBlur={handleBalanceSave}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleBalanceSave()
              if (e.key === 'Escape') setEditingBalance(false)
            }}
            autoFocus
          />
        ) : (
          <span
            className="currency-value"
            onClick={() => { setBalanceDraft(String(balance)); setEditingBalance(true) }}
            title="Click to correct"
          >
            {balance}
          </span>
        )}

        <button
          className="currency-edit-btn"
          title="Correct balance"
          onClick={() => { setBalanceDraft(String(balance)); setEditingBalance(true) }}
        >
          ✎
        </button>
      </div>

      {/* Undo toast */}
      {toast && (
        <div className="currency-toast">
          <span>AI: {deltaSign}{toast.delta} {toast.label}</span>
          <button className="currency-toast-undo" onClick={handleUndoToast}>Undo</button>
        </div>
      )}

      {sorted.length === 0 && (
        <p className="empty">No items yet.</p>
      )}

      {sorted.map((item) => (
        <div key={item.id} className={`inventory-item${item.equipped ? ' equipped' : ''}`}>
          <div className="inventory-item-row">
            {item.equipped && <span className="inventory-equipped-icon" title="Equipped">⚔</span>}
            <span className="inventory-item-name">{item.name}</span>
            {item.quantity > 1 && (
              <span className="inventory-item-qty">×{item.quantity}</span>
            )}
            <div className="inventory-actions">
              <button
                className={`inventory-equip-btn${item.equipped ? ' active' : ''}`}
                onClick={() => handleEquipToggle(item)}
                title={item.equipped ? 'Unequip' : 'Equip'}
              >
                {item.equipped ? '▣' : '□'}
              </button>
              <button
                className="inventory-delete-btn"
                onClick={() => handleDelete(item.id)}
                title="Remove item"
              >
                ×
              </button>
            </div>
          </div>
          {item.description && (
            <div className="inventory-item-desc">{item.description}</div>
          )}
        </div>
      ))}

      <div className="inventory-add-form">
        <input
          className="inventory-add-input"
          placeholder="Item name…"
          value={addName}
          onChange={(e) => setAddName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleAdd()
          }}
        />
        <button
          className="inventory-add-btn"
          onClick={handleAdd}
          disabled={saving || !addName.trim()}
        >
          {saving ? '…' : 'Add'}
        </button>
      </div>
    </div>
  )
}
