package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient wraps the OpenAI SDK
type OpenAIClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(cfg Config) (*OpenAIClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai API key is required")
	}
	
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
	)
	
	model := cfg.Model
	if model == "" {
		model = string(openai.ChatModelGPT4oMini)
	}
	
	return &OpenAIClient{
		client: client,
		model:  model,
	}, nil
}

// Complete implements the Client interface
func (c *OpenAIClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Convert our messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0)
	
	// Add system message if provided
	if req.SystemPrompt != "" {
		openaiMessages = append(openaiMessages, openai.SystemMessage(req.SystemPrompt))
	}
	
	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			// Check if this is a tool result message
			hasToolResults := false
			for _, block := range msg.Content {
				if block.Type == "tool_result" {
					hasToolResults = true
					break
				}
			}
			
			if hasToolResults {
				// Convert tool results to tool messages
				for _, block := range msg.Content {
					if block.Type == "tool_result" {
						openaiMessages = append(openaiMessages, openai.ToolMessage(block.ToolUseID, block.Content))
					}
				}
			} else {
				// Regular user message
				text := ExtractText(msg.Content)
				if text != "" {
					openaiMessages = append(openaiMessages, openai.UserMessage(text))
				}
			}
			
		case "assistant":
			// Check if this has tool calls
			toolCalls := []openai.ChatCompletionMessageToolCallParam{}
			text := ""
			
			for _, block := range msg.Content {
				if block.Type == "text" {
					text += block.Text
				} else if block.Type == "tool_use" {
					argsBytes, _ := json.Marshal(block.Input)
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
						ID:   openai.F(block.ID),
						Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction),
						Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      openai.F(block.Name),
							Arguments: openai.F(string(argsBytes)),
						}),
					})
				}
			}
			
			if len(toolCalls) > 0 {
				msgParam := openai.AssistantMessage(text)
				msgParam.ToolCalls = openai.F(toolCalls)
				openaiMessages = append(openaiMessages, msgParam)
			} else if text != "" {
				openaiMessages = append(openaiMessages, openai.AssistantMessage(text))
			}
		}
	}
	
	// Convert tools to OpenAI format
	var openaiTools []openai.ChatCompletionToolParam
	if len(req.Tools) > 0 {
		openaiTools = make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
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
			
			params := map[string]interface{}{
				"type":       tool.Parameters.Type,
				"properties": properties,
			}
			if len(tool.Parameters.Required) > 0 {
				params["required"] = tool.Parameters.Required
			}
			
			openaiTools = append(openaiTools, openai.ChatCompletionToolParam{
				Type: openai.F(openai.ChatCompletionToolTypeFunction),
				Function: openai.F(openai.FunctionDefinitionParam{
					Name:        openai.F(tool.Name),
					Description: openai.F(tool.Description),
					Parameters:  openai.F(openai.FunctionParameters(params)),
				}),
			})
		}
	}
	
	// Build request
	model := req.Model
	if model == "" {
		model = c.model
	}
	
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}
	
	params := openai.ChatCompletionNewParams{
		Model:     openai.F(openai.ChatModel(model)),
		Messages:  openai.F(openaiMessages),
		MaxTokens: openai.Int(int64(maxTokens)),
	}
	
	if len(openaiTools) > 0 {
		params.Tools = openai.F(openaiTools)
	}
	
	if req.Temperature > 0 {
		params.Temperature = openai.F(req.Temperature)
	}
	
	// Call OpenAI API
	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai API call: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	
	choice := resp.Choices[0]
	
	// Convert response to our format
	contentBlocks := make([]ContentBlock, 0)
	
	// Add text content if present
	if choice.Message.Content != "" {
		contentBlocks = append(contentBlocks, ContentBlock{
			Type: "text",
			Text: choice.Message.Content,
		})
	}
	
	// Add tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		for _, toolCall := range choice.Message.ToolCalls {
			var input map[string]interface{}
			json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
			
			contentBlocks = append(contentBlocks, ContentBlock{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: input,
			})
		}
	}
	
	// Determine stop reason
	stopReason := "end_turn"
	switch choice.FinishReason {
	case openai.ChatCompletionChoicesFinishReasonStop:
		stopReason = "end_turn"
	case openai.ChatCompletionChoicesFinishReasonToolCalls:
		stopReason = "tool_use"
	case openai.ChatCompletionChoicesFinishReasonLength:
		stopReason = "max_tokens"
	}
	
	return &CompletionResponse{
		Content:    contentBlocks,
		StopReason: stopReason,
		Model:      string(resp.Model),
		Usage: Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
		},
	}, nil
}

// GetProviderName returns the provider type
func (c *OpenAIClient) GetProviderName() Provider {
	return ProviderOpenAI
}

// GetDefaultModel returns the default model
func (c *OpenAIClient) GetDefaultModel() string {
	return c.model
}
