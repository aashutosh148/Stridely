package tools

import (
	"context"
	"encoding/json"
	"github.com/aashutosh148/Stridely/pacer-api/llm"
)

type GetActivitiesTool struct { deps *Dependencies }
func (t *GetActivitiesTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_activities", Description: "Get Strava activities"}
}
func (t *GetActivitiesTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	page := 1
	if v, ok := input["page"].(float64); ok { page = int(v) }
	perPage := 30
	if v, ok := input["per_page"].(float64); ok { perPage = int(v) }
	
	// t.deps.Strava.GetActivities...
	res := []map[string]any{{"id": 12345, "name": "Morning Run", "distance": 5000, "page": page, "per_page": perPage}}
	b, _ := json.Marshal(res)
	return string(b), nil
}

type GetActivityDetailTool struct { deps *Dependencies }
func (t *GetActivityDetailTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_activity_detail", Description: "Get Strava activity detail"}
}
func (t *GetActivityDetailTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	activityID, _ := input["activity_id"].(string)
	res := map[string]any{"id": activityID, "name": "Detailed Run", "type": "Run", "suffer_score": 50}
	b, _ := json.Marshal(res)
	return string(b), nil
}

type GetAthleteStatsTool struct { deps *Dependencies }
func (t *GetAthleteStatsTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_athlete_stats", Description: "Get athlete stats"}
}
func (t *GetAthleteStatsTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	res := map[string]any{"recent_run_totals": map[string]any{"count": 14, "distance": 120000}}
	b, _ := json.Marshal(res)
	return string(b), nil
}

type GetGearTool struct { deps *Dependencies }
func (t *GetGearTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_gear", Description: "Get gear"}
}
func (t *GetGearTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	gearID, _ := input["gear_id"].(string)
	res := map[string]any{"id": gearID, "name": "Pegasus 39", "distance": 250000}
	b, _ := json.Marshal(res)
	return string(b), nil
}

type GetSegmentEffortsTool struct { deps *Dependencies }
func (t *GetSegmentEffortsTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_segment_efforts", Description: "Get segment efforts"}
}
func (t *GetSegmentEffortsTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	segmentID, _ := input["segment_id"].(string)
	res := []map[string]any{{"id": 987, "elapsed_time": 300, "segment_id": segmentID}}
	b, _ := json.Marshal(res)
	return string(b), nil
}
