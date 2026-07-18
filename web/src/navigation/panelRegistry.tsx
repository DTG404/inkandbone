import type { ReactNode } from 'react'

export type PanelID = 'handouts' | 'decks' | 'oracle' | 'compendium' | 'notes' | 'journal' | 'npcs' | 'relationships' | 'factions' | 'calendar' | 'objectives' | 'npcstats' | 'adventures' | 'secrets' | 'gmtools'
export type PanelGroup = 'Play' | 'World' | 'GM'
export type MobileDestination = 'story' | 'character' | 'world' | 'gm'

export interface PanelDefinition {
  id: PanelID
  label: string
  group: PanelGroup
}

export interface PanelRegistryEntry extends PanelDefinition {
  render: () => ReactNode
}

export const PANEL_DEFINITIONS: readonly PanelDefinition[] = [
  { id: 'handouts', label: 'Handouts', group: 'Play' },
  { id: 'decks', label: 'Decks', group: 'Play' },
  { id: 'oracle', label: 'Oracle', group: 'Play' },
  { id: 'compendium', label: 'Compendium', group: 'World' },
  { id: 'notes', label: 'Notes', group: 'World' },
  { id: 'journal', label: 'Journal', group: 'World' },
  { id: 'npcs', label: 'NPCs', group: 'World' },
  { id: 'relationships', label: 'Relationships', group: 'World' },
  { id: 'factions', label: 'Factions', group: 'World' },
  { id: 'calendar', label: 'Calendar', group: 'World' },
  { id: 'objectives', label: 'Objectives', group: 'GM' },
  { id: 'npcstats', label: 'NPC Stat Blocks', group: 'GM' },
  { id: 'adventures', label: 'Adventures', group: 'GM' },
  { id: 'secrets', label: 'Secrets', group: 'GM' },
  { id: 'gmtools', label: 'GM Tools', group: 'GM' },
]

export function createPanelRegistry(renderers: Partial<Record<PanelID, () => ReactNode>>): PanelRegistryEntry[] {
  return PANEL_DEFINITIONS.map((definition) => ({
    ...definition,
    render: renderers[definition.id] ?? (() => null),
  }))
}
