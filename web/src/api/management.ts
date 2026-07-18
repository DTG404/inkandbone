import { request } from './transport'
import type { Ruleset } from './characters'

export interface RulebookSource {
  source: string;
  chunks: number;
}
export async function fetchRulesets(): Promise<Ruleset[]> {
  const res = await request('/api/rulesets')
  if (!res.ok) throw new Error(`fetchRulesets failed: ${res.status}`)
  return res.json()
}

export async function fetchCampaigns(): Promise<import('../types').Campaign[]> {
  const res = await request('/api/campaigns')
  if (!res.ok) throw new Error(`fetchCampaigns failed: ${res.status}`)
  return res.json()
}

export async function fetchCharacters(campaignId: number): Promise<import('../types').Character[]> {
  const res = await request(`/api/campaigns/${campaignId}/characters`)
  if (!res.ok) throw new Error(`fetchCharacters failed: ${res.status}`)
  return res.json()
}


export async function createCampaign(name: string, description: string, rulesetId: number): Promise<{ id: number }> {
  const res = await request('/api/campaigns', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, ruleset_id: rulesetId }),
  })
  if (!res.ok) throw new Error(`createCampaign failed: ${res.status}`)
  return res.json()
}

export async function deleteCampaign(id: number): Promise<void> {
  const res = await request(`/api/campaigns/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteCampaign failed: ${res.status}`)
}

export async function patchCampaign(id: number, updates: { chronicle_night?: number; active?: boolean }): Promise<void> {
  const res = await request(`/api/campaigns/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCampaign failed: ${res.status}`)
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
  const res = await request('/api/settings', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`patchSettings failed: ${res.status}`)
}

export async function fetchRulebookSources(rulesetId: number): Promise<RulebookSource[]> {
  const res = await request(`/api/rulesets/${rulesetId}/rulebook`)
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
  const res = await request(`/api/rulesets/${rulesetId}/rulebook`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) throw new Error(`uploadRulebook failed: ${res.status}`)
  return res.json()
}


export interface CampaignConfig {
  description: string
  gm_notes: string
  system_prompt_override: string
  content_boundaries: string
  narrative_locale: string
  character_count: number
  session_count: number
  ruleset_name: string
}

export async function fetchCampaignConfig(campaignId: number): Promise<CampaignConfig> {
  const res = await request(`/api/campaigns/${campaignId}/config`)
  if (!res.ok) throw new Error(`fetchCampaignConfig failed: ${res.status}`)
  return res.json()
}

export async function patchCampaignConfig(
  campaignId: number,
  updates: { description?: string; gm_notes?: string; system_prompt_override?: string; content_boundaries?: string; narrative_locale?: string },
): Promise<void> {
  const res = await request(`/api/campaigns/${campaignId}/config`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCampaignConfig failed: ${res.status}`)
}
