package ai

import (
	"encoding/json"
	"io"
	"net/http"
)

// SSEEvent is the only server-to-browser GM streaming frame shape.
type SSEEvent struct {
	Type      string `json:"type"`
	Delta     string `json:"delta,omitempty"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteSSE writes one JSON event in one data line followed by a blank LF line.
func WriteSSE(w io.Writer, event SSEEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	frame := make([]byte, 0, len(payload)+8)
	frame = append(frame, "data: "...)
	frame = append(frame, payload...)
	frame = append(frame, '\n', '\n')
	if _, err := w.Write(frame); err != nil {
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}
