package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/pacer-api/llm"
	"github.com/yourname/pacer-api/models"
)

// MemoryInterface defines the memory operations needed by tools
type MemoryInterface interface {
	StoreEpisodic(ctx context.Context, memory *models.EpisodicMemory) (uuid.UUID, error)
	SearchEpisodic(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.SearchResult, error)
	GetSemanticFacts(ctx context.Context, userID uuid.UUID, minConfidence float64) ([]models.SemanticFact, error)
}

// StoreEpisodicTool stores a new episodic memory
// LLD Section 8.2
type StoreEpisodicTool struct {
	MemorySvc MemoryInterface
}

func (t *StoreEpisodicTool) Name() string {
	return "memory.store_episodic"
}

func (t *StoreEpisodicTool) Description() string {
	return "Store a new episodic memory (activity, race, injury, milestone, note, coaching moment). Use this to remember important events and insights about the athlete."
}

func (t *StoreEpisodicTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"memory_type": {
					Type:        "string",
					Description: "Type of memory: activity, race, injury, milestone, note, coaching_moment, weekly_summary",
				},
				"event_date": {
					Type:        "string",
					Description: "Date of the event (YYYY-MM-DD format)",
				},
				"title": {
					Type:        "string",
					Description: "Short title for the memory",
				},
				"summary": {
					Type:        "string",
					Description: "Detailed summary of the event or insight",
				},
				"content": {
					Type:        "object",
					Description: "Additional structured data (JSON object with arbitrary fields)",
				},
				"importance_score": {
					Type:        "number",
					Description: "Importance score 0.0-1.0 (default 0.5)",
				},
				"tags": {
					Type:        "array",
					Description: "Tags for categorization",
				},
			},
			Required: []string{"memory_type", "event_date", "title", "summary"},
		},
	}
}

func (t *StoreEpisodicTool) Execute(ctx context.Context, userIDStr string, input map[string]any) (string, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %v", err)
	}
	memoryType, ok := input["memory_type"].(string)
	if !ok {
		return "", fmt.Errorf("memory_type is required")
	}

	eventDateStr, ok := input["event_date"].(string)
	if !ok {
		return "", fmt.Errorf("event_date is required")
	}

	eventDate, err := time.Parse("2006-01-02", eventDateStr)
	if err != nil {
		return "", fmt.Errorf("invalid event_date format (use YYYY-MM-DD): %w", err)
	}

	title, ok := input["title"].(string)
	if !ok {
		return "", fmt.Errorf("title is required")
	}

	summary, ok := input["summary"].(string)
	if !ok {
		return "", fmt.Errorf("summary is required")
	}

	content := make(map[string]interface{})
	if c, ok := input["content"].(map[string]interface{}); ok {
		content = c
	}

	importanceScore := 0.5
	if score, ok := input["importance_score"].(float64); ok {
		importanceScore = score
	}

	tags := []string{}
	if tagsRaw, ok := input["tags"].([]interface{}); ok {
		for _, tag := range tagsRaw {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	memory := &models.EpisodicMemory{
		UserID:          userID,
		MemoryType:      models.MemoryType(memoryType),
		EventDate:       eventDate,
		Title:           title,
		Summary:         summary,
		Content:         content,
		ImportanceScore: importanceScore,
		Tags:            tags,
		Compressed:      false,
	}

	memoryID, err := t.MemorySvc.StoreEpisodic(ctx, memory)
	if err != nil {
		return "", fmt.Errorf("store episodic memory: %w", err)
	}

	result := map[string]interface{}{
		"memory_id":        memoryID.String(),
		"memory_type":      memoryType,
		"title":            title,
		"importance_score": importanceScore,
		"status":           "stored",
		"embedding_status": "generating_async",
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), nil
}

// SearchEpisodicTool performs semantic search on episodic memories
// LLD Section 7.4 - Full pgvector cosine similarity search
type SearchEpisodicTool struct {
	MemorySvc MemoryInterface
}

func (t *SearchEpisodicTool) Name() string {
	return "memory.search_episodic"
}

func (t *SearchEpisodicTool) Description() string {
	return "Search episodic memories using natural language query. Returns semantically similar memories ranked by relevance. Use this to recall past events, patterns, or insights."
}

func (t *SearchEpisodicTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"query": {
					Type:        "string",
					Description: "Natural language search query (e.g., 'times I struggled with fatigue', 'successful tempo runs')",
				},
				"limit": {
					Type:        "number",
					Description: "Maximum number of results to return (default 5, max 20)",
				},
			},
			Required: []string{"query"},
		},
	}
}

func (t *SearchEpisodicTool) Execute(ctx context.Context, userIDStr string, input map[string]any) (string, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %v", err)
	}
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	limit := 5
	if limitRaw, ok := input["limit"].(float64); ok {
		limit = int(limitRaw)
		if limit > 20 {
			limit = 20
		}
		if limit < 1 {
			limit = 1
		}
	}

	results, err := t.MemorySvc.SearchEpisodic(ctx, userID, query, limit)
	if err != nil {
		return "", fmt.Errorf("search episodic: %w", err)
	}

	// Format results
	formattedResults := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		formattedResults = append(formattedResults, map[string]interface{}{
			"memory_id":        r.Memory.ID.String(),
			"memory_type":      r.Memory.MemoryType,
			"event_date":       r.Memory.EventDate.Format("2006-01-02"),
			"title":            r.Memory.Title,
			"summary":          r.Memory.Summary,
			"importance_score": r.Memory.ImportanceScore,
			"tags":             r.Memory.Tags,
			"similarity":       fmt.Sprintf("%.3f", r.Similarity),
		})
	}

	response := map[string]interface{}{
		"query":        query,
		"total_found":  len(results),
		"memories":     formattedResults,
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return string(resultJSON), nil
}

// GetSemanticFactsTool retrieves semantic facts about the athlete
// LLD Section 8.3
type GetSemanticFactsTool struct {
	MemorySvc MemoryInterface
}

func (t *GetSemanticFactsTool) Name() string {
	return "memory.get_semantic_facts"
}

func (t *GetSemanticFactsTool) Description() string {
	return "Retrieve learned semantic facts about the athlete (preferences, injury history, goals, patterns). These are high-confidence generalizations extracted from episodic memories."
}

func (t *GetSemanticFactsTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"min_confidence": {
					Type:        "number",
					Description: "Minimum confidence threshold 0.0-1.0 (default 0.7)",
				},
			},
			Required: []string{},
		},
	}
}

func (t *GetSemanticFactsTool) Execute(ctx context.Context, userIDStr string, input map[string]any) (string, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %v", err)
	}
	minConfidence := 0.7
	if conf, ok := input["min_confidence"].(float64); ok {
		minConfidence = conf
	}

	facts, err := t.MemorySvc.GetSemanticFacts(ctx, userID, minConfidence)
	if err != nil {
		return "", fmt.Errorf("get semantic facts: %w", err)
	}

	// Format results
	formattedFacts := make([]map[string]interface{}, 0, len(facts))
	for _, fact := range facts {
		formattedFacts = append(formattedFacts, map[string]interface{}{
			"fact_key":       fact.FactKey,
			"fact_value":     fact.FactValue,
			"confidence":     fmt.Sprintf("%.2f", fact.Confidence),
			"evidence_count": fact.EvidenceCount,
			"first_observed": fact.FirstObserved.Format("2006-01-02"),
			"last_updated":   fact.LastUpdated.Format("2006-01-02"),
		})
	}

	response := map[string]interface{}{
		"min_confidence": minConfidence,
		"total_facts":    len(facts),
		"facts":          formattedFacts,
	}

	resultJSON, _ := json.MarshalIndent(response, "", "  ")
	return string(resultJSON), nil
}
