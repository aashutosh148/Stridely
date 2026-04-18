package llm

import (
	"encoding/json"
)

// Provider represents the LLM provider type
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderGemini    Provider = "gemini"
)

// ToolDefinition represents a provider-agnostic tool definition
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ToolParameters         `json:"parameters"`
}

// ToolParameters defines the input schema for a tool
type ToolParameters struct {
	Type       string                            `json:"type"`
	Properties map[string]PropertyDefinition     `json:"properties"`
	Required   []string                          `json:"required"`
}

// PropertyDefinition defines a single parameter property
type PropertyDefinition struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string        `json:"role"` // "user", "assistant", "system"
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a piece of content in a message
type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result"
	
	// For text blocks
	Text string `json:"text,omitempty"`
	
	// For tool_use blocks
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	
	// For tool_result blocks
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// NewTextMessage creates a user message with text content
func NewTextMessage(role, text string) Message {
	return Message{
		Role: role,
		Content: []ContentBlock{
			{Type: "text", Text: text},
		},
	}
}

// NewToolUseBlock creates a tool use content block
func NewToolUseBlock(id, name string, input map[string]interface{}) ContentBlock {
	return ContentBlock{
		Type:  "tool_use",
		ID:    id,
		Name:  name,
		Input: input,
	}
}

// NewToolResultBlock creates a tool result content block
func NewToolResultBlock(toolUseID, content string, isError bool) ContentBlock {
	return ContentBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}
}

// CompletionRequest represents a chat completion request
type CompletionRequest struct {
	Model         string           `json:"model"`
	Messages      []Message        `json:"messages"`
	SystemPrompt  string           `json:"system_prompt,omitempty"`
	Tools         []ToolDefinition `json:"tools,omitempty"`
	MaxTokens     int              `json:"max_tokens"`
	Temperature   float64          `json:"temperature"`
	StopSequences []string         `json:"stop_sequences,omitempty"`
}

// CompletionResponse represents a chat completion response
type CompletionResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"` // "end_turn", "tool_use", "max_tokens"
	Model      string         `json:"model"`
	Usage      Usage          `json:"usage"`
}

// Usage tracks token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ExtractText extracts all text content from content blocks
func ExtractText(content []ContentBlock) string {
	var result string
	for _, block := range content {
		if block.Type == "text" {
			result += block.Text
		}
	}
	return result
}

// ExtractToolUses extracts all tool use blocks
func ExtractToolUses(content []ContentBlock) []ContentBlock {
	var toolUses []ContentBlock
	for _, block := range content {
		if block.Type == "tool_use" {
			toolUses = append(toolUses, block)
		}
	}
	return toolUses
}

// MarshalJSON is a helper to convert results to JSON strings
func MarshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
