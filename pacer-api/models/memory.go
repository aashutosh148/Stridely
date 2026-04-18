package models

import (
	"time"

	"github.com/google/uuid"
)

// Memory-related models (Session 6)

type MemoryType string

const (
	MemoryTypeActivity       MemoryType = "activity"
	MemoryTypeRace           MemoryType = "race"
	MemoryTypeInjury         MemoryType = "injury"
	MemoryTypeMilestone      MemoryType = "milestone"
	MemoryTypeNote           MemoryType = "note"
	MemoryTypeCoachingMoment MemoryType = "coaching_moment"
	MemoryTypeWeeklySummary  MemoryType = "weekly_summary"
)

// EpisodicMemory represents a stored memory record
type EpisodicMemory struct {
	ID              uuid.UUID              `json:"id"`
	UserID          uuid.UUID              `json:"user_id"`
	MemoryType      MemoryType             `json:"memory_type"`
	EventDate       time.Time              `json:"event_date"`
	Title           string                 `json:"title"`
	Summary         string                 `json:"summary"`
	Content         map[string]interface{} `json:"content"`
	ImportanceScore float64                `json:"importance_score"`
	Tags            []string               `json:"tags"`
	Compressed      bool                   `json:"compressed"`
	CreatedAt       time.Time              `json:"created_at"`
}

// SearchResult represents a memory search result with relevance score
type SearchResult struct {
	Memory     EpisodicMemory `json:"memory"`
	Similarity float64        `json:"similarity"`
}

// SemanticFact represents a learned fact about the user
type SemanticFact struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	FactKey        string                 `json:"fact_key"`
	FactValue      map[string]interface{} `json:"fact_value"`
	Confidence     float64                `json:"confidence"`
	EvidenceCount  int                    `json:"evidence_count"`
	LastUpdated    time.Time              `json:"last_updated"`
	FirstObserved  time.Time              `json:"first_observed"`
	Notes          string                 `json:"notes,omitempty"`
}

// FatigueGenome represents user-specific fatigue response patterns
type FatigueGenome struct {
	ID              uuid.UUID              `json:"id"`
	UserID          uuid.UUID              `json:"user_id"`
	ModelVersion    int                    `json:"model_version"`
	DataPoints      int                    `json:"data_points"`
	Confidence      string                 `json:"confidence"` // insufficient, low, medium, high
	GenomeData      map[string]interface{} `json:"genome_data"`
	LastCalibrated  time.Time              `json:"last_calibrated"`
}
