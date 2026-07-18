import { request } from './transport'
import type { Adventure, Faction, Relationship, NpcStat, Secret } from '../types'

export async function listRelationships(campaignId: number): Promise<Relationship[]> {
  const res = await request(`/api/campaigns/${campaignId}/relationships`)
  if (!res.ok) throw new Error('List relationships failed')
  return res.json()
}

export async function createRelationship(campaignId: number, fromName: string, toName: string, type: string, description: string): Promise<{ id: number }> {
  const res = await request(`/api/campaigns/${campaignId}/relationships`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ from_name: fromName, to_name: toName, relationship_type: type, description }),
  })
  if (!res.ok) throw new Error('Create relationship failed')
  return res.json()
}

export async function updateRelationship(id: number, type: string, description: string): Promise<void> {
  const res = await request(`/api/relationships/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ relationship_type: type, description }),
  })
  if (!res.ok) throw new Error('Update relationship failed')
}

export async function deleteRelationship(id: number): Promise<void> {
  const res = await request(`/api/relationships/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete relationship failed')
}

// Adventures
export async function listAdventures(campaignId: number): Promise<Adventure[]> {
  const res = await request(`/api/campaigns/${campaignId}/adventures`)
  if (!res.ok) throw new Error('List adventures failed')
  return res.json()
}

export async function getAdventure(id: number): Promise<Adventure> {
  const res = await request(`/api/adventures/${id}`)
  if (!res.ok) throw new Error('Get adventure failed')
  return res.json()
}

export async function createAdventure(
  campaignId: number,
  title: string,
  description: string,
  status: string,
): Promise<{ id: number }> {
  const res = await request(`/api/campaigns/${campaignId}/adventures`, {
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
  const res = await request(`/api/adventures/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, description, status }),
  })
  if (!res.ok) throw new Error('Update adventure failed')
}

export async function deleteAdventure(id: number): Promise<void> {
  const res = await request(`/api/adventures/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete adventure failed')
}

export async function setSessionAdventure(sessionId: number, adventureId: number | null): Promise<void> {
  const res = await request(`/api/sessions/${sessionId}/adventure`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ adventure_id: adventureId }),
  })
  if (!res.ok) throw new Error('Set session adventure failed')
}

// Factions
export async function listFactions(campaignId: number): Promise<Faction[]> {
  const res = await request(`/api/campaigns/${campaignId}/factions`)
  if (!res.ok) throw new Error('List factions failed')
  return res.json()
}

export async function getFaction(id: number): Promise<Faction> {
  const res = await request(`/api/factions/${id}`)
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
  const res = await request(`/api/campaigns/${campaignId}/factions`, {
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
  const res = await request(`/api/factions/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, faction_type: factionType, influence, resources_json: resourcesJSON, color }),
  })
  if (!res.ok) throw new Error('Update faction failed')
}

export async function deleteFaction(id: number): Promise<void> {
  const res = await request(`/api/factions/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete faction failed')
}

// NPC Stat Blocks
export async function listNpcStats(campaignId: number): Promise<NpcStat[]> {
  const res = await request(`/api/campaigns/${campaignId}/npc-stats`)
  if (!res.ok) throw new Error('List NPC stats failed')
  return res.json()
}

export async function getNpcStat(id: number): Promise<NpcStat> {
  const res = await request(`/api/npc-stats/${id}`)
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
  const res = await request(`/api/campaigns/${campaignId}/npc-stats`, {
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
  const res = await request(`/api/npc-stats/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, role, data_json: dataJSON, hp_max: hpMax, armor_class: armorClass, initiative_mod: initiativeMod, skills, abilities, loot, notes }),
  })
  if (!res.ok) throw new Error('Update NPC stat failed')
}

export async function deleteNpcStat(id: number): Promise<void> {
  const res = await request(`/api/npc-stats/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete NPC stat failed')
}


export async function listSecrets(campaignId: number): Promise<Secret[]> {
  const res = await request(`/api/campaigns/${campaignId}/secrets`)
  if (!res.ok) throw new Error('List secrets failed')
  return res.json()
}

export async function getSecret(id: number): Promise<Secret> {
  const res = await request(`/api/secrets/${id}`)
  if (!res.ok) throw new Error('Get secret failed')
  return res.json()
}

export async function createSecret(campaignId: number, title: string, content: string, category: string): Promise<{ id: number }> {
  const res = await request(`/api/campaigns/${campaignId}/secrets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, content, category }),
  })
  if (!res.ok) throw new Error('Create secret failed')
  return res.json()
}

export async function revealSecret(id: number, sessionId: number): Promise<void> {
  const res = await request(`/api/secrets/${id}/reveal`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId }),
  })
  if (!res.ok) throw new Error('Reveal secret failed')
}

export async function updateSecret(id: number, title: string, content: string, category: string): Promise<void> {
  const res = await request(`/api/secrets/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, content, category }),
  })
  if (!res.ok) throw new Error('Update secret failed')
}

export async function deleteSecret(id: number): Promise<void> {
  const res = await request(`/api/secrets/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete secret failed')
}

// Calendar
export async function getCampaignCalendar(campaignId: number): Promise<import('../types').CampaignCalendarInfo> {
  const res = await request(`/api/campaigns/${campaignId}/calendar`)
  if (!res.ok) throw new Error('Get calendar failed')
  return res.json()
}

export async function patchCampaignCalendar(
  campaignId: number,
  updates: { in_game_year?: number; in_game_month?: number; in_game_day?: number; advance_days?: number; calendar_config?: string },
): Promise<import('../types').CampaignCalendarInfo> {
  const res = await request(`/api/campaigns/${campaignId}/calendar`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error('Patch calendar failed')
  return res.json()
}

export async function listCalendarEvents(campaignId: number): Promise<import('../types').CalendarEvent[]> {
  const res = await request(`/api/campaigns/${campaignId}/calendar-events`)
  if (!res.ok) throw new Error('List calendar events failed')
  return res.json()
}

export async function createCalendarEvent(
  campaignId: number,
  event: { in_game_year: number; in_game_month: number; in_game_day: number; title: string; description: string; event_type: string; session_id?: number | null },
): Promise<{ id: number }> {
  const res = await request(`/api/campaigns/${campaignId}/calendar-events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(event),
  })
  if (!res.ok) throw new Error('Create calendar event failed')
  return res.json()
}

export async function deleteCalendarEvent(id: number): Promise<void> {
  const res = await request(`/api/calendar-events/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Delete calendar event failed')
}

// Campaign Config (GM Screen)
