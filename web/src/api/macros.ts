import { request } from './transport'
import type { Macro } from '../types'

export async function fetchMacros(characterId: number): Promise<Macro[]> {
  const res = await request(`/api/characters/${characterId}/macros`)
  if (!res.ok) throw new Error(`fetchMacros failed: ${res.status}`)
  return res.json()
}
export async function createMacro(
  characterId: number,
  macro: { label: string; action_text: string; color: string },
): Promise<{ id: number }> {
  const res = await request(`/api/characters/${characterId}/macros`, {
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
  const res = await request(`/api/macros/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(macro),
  })
  if (!res.ok) throw new Error(`updateMacro failed: ${res.status}`)
}

export async function deleteMacro(id: number): Promise<void> {
  const res = await request(`/api/macros/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteMacro failed: ${res.status}`)
}

export async function reorderMacros(characterId: number, ids: number[]): Promise<void> {
  const res = await request(`/api/characters/${characterId}/macros/reorder`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids }),
  })
  if (!res.ok) throw new Error(`reorderMacros failed: ${res.status}`)
}
