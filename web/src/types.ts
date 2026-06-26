export interface Campaign {
  id: number
  ruleset_id: number
  name: string
  description: string
  active: boolean
  chronicle_night: number
  chronicle_night_start_dow: number  // 0=Sun…6=Sat, -1=undetected
  created_at: string
}

export interface Character {
  id: number
  campaign_id: number
  name: string
  data_json: string
  portrait_path: string
  currency_balance: number
  currency_label: string
  created_at: string
}

export interface Session {
  id: number
  campaign_id: number
  title: string
  date: string
  summary: string
  notes: string
  scene_tags: string
  adventure_id: number | null
  created_at: string
}

export interface Message {
  id: number
  session_id: number
  role: string
  content: string
  created_at: string
  whisper?: boolean
  character_id?: number | null
}

export interface SessionNPC {
  id: number
  session_id: number
  name: string
  note: string
  created_at: string
}

export interface CombatEncounter {
  id: number
  session_id: number
  name: string
  active: boolean
  active_turn_index: number
  created_at: string
}

export interface Combatant {
  id: number
  encounter_id: number
  character_id: number | null
  name: string
  initiative: number
  hp_current: number
  hp_max: number
  conditions_json: string
  is_player: boolean
  sort_order: number
}

export interface CombatSnapshot {
  encounter: CombatEncounter
  combatants: Combatant[]
}

export interface GameContext {
  campaign: Campaign | null
  character: Character | null
  session: Session | null
  recent_messages: Message[]
  active_combat: CombatSnapshot | null
}

export interface WorldNote {
  id: number
  campaign_id: number
  title: string
  content: string
  category: string
  tags_json: string
  personality_json: string
  is_revealed: boolean
  created_at: string
}

export interface DiceRoll {
  id: number
  session_id: number
  expression: string
  result: number
  breakdown_json: string
  created_at: string
}

export interface TimelineEntry {
  type: 'message' | 'dice_roll' | 'world_note_event' | 'combat_event'
  timestamp: string
  data: Record<string, unknown>
}

export interface Objective {
  id: number
  campaign_id: number
  title: string
  description: string
  status: 'active' | 'completed' | 'failed'
  parent_id: number | null
  created_at: string
}

export interface Item {
  id: number
  character_id: number
  name: string
  description: string
  quantity: number
  equipped: boolean
  created_at: string
}

export interface XPEntry {
  id: number
  session_id: number
  note: string
  amount: number | null
  created_at: string
}

export interface Adventure {
  id: number
  campaign_id: number
  title: string
  description: string
  status: 'upcoming' | 'active' | 'completed' | 'abandoned'
  sort_order: number
  created_at: string
}

export interface NpcStat {
  id: number
  campaign_id: number
  name: string
  role: string
  data_json: string
  hp_max: number
  armor_class: number | null
  initiative_mod: number
  skills: string
  abilities: string
  loot: string
  notes: string
  created_at: string
}

export interface Faction {
  id: number
  campaign_id: number
  name: string
  description: string
  faction_type: string
  influence: number
  resources_json: string
  color: string
  created_at: string
}

export interface Relationship {
  id: number
  campaign_id: number
  from_name: string
  to_name: string
  relationship_type: string
  description: string
  created_at: string
}

export interface XPSuggestion {
  field: string
  display_name: string
  current_value: number
  new_value: number
  xp_cost: number
  reasoning: string
}

export interface Secret {
  id: number
  campaign_id: number
  title: string
  content: string
  category: string
  revealed: boolean
  revealed_at_session_id: number | null
  created_at: string
}

export interface CalendarEvent {
  id: number
  campaign_id: number
  in_game_year: number
  in_game_month: number
  in_game_day: number
  title: string
  description: string
  event_type: string
  session_id: number | null
  created_at: string
}

export interface CampaignCalendarInfo {
  in_game_year: number
  in_game_month: number
  in_game_day: number
  calendar_config: string
}

export interface XPSpendSuggestionsEvent {
  character_id: number
  character_name: string
  current_xp: number
  xp_label: string
  suggestions: XPSuggestion[]
}

export interface Macro {
  id: number
  character_id: number
  label: string
  action_text: string
  color: string
  sort_order: number
  created_at: string
}

export interface DeckCard { front: string; back?: string }
export interface Deck {
  id: number
  campaign_id: number
  name: string
  cards_json: string
  shuffled_order_json: string
  draw_index: number
  created_at: string
}
export interface DeckDraw {
  id: number
  session_id: number
  deck_id: number
  card_json: string
  drawn_at: string
}
