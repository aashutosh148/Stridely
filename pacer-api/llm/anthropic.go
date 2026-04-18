package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicClient wraps the Anthropic SDK
type AnthropicClient struct {
	client anthropic.Client
	model  string
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(cfg Config) (*AnthropicClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}
	
	client := anthropic.NewClient(
		option.WithAPIKey(cfg.APIKey),
	)
	
	model := cfg.Model
	if model == "" {
		model = string(anthropic.ModelClaudeSonnet4_5)
	}
	
	return &AnthropicClient{
		client: client,
		model:  model,
	}, nil
}

// Complete implements the Client interface
func (c *AnthropicClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Convert our messages to Anthropic format
	anthropicMessages := make([]anthropic.MessageParam, 0, len(req.Messages))
	
	for _, msg := range req.Messages {
		contentBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))
		
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				contentBlocks = append(contentBlocks, anthropic.NewTextBlock(block.Text))
			case "tool_use":
				contentBlocks = append(contentBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))
			case "tool_result":
				contentBlocks = append(contentBlocks, anthropic.NewToolResultBlock(block.ToolUseID, block.Content, block.IsError))
			}
		}
		
		if msg.Role == "user" {
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(contentBlocks...))
		} else if msg.Role == "assistant" {
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(contentBlocks...))
		}
	}
	
	// Convert tools to Anthropic format
	anthropicTools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
	for _, tool := range req.Tools {
		properties := make(map[string]interface{})
		for key, prop := range tool.Parameters.Properties {
			propMap := map[string]interface{}{
				"type":        prop.Type,
				"description": prop.Description,
			}
			if len(prop.Enum) > 0 {
				propMap["enum"] = prop.Enum
			}
			properties[key] = propMap
		}
		
		toolParam := anthropic.ToolParam{
			Name:        tool.Name,
			Description: anthropic.String(tool.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: properties,
				Required:   tool.Parameters.Required,
			},
		}
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{OfTool: &toolParam})
	}
	
	// Build the request
	model := req.Model
	if model == "" {
		model = c.model
	}
	
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}
	
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: int64(maxTokens),
		Messages:  anthropicMessages,
	}
	
	// Add system prompt if provided
	if req.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{
				Text: req.SystemPrompt,
				Type: "text",
			},
		}
	}
	
	// Add tools if provided
	if len(anthropicTools) > 0 {
		params.Tools = anthropicTools
	}
	
	// Add temperature if specified
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}
	
	// Call Anthropic API
	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic API call: %w", err)
	}
	
	// Convert response to our format
	contentBlocks := make([]ContentBlock, 0, len(resp.Content))
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			contentBlocks = append(contentBlocks, ContentBlock{
				Type: "text",
				Text: block.Text,
			})
		case "tool_use":
			var input map[string]interface{}
			_ = json.Unmarshal(block.Input, &input)
			
			contentBlocks = append(contentBlocks, ContentBlock{
				Type:  "tool_use",
				ID:    block.ID,
				Name:  block.Name,
				Input: input,
			})
		}
	}
	
	stopReason := "end_turn"
	switch resp.StopReason {
	case anthropic.StopReasonEndTurn:
		stopReason = "end_turn"
	case anthropic.StopReasonToolUse:
		stopReason = "tool_use"
	case anthropic.StopReasonMaxTokens:
		stopReason = "max_tokens"
	}
	
	return &CompletionResponse{
		Content:    contentBlocks,
		StopReason: stopReason,
		Model:      string(resp.Model),
		Usage: Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
		},
	}, nil
}

// GetProviderName returns the provider type
func (c *AnthropicClient) GetProviderName() Provider {
	return ProviderAnthropic
}

// GetDefaultModel returns the default model
func (c *AnthropicClient) GetDefaultModel() string {
	return c.model
}
