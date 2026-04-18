package tools

import (
	"context"
	"encoding/json"
	"github.com/aashutosh148/Stridely/pacer-api/llm"
)

type AnalyzeCourseTool struct { deps *Dependencies }
func (t *AnalyzeCourseTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "analyze_course", Description: "Analyze course"}
}
func (t *AnalyzeCourseTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	gpx, _ := input["gpx_data"].(string)
	
	// Mocking RaceService.AnalyzeCourse usage
	res := map[string]any{
		"difficulty": 0.8, 
		"key_hills": []string{"Extracted Hill (gradient > 3%, length > 200m)"},
		"parsed_gpx_length": len(gpx),
	}
	b, _ := json.Marshal(res)
	return string(b), nil
}

type GetWeatherTool struct { deps *Dependencies }
func (t *GetWeatherTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "get_weather", Description: "Get weather"}
}
func (t *GetWeatherTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	lat, _ := input["lat"].(float64)
	lng, _ := input["lng"].(float64)
	date, _ := input["date"].(string)
	
	// Mocking RaceService.GetWeatherForecast usage
	res := map[string]any{"temp": 15.0, "lat": lat, "lng": lng, "date": date}
	b, _ := json.Marshal(res)
	return string(b), nil
}

type GeneratePacingStrategyTool struct { deps *Dependencies }
func (t *GeneratePacingStrategyTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "generate_pacing_strategy", Description: "Generate pacing strategy"}
}
func (t *GeneratePacingStrategyTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	// Mocking RaceService.GeneratePacingStrategy usage
	res := []map[string]any{
		{"km": 1, "pace": "5:00", "elevation_adj": "0s"},
		{"km": 2, "pace": "4:55", "elevation_adj": "-5s"},
		{"km": 3, "pace": "5:10", "elevation_adj": "+10s (hill)"},
	}
	b, _ := json.Marshal(res)
	return string(b), nil
}

type GenerateFuelingPlanTool struct { deps *Dependencies }
func (t *GenerateFuelingPlanTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{Name: "generate_fueling_plan", Description: "Generate fueling plan"}
}
func (t *GenerateFuelingPlanTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	// Mocking RaceService.GenerateFuelingPlan usage
	res := map[string]any{
		"total_gels": 4,
		"water_ml_per_hour": 1200,
		"stations": []map[string]any{
			{"km": 5, "action": "Water"},
			{"km": 10, "action": "Gel + Water"},
		},
	}
	b, _ := json.Marshal(res)
	return string(b), nil
}
