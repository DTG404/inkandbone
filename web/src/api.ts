import type { GameContext, WorldNote, DiceRoll, TimelineEntry, SessionNPC, Objective, Item, XPEntry, Adventure, Faction, Relationship, NpcStat, Secret, Macro, Deck, DeckCard, DeckDraw } from './types'

export interface CampaignMap {
  id: number;
  campaign_id: number;
  name: string;
  image_path: string;
  created_at: string;
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

export async function fetchContext(): Promise<GameContext> {
  const res = await fetch('/api/context')
  if (!res.ok) throw new Error(`GET /api/context failed: ${res.status}`)
  return res.json()
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
  const res = await fetch(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function patchWorldNoteRevealed(noteId: number, isRevealed: boolean): Promise<void> {
  const res = await fetch(`/api/world-notes/${noteId}/reveal`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_revealed: isRevealed }),
  })
  if (!res.ok) throw new Error(`patchWorldNoteRevealed failed: ${res.status}`)
}

export async function fetchDiceRolls(sessionId: number): Promise<DiceRoll[]> {
  const url = `/api/sessions/${sessionId}/dice-rolls`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function fetchTimeline(sessionId: number): Promise<TimelineEntry[]> {
  const url = `/api/sessions/${sessionId}/timeline`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function fetchMaps(campaignId: number): Promise<CampaignMap[]> {
  const url = `/api/campaigns/${campaignId}/maps`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function fetchMapPins(mapId: number): Promise<MapPin[]> {
  const url = `/api/maps/${mapId}/pins`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function patchSession(sessionId: number, updates: { scene_tags?: string; summary?: string; notes?: string }): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchSession failed: ${res.status}`)
}

export async function patchSessionSummary(sessionId: number, summary: string): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ summary }),
  })
  if (!res.ok) throw new Error(`PATCH /api/sessions/${sessionId} failed: ${res.status}`)
}

export async function patchSessionNotes(sessionId: number, notes: string): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ notes }),
  })
  if (!res.ok) throw new Error(`patchSessionNotes failed: ${res.status}`)
}

export async function generateRecap(sessionId: number): Promise<{ summary: string }> {
  const url = `/api/sessions/${sessionId}/recap`
  const res = await fetch(url, { method: 'POST' })
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`)
  return res.json()
}

export async function draftWorldNote(campaignId: number, hint: string): Promise<{ id: number; title: string; content: string }> {
  const url = `/api/campaigns/${campaignId}/world-notes/draft`
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ hint }),
  })
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`)
  return res.json()
}

export async function patchWorldNotePersonality(noteId: number, personalityJson: string): Promise<void> {
  const url = `/api/world-notes/${noteId}/personality`
  const res = await fetch(url, {
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
  const res = await fetch(url, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`)
  return res.json()
}

export interface Ruleset {
  id: number;
  name: string;
  schema_json: string;
  version: string;
}

export async function fetchRuleset(rulesetId: number): Promise<Ruleset> {
  const res = await fetch(`/api/rulesets/${rulesetId}`)
  if (!res.ok) throw new Error(`fetchRuleset failed: ${res.status}`)
  return res.json()
}

export async function patchCharacter(characterId: number, updates: Record<string, unknown>): Promise<void> {
  const res = await fetch(`/api/characters/${characterId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ data_json: JSON.stringify(updates) }),
  })
  if (!res.ok) throw new Error(`patchCharacter failed: ${res.status}`)
}

export async function uploadPortrait(characterId: number, file: File): Promise<{ portrait_path: string }> {
  const form = new FormData()
  form.append('portrait', file)
  const res = await fetch(`/api/characters/${characterId}/portrait`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) throw new Error(`uploadPortrait failed: ${res.status}`)
  return res.json()
}

export async function sendMessage(sessionId: number, content: string, whisper?: boolean, characterId?: number | null): Promise<void> {
  const body: Record<string, unknown> = { role: 'user', content }
  if (whisper) body['whisper'] = true
  if (characterId != null) body['character_id'] = characterId
  const res = await fetch(`/api/sessions/${sessionId}/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`sendMessage failed: ${res.status}`)
}

export async function generateMap(campaignId: number, name: string, context: string): Promise<CampaignMap> {
  const res = await fetch(`/api/campaigns/${campaignId}/maps/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, context }),
  })
  if (!res.ok) throw new Error(`generateMap failed: ${res.status}`)
  return res.json()
}

export async function gmRespondStream(
  sessionId: number,
  onChunk: (text: string) => void,
): Promise<string> {
  const res = await fetch(`/api/sessions/${sessionId}/gm-respond-stream`, { method: 'POST' })
  if (!res.ok) throw new Error(`gmRespondStream failed: ${res.status}`)
  const reader = res.body?.getReader()
  if (!reader) return ''
  const decoder = new TextDecoder()
  let accumulated = ''
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() ?? ''
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const chunk = line.slice(6)
        accumulated += chunk
        onChunk(chunk)
      }
    }
  }
  // flush remaining buffer
  if (buffer.startsWith('data: ')) {
    const chunk = buffer.slice(6)
    accumulated += chunk
    onChunk(chunk)
  }
  return accumulated
}

export async function rollDice(
  sessionId: number,
  expression: string,
): Promise<{ expression: string; result: number; rolls: number[] }> {
  const res = await fetch(`/api/sessions/${sessionId}/dice-rolls`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ expression }),
  })
  if (!res.ok) throw new Error(`rollDice failed: ${res.status}`)
  return res.json()
}

export async function patchCombatant(
  combatantId: number,
  updates: { conditions_json?: string; hp_current?: number; initiative?: number },
): Promise<void> {
  const res = await fetch(`/api/combatants/${combatantId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCombatant failed: ${res.status}`)
}

export async function reorderCombatants(encounterId: number, ids: number[]): Promise<void> {
  const res = await fetch(`/api/encounters/${encounterId}/combatants/reorder`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  })
  if (!res.ok) throw new Error(`reorderCombatants failed: ${res.status}`)
}

export async function createMapPin(
  mapId: number,
  pin: { x: number; y: number; label: string; note: string; color: string },
): Promise<MapPin> {
  const res = await fetch(`/api/maps/${mapId}/pins`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(pin),
  })
  if (!res.ok) throw new Error(`createMapPin failed: ${res.status}`)
  return res.json()
}

export async function fetchNPCs(sessionId: number): Promise<SessionNPC[]> {
  const res = await fetch(`/api/sessions/${sessionId}/npcs`)
  if (!res.ok) throw new Error(`fetchNPCs failed: ${res.status}`)
  return res.json()
}

export async function createNPC(sessionId: number, name: string, note: string): Promise<SessionNPC> {
  const res = await fetch(`/api/sessions/${sessionId}/npcs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, note }),
  })
  if (!res.ok) throw new Error(`createNPC failed: ${res.status}`)
  return res.json()
}

export async function patchNPC(npcId: number, note: string): Promise<void> {
  const res = await fetch(`/api/npcs/${npcId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note }),
  })
  if (!res.ok) throw new Error(`patchNPC failed: ${res.status}`)
}

export async function deleteNPC(npcId: number): Promise<void> {
  const res = await fetch(`/api/npcs/${npcId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteNPC failed: ${res.status}`)
}

export async function ingestRulebook(rulesetId: number, text: string): Promise<{ chunks_created: number }> {
  const res = await fetch(`/api/rulesets/${rulesetId}/rulebook`, {
    method: 'POST',
    headers: { 'Content-Type': 'text/plain' },
    body: text,
  })
  if (!res.ok) throw new Error(`ingestRulebook failed: ${res.status}`)
  return res.json()
}

export async function fetchObjectives(campaignId: number): Promise<Objective[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/objectives`)
  if (!res.ok) throw new Error(`fetchObjectives failed: ${res.status}`)
  return res.json()
}

export async function createObjective(campaignId: number, title: string, description: string, parentId?: number): Promise<Objective> {
  const res = await fetch(`/api/campaigns/${campaignId}/objectives`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, description, parent_id: parentId ?? null }),
  })
  if (!res.ok) throw new Error(`createObjective failed: ${res.status}`)
  return res.json()
}

export async function patchObjective(id: number, status: string): Promise<void> {
  const res = await fetch(`/api/objectives/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  })
  if (!res.ok) throw new Error(`patchObjective failed: ${res.status}`)
}

export async function deleteObjective(id: number): Promise<void> {
  const res = await fetch(`/api/objectives/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteObjective failed: ${res.status}`)
}

export async function deduplicateObjectives(campaignId: number): Promise<{ deleted: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/objectives/dedup`, { method: 'POST' })
  if (!res.ok) throw new Error(`deduplicateObjectives failed: ${res.status}`)
  return res.json()
}

export async function fetchItems(characterId: number): Promise<Item[]> {
  const res = await fetch(`/api/characters/${characterId}/items`)
  if (!res.ok) throw new Error(`fetchItems failed: ${res.status}`)
  return res.json()
}

export async function createItem(
  characterId: number,
  name: string,
  description: string,
  quantity: number,
): Promise<Item> {
  const res = await fetch(`/api/characters/${characterId}/items`, {
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
  const res = await fetch(`/api/items/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchItem failed: ${res.status}`)
}

export async function deleteItem(id: number): Promise<void> {
  const res = await fetch(`/api/items/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteItem failed: ${res.status}`)
}

export async function advanceTurn(encounterId: number): Promise<void> {
  const res = await fetch(`/api/combat-encounters/${encounterId}/next-turn`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(`advanceTurn failed: ${res.status}`)
}

export async function fetchXP(sessionId: number): Promise<XPEntry[]> {
  const res = await fetch(`/api/sessions/${sessionId}/xp`)
  if (!res.ok) throw new Error(`fetchXP failed: ${res.status}`)
  return res.json()
}

export async function createXP(sessionId: number, note: string, amount?: number): Promise<XPEntry> {
  const res = await fetch(`/api/sessions/${sessionId}/xp`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note, amount: amount ?? null }),
  })
  if (!res.ok) throw new Error(`createXP failed: ${res.status}`)
  return res.json()
}

export async function deleteXP(id: number): Promise<void> {
  const res = await fetch(`/api/xp/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteXP failed: ${res.status}`)
}

export async function postImprovise(sessionId: number): Promise<string> {
  const res = await fetch(`/api/sessions/${sessionId}/improvise`, { method: 'POST' })
  if (!res.ok) throw new Error('Improvise failed')
  const data = await res.json()
  return data.result
}

export async function postPreSessionBrief(campaignId: number): Promise<string> {
  const res = await fetch(`/api/campaigns/${campaignId}/pre-session-brief`, { method: 'POST' })
  if (!res.ok) throw new Error('Pre-session brief failed')
  const data = await res.json()
  return data.result
}

export async function postDetectThreads(sessionId: number): Promise<string> {
  const res = await fetch(`/api/sessions/${sessionId}/detect-threads`, { method: 'POST' })
  if (!res.ok) throw new Error('Detect threads failed')
  const data = await res.json()
  return data.result
}

export async function postCampaignAsk(campaignId: number, question: string): Promise<string> {
  const res = await fetch(`/api/campaigns/${campaignId}/ask`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ question }),
  })
  if (!res.ok) throw new Error('Campaign ask failed')
  const data = await res.json()
  return data.result
}

// --- Management API ---

export interface RulebookSource {
  source: string;
  chunks: number;
}

export async function fetchRulesets(): Promise<Ruleset[]> {
  const res = await fetch('/api/rulesets')
  if (!res.ok) throw new Error(`fetchRulesets failed: ${res.status}`)
  return res.json()
}

export async function fetchCampaigns(): Promise<import('./types').Campaign[]> {
  const res = await fetch('/api/campaigns')
  if (!res.ok) throw new Error(`fetchCampaigns failed: ${res.status}`)
  return res.json()
}

export async function fetchCharacters(campaignId: number): Promise<import('./types').Character[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/characters`)
  if (!res.ok) throw new Error(`fetchCharacters failed: ${res.status}`)
  return res.json()
}

export async function fetchSessions(campaignId: number): Promise<import('./types').Session[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/sessions`)
  if (!res.ok) throw new Error(`fetchSessions failed: ${res.status}`)
  return res.json()
}

export async function createCampaign(name: string, description: string, rulesetId: number): Promise<{ id: number }> {
  const res = await fetch('/api/campaigns', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, ruleset_id: rulesetId }),
  })
  if (!res.ok) throw new Error(`createCampaign failed: ${res.status}`)
  return res.json()
}

export async function deleteCampaign(id: number): Promise<void> {
  const res = await fetch(`/api/campaigns/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteCampaign failed: ${res.status}`)
}

export async function patchCampaign(id: number, updates: { chronicle_night?: number; active?: boolean }): Promise<void> {
  const res = await fetch(`/api/campaigns/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCampaign failed: ${res.status}`)
}

export async function suggestAdvances(characterId: number, hintXP?: number): Promise<void> {
  const res = await fetch(`/api/characters/${characterId}/suggest-advances`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ hint_xp: hintXP ?? 0 }),
  })
  if (!res.ok) throw new Error(`suggestAdvances failed: ${res.status}`)
}

export async function fetchCharacterOptions(rulesetId: number): Promise<Record<string, string[]>> {
  const res = await fetch(`/api/rulesets/${rulesetId}/character-options`)
  if (!res.ok) throw new Error(`fetchCharacterOptions failed: ${res.status}`)
  return res.json()
}

export async function createCharacter(
  campaignId: number,
  name: string,
  overrides?: Record<string, string>,
): Promise<import('./types').Character> {
  const res = await fetch(`/api/campaigns/${campaignId}/characters`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, overrides }),
  })
  if (!res.ok) throw new Error(`createCharacter failed: ${res.status}`)
  return res.json()
}

export async function deleteCharacter(id: number): Promise<void> {
  const res = await fetch(`/api/characters/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteCharacter failed: ${res.status}`)
}

export async function createSession(
  campaignId: number,
  title: string,
  date: string,
): Promise<import('./types').Session> {
  const res = await fetch(`/api/campaigns/${campaignId}/sessions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, date }),
  })
  if (!res.ok) throw new Error(`createSession failed: ${res.status}`)
  return res.json()
}

export async function deleteSession(id: number): Promise<void> {
  const res = await fetch(`/api/sessions/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteSession failed: ${res.status}`)
}

export async function patchSettings(settings: {
  campaign_id?: number | null;
  character_id?: number | null;
  session_id?: number | null;
}): Promise<void> {
  const body: Record<string, number> = {}
  if (settings.campaign_id !== undefined) body['campaign_id'] = settings.campaign_id ?? 0
  if (settings.character_id !== undefined) body['character_id'] = settings.character_id ?? 0
  if (settings.session_id !== undefined) body['session_id'] = settings.session_id ?? 0
  const res = await fetch('/api/settings', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`patchSettings failed: ${res.status}`)
}

export async function fetchRulebookSources(rulesetId: number): Promise<RulebookSource[]> {
  const res = await fetch(`/api/rulesets/${rulesetId}/rulebook`)
  if (!res.ok) throw new Error(`fetchRulebookSources failed: ${res.status}`)
  return res.json()
}

export async function uploadRulebook(
  rulesetId: number,
  file: File,
  source: string,
): Promise<{ chunks_created: number; source: string }> {
  const form = new FormData()
  form.append('rulebook', file)
  form.append('source', source)
  const res = await fetch(`/api/rulesets/${rulesetId}/rulebook`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) throw new Error(`uploadRulebook failed: ${res.status}`)
  return res.json()
}

export interface RulebookResult {
  heading: string
  content: string
  source: string
}

export async function searchRulebook(rulesetId: number, query: string, signal?: AbortSignal): Promise<{ results: RulebookResult[]; mode: string }> {
  const res = await fetch(`/api/rulesets/${rulesetId}/rulebook/search`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query }),
    signal,
  })
  if (!res.ok) throw new Error(`searchRulebook failed: ${res.status}`)
  return res.json()
}

// Oracle
export async function postOracleRoll(table: string, roll: number, rulesetId?: number): Promise<{ result: string; table: string; roll: number }> {
  const res = await fetch('/api/oracle/roll', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ table, roll, ruleset_id: rulesetId }),
  })
  if (!res.ok) throw new Error('Oracle roll failed')
  return res.json()
}

// Tension
export async function getTension(sessionId: number): Promise<number> {
  const res = await fetch(`/api/sessions/${sessionId}/tension`)
  if (!res.ok) throw new Error('Get tension failed')
  const data = await res.json()
  return data.tension_level
}

export async function patchTension(sessionId: number, level: number): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}/tension`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tension_level: level }),
  })
  if (!res.ok) throw new Error('Patch tension failed')
}

// Relationships
export async function listRelationships(campaignId: number): Promise<Relationship[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/relationships`)
  if (!res.ok) throw new Error('List relationships failed')
  return res.json()
}

export async function createRelationship(campaignId: number, fromName: string, toName: string, type: string, description: string): Promise<{ id: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/relationships`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ from_name: fromName, to_name: toName, relationship_type: type, description }),
  })
  if (!res.ok) throw new Error('Create relationship failed')
  return res.json()
}

export async function updateRelationship(id: number, type: string, description: string): Promise<void> {
  const res = await fetch(`/api/relationships/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ relationship_type: type, description }),
  })
  if (!res.ok) throw new Error('Update relationship failed')
}

export async function deleteRelationship(id: number): Promise<void> {
  const res = await fetch(`/api/relationships/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete relationship failed')
}

// Adventures
export async function listAdventures(campaignId: number): Promise<Adventure[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/adventures`)
  if (!res.ok) throw new Error('List adventures failed')
  return res.json()
}

export async function getAdventure(id: number): Promise<Adventure> {
  const res = await fetch(`/api/adventures/${id}`)
  if (!res.ok) throw new Error('Get adventure failed')
  return res.json()
}

export async function createAdventure(
  campaignId: number,
  title: string,
  description: string,
  status: string,
): Promise<{ id: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/adventures`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, description, status, sort_order: 0 }),
  })
  if (!res.ok) throw new Error('Create adventure failed')
  return res.json()
}

export async function updateAdventure(
  id: number,
  title: string,
  description: string,
  status: string,
): Promise<void> {
  const res = await fetch(`/api/adventures/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, description, status }),
  })
  if (!res.ok) throw new Error('Update adventure failed')
}

export async function deleteAdventure(id: number): Promise<void> {
  const res = await fetch(`/api/adventures/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete adventure failed')
}

export async function setSessionAdventure(sessionId: number, adventureId: number | null): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}/adventure`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ adventure_id: adventureId }),
  })
  if (!res.ok) throw new Error('Set session adventure failed')
}

// Factions
export async function listFactions(campaignId: number): Promise<Faction[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/factions`)
  if (!res.ok) throw new Error('List factions failed')
  return res.json()
}

export async function getFaction(id: number): Promise<Faction> {
  const res = await fetch(`/api/factions/${id}`)
  if (!res.ok) throw new Error('Get faction failed')
  return res.json()
}

export async function createFaction(
  campaignId: number,
  name: string,
  description: string,
  factionType: string,
  influence: number,
  resourcesJSON: string,
  color: string,
): Promise<{ id: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/factions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, faction_type: factionType, influence, resources_json: resourcesJSON, color }),
  })
  if (!res.ok) throw new Error('Create faction failed')
  return res.json()
}

export async function updateFaction(
  id: number,
  name: string,
  description: string,
  factionType: string,
  influence: number,
  resourcesJSON: string,
  color: string,
): Promise<void> {
  const res = await fetch(`/api/factions/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, faction_type: factionType, influence, resources_json: resourcesJSON, color }),
  })
  if (!res.ok) throw new Error('Update faction failed')
}

export async function deleteFaction(id: number): Promise<void> {
  const res = await fetch(`/api/factions/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete faction failed')
}

// NPC Stat Blocks
export async function listNpcStats(campaignId: number): Promise<NpcStat[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/npc-stats`)
  if (!res.ok) throw new Error('List NPC stats failed')
  return res.json()
}

export async function getNpcStat(id: number): Promise<NpcStat> {
  const res = await fetch(`/api/npc-stats/${id}`)
  if (!res.ok) throw new Error('Get NPC stat failed')
  return res.json()
}

export async function createNpcStat(
  campaignId: number,
  name: string,
  role: string,
  dataJSON: string,
  hpMax: number,
  armorClass: number | null,
  initiativeMod: number,
  skills: string,
  abilities: string,
  loot: string,
  notes: string,
): Promise<{ id: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/npc-stats`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, role, data_json: dataJSON, hp_max: hpMax, armor_class: armorClass, initiative_mod: initiativeMod, skills, abilities, loot, notes }),
  })
  if (!res.ok) throw new Error('Create NPC stat failed')
  return res.json()
}

export async function updateNpcStat(
  id: number,
  name: string,
  role: string,
  dataJSON: string,
  hpMax: number,
  armorClass: number | null,
  initiativeMod: number,
  skills: string,
  abilities: string,
  loot: string,
  notes: string,
): Promise<void> {
  const res = await fetch(`/api/npc-stats/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, role, data_json: dataJSON, hp_max: hpMax, armor_class: armorClass, initiative_mod: initiativeMod, skills, abilities, loot, notes }),
  })
  if (!res.ok) throw new Error('Update NPC stat failed')
}

export async function deleteNpcStat(id: number): Promise<void> {
  const res = await fetch(`/api/npc-stats/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete NPC stat failed')
}

export async function reanalyzeSession(sessionId: number): Promise<void> {
  const res = await fetch(`/api/sessions/${sessionId}/reanalyze`, { method: 'POST' })
  if (!res.ok) throw new Error('Reanalyze failed')
}

export async function patchCurrency(
  characterId: number,
  updates: { currency_balance?: number; currency_label?: string },
): Promise<void> {
  const res = await fetch(`/api/characters/${characterId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCurrency failed: ${res.status}`)
}

// Secrets
export async function listSecrets(campaignId: number): Promise<Secret[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/secrets`)
  if (!res.ok) throw new Error('List secrets failed')
  return res.json()
}

export async function getSecret(id: number): Promise<Secret> {
  const res = await fetch(`/api/secrets/${id}`)
  if (!res.ok) throw new Error('Get secret failed')
  return res.json()
}

export async function createSecret(campaignId: number, title: string, content: string, category: string): Promise<{ id: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/secrets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, content, category }),
  })
  if (!res.ok) throw new Error('Create secret failed')
  return res.json()
}

export async function revealSecret(id: number, sessionId: number): Promise<void> {
  const res = await fetch(`/api/secrets/${id}/reveal`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId }),
  })
  if (!res.ok) throw new Error('Reveal secret failed')
}

export async function updateSecret(id: number, title: string, content: string, category: string): Promise<void> {
  const res = await fetch(`/api/secrets/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, content, category }),
  })
  if (!res.ok) throw new Error('Update secret failed')
}

export async function deleteSecret(id: number): Promise<void> {
  const res = await fetch(`/api/secrets/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete secret failed')
}

// Calendar
export async function getCampaignCalendar(campaignId: number): Promise<import('./types').CampaignCalendarInfo> {
  const res = await fetch(`/api/campaigns/${campaignId}/calendar`)
  if (!res.ok) throw new Error('Get calendar failed')
  return res.json()
}

export async function patchCampaignCalendar(
  campaignId: number,
  updates: { in_game_year?: number; in_game_month?: number; in_game_day?: number; advance_days?: number; calendar_config?: string },
): Promise<import('./types').CampaignCalendarInfo> {
  const res = await fetch(`/api/campaigns/${campaignId}/calendar`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error('Patch calendar failed')
  return res.json()
}

export async function listCalendarEvents(campaignId: number): Promise<import('./types').CalendarEvent[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/calendar-events`)
  if (!res.ok) throw new Error('List calendar events failed')
  return res.json()
}

export async function createCalendarEvent(
  campaignId: number,
  event: { in_game_year: number; in_game_month: number; in_game_day: number; title: string; description: string; event_type: string; session_id?: number | null },
): Promise<{ id: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/calendar-events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(event),
  })
  if (!res.ok) throw new Error('Create calendar event failed')
  return res.json()
}

export async function deleteCalendarEvent(id: number): Promise<void> {
  const res = await fetch(`/api/calendar-events/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete calendar event failed')
}

// Campaign Config (GM Screen)
export interface CampaignConfig {
  description: string
  gm_notes: string
  system_prompt_override: string
  character_count: number
  session_count: number
  ruleset_name: string
}

export async function fetchCampaignConfig(campaignId: number): Promise<CampaignConfig> {
  const res = await fetch(`/api/campaigns/${campaignId}/config`)
  if (!res.ok) throw new Error(`fetchCampaignConfig failed: ${res.status}`)
  return res.json()
}

export async function patchCampaignConfig(
  campaignId: number,
  updates: { description?: string; gm_notes?: string; system_prompt_override?: string },
): Promise<void> {
  const res = await fetch(`/api/campaigns/${campaignId}/config`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCampaignConfig failed: ${res.status}`)
}

export async function fetchTalentDescription(name: string, system = 'wrath_glory'): Promise<string> {
  const res = await fetch(`/api/talent-description?name=${encodeURIComponent(name)}&system=${encodeURIComponent(system)}`)
  if (!res.ok) return ''
  const data = await res.json() as { description: string }
  return data.description ?? ''
}

export interface AutomationSetting {
  key: string
  label: string
  enabled: boolean
}

export async function fetchAutomationSettings(): Promise<AutomationSetting[]> {
  const res = await fetch('/api/settings/automations')
  if (!res.ok) throw new Error('fetchAutomationSettings failed')
  return res.json()
}

export async function patchAutomationSetting(key: string, enabled: boolean): Promise<void> {
  const res = await fetch('/api/settings/automations', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ key, enabled }),
  })
  if (!res.ok) throw new Error('patchAutomationSetting failed')
}

export async function fetchMacros(characterId: number): Promise<Macro[]> {
  const res = await fetch(`/api/characters/${characterId}/macros`)
  if (!res.ok) throw new Error(`fetchMacros failed: ${res.status}`)
  return res.json()
}

export async function createMacro(
  characterId: number,
  macro: { label: string; action_text: string; color: string },
): Promise<{ id: number }> {
  const res = await fetch(`/api/characters/${characterId}/macros`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(macro),
  })
  if (!res.ok) throw new Error(`createMacro failed: ${res.status}`)
  return res.json()
}

export async function updateMacro(
  id: number,
  macro: { label: string; action_text: string; color: string },
): Promise<void> {
  const res = await fetch(`/api/macros/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(macro),
  })
  if (!res.ok) throw new Error(`updateMacro failed: ${res.status}`)
}

export async function deleteMacro(id: number): Promise<void> {
  const res = await fetch(`/api/macros/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteMacro failed: ${res.status}`)
}

export async function reorderMacros(characterId: number, ids: number[]): Promise<void> {
  const res = await fetch(`/api/characters/${characterId}/macros/reorder`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  })
  if (!res.ok) throw new Error(`reorderMacros failed: ${res.status}`)
}

export async function listDecks(campaignId: number): Promise<Deck[]> {
  const res = await fetch(`/api/campaigns/${campaignId}/decks`)
  if (!res.ok) throw new Error(`listDecks failed: ${res.status}`)
  return res.json()
}

export async function createDeck(campaignId: number, name: string, cards: DeckCard[]): Promise<{ id: number }> {
  const res = await fetch(`/api/campaigns/${campaignId}/decks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, cards }),
  })
  if (!res.ok) throw new Error(`createDeck failed: ${res.status}`)
  return res.json()
}

export async function deleteDeck(deckId: number): Promise<void> {
  const res = await fetch(`/api/decks/${deckId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteDeck failed: ${res.status}`)
}

export async function shuffleDeck(deckId: number): Promise<void> {
  const res = await fetch(`/api/decks/${deckId}/shuffle`, { method: 'POST' })
  if (!res.ok) throw new Error(`shuffleDeck failed: ${res.status}`)
}

export async function drawCard(deckId: number, sessionId: number): Promise<{ card?: DeckCard; draw_index?: number; total?: number; exhausted?: boolean }> {
  const res = await fetch(`/api/decks/${deckId}/draw`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId }),
  })
  if (!res.ok) throw new Error(`drawCard failed: ${res.status}`)
  return res.json()
}

export async function listDeckDraws(sessionId: number): Promise<DeckDraw[]> {
  const res = await fetch(`/api/sessions/${sessionId}/deck-draws`)
  if (!res.ok) throw new Error(`listDeckDraws failed: ${res.status}`)
  return res.json()
}
