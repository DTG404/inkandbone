package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const ollamaEmbedURL = "http://localhost:11434/api/embeddings"

// EmbedText calls Ollama's native embeddings endpoint with nomic-embed-text.
// Returns error if Ollama is unreachable.
func EmbedText(ctx context.Context, text string) ([]float32, error) {
	payload := map[string]string{"model": "nomic-embed-text", "prompt": text}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaEmbedURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}
	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	out := make([]float32, len(result.Embedding))
	for i, v := range result.Embedding {
		out[i] = float32(v)
	}
	return out, nil
}
