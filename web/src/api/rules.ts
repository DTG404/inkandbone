import { request } from './transport'

export interface RulebookResult {
  heading: string
  content: string
  source: string
}

export async function searchRulebook(rulesetId: number, query: string, signal?: AbortSignal): Promise<{ results: RulebookResult[]; mode: string }> {
  const res = await request(`/api/rulesets/${rulesetId}/rulebook/search`, {
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
  const res = await request('/api/oracle/roll', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ table, roll, ruleset_id: rulesetId }),
  })
  if (!res.ok) throw new Error('Oracle roll failed')
  return res.json()
}

// Tension
export async function getTension(sessionId: number): Promise<number> {
  const res = await request(`/api/sessions/${sessionId}/tension`)
  if (!res.ok) throw new Error('Get tension failed')
  const data = await res.json()
  return data.tension_level
}

export async function patchTension(sessionId: number, level: number): Promise<void> {
  const res = await request(`/api/sessions/${sessionId}/tension`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tension_level: level }),
  })
  if (!res.ok) throw new Error('Patch tension failed')
}

// Relationships
