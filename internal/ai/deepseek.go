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
	deepseekURL      = "https://api.deepseek.com/chat/completions"
	DeepSeekModel    = "deepseek-v4-flash"
)

// DeepSeekClient calls the DeepSeek API directly via its OpenAI-compatible endpoint.
// It implements Completer, Responder, and Streamer.
type DeepSeekClient struct {
	apiKey string
	model  string
	http   *http.Client
	think  bool // strip <think>...</think> blocks from responses
}

// NewDeepSeekClient returns a GM client for DeepSeek V4 Flash.
// DeepSeek defaults to thinking mode (reasoning tokens before output), but this
// client explicitly disables it via "thinking":{"type":"disabled"} on every request.
// Reasoning: thinking mode delays streaming start (bad for GM narration) and eats
// into maxTokens budgets for structured automation tasks. The think-stripping field
// (think=true) is kept as a safety net for any residual <think> blocks the API might
// emit despite the disable parameter.
func NewDeepSeekClient(apiKey string) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey: apiKey,
		model:  DeepSeekModel,
		http:   &http.Client{},
		think:  true,
	}
}

func newDeepSeekClientWithModel(apiKey, model string) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{},
		think:  true,
	}
}

// newDeepSeekAutoClient returns a fast automation client (no think-block stripping needed).
func newDeepSeekAutoClient(apiKey, model string) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{},
	}
}

func (c *DeepSeekClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return c.chatOnce(ctx, "", []ChatMessage{{Role: "user", Content: prompt}}, maxTokens)
}

func (c *DeepSeekClient) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	text, err := c.chatOnce(ctx, system, history, maxTokens)
	if err != nil {
		return "", err
	}
	if c.think {
		text = stripThinkBlock(text)
	}
	return stripEmDash(text), nil
}

func (c *DeepSeekClient) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	payload := map[string]any{
		"model":      c.model,
		"max_tokens": maxTokens,
		"messages":   ollamaMessages(system, history),
		"stream":     true,
		"thinking":   map[string]any{"type": "disabled"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deepseekURL, bytes.NewReader(body))
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
		return "", fmt.Errorf("deepseek returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	flusher, canFlush := w.(http.Flusher)
	var (
		fullText     strings.Builder
		thinkBuf     strings.Builder
		thinkingDone = !c.think
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
					Reasoning string `json:"reasoning"`
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
			thinkBuf.WriteString(chunk)
			buf := thinkBuf.String()

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

func (c *DeepSeekClient) chatOnce(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	payload := map[string]any{
		"model":      c.model,
		"max_tokens": maxTokens,
		"messages":   ollamaMessages(system, history),
		"stream":     false,
		"thinking":   map[string]any{"type": "disabled"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deepseekURL, bytes.NewReader(body))
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
		return "", fmt.Errorf("deepseek returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
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
		return "", fmt.Errorf("empty response from DeepSeek")
	}
	return result.Choices[0].Message.Content, nil
}

// DualDeepSeekClient routes GM narrative calls (Respond, StreamRespond) to one
// DeepSeek model and all automation calls (Generate) to a smaller/faster model.
type DualDeepSeekClient struct {
	gm   *DeepSeekClient
	auto *DeepSeekClient
}

// NewDualDeepSeekClient creates a client that sends GM calls to the default
// DeepSeek V4 Flash model and automation calls to a faster model.
func NewDualDeepSeekClient(deepseekKey, autoModel string) *DualDeepSeekClient {
	return &DualDeepSeekClient{
		gm:   newDeepSeekClientWithModel(deepseekKey, DeepSeekModel),
		auto: newDeepSeekAutoClient(deepseekKey, autoModel),
	}
}

func (d *DualDeepSeekClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return d.auto.Generate(ctx, prompt, maxTokens)
}

func (d *DualDeepSeekClient) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	return d.gm.Respond(ctx, system, history, maxTokens)
}

func (d *DualDeepSeekClient) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	return d.gm.StreamRespond(ctx, system, history, maxTokens, w)
}
