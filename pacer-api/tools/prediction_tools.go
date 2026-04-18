package tools

import (
	"context"
	"encoding/json"
	"github.com/aashutosh148/Stridely/pacer-api/llm"
)

type PredictFinishTimeTool struct { deps *Dependencies }
func (t *PredictFinishTimeTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "predict_finish_time", Description: "Predict finish time"}
}
func (t *PredictFinishTimeTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
    res := map[string]any{"predicted_time_seconds": 3600, "confidence_band_low": 3500, "confidence_band_high": 3700, "weekly_delta_seconds": -10}
	b, _ := json.Marshal(res)
    return string(b), nil
}

type PredictInjuryRiskTool struct { deps *Dependencies }
func (t *PredictInjuryRiskTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "predict_injury_risk", Description: "Predict injury risk"}
}
func (t *PredictInjuryRiskTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	return `{"score": 0.15}`, nil
}

type PredictWallPointTool struct { deps *Dependencies }
func (t *PredictWallPointTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "predict_wall_point", Description: "Predict wall point"}
}
func (t *PredictWallPointTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	return `{"distance_km": 30.5}`, nil
}
