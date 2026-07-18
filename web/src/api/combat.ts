import { request } from './transport'

export async function patchCombatant(
  combatantId: number,
  updates: { conditions_json?: string; hp_current?: number; initiative?: number },
): Promise<void> {
  const res = await request(`/api/combatants/${combatantId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchCombatant failed: ${res.status}`)
}
export async function reorderCombatants(encounterId: number, ids: number[]): Promise<void> {
  const res = await request(`/api/encounters/${encounterId}/combatants/reorder`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  })
  if (!res.ok) throw new Error(`reorderCombatants failed: ${res.status}`)
}


export async function advanceTurn(encounterId: number): Promise<void> {
  const res = await request(`/api/combat-encounters/${encounterId}/next-turn`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(`advanceTurn failed: ${res.status}`)
}
