package mcp

import (
	"encoding/json"
	"testing"

	"github.com/digitalghost404/inkandbone/internal/api"
	"github.com/stretchr/testify/require"
)

// eventPayload decodes the generated payload through its public wire shape.
func eventPayload(t *testing.T, event api.Event) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(event.Payload)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(encoded, &payload))
	return payload
}
