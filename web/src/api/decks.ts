import { request } from './transport'
import type { Deck, DeckCard, DeckDraw } from '../types'

export async function listDecks(campaignId: number): Promise<Deck[]> {
  const res = await request(`/api/campaigns/${campaignId}/decks`)
  if (!res.ok) throw new Error(`listDecks failed: ${res.status}`)
  return res.json()
}
export async function createDeck(campaignId: number, name: string, cards: DeckCard[]): Promise<{ id: number }> {
  const res = await request(`/api/campaigns/${campaignId}/decks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, cards }),
  })
  if (!res.ok) throw new Error(`createDeck failed: ${res.status}`)
  return res.json()
}

export async function deleteDeck(deckId: number): Promise<void> {
  const res = await request(`/api/decks/${deckId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteDeck failed: ${res.status}`)
}

export async function shuffleDeck(deckId: number): Promise<void> {
  const res = await request(`/api/decks/${deckId}/shuffle`, { method: 'POST' })
  if (!res.ok) throw new Error(`shuffleDeck failed: ${res.status}`)
}

export async function drawCard(deckId: number, sessionId: number): Promise<{ card?: DeckCard; draw_index?: number; total?: number; exhausted?: boolean }> {
  const res = await request(`/api/decks/${deckId}/draw`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId }),
  })
  if (!res.ok) throw new Error(`drawCard failed: ${res.status}`)
  return res.json()
}

export async function listDeckDraws(sessionId: number): Promise<DeckDraw[]> {
  const res = await request(`/api/sessions/${sessionId}/deck-draws`)
  if (!res.ok) throw new Error(`listDeckDraws failed: ${res.status}`)
  return res.json()
}
