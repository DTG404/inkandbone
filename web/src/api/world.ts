import { request } from './transport'
import type { WorldNote, SessionNPC, Objective, Item } from '../types'

export interface CampaignMap {
  id: number;
  campaign_id: number;
  name: string;
  image_path: string;
  created_at: string;
}

export function mapAssetURL(mapId: number): string {
  return `/api/assets/maps/${mapId}`
}

export function portraitAssetURL(characterId: number): string {
  return `/api/assets/portraits/${characterId}`
}

export interface MapPin {
  id: number;
  map_id: number;
  x: number;
  y: number;
  label: string;
  note: string;
  color: string;
  created_at: string;
}


export async function fetchWorldNotes(campaignId: number, q?: string, tag?: string, revealed?: boolean): Promise<WorldNote[]> {
  const params = new URLSearchParams()
  if (q) params.set('q', q)
  if (tag) params.set('tag', tag)
  if (revealed !== undefined) params.set('revealed', String(revealed))
  const qs = params.toString()
  const url = qs
    ? `/api/campaigns/${campaignId}/world-notes?${qs}`
    : `/api/campaigns/${campaignId}/world-notes`
  const res = await request(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function patchWorldNoteRevealed(noteId: number, isRevealed: boolean): Promise<void> {
  const res = await request(`/api/world-notes/${noteId}/reveal`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_revealed: isRevealed }),
  })
  if (!res.ok) throw new Error(`patchWorldNoteRevealed failed: ${res.status}`)
}


export async function fetchMaps(campaignId: number): Promise<CampaignMap[]> {
  const url = `/api/campaigns/${campaignId}/maps`
  const res = await request(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function fetchMapPins(mapId: number): Promise<MapPin[]> {
  const url = `/api/maps/${mapId}/pins`
  const res = await request(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}


export async function draftWorldNote(campaignId: number, hint: string): Promise<{ id: number; title: string; content: string }> {
  const url = `/api/campaigns/${campaignId}/world-notes/draft`
  const res = await request(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ hint }),
  })
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`)
  return res.json()
}

export async function patchWorldNotePersonality(noteId: number, personalityJson: string): Promise<void> {
  const url = `/api/world-notes/${noteId}/personality`
  const res = await request(url, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ personality_json: personalityJson }),
  })
  if (!res.ok) throw new Error(`PATCH ${url} failed: ${res.status}`)
}

export async function uploadMap(campaignId: number, file: File): Promise<CampaignMap> {
  const url = `/api/campaigns/${campaignId}/maps`
  const form = new FormData()
  form.append('image', file)
  form.append('name', file.name.replace(/\.[^.]+$/, ''))
  const res = await request(url, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`)
  return res.json()
}


export async function generateMap(campaignId: number, name: string, context: string): Promise<CampaignMap> {
  const res = await request(`/api/campaigns/${campaignId}/maps/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, context }),
  })
  if (!res.ok) throw new Error(`generateMap failed: ${res.status}`)
  return res.json()
}


export async function createMapPin(
  mapId: number,
  pin: { x: number; y: number; label: string; note: string; color: string },
): Promise<MapPin> {
  const res = await request(`/api/maps/${mapId}/pins`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(pin),
  })
  if (!res.ok) throw new Error(`createMapPin failed: ${res.status}`)
  return res.json()
}

export async function fetchNPCs(sessionId: number): Promise<SessionNPC[]> {
  const res = await request(`/api/sessions/${sessionId}/npcs`)
  if (!res.ok) throw new Error(`fetchNPCs failed: ${res.status}`)
  return res.json()
}

export async function createNPC(sessionId: number, name: string, note: string): Promise<SessionNPC> {
  const res = await request(`/api/sessions/${sessionId}/npcs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, note }),
  })
  if (!res.ok) throw new Error(`createNPC failed: ${res.status}`)
  return res.json()
}

export async function patchNPC(npcId: number, note: string): Promise<void> {
  const res = await request(`/api/npcs/${npcId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note }),
  })
  if (!res.ok) throw new Error(`patchNPC failed: ${res.status}`)
}

export async function deleteNPC(npcId: number): Promise<void> {
  const res = await request(`/api/npcs/${npcId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteNPC failed: ${res.status}`)
}

export async function ingestRulebook(rulesetId: number, text: string): Promise<{ chunks_created: number }> {
  const res = await request(`/api/rulesets/${rulesetId}/rulebook`, {
    method: 'POST',
    headers: { 'Content-Type': 'text/plain' },
    body: text,
  })
  if (!res.ok) throw new Error(`ingestRulebook failed: ${res.status}`)
  return res.json()
}

export async function fetchObjectives(campaignId: number): Promise<Objective[]> {
  const res = await request(`/api/campaigns/${campaignId}/objectives`)
  if (!res.ok) throw new Error(`fetchObjectives failed: ${res.status}`)
  return res.json()
}

export async function createObjective(campaignId: number, title: string, description: string, parentId?: number): Promise<Objective> {
  const res = await request(`/api/campaigns/${campaignId}/objectives`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, description, parent_id: parentId ?? null }),
  })
  if (!res.ok) throw new Error(`createObjective failed: ${res.status}`)
  return res.json()
}

export async function patchObjective(id: number, status: string): Promise<void> {
  const res = await request(`/api/objectives/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  })
  if (!res.ok) throw new Error(`patchObjective failed: ${res.status}`)
}

export async function deleteObjective(id: number): Promise<void> {
  const res = await request(`/api/objectives/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteObjective failed: ${res.status}`)
}

export async function deduplicateObjectives(campaignId: number): Promise<{ deleted: number }> {
  const res = await request(`/api/campaigns/${campaignId}/objectives/dedup`, { method: 'POST' })
  if (!res.ok) throw new Error(`deduplicateObjectives failed: ${res.status}`)
  return res.json()
}

export async function fetchItems(characterId: number): Promise<Item[]> {
  const res = await request(`/api/characters/${characterId}/items`)
  if (!res.ok) throw new Error(`fetchItems failed: ${res.status}`)
  return res.json()
}

export async function createItem(
  characterId: number,
  name: string,
  description: string,
  quantity: number,
): Promise<Item> {
  const res = await request(`/api/characters/${characterId}/items`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, quantity }),
  })
  if (!res.ok) throw new Error(`createItem failed: ${res.status}`)
  return res.json()
}

export async function patchItem(
  id: number,
  updates: { name?: string; description?: string; quantity?: number; equipped?: boolean },
): Promise<void> {
  const res = await request(`/api/items/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchItem failed: ${res.status}`)
}

export async function deleteItem(id: number): Promise<void> {
  const res = await request(`/api/items/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteItem failed: ${res.status}`)
}


export interface MapToken {
  id: number
  map_id: number
  entity_type: 'character' | 'npc'
  entity_id: number
  name: string
  x: number
  y: number
}

export async function fetchMapTokens(mapId: number): Promise<MapToken[]> {
  const res = await request(`/api/maps/${mapId}/tokens`)
  if (!res.ok) throw new Error(`fetchMapTokens failed: ${res.status}`)
  return res.json()
}

export async function placeToken(mapId: number, entityType: string, entityId: number, x: number, y: number): Promise<MapToken> {
  const res = await request(`/api/maps/${mapId}/tokens`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ entity_type: entityType, entity_id: entityId, x, y }),
  })
  if (!res.ok) throw new Error(`placeToken failed: ${res.status}`)
  return res.json()
}

export async function moveToken(tokenId: number, x: number, y: number): Promise<void> {
  const res = await request(`/api/map-tokens/${tokenId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ x, y }),
  })
  if (!res.ok) throw new Error(`moveToken failed: ${res.status}`)
}

export async function removeToken(tokenId: number): Promise<void> {
  const res = await request(`/api/map-tokens/${tokenId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`removeToken failed: ${res.status}`)
}

export interface MapZone {
  id: number
  map_id: number
  name: string
  x: number
  y: number
  width: number
  height: number
  is_revealed: boolean
}

export async function fetchMapZones(mapId: number): Promise<MapZone[]> {
  const res = await request(`/api/maps/${mapId}/zones`)
  if (!res.ok) throw new Error(`fetchMapZones failed: ${res.status}`)
  return res.json()
}

export async function createMapZone(mapId: number, name: string, x: number, y: number, width: number, height: number): Promise<{ id: number }> {
  const res = await request(`/api/maps/${mapId}/zones`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, x, y, width, height }),
  })
  if (!res.ok) throw new Error(`createMapZone failed: ${res.status}`)
  return res.json()
}

export async function patchMapZone(zoneId: number, updates: Partial<Pick<MapZone, 'name' | 'x' | 'y' | 'width' | 'height' | 'is_revealed'>>): Promise<void> {
  const res = await request(`/api/map-zones/${zoneId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchMapZone failed: ${res.status}`)
}

export async function deleteMapZone(zoneId: number): Promise<void> {
  const res = await request(`/api/map-zones/${zoneId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteMapZone failed: ${res.status}`)
}
