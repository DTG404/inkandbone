package api

import (
	"context"
	"sync"
)

// EventType identifies what changed in the DB so the frontend knows what to refresh.
type EventType string

const (
	EventCharacterUpdated      EventType = "character_updated"
	EventMessageCreated        EventType = "message_created"
	EventCombatStarted         EventType = "combat_started"
	EventCombatantUpdated      EventType = "combatant_updated"
	EventCombatEnded           EventType = "combat_ended"
	EventWorldNoteCreated      EventType = "world_note_created"
	EventWorldNoteUpdated      EventType = "world_note_updated"
	EventMapPinAdded           EventType = "map_pin_added"
	EventMapCreated            EventType = "map_created"
	EventDiceRolled            EventType = "dice_rolled"
	EventSessionStarted        EventType = "session_started"
	EventSessionEnded          EventType = "session_ended"
	EventCampaignCreated       EventType = "campaign_created"
	EventCampaignClosed        EventType = "campaign_closed"
	EventCampaignDeleted       EventType = "campaign_deleted"
	EventCampaignReopened      EventType = "campaign_reopened"
	EventCharacterCreated      EventType = "character_created"
	EventSessionUpdated        EventType = "session_updated"
	EventNPCUpdated            EventType = "npc_updated"
	EventObjectiveUpdated      EventType = "objective_updated"
	EventItemUpdated           EventType = "item_updated"
	EventTurnAdvanced          EventType = "turn_advanced"
	EventXPAdded               EventType = "xp_added"
	EventContextUpdated        EventType = "context_updated"
	EventOracleRolled          EventType = "oracle_rolled"
	EventTensionUpdated        EventType = "tension_updated"
	EventRelationshipUpdated   EventType = "relationship_updated"
	EventFactionUpdated        EventType = "faction_updated"
	EventXPSpendSuggestions    EventType = "xp_spend_suggestions"
	EventSessionDeleted        EventType = "session_deleted"
	EventAdventureUpdated      EventType = "adventure_updated"
	EventNpcStatUpdated        EventType = "npc_stat_updated"
	EventSecretsUpdated        EventType = "secrets_updated"
	EventCalendarUpdated       EventType = "calendar_updated"
	EventCampaignConfigUpdated EventType = "campaign_config_updated"
	EventExpectedAction        EventType = "expected_action"
	EventTyping                EventType = "typing"
	EventWorldNoteRevealed     EventType = "world_note_revealed"
	EventCardDrawn             EventType = "card_drawn"
	EventTokenPlaced           EventType = "token_placed"
	EventTokenMoved            EventType = "token_moved"
	EventTokenRemoved          EventType = "token_removed"
	EventZoneRevealed          EventType = "zone_revealed"
	EventSecretRevealed        EventType = "secret_revealed"
	EventMapFX                 EventType = "map_fx"
	EventResyncRequired        EventType = "resync_required"
)

// Event is published by MCP tool handlers and broadcast to WebSocket clients.
type Event struct {
	Type     EventType `json:"type"`
	Sequence uint64    `json:"sequence"`
	Payload  any       `json:"payload"`
}

type ResyncRequiredPayload struct {
	FromSequence uint64 `json:"from_sequence"`
	ToSequence   uint64 `json:"to_sequence"`
}

type busSubscriber struct {
	ch       chan Event
	wake     chan struct{}
	lostFrom uint64
	lostTo   uint64
}

// Bus is a fan-out pub/sub for Events. Publishers call Publish; the WebSocket
// hub calls Subscribe to receive a channel of all events.
type Bus struct {
	mu          sync.Mutex
	subscribers []*busSubscriber
	sequence    uint64
}

func NewBus() *Bus { return &Bus{} }

// SubscribeContext returns a buffered channel that receives all future events.
func (b *Bus) SubscribeContext(ctx context.Context) chan Event {
	subscriber := &busSubscriber{ch: make(chan Event, 64), wake: make(chan struct{}, 1)}
	b.mu.Lock()
	b.subscribers = append(b.subscribers, subscriber)
	b.mu.Unlock()
	go b.runSubscriber(ctx, subscriber)
	return subscriber.ch
}

// Publish sends an event to all subscribers. Non-blocking: full channels are skipped.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sequence++
	e.Sequence = b.sequence
	for _, subscriber := range b.subscribers {
		if subscriber.lostFrom != 0 {
			subscriber.lostTo = e.Sequence
			select {
			case subscriber.wake <- struct{}{}:
			default:
			}
			continue
		}
		select {
		case subscriber.ch <- e:
		default:
			if subscriber.lostFrom == 0 {
				subscriber.lostFrom = e.Sequence
			}
			subscriber.lostTo = e.Sequence
			select {
			case subscriber.wake <- struct{}{}:
			default:
			}
		}
	}
}

func (b *Bus) runSubscriber(ctx context.Context, subscriber *busSubscriber) {
	defer func() {
		b.mu.Lock()
		for i, current := range b.subscribers {
			if current == subscriber {
				b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
		close(subscriber.ch)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-subscriber.wake:
			if !b.deliverPendingResync(ctx, subscriber) {
				return
			}
		}
	}
}

func (b *Bus) deliverPendingResync(ctx context.Context, subscriber *busSubscriber) bool {
	for {
		b.mu.Lock()
		from, to := subscriber.lostFrom, subscriber.lostTo
		b.mu.Unlock()
		if from == 0 {
			return true
		}
		event := Event{
			Type:     EventResyncRequired,
			Sequence: to,
			Payload:  ResyncRequiredPayload{FromSequence: from, ToSequence: to},
		}
		select {
		case subscriber.ch <- event:
		case <-ctx.Done():
			return false
		}

		b.mu.Lock()
		if subscriber.lostTo <= to {
			subscriber.lostFrom = 0
			subscriber.lostTo = 0
			b.mu.Unlock()
			return true
		}
		subscriber.lostFrom = to + 1
		b.mu.Unlock()
	}
}
