package llm

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient wraps the Google Gemini SDK
type GeminiClient struct {
	client *genai.Client
	model  string
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(cfg Config) (*GeminiClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}
	
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	
	model := cfg.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}
	
	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

// Complete implements the Client interface
func (c *GeminiClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	model := c.client.GenerativeModel(c.model)
	
	// Set temperature if specified
	if req.Temperature > 0 {
		model.Temperature = genai.Ptr(float32(req.Temperature))
	}
	
	// Set max tokens
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}
	model.MaxOutputTokens = genai.Ptr(int32(maxTokens))
	
	// Convert tools to Gemini format
	if len(req.Tools) > 0 {
		geminiTools := []*genai.Tool{}
		functionDecls := []*genai.FunctionDeclaration{}
		
		for _, tool := range req.Tools {
			params := &genai.Schema{
				Type:       genai.TypeObject,
				Properties: make(map[string]*genai.Schema),
				Required:   tool.Parameters.Required,
			}
			
			for key, prop := range tool.Parameters.Properties {
				propSchema := &genai.Schema{
					Description: prop.Description,
				}
				
				// Map type
				switch prop.Type {
				case "string":
					propSchema.Type = genai.TypeString
				case "number":
					propSchema.Type = genai.TypeNumber
				case "integer":
					propSchema.Type = genai.TypeInteger
				case "boolean":
					propSchema.Type = genai.TypeBoolean
				case "array":
					propSchema.Type = genai.TypeArray
				case "object":
					propSchema.Type = genai.TypeObject
				}
				
				if len(prop.Enum) > 0 {
					propSchema.Enum = prop.Enum
				}
				
				params.Properties[key] = propSchema
			}
			
			functionDecls = append(functionDecls, &genai.FunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			})
		}
		
		geminiTools = append(geminiTools, &genai.Tool{
			FunctionDeclarations: functionDecls,
		})
		
		model.Tools = geminiTools
	}
	
	// Build conversation history
	var chat *genai.ChatSession
	
	// Add system prompt if provided
	if req.SystemPrompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(req.SystemPrompt)},
		}
	}
	
	// Start chat session
	chat = model.StartChat()
	
	// Add message history
	for i, msg := range req.Messages {
		parts := []genai.Part{}
		
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				parts = append(parts, genai.Text(block.Text))
			case "tool_result":
				// Convert tool result to function response
				parts = append(parts, genai.FunctionResponse{
					Name: block.ToolUseID,
					Response: map[string]interface{}{
						"result": block.Content,
					},
				})
			}
		}
		
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		
		// For all messages except the last one, add to history
		if i < len(req.Messages)-1 {
			chat.History = append(chat.History, &genai.Content{
				Parts: parts,
				Role:  role,
			})
		} else {
			// Last message will be sent via SendMessage
			// Build the last message
			resp, err := chat.SendMessage(ctx, parts...)
			if err != nil {
				return nil, fmt.Errorf("gemini API call: %w", err)
			}
			
			return c.convertResponse(resp)
		}
	}
	
	// If we only have system prompt, send an empty message
	if len(req.Messages) == 0 {
		resp, err := chat.SendMessage(ctx, genai.Text(""))
		if err != nil {
			return nil, fmt.Errorf("gemini API call: %w", err)
		}
		return c.convertResponse(resp)
	}
	
	return nil, fmt.Errorf("no messages to send")
}

func (c *GeminiClient) convertResponse(resp *genai.GenerateContentResponse) (*CompletionResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}
	
	candidate := resp.Candidates[0]
	
	// Convert response to our format
	contentBlocks := make([]ContentBlock, 0)
	
	for _, part := range candidate.Content.Parts {
		switch p := part.(type) {
		case genai.Text:
			contentBlocks = append(contentBlocks, ContentBlock{
				Type: "text",
				Text: string(p),
			})
		case genai.FunctionCall:
			// Convert function call to tool use
			contentBlocks = append(contentBlocks, ContentBlock{
				Type:  "tool_use",
				ID:    p.Name, // Gemini doesn't have separate IDs, use name
				Name:  p.Name,
				Input: p.Args,
			})
		}
	}
	
	// Determine stop reason
	stopReason := "end_turn"
	if candidate.FinishReason == genai.FinishReasonStop {
		stopReason = "end_turn"
	} else if len(candidate.Content.Parts) > 0 {
		// Check if last part is a function call
		for _, part := range candidate.Content.Parts {
			if _, ok := part.(genai.FunctionCall); ok {
				stopReason = "tool_use"
				break
			}
		}
	}
	
	usage := Usage{}
	if resp.UsageMetadata != nil {
		usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}
	
	return &CompletionResponse{
		Content:    contentBlocks,
		StopReason: stopReason,
		Model:      c.model,
		Usage:      usage,
	}, nil
}

// GetProviderName returns the provider type
func (c *GeminiClient) GetProviderName() Provider {
	return ProviderGemini
}

// GetDefaultModel returns the default model
func (c *GeminiClient) GetDefaultModel() string {
	return c.model
}

// Close closes the Gemini client
func (c *GeminiClient) Close() error {
	return c.client.Close()
}
