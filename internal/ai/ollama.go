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

const defaultOllamaURL = "http://localhost:11434"

// OllamaClient calls a local Ollama instance via its OpenAI-compatible API.
// It implements Completer, Responder, and Streamer.
type OllamaClient struct {
	model   string
	baseURL string
	http    *http.Client
	options map[string]any // extra Ollama model options (num_ctx, temperature, etc.)
	think   bool           // prepend /think to system prompt for Qwen3 reasoning mode
}

func (*OllamaClient) ProviderName() string { return "ollama" }

// NewOllamaClient creates an OllamaClient for the given model using the default
// localhost:11434 base URL. Override with OLLAMA_HOST env var via NewOllamaClientWithURL.
func NewOllamaClient(model string) *OllamaClient {
	return &OllamaClient{model: model, baseURL: defaultOllamaURL, http: NewHTTPClient()}
}

// NewOllamaClientWithURL is like NewOllamaClient but uses the given base URL (for tests).
func NewOllamaClientWithURL(model, baseURL string) *OllamaClient {
	return &OllamaClient{model: model, baseURL: baseURL, http: NewHTTPClient()}
}

// NewOllamaGMClient creates an OllamaClient tuned for GM roleplay:
//   - num_ctx 8192: enough for full session history
//   - temperature 0.72: focused but not mechanical
//   - repeat_penalty 1.05: prevents looping prose
//   - think: prepends /think to system prompt so Qwen3 reasons before responding
func NewOllamaGMClient(model string) *OllamaClient {
	return &OllamaClient{
		model:   model,
		baseURL: defaultOllamaURL,
		http:    NewHTTPClient(),
		options: map[string]any{
			"num_ctx":        16384,
			"temperature":    0.85,
			"repeat_penalty": 1.15,
			"repeat_last_n":  128,
			"top_p":          0.92,
			"top_k":          60,
		},
		think: false,
	}
}

// Generate implements Completer. Sends a single-turn prompt and returns the response.
func (c *OllamaClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	ctx, cancel := withAutomationDeadline(ctx)
	defer cancel()
	return c.chatOnce(ctx, "", []ChatMessage{{Role: "user", Content: prompt}}, maxTokens)
}

// Respond implements Responder. Sends a multi-turn conversation with a system prompt.
func (c *OllamaClient) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	ctx, cancel := withGMDeadline(ctx)
	defer cancel()
	text, err := c.chatOnce(ctx, system, history, maxTokens)
	if err != nil {
		return "", err
	}
	if c.think {
		text = stripThinkBlock(text)
	}
	return stripEmDash(text), nil
}

// StreamRespond implements Streamer. Streams the response as SSE data lines to w.
func (c *OllamaClient) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	ctx, cancel := withGMDeadline(ctx)
	defer cancel()

	msgs := ollamaMessages(c.applyThink(system), history)
	payload := map[string]any{
		"model":      c.model,
		"messages":   msgs,
		"max_tokens": maxTokens,
		"stream":     true,
	}
	if len(c.options) > 0 {
		payload["options"] = c.options
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	flusher, canFlush := w.(http.Flusher)
	var (
		fullText     strings.Builder
		thinkBuf     strings.Builder
		thinkingDone = !c.think // if think mode is off, stream immediately
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
					Content string `json:"content"`
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
			// Buffer until we see </think>. Everything inside is reasoning — never stream it.
			thinkBuf.WriteString(chunk)
			buf := thinkBuf.String()

			// Early exit from think-buffering: if we've accumulated enough to know the
			// model didn't open a <think> block (e.g. abliterated models that ignore /think),
			// switch to streaming immediately rather than swallowing the entire response.
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

			if idx := strings.Index(buf, "</think>"); idx != -1 {
				thinkingDone = true
				after := strings.TrimLeft(buf[idx+len("</think>"):], "\n")
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
	// End-of-stream safety: if think-buffering never resolved (model stopped mid-think-block
	// or never emitted </think>), flush whatever is buffered as the response — but only if
	// the buffer doesn't start with <think> (which would mean it's reasoning, not content).
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

func (c *OllamaClient) chatOnce(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	msgs := ollamaMessages(c.applyThink(system), history)
	payload := map[string]any{
		"model":      c.model,
		"messages":   msgs,
		"max_tokens": maxTokens,
		"stream":     false,
	}
	if len(c.options) > 0 {
		payload["options"] = c.options
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
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
		return "", fmt.Errorf("empty response from Ollama")
	}
	return result.Choices[0].Message.Content, nil
}

// stripThinkBlock removes all <think>...</think> blocks from s.
// Qwen3 thinking models emit chain-of-thought inside these tags; occasionally
// multiple blocks appear (e.g. mid-response reasoning). This is called on the
// non-streaming path; the streaming path buffers tokens until it sees </think>
// and discards them inline.
func stripThinkBlock(s string) string {
	for {
		end := strings.Index(s, "</think>")
		if end == -1 {
			break
		}
		after := strings.TrimLeft(s[end+len("</think>"):], "\n")
		start := strings.Index(s, "<think>")
		if start == -1 || start > end {
			// No opening tag before the closing tag — strip everything up to </think>.
			s = after
		} else {
			s = s[:start] + after
		}
	}
	return strings.TrimLeft(s, "\n")
}

// applyThink prepends /think to the system prompt when thinking mode is enabled.
// Qwen3 models recognise this token and reason silently before generating output,
// which significantly improves instruction following without changing visible output.
func (c *OllamaClient) applyThink(system string) string {
	if c.think && system != "" {
		return "/think\n\n" + system
	}
	return system
}

// ollamaMessages converts a system prompt and history into OpenAI-format messages.
func ollamaMessages(system string, history []ChatMessage) []map[string]string {
	var msgs []map[string]string
	if system != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": system})
	}
	for _, m := range history {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	return msgs
}

// DualOllamaClient routes GM streaming/response calls to one model and all
// automation Generate calls to another. This lets you use a RP-tuned model
// (e.g. hermes3:8b) for narrative prose and a stronger instruction-following
// model (e.g. phi4:14b) for structured JSON tasks.
type DualOllamaClient struct {
	gm   *OllamaClient // handles Respond + StreamRespond
	auto *OllamaClient // handles Generate
}

func (*DualOllamaClient) ProviderName() string { return "ollama" }

// NewDualOllamaClient creates a split-model Ollama client.
func NewDualOllamaClient(gmModel, autoModel string) *DualOllamaClient {
	return &DualOllamaClient{
		gm:   NewOllamaClient(gmModel),
		auto: NewOllamaClient(autoModel),
	}
}

func (d *DualOllamaClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return d.auto.Generate(ctx, prompt, maxTokens)
}

func (d *DualOllamaClient) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	return d.gm.Respond(ctx, system, history, maxTokens)
}

func (d *DualOllamaClient) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	return d.gm.StreamRespond(ctx, system, history, maxTokens, w)
}

// HybridClient routes GM streaming/response calls to a local Ollama model and
// all automation Generate calls to the Anthropic API (Claude Haiku). Use this
// when you want an uncensored local model for roleplay but Claude for structured
// JSON tasks (objective detection, NPC extraction, recaps, etc.).
type HybridClient struct {
	gm   *OllamaClient // handles Respond + StreamRespond
	auto *Client       // handles Generate
}

func (*HybridClient) ProviderName() string { return "anthropic" }

// NewHybridClient creates a client that sends GM calls to Ollama and automation
// calls to Anthropic. The GM client is tuned for roleplay quality (see NewOllamaGMClient).
func NewHybridClient(gmModel, anthropicKey string) *HybridClient {
	return &HybridClient{
		gm:   NewOllamaGMClient(gmModel),
		auto: NewClient(anthropicKey),
	}
}

func (h *HybridClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return h.auto.Generate(ctx, prompt, maxTokens)
}

func (h *HybridClient) Respond(ctx context.Context, system string, history []ChatMessage, maxTokens int) (string, error) {
	return h.gm.Respond(ctx, system, history, maxTokens)
}

func (h *HybridClient) StreamRespond(ctx context.Context, system string, history []ChatMessage, maxTokens int, w http.ResponseWriter) (string, error) {
	return h.gm.StreamRespond(ctx, system, history, maxTokens, w)
}
