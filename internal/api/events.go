package api

import "sync"

// EventType identifies what changed in the DB so the frontend knows what to refresh.
type EventType string

const (
	EventCharacterUpdated EventType = "character_updated"
	EventMessageCreated   EventType = "message_created"
	EventCombatStarted    EventType = "combat_started"
	EventCombatantUpdated EventType = "combatant_updated"
	EventCombatEnded      EventType = "combat_ended"
	EventWorldNoteCreated EventType = "world_note_created"
	EventWorldNoteUpdated EventType = "world_note_updated"
	EventMapPinAdded      EventType = "map_pin_added"
	EventMapCreated       EventType = "map_created"
	EventDiceRolled       EventType = "dice_rolled"
	EventSessionStarted   EventType = "session_started"
	EventSessionEnded     EventType = "session_ended"
	EventCampaignCreated  EventType = "campaign_created"
	EventCampaignClosed   EventType = "campaign_closed"
	EventCampaignDeleted  EventType = "campaign_deleted"
	EventCampaignReopened EventType = "campaign_reopened"
	EventCharacterCreated EventType = "character_created"
	EventSessionUpdated   EventType = "session_updated"
	EventNPCUpdated       EventType = "npc_updated"
	EventObjectiveUpdated EventType = "objective_updated"
	EventItemUpdated      EventType = "item_updated"
	EventTurnAdvanced     EventType = "turn_advanced"
	EventXPAdded          EventType = "xp_added"
	EventContextUpdated     EventType = "context_updated"
	EventOracleRolled       EventType = "oracle_rolled"
	EventTensionUpdated     EventType = "tension_updated"
	EventRelationshipUpdated  EventType = "relationship_updated"
	EventFactionUpdated        EventType = "faction_updated"
	EventXPSpendSuggestions   EventType = "xp_spend_suggestions"
	EventSessionDeleted       EventType = "session_deleted"
	EventAdventureUpdated     EventType = "adventure_updated"
	EventNpcStatUpdated       EventType = "npc_stat_updated"
	EventSecretsUpdated       EventType = "secrets_updated"
	EventCalendarUpdated         EventType = "calendar_updated"
	EventCampaignConfigUpdated   EventType = "campaign_config_updated"
	EventExpectedAction         EventType = "expected_action"
	EventTyping                 EventType = "typing"
	EventWorldNoteRevealed      EventType = "world_note_revealed"
	EventCardDrawn              EventType = "card_drawn"
)

// Event is published by MCP tool handlers and broadcast to WebSocket clients.
type Event struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
}

// Bus is a fan-out pub/sub for Events. Publishers call Publish; the WebSocket
// hub calls Subscribe to receive a channel of all events.
type Bus struct {
	mu          sync.Mutex
	subscribers []chan Event
}

func NewBus() *Bus { return &Bus{} }

// Subscribe returns a buffered channel that receives all future events.
// The caller must drain or close the channel; a full channel drops events.
func (b *Bus) Subscribe() chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

// Publish sends an event to all subscribers. Non-blocking: full channels are skipped.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- e:
		default:
		}
	}
}
