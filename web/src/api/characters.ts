import { request } from './transport'

export interface Ruleset {
  id: number;
  name: string;
  schema_json: string;
  version: string;
}
export async function fetchRuleset(rulesetId: number): Promise<Ruleset> {
  const res = await request(`/api/rulesets/${rulesetId}`)
  if (!res.ok) throw new Error(`fetchRuleset failed: ${res.status}`)
  return res.json()
}

export interface AdvancementConfig {
  minimum_xp: number
  supported: boolean
}

export async function fetchAdvancementConfig(rulesetId: number): Promise<AdvancementConfig> {
  const res = await request(`/api/rulesets/${rulesetId}/advancement-config`)
  if (!res.ok) throw new Error(`fetchAdvancementConfig failed: ${res.status}`)
  return res.json()
}

export async function patchCharacter(characterId: number, updates: Record<string, unknown>): Promise<void> {
  const res = await request(`/api/characters/${characterId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ data_json: JSON.stringify(updates) }),
  })
  if (!res.ok) throw new Error(`patchCharacter failed: ${res.status}`)
}

export async function uploadPortrait(characterId: number, file: File): Promise<{ portrait_path: string }> {
  const form = new FormData()
  form.append('portrait', file)
  const res = await request(`/api/characters/${characterId}/portrait`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) throw new Error(`uploadPortrait failed: ${res.status}`)
  return res.json()
}


export async function suggestAdvances(characterId: number, hintXP?: number): Promise<void> {
  const res = await request(`/api/characters/${characterId}/suggest-advances`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ hint_xp: hintXP ?? 0 }),
  })
  if (!res.ok) throw new Error(`suggestAdvances failed: ${res.status}`)
}

export async function fetchCharacterOptions(rulesetId: number): Promise<Record<string, string[]>> {
  const res = await request(`/api/rulesets/${rulesetId}/character-options`)
  if (!res.ok) throw new Error(`fetchCharacterOptions failed: ${res.status}`)
  return res.json()
}

export async function createCharacter(
  campaignId: number,
  name: string,
  overrides?: Record<string, string>,
): Promise<import('../types').Character> {
  const res = await request(`/api/campaigns/${campaignId}/characters`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, overrides }),
  })
  if (!res.ok) throw new Error(`createCharacter failed: ${res.status}`)
  return res.json()
}

export async function deleteCharacter(id: number): Promise<void> {
  const res = await request(`/api/characters/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteCharacter failed: ${res.status}`)
}


export async function patchCurrency(
  characterId: number,
  updates: { currency_balance?: number; currency_label?: string },
): Promise<void> {
  const res = await request(`/api/characters/${characterId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCurrency failed: ${res.status}`)
}

// Secrets

export async function fetchTalentDescription(name: string, system = 'wrath_glory'): Promise<string> {
  const res = await request(`/api/talent-description?name=${encodeURIComponent(name)}&system=${encodeURIComponent(system)}`)
  if (!res.ok) return ''
  const data = await res.json() as { description: string }
  return data.description ?? ''
}
