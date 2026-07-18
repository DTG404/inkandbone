package mcp

import (
	"context"
	"fmt"

	"github.com/digitalghost404/inkandbone/internal/api"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleAddMapPin(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	mapID, ok := optInt64(req, "map_id")
	if !ok {
		return mcplib.NewToolResultError("map_id is required"), nil
	}
	x, ok := optFloat64(req, "x")
	if !ok {
		return mcplib.NewToolResultError("x is required"), nil
	}
	y, ok := optFloat64(req, "y")
	if !ok {
		return mcplib.NewToolResultError("y is required"), nil
	}
	label, ok := reqStr(req, "label")
	if !ok {
		return mcplib.NewToolResultError("label is required"), nil
	}
	note := optStr(req, "note")
	color := optStr(req, "color")

	pinID, err := s.db.AddMapPin(mapID, x, y, label, note, color)
	if err != nil {
		return mcplib.NewToolResultError("add pin: " + err.Error()), nil
	}

	s.bus.Publish(api.Event{Type: api.EventMapPinAdded, Payload: &api.MapPinAddedPayload{PinID: api.RealtimeInt64(pinID), MapID: api.RealtimeInt64(mapID), Label: api.RealtimePtr(label)}})
	return mcplib.NewToolResultText(fmt.Sprintf("pin %d added to map %d: %s", pinID, mapID, label)), nil
}
