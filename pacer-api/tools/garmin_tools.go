package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yourname/pacer-api/llm"
)

type GetDailySummaryTool struct{ deps *Dependencies }

func (t *GetDailySummaryTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_daily_summary", Description: "Get daily summary"}
}
func (t *GetDailySummaryTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	return garminUnavailablePayload("daily_summary")
}

type GetHrvDataTool struct{ deps *Dependencies }

func (t *GetHrvDataTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_hrv_data", Description: "Get HRV data"}
}
func (t *GetHrvDataTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	return garminUnavailablePayload("hrv")
}

type GetSleepDataTool struct{ deps *Dependencies }

func (t *GetSleepDataTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_sleep_data", Description: "Get sleep data"}
}
func (t *GetSleepDataTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	return garminUnavailablePayload("sleep")
}

type GetActivityMetricsTool struct{ deps *Dependencies }

func (t *GetActivityMetricsTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_activity_metrics", Description: "Get activity metrics"}
}
func (t *GetActivityMetricsTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	return garminUnavailablePayload("activity_metrics")
}

type GetTrainingLoadHistoryTool struct{ deps *Dependencies }

func (t *GetTrainingLoadHistoryTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_training_load_history", Description: "Get training load history"}
}
func (t *GetTrainingLoadHistoryTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	return garminUnavailablePayload("training_load_history")
}

func garminUnavailablePayload(kind string) (string, error) {
	payload := map[string]any{
		"kind":                    kind,
		"garmin_data_unavailable": true,
		"data":                    nil,
		"as_of":                   time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(payload)
	return string(b), nil
}
