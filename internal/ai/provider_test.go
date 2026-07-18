package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderNamesIdentifyAutomationProvider(t *testing.T) {
	tests := []struct {
		name   string
		client ProviderNamer
		want   string
	}{
		{"anthropic", NewClient("key"), "anthropic"},
		{"deepseek", NewDeepSeekClient("key"), "deepseek"},
		{"dual deepseek", NewDualDeepSeekClient("key", "auto"), "deepseek"},
		{"openrouter", NewOpenRouterClient("key"), "openrouter"},
		{"dual openrouter", NewDualOpenRouterClient("key", "auto"), "openrouter"},
		{"ollama", NewOllamaClient("model"), "ollama"},
		{"dual ollama", NewDualOllamaClient("gm", "auto"), "ollama"},
		{"hybrid", NewHybridClient("gm", "key"), "anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.client.ProviderName())
		})
	}
}
