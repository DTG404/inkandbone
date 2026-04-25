package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultURL       = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	model            = "claude-haiku-4-5-20251001" // legacy — Anthropic fallback client
)

// Completer generates text from a prompt. Implemented by *Client; nil means AI is disabled.
// maxTokens caps the output length; pass a tight value to avoid wasted spend.
type Completer interface {
	Generate(ctx context.Context, prompt string, maxTokens int) (string, error)
}

// ChatMessage is a single turn in a conversation.
type ChatMessage struct {
	Role    string
	Content string
}

// Responder generates a reply from a system prompt and conversation history.
type Responder interface {
	Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error)
}

// Streamer streams a reply from the Anthropic API as SSE chunks, writing each
// text delta directly to the ResponseWriter. Returns the full accumulated text.
type Streamer interface {
	StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error)
}

// Client calls the Anthropic Messages API over plain HTTP.
type Client struct {
	apiKey string
	url    string
	http   *http.Client
}

// NewClient returns a Client using the production Anthropic API URL.
func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, url: defaultURL, http: &http.Client{}}
}

// NewClientWithURL returns a Client using a custom URL (for tests).
func NewClientWithURL(apiKey, url string) *Client {
	return &Client{apiKey: apiKey, url: url, http: &http.Client{}}
}

func (c *Client) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   []map[string]any{{"role": "user", "content": prompt}},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("anthropic API returned %d", resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}
	return result.Content[0].Text, nil
}

func (c *Client) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	msgs := make([]map[string]any, len(history))
	for i, m := range history {
		msgs[i] = map[string]any{"role": m.Role, "content": m.Content}
	}
	body, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   msgs,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("anthropic API returned %d", resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}
	return stripEmDash(result.Content[0].Text), nil
}

// stripEmDash replaces em-dashes with context-appropriate punctuation so they
// never appear in GM output regardless of model behaviour.
// " — " (spaced) becomes ", "; bare "—" becomes ", ".
func stripEmDash(s string) string {
	s = strings.ReplaceAll(s, " — ", ", ")
	s = strings.ReplaceAll(s, "— ", ", ")
	s = strings.ReplaceAll(s, " —", ",")
	s = strings.ReplaceAll(s, "—", ", ")
	return s
}

// StreamRespond sends a streaming request to the Anthropic Messages API and
// writes each text delta as an SSE data line to w. It returns the full
// accumulated response text so the caller can persist it.
func (c *Client) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	msgs := make([]map[string]any, len(history))
	for i, m := range history {
		msgs[i] = map[string]any{"role": m.Role, "content": m.Content}
	}
	body, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   msgs,
		"stream":     true,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	flusher, canFlush := w.(http.Flusher)

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		data, found := strings.CutPrefix(line, "data: ")
		if !found {
			continue
		}
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" && event.Delta.Text != "" {
			text := stripEmDash(event.Delta.Text)
			fullText.WriteString(text)
			fmt.Fprintf(w, "data: %s\n\n", text) //nolint:errcheck
			if canFlush {
				flusher.Flush()
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fullText.String(), fmt.Errorf("read stream: %w", err)
	}
	return fullText.String(), nil
}
