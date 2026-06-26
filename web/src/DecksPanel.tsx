import { useState, useEffect, useCallback } from 'react'
import { listDecks, createDeck, deleteDeck, shuffleDeck, drawCard, listDeckDraws } from './api'
import type { Deck, DeckCard, DeckDraw } from './types'

interface Props {
  campaignId: number
  sessionId: number
  lastEvent: unknown
}

function parseDeckCards(json: string): DeckCard[] {
  try { return JSON.parse(json) as DeckCard[] } catch { return [] }
}

function parseDeckOrder(json: string): number[] {
  try { return JSON.parse(json) as number[] } catch { return [] }
}

export function DecksPanel({ campaignId, sessionId, lastEvent }: Props) {
  const [decks, setDecks] = useState<Deck[]>([])
  const [draws, setDraws] = useState<DeckDraw[]>([])
  const [lastCard, setLastCard] = useState<{ card: DeckCard; deckName: string } | null>(null)
  const [editMode, setEditMode] = useState(false)
  const [newName, setNewName] = useState('')
  const [newCards, setNewCards] = useState<{ front: string; back: string }[]>([{ front: '', back: '' }])

  const load = useCallback(() => {
    listDecks(campaignId).then(setDecks).catch(console.error)
    listDeckDraws(sessionId).then(setDraws).catch(console.error)
  }, [campaignId, sessionId])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    const e = lastEvent as { type?: string; payload?: Record<string, unknown> } | null
    if (e?.type === 'card_drawn' && e.payload) {
      const p = e.payload
      setLastCard({ card: p['card'] as DeckCard, deckName: p['deck_name'] as string })
      load()
    }
  }, [lastEvent, load])

  async function handleShuffle(deckId: number) {
    await shuffleDeck(deckId)
    load()
  }

  async function handleDraw(deck: Deck) {
    const result = await drawCard(deck.id, sessionId)
    if (result.exhausted) return
    if (result.card) setLastCard({ card: result.card, deckName: deck.name })
    load()
  }

  async function handleCreate() {
    const cards = newCards.filter(c => c.front.trim())
    if (!newName.trim() || cards.length === 0) return
    await createDeck(campaignId, newName.trim(), cards.map(c => ({ front: c.front, back: c.back || undefined })))
    setNewName('')
    setNewCards([{ front: '', back: '' }])
    load()
  }

  async function handleDelete(deck: Deck) {
    if (!confirm(`Delete "${deck.name}"?`)) return
    await deleteDeck(deck.id)
    load()
  }

  const recentDraws = draws.slice(0, 5)

  return (
    <div className="decks-panel">
      <div className="decks-header">
        <span>Decks</span>
        <button className="decks-edit-toggle" onClick={() => setEditMode(!editMode)}>⚙</button>
      </div>

      {!editMode && (
        <>
          {decks.map((deck) => {
            const order = parseDeckOrder(deck.shuffled_order_json)
            const total = parseDeckCards(deck.cards_json).length
            const exhausted = order.length === 0 || deck.draw_index >= order.length
            return (
              <div key={deck.id} className="deck-row">
                <span className="deck-name">{deck.name}</span>
                <span className="deck-progress">{deck.draw_index} / {total}</span>
                <button onClick={() => handleShuffle(deck.id)}>↺ Shuffle</button>
                <button onClick={() => handleDraw(deck)} disabled={exhausted}>
                  {exhausted ? 'Deck empty — reshuffle' : '▶ Draw'}
                </button>
              </div>
            )
          })}

          {lastCard && (
            <div className="last-drawn-card">
              <div className="last-drawn-deck-name">{lastCard.deckName}</div>
              <div className="last-drawn-front">{lastCard.card.front}</div>
              {lastCard.card.back && <div className="last-drawn-back">{lastCard.card.back}</div>}
            </div>
          )}

          {recentDraws.length > 0 && (
            <div className="draw-history">
              <strong>Recent draws</strong>
              {recentDraws.map((d) => {
                let card: DeckCard = { front: '?' }
                try { card = JSON.parse(d.card_json) } catch { /* skip */ }
                return <div key={d.id} className="draw-history-item">{card.front}</div>
              })}
            </div>
          )}
        </>
      )}

      {editMode && (
        <>
          <div className="deck-create-form">
            <input placeholder="Deck name" value={newName} onChange={e => setNewName(e.target.value)} />
            {newCards.map((c, i) => (
              <div key={i} className="deck-card-row">
                <input placeholder="Front" value={c.front} onChange={e => { const n = [...newCards]; n[i] = { ...n[i], front: e.target.value }; setNewCards(n) }} />
                <input placeholder="Back (optional)" value={c.back} onChange={e => { const n = [...newCards]; n[i] = { ...n[i], back: e.target.value }; setNewCards(n) }} />
                <button onClick={() => setNewCards(newCards.filter((_, j) => j !== i))}>×</button>
              </div>
            ))}
            {newCards.length < 200 && (
              <button onClick={() => setNewCards([...newCards, { front: '', back: '' }])}>+ Card</button>
            )}
            <button onClick={handleCreate}>Create Deck</button>
          </div>
          <div className="deck-list-edit">
            {decks.map(deck => (
              <div key={deck.id} className="deck-edit-row">
                <span>{deck.name}</span>
                <button onClick={() => handleDelete(deck)}>× Delete</button>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
