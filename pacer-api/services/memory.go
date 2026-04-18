package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/aashutosh148/Stridely/pacer-api/models"
)

// MemoryService handles memory operations (LLD Section 8)
type MemoryService struct {
	Pool           *pgxpool.Pool
	OpenAIAPIKey   string
	embeddingURL   string
	embeddingModel string
}

// NewMemoryService creates a new memory service
func NewMemoryService(pool *pgxpool.Pool, openAIKey string) *MemoryService {
	return &MemoryService{
		Pool:           pool,
		OpenAIAPIKey:   openAIKey,
		embeddingURL:   "https://api.openai.com/v1/embeddings",
		embeddingModel: "text-embedding-3-small",
	}
}

// OpenAI Embedding API types
type embeddingRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format"`
}

type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed generates an embedding vector using OpenAI text-embedding-3-small
func (m *MemoryService) Embed(ctx context.Context, text string) ([]float64, error) {
	if m.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	reqBody := embeddingRequest{
		Input:          text,
		Model:          m.embeddingModel,
		EncodingFormat: "float",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.embeddingURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.OpenAIAPIKey))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
	}

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embResp.Data[0].Embedding, nil
}

// StoreEpisodic stores a new episodic memory with async embedding generation
// LLD Section 8.2
func (m *MemoryService) StoreEpisodic(ctx context.Context, memory *models.EpisodicMemory) (uuid.UUID, error) {
	if memory.ID == uuid.Nil {
		memory.ID = uuid.New()
	}

	contentJSON, err := json.Marshal(memory.Content)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal content: %w", err)
	}

	// Insert memory record without embedding first
	_, err = m.Pool.Exec(ctx, `
		INSERT INTO episodic_memories (
			id, user_id, memory_type, event_date, title, summary, content,
			importance_score, tags, compressed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, memory.ID, memory.UserID, memory.MemoryType, memory.EventDate,
		memory.Title, memory.Summary, contentJSON, memory.ImportanceScore,
		memory.Tags, memory.Compressed)

	if err != nil {
		return uuid.Nil, fmt.Errorf("insert episodic memory: %w", err)
	}

	// Generate embedding asynchronously
	go func() {
		bgCtx := context.Background()
		
		// Combine title + summary for embedding
		embeddingText := fmt.Sprintf("%s\n%s", memory.Title, memory.Summary)
		
		embedding, err := m.Embed(bgCtx, embeddingText)
		if err != nil {
			slog.Error("failed to generate embedding",
				"memory_id", memory.ID,
				"error", err,
			)
			return
		}

		// Convert []float64 to pgvector format string
		embeddingJSON, _ := json.Marshal(embedding)
		
		_, err = m.Pool.Exec(bgCtx, `
			UPDATE episodic_memories
			SET embedding = $1::vector
			WHERE id = $2
		`, string(embeddingJSON), memory.ID)

		if err != nil {
			slog.Error("failed to store embedding",
				"memory_id", memory.ID,
				"error", err,
			)
		} else {
			slog.Info("embedding stored",
				"memory_id", memory.ID,
			)
		}
	}()

	slog.Info("episodic memory stored",
		"memory_id", memory.ID,
		"type", memory.MemoryType,
		"user_id", memory.UserID,
	)

	return memory.ID, nil
}

// SearchEpisodic performs semantic search using pgvector cosine similarity
// LLD Section 7.4 - Returns top-k most relevant memories
func (m *MemoryService) SearchEpisodic(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	// Generate query embedding
	queryEmbedding, err := m.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	embeddingJSON, _ := json.Marshal(queryEmbedding)

	// Cosine similarity search (LLD 7.4 spec)
	// 1 - (embedding <=> query) gives cosine similarity (higher = more similar)
	rows, err := m.Pool.Query(ctx, `
		SELECT
			id, user_id, memory_type, event_date, title, summary, content,
			importance_score, tags, compressed, created_at,
			1 - (embedding <=> $1::vector) AS similarity
		FROM episodic_memories
		WHERE user_id = $2
		  AND embedding IS NOT NULL
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`, string(embeddingJSON), userID, limit)

	if err != nil {
		return nil, fmt.Errorf("search episodic: %w", err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var sr models.SearchResult
		var contentJSON []byte
		var tags []string

		err := rows.Scan(
			&sr.Memory.ID,
			&sr.Memory.UserID,
			&sr.Memory.MemoryType,
			&sr.Memory.EventDate,
			&sr.Memory.Title,
			&sr.Memory.Summary,
			&contentJSON,
			&sr.Memory.ImportanceScore,
			&tags,
			&sr.Memory.Compressed,
			&sr.Memory.CreatedAt,
			&sr.Similarity,
		)

		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		if err := json.Unmarshal(contentJSON, &sr.Memory.Content); err != nil {
			return nil, fmt.Errorf("unmarshal content: %w", err)
		}

		sr.Memory.Tags = tags
		results = append(results, sr)
	}

	return results, nil
}

// GetSemanticFacts retrieves semantic facts filtered by confidence threshold
// LLD Section 8.3
func (m *MemoryService) GetSemanticFacts(ctx context.Context, userID uuid.UUID, minConfidence float64) ([]models.SemanticFact, error) {
	if minConfidence <= 0 {
		minConfidence = 0.7 // Default threshold
	}

	rows, err := m.Pool.Query(ctx, `
		SELECT id, user_id, fact_key, fact_value, confidence, evidence_count,
		       last_updated, first_observed, notes
		FROM semantic_facts
		WHERE user_id = $1
		  AND confidence >= $2
		ORDER BY confidence DESC, evidence_count DESC
	`, userID, minConfidence)

	if err != nil {
		return nil, fmt.Errorf("query semantic facts: %w", err)
	}
	defer rows.Close()

	var facts []models.SemanticFact
	for rows.Next() {
		var fact models.SemanticFact
		var valueJSON []byte
		var notes *string

		err := rows.Scan(
			&fact.ID,
			&fact.UserID,
			&fact.FactKey,
			&valueJSON,
			&fact.Confidence,
			&fact.EvidenceCount,
			&fact.LastUpdated,
			&fact.FirstObserved,
			&notes,
		)

		if err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}

		if err := json.Unmarshal(valueJSON, &fact.FactValue); err != nil {
			return nil, fmt.Errorf("unmarshal fact value: %w", err)
		}

		if notes != nil {
			fact.Notes = *notes
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// UpdateSemanticFact updates a semantic fact using Bayesian confidence update
// LLD Section 8.3 - Bayesian evidence accumulation
func (m *MemoryService) UpdateSemanticFact(ctx context.Context, userID uuid.UUID, factKey string, newValue map[string]interface{}, evidence bool) error {
	valueJSON, err := json.Marshal(newValue)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	// Check if fact exists
	var existingID uuid.UUID
	var currentConfidence float64
	var evidenceCount int

	err = m.Pool.QueryRow(ctx, `
		SELECT id, confidence, evidence_count
		FROM semantic_facts
		WHERE user_id = $1 AND fact_key = $2
	`, userID, factKey).Scan(&existingID, &currentConfidence, &evidenceCount)

	if err != nil && err.Error() != "no rows in result set" {
		return fmt.Errorf("query existing fact: %w", err)
	}

	if err != nil && err.Error() == "no rows in result set" {
		// Create new fact
		initialConfidence := 0.3
		if evidence {
			initialConfidence = 0.6
		}

		_, err = m.Pool.Exec(ctx, `
			INSERT INTO semantic_facts (user_id, fact_key, fact_value, confidence, evidence_count)
			VALUES ($1, $2, $3, $4, 1)
		`, userID, factKey, valueJSON, initialConfidence)

		if err != nil {
			return fmt.Errorf("insert semantic fact: %w", err)
		}

		slog.Info("semantic fact created",
			"user_id", userID,
			"fact_key", factKey,
			"confidence", initialConfidence,
		)

		return nil
	}

	// Bayesian update (LLD Section 8.3)
	// P(fact|evidence) = P(evidence|fact) * P(fact) / P(evidence)
	// Simplified: confidence += 0.1 if evidence, -= 0.05 if counter-evidence
	newConfidence := currentConfidence
	if evidence {
		// Evidence supports the fact
		newConfidence = currentConfidence + (1-currentConfidence)*0.15
	} else {
		// Counter-evidence
		newConfidence = currentConfidence * 0.85
	}

	// Clamp to [0, 1]
	newConfidence = math.Max(0, math.Min(1, newConfidence))

	_, err = m.Pool.Exec(ctx, `
		UPDATE semantic_facts
		SET fact_value = $1,
		    confidence = $2,
		    evidence_count = evidence_count + 1,
		    last_updated = NOW()
		WHERE id = $3
	`, valueJSON, newConfidence, existingID)

	if err != nil {
		return fmt.Errorf("update semantic fact: %w", err)
	}

	slog.Info("semantic fact updated",
		"fact_key", factKey,
		"old_confidence", currentConfidence,
		"new_confidence", newConfidence,
		"evidence_count", evidenceCount+1,
	)

	return nil
}

// PostLoopUpdate stores coaching moment and extracts semantic signals after agent turn
// LLD Section 8.4
func (m *MemoryService) PostLoopUpdate(ctx context.Context, userID uuid.UUID, conversationSummary string, insights []string) error {
	// 1. Store coaching moment as episodic memory
	memory := &models.EpisodicMemory{
		UserID:          userID,
		MemoryType:      models.MemoryTypeCoachingMoment,
		EventDate:       time.Now(),
		Title:           "Coaching conversation",
		Summary:         conversationSummary,
		Content:         map[string]interface{}{"insights": insights},
		ImportanceScore: 0.6,
		Tags:            []string{"coaching", "conversation"},
		Compressed:      false,
	}

	memoryID, err := m.StoreEpisodic(ctx, memory)
	if err != nil {
		return fmt.Errorf("store coaching moment: %w", err)
	}

	slog.Info("coaching moment stored", "memory_id", memoryID)

	// 2. Extract semantic signals (simple keyword-based extraction)
	// In production, this would use LLM to extract structured facts
	for _, insight := range insights {
		// Example patterns (simplified)
		// Real implementation would parse LLM-generated structured data
		
		// Check for preference signals
		if containsKeyword(insight, []string{"prefer", "like", "enjoy"}) {
			m.UpdateSemanticFact(ctx, userID, "training_preferences", map[string]interface{}{
				"note": insight,
			}, true)
		}

		// Check for injury signals
		if containsKeyword(insight, []string{"pain", "injury", "sore", "ache"}) {
			m.UpdateSemanticFact(ctx, userID, "injury_history", map[string]interface{}{
				"recent_issue": insight,
				"date": time.Now().Format("2006-01-02"),
			}, true)
		}

		// Check for goal signals
		if containsKeyword(insight, []string{"goal", "target", "aim", "want to"}) {
			m.UpdateSemanticFact(ctx, userID, "athlete_goals", map[string]interface{}{
				"stated_goal": insight,
			}, true)
		}
	}

	return nil
}

// GetFatigueGenome retrieves fatigue genome or returns population defaults
// LLD Section 8
func (m *MemoryService) GetFatigueGenome(ctx context.Context, userID uuid.UUID) (*models.FatigueGenome, error) {
	var genome models.FatigueGenome
	var genomeJSON []byte

	err := m.Pool.QueryRow(ctx, `
		SELECT id, user_id, model_version, data_points, confidence, genome_data, last_calibrated
		FROM fatigue_genome
		WHERE user_id = $1
	`, userID).Scan(
		&genome.ID,
		&genome.UserID,
		&genome.ModelVersion,
		&genome.DataPoints,
		&genome.Confidence,
		&genomeJSON,
		&genome.LastCalibrated,
	)

	if err != nil && err.Error() == "no rows in result set" {
		// Return population defaults (LLD Section 8)
		slog.Info("no fatigue genome found, using population defaults", "user_id", userID)
		
		return &models.FatigueGenome{
			UserID:       userID,
			ModelVersion: 1,
			DataPoints:   0,
			Confidence:   "insufficient",
			GenomeData: map[string]interface{}{
				"ctl_decay_rate":         0.07,  // Standard Banister model
				"atl_decay_rate":         0.2,   // Standard Banister model
				"recovery_coefficient":   1.0,   // Population average
				"injury_susceptibility":  0.5,   // Medium
				"overreaching_threshold": -30.0, // TSB threshold
				"freshness_threshold":    10.0,  // TSB threshold
			},
			LastCalibrated: time.Now(),
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("query fatigue genome: %w", err)
	}

	if err := json.Unmarshal(genomeJSON, &genome.GenomeData); err != nil {
		return nil, fmt.Errorf("unmarshal genome data: %w", err)
	}

	return &genome, nil
}

// Helper function for keyword matching
func containsKeyword(text string, keywords []string) bool {
	textLower := strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
