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
	openRouterURL   = "https://openrouter.ai/api/v1/chat/completions"
	OpenRouterModel = "deepseek/deepseek-v4-flash"
)

// OpenRouterClient calls the OpenRouter API using the OpenAI-compatible endpoint.
// It implements Completer, Responder, and Streamer.
type OpenRouterClient struct {
	apiKey      string
	model       string
	http        *http.Client
	think       bool // strip <think>...</think> blocks from responses
	suppressReasoning bool // send "reasoning": {"exclude": true} to suppress server-side thinking tokens
	options     map[string]any
}

// NewOpenRouterClient returns a GM client for DeepSeek V4 Flash:
// reasoning suppressed server-side and tuned for prose quality.
func NewOpenRouterClient(apiKey string) *OpenRouterClient {
	return &OpenRouterClient{
		apiKey:            apiKey,
		model:             OpenRouterModel,
		http:              &http.Client{},
		think:             true,
		suppressReasoning: true,
		options: map[string]any{
			"temperature":    0.85,
			"repeat_penalty": 1.15,
			"top_p":          0.92,
			"top_k":          60,
		},
	}
}

func newOpenRouterClientWithModel(apiKey, model string) *OpenRouterClient {
	return &OpenRouterClient{
		apiKey:            apiKey,
		model:             model,
		http:              &http.Client{},
		think:             true,
		suppressReasoning: true,
	}
}

// newOpenRouterAutoClient returns a fast automation client (no thinking suppression needed
// for small models that don't emit reasoning tokens).
func newOpenRouterAutoClient(apiKey, model string) *OpenRouterClient {
	return &OpenRouterClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{},
	}
}

func (c *OpenRouterClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return c.chatOnce(ctx, "", []ChatMessage{{Role: "user", Content: prompt}}, maxTokens)
}

func (c *OpenRouterClient) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	text, err := c.chatOnce(ctx, system, history, maxTokens)
	if err != nil {
		return "", err
	}
	if c.think {
		text = stripThinkBlock(text)
	}
	return stripEmDash(text), nil
}

func (c *OpenRouterClient) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	payload := map[string]any{
		"model":      c.model,
		"max_tokens": maxTokens,
		"messages":   ollamaMessages(system, history),
		"stream":     true,
	}
	if len(c.options) > 0 {
		for k, v := range c.options {
			payload[k] = v
		}
	}
	if c.suppressReasoning {
		payload["reasoning"] = map[string]any{"exclude": true}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openrouter returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	flusher, canFlush := w.(http.Flusher)
	var (
		fullText     strings.Builder
		thinkBuf     strings.Builder
		thinkingDone = !c.think // if think-stripping is off, stream immediately
	)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		data, found := strings.CutPrefix(line, "data: ")
		if !found || data == "[DONE]" {
			continue
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					Reasoning string `json:"reasoning"` // ignored — already excluded server-side
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Choices) == 0 || event.Choices[0].Delta.Content == "" {
			continue
		}
		chunk := event.Choices[0].Delta.Content

		if !thinkingDone {
			// Buffer until </think>. Everything inside is reasoning — never stream it.
			thinkBuf.WriteString(chunk)
			buf := thinkBuf.String()

			// Early exit: if we've accumulated enough to know the model didn't open a
			// <think> block, switch to streaming immediately.
			const thinkOpen = "<think>"
			if thinkBuf.Len() >= len(thinkOpen) && !strings.HasPrefix(buf, thinkOpen) {
				thinkingDone = true
				thinkBuf.Reset()
				text := stripEmDash(buf)
				fullText.WriteString(text)
				fmt.Fprintf(w, "data: %s\n\n", text) //nolint:errcheck
				if canFlush {
					flusher.Flush()
				}
				continue
			}

			if _, rest, found := strings.Cut(buf, "</think>"); found {
				thinkingDone = true
				after := strings.TrimLeft(rest, "\n")
				thinkBuf.Reset()
				if after != "" {
					after = stripEmDash(after)
					fullText.WriteString(after)
					fmt.Fprintf(w, "data: %s\n\n", after) //nolint:errcheck
					if canFlush {
						flusher.Flush()
					}
				}
			}
			continue
		}

		text := stripEmDash(chunk)
		fullText.WriteString(text)
		fmt.Fprintf(w, "data: %s\n\n", text) //nolint:errcheck
		if canFlush {
			flusher.Flush()
		}
	}
	if err := scanner.Err(); err != nil {
		return fullText.String(), fmt.Errorf("read stream: %w", err)
	}
	// Safety: if think-buffering never resolved, flush as content only if it's not a raw <think> block.
	if !thinkingDone && thinkBuf.Len() > 0 {
		if !strings.HasPrefix(thinkBuf.String(), "<think>") {
			text := stripEmDash(thinkBuf.String())
			fullText.WriteString(text)
			fmt.Fprintf(w, "data: %s\n\n", text) //nolint:errcheck
			if canFlush {
				flusher.Flush()
			}
		}
	}
	return fullText.String(), nil
}

func (c *OpenRouterClient) chatOnce(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	payload := map[string]any{
		"model":      c.model,
		"max_tokens": maxTokens,
		"messages":   ollamaMessages(system, history),
		"stream":     false,
	}
	if len(c.options) > 0 {
		for k, v := range c.options {
			payload[k] = v
		}
	}
	if c.suppressReasoning {
		payload["reasoning"] = map[string]any{"exclude": true}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openrouter returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenRouter")
	}
	return result.Choices[0].Message.Content, nil
}

// DualOpenRouterClient routes GM narrative calls (Respond, StreamRespond) to the
// NVIDIA model and all automation calls (Generate) to a fast small model.
type DualOpenRouterClient struct {
	gm   *OpenRouterClient // NVIDIA nemotron — GM narration
	auto *OpenRouterClient // small/fast model — structured automation tasks
}

// NewDualOpenRouterClient creates a client that sends GM calls to OpenRouter (NVIDIA)
// and automation calls to a fast OpenRouter model specified by autoModel.
func NewDualOpenRouterClient(openrouterKey, autoModel string) *DualOpenRouterClient {
	return &DualOpenRouterClient{
		gm:   newOpenRouterClientWithModel(openrouterKey, OpenRouterModel),
		auto: newOpenRouterAutoClient(openrouterKey, autoModel),
	}
}

func (d *DualOpenRouterClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return d.auto.Generate(ctx, prompt, maxTokens)
}

func (d *DualOpenRouterClient) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	return d.gm.Respond(ctx, system, history, maxTokens)
}

func (d *DualOpenRouterClient) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	return d.gm.StreamRespond(ctx, system, history, maxTokens, w)
}
