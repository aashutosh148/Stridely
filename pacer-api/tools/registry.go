package tools

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/pacer-api/db"
	"github.com/yourname/pacer-api/llm"
	"github.com/yourname/pacer-api/models"
)

// Tool interface that all tools must implement
type Tool interface {
	Definition() llm.ToolDefinition
	Execute(ctx context.Context, userID string, input map[string]any) (string, error)
}

// PlanningInterface defines the interface for planning service operations
type PlanningInterface interface {
	GenerateBlock(ctx context.Context, userID uuid.UUID, params models.BlockParams) (*models.TrainingBlock, []models.WeekPlan, error)
	AdjustWeeklyPlan(ctx context.Context, userID uuid.UUID, readinessScore int, missedSessions int) error
	SubstituteWorkout(ctx context.Context, workoutID uuid.UUID, reason string) (*models.Workout, error)
}

// StravaInterface defines Strava service operations
type StravaInterface interface {
	// Add methods as needed
}

// Dependencies holds all services needed by tools
type Dependencies struct {
	DB       *db.Postgres
	Redis    *db.Redis
	Strava   StravaInterface
	Planning PlanningInterface
	Memory   MemoryInterface
}

// Registry manages all available tools
type Registry struct {
	tools    map[string]Tool
	db       *db.Postgres
	redis    *db.Redis
	strava   StravaInterface
	planning PlanningInterface
	memory   MemoryInterface
}

// NewRegistry creates and populates the tool registry
func NewRegistry(deps *Dependencies) *Registry {
	r := &Registry{
		tools:    make(map[string]Tool),
		db:       deps.DB,
		redis:    deps.Redis,
		strava:   deps.Strava,
		planning: deps.Planning,
		memory:   deps.Memory,
	}

	// Analysis group - Session 4
	r.register(NewCalculateTSSTool(deps))
	r.register(NewCalculateCTLTool(deps))
	r.register(NewCalculateATLTool(deps))
	r.register(NewCalculateTSBTool(deps))
	r.register(NewEstimateLactateThresholdTool(deps))
	r.register(NewCardiacDecouplingTool(deps))
	r.register(NewDetectLoadSpikeTool(deps))
	// Note: Running economy tool skipped - requires Garmin data

	// Planning group - Session 5
	r.register(NewGenerateTrainingBlockTool(deps, deps.Planning))
	r.register(NewAdjustWeeklyPlanTool(deps, deps.Planning))
	r.register(NewSubstituteWorkoutTool(deps, deps.Planning))
	r.register(NewGenerateTaperTool(deps, deps.Planning))

	// Memory group - Session 6
	r.register(&StoreEpisodicTool{MemorySvc: deps.Memory})
	r.register(&SearchEpisodicTool{MemorySvc: deps.Memory})
	r.register(&GetSemanticFactsTool{MemorySvc: deps.Memory})

	// Strava group - Future session
	r.register(&GetActivitiesTool{deps: deps})
	r.register(&GetActivityDetailTool{deps: deps})
	r.register(&GetAthleteStatsTool{deps: deps})
	r.register(&GetGearTool{deps: deps})
	r.register(&GetSegmentEffortsTool{deps: deps})

	// Garmin group
	r.register(&GetDailySummaryTool{deps: deps})
	r.register(&GetHrvDataTool{deps: deps})
	r.register(&GetSleepDataTool{deps: deps})
	r.register(&GetActivityMetricsTool{deps: deps})
	r.register(&GetTrainingLoadHistoryTool{deps: deps})

	// Prediction group - Future session
	r.register(&PredictFinishTimeTool{deps: deps})
	r.register(&PredictInjuryRiskTool{deps: deps})
	r.register(&PredictWallPointTool{deps: deps})

	// Race group
	r.register(&AnalyzeCourseTool{deps: deps})
	r.register(&GetWeatherTool{deps: deps})
	r.register(&GeneratePacingStrategyTool{deps: deps})
	r.register(&GenerateFuelingPlanTool{deps: deps})

	return r
}

// register adds a tool to the registry
func (r *Registry) register(tool Tool) {
	def := tool.Definition()
	r.tools[def.Name] = tool
}

// GetAllDefinitions returns all tool definitions
func (r *Registry) GetAllDefinitions() []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

// ExecuteAll runs multiple tools in parallel and returns results
func (r *Registry) ExecuteAll(ctx context.Context, userID string, toolUses []llm.ContentBlock) []llm.ContentBlock {
	results := make([]llm.ContentBlock, len(toolUses))
	var wg sync.WaitGroup

	for i, tu := range toolUses {
		wg.Add(1)
		go func(idx int, toolUse llm.ContentBlock) {
			defer wg.Done()
			start := time.Now()

			tool, ok := r.tools[toolUse.Name]
			if !ok {
				results[idx] = llm.NewToolResultBlock(toolUse.ID, "tool not found", true)
				slog.Warn("tool not found", "tool", toolUse.Name, "user_id", userID)
				return
			}

			output, err := tool.Execute(ctx, userID, toolUse.Input)
			if err != nil {
				errMsg := normalizeToolError(err)
				results[idx] = llm.NewToolResultBlock(toolUse.ID, errMsg, true)
				slog.Warn("tool execution failed",
					"tool", toolUse.Name,
					"user_id", userID,
					"duration_ms", time.Since(start).Milliseconds(),
					"error", errMsg,
				)
				return
			}

			results[idx] = llm.NewToolResultBlock(toolUse.ID, output, false)
			slog.Info("tool_executed",
				"tool", toolUse.Name,
				"user_id", userID,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}(i, tu)
	}

	wg.Wait()
	return results
}

// Execute runs a single tool (convenience method)
func (r *Registry) Execute(ctx context.Context, userID string, toolName string, input map[string]any) (string, error) {
	tool, ok := r.tools[toolName]
	if !ok {
		return "", ErrToolNotFound
	}
	return tool.Execute(ctx, userID, input)
}

// marshalJSON is a helper to convert results to JSON strings
func marshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Common errors
var (
	ErrToolNotFound = newToolError("tool not found")
)

type toolError struct {
	message string
}

func (e *toolError) Error() string {
	return e.message
}

func newToolError(msg string) error {
	return &toolError{message: msg}
}

func normalizeToolError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate"):
		return "rate limited, try in 1min"
	case strings.Contains(lower, "timeout") && strings.Contains(lower, "database"):
		return "database timeout"
	case errors.Is(err, context.DeadlineExceeded):
		return "database timeout"
	default:
		return msg
	}
}
