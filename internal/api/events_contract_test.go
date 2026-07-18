package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBusPublishesGeneratedPayloadStruct(t *testing.T) {
	bus := NewBus()
	events := bus.SubscribeContext(context.Background())
	bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{
		"session_id": int64(7),
		"result":     18,
	}})
	event := <-events
	payload, ok := event.Payload.(*DiceRolledPayload)
	require.True(t, ok)
	require.Equal(t, int64(7), *payload.SessionID)
	require.Equal(t, int64(18), *payload.Result)
}

func TestBusPreservesExplicitOptionalZeroAndFalseValues(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		payload   map[string]any
		keys      []string
	}{
		{
			name:      "token coordinates at origin",
			eventType: EventTokenMoved,
			payload:   map[string]any{"map_id": int64(1), "token_id": int64(2), "x": 0.0, "y": 0.0},
			keys:      []string{"x", "y"},
		},
		{
			name:      "zero dice result and false flags",
			eventType: EventDiceRolled,
			payload: map[string]any{
				"session_id": int64(3), "result": 0, "hidden": false, "bestial_fail": false,
			},
			keys: []string{"result", "hidden", "bestial_fail"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bus := NewBus()
			events := bus.SubscribeContext(t.Context())
			bus.Publish(Event{Type: test.eventType, Payload: test.payload})
			event := <-events
			encoded, err := json.Marshal(event.Payload)
			require.NoError(t, err)
			var wire map[string]any
			require.NoError(t, json.Unmarshal(encoded, &wire))
			for _, key := range test.keys {
				require.Contains(t, wire, key)
				require.EqualValues(t, test.payload[key], wire[key])
			}
		})
	}
}

func TestDecodeRealtimePayloadRejectsUnknownFields(t *testing.T) {
	_, err := DecodeRealtimePayload(EventDiceRolled, map[string]any{"result": 1, "drifted": true})
	require.ErrorContains(t, err, "unknown field")
}
