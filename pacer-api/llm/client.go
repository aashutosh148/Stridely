package llm

import (
	"context"
)

// Provider interface that all LLM providers must implement
type Client interface {
	// Complete generates a chat completion
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	
	// GetProviderName returns the provider type
	GetProviderName() Provider
	
	// GetDefaultModel returns the default model for this provider
	GetDefaultModel() string
}

// Config holds LLM configuration
type Config struct {
	Provider Provider
	APIKey   string
	Model    string
}

// NewClient creates a new LLM client based on the provider
func NewClient(cfg Config) (Client, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		return NewAnthropicClient(cfg)
	case ProviderOpenAI:
		return NewOpenAIClient(cfg)
	case ProviderGemini:
		return NewGeminiClient(cfg)
	default:
		return NewAnthropicClient(cfg) // Default to Anthropic
	}
}
