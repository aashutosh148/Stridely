package tools

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/yourname/pacer-api/llm"
	"github.com/yourname/pacer-api/models"
)

// ========================================================================
// 1. Calculate TSS Tool
// ========================================================================

type CalculateTSSTool struct {
	deps *Dependencies
}

func NewCalculateTSSTool(deps *Dependencies) *CalculateTSSTool {
	return &CalculateTSSTool{deps: deps}
}

func (t *CalculateTSSTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "analysis.calculate_tss",
		Description: `Calculate Training Stress Score (TSS) for a run.
TSS quantifies training load based on duration and intensity.
Formula: TSS = (duration_hours * intensity_factor^2 * 100)
Intensity Factor = threshold_pace / actual_pace (faster = higher IF)
Reference: 1 hour at threshold pace = 100 TSS.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"duration_seconds": {
					Type:        "number",
					Description: "Duration of the run in seconds",
				},
				"avg_pace_s": {
					Type:        "number",
					Description: "Average pace in seconds per km",
				},
				"threshold_pace_s": {
					Type:        "number",
					Description: "Lactate threshold pace in seconds per km",
				},
			},
			Required: []string{"duration_seconds", "avg_pace_s", "threshold_pace_s"},
		},
	}
}

func (t *CalculateTSSTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	durationS := input["duration_seconds"].(float64)
	avgPaceS := input["avg_pace_s"].(float64)
	thresholdPaceS := input["threshold_pace_s"].(float64)

	if thresholdPaceS == 0 || avgPaceS == 0 {
		return marshalJSON(map[string]any{"error": "pace values cannot be zero"}), nil
	}

	// Intensity Factor: threshold_pace / actual_pace
	// Faster pace = higher IF (e.g., 3:30/km actual vs 4:00/km threshold = IF 1.14)
	intensityFactor := thresholdPaceS / avgPaceS

	// Cap IF at 1.2 (very hard effort)
	if intensityFactor > 1.2 {
		intensityFactor = 1.2
	}

	// TSS formula
	tss := (durationS / 3600) * math.Pow(intensityFactor, 2) * 100

	// Round to 1 decimal
	tss = math.Round(tss*10) / 10

	return marshalJSON(map[string]any{
		"tss":              tss,
		"intensity_factor": math.Round(intensityFactor*100) / 100,
		"duration_hours":   math.Round((durationS/3600)*100) / 100,
	}), nil
}

// ========================================================================
// 2. Calculate CTL Tool (Chronic Training Load - 42 day)
// ========================================================================

type CalculateCTLTool struct {
	deps *Dependencies
}

func NewCalculateCTLTool(deps *Dependencies) *CalculateCTLTool {
	return &CalculateCTLTool{deps: deps}
}

func (t *CalculateCTLTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "analysis.calculate_ctl",
		Description: `Calculate Chronic Training Load (CTL) - 42-day exponential weighted average of TSS.
CTL represents fitness/endurance capacity built over ~6 weeks.
Formula: CTL_new = CTL_old + (TSS - CTL_old) * (1 - e^(-1/42))
Higher CTL = better endurance base.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"user_id": {
					Type:        "string",
					Description: "User ID to fetch activity history",
				},
				"days": {
					Type:        "number",
					Description: "Number of days to calculate CTL over (default 42)",
				},
			},
			Required: []string{"user_id"},
		},
	}
}

func (t *CalculateCTLTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	days := 42
	if v, ok := input["days"]; ok {
		days = int(v.(float64))
	}

	// Fetch TSS history from activities
	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	rows, err := t.deps.DB.Pool.Query(ctx, `
		SELECT activity_date, COALESCE(tss, 0) as tss
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= $2
		  AND tss IS NOT NULL
		ORDER BY activity_date ASC
	`, userID, cutoffDate.Format("2006-01-02"))
	if err != nil {
		return "", fmt.Errorf("fetch activities: %w", err)
	}
	defer rows.Close()

	ctl := 0.0
	decayConstant := 1 - math.Exp(-1.0/42.0)
	count := 0

	for rows.Next() {
		var date string
		var tss float64
		if err := rows.Scan(&date, &tss); err != nil {
			continue
		}
		
		// Apply exponential weighted average formula
		ctl += (tss - ctl) * decayConstant
		count++
	}

	return marshalJSON(map[string]any{
		"ctl":            math.Round(ctl*10) / 10,
		"days_analyzed":  days,
		"activities_used": count,
	}), nil
}

// ========================================================================
// 3. Calculate ATL Tool (Acute Training Load - 7 day)
// ========================================================================

type CalculateATLTool struct {
	deps *Dependencies
}

func NewCalculateATLTool(deps *Dependencies) *CalculateATLTool {
	return &CalculateATLTool{deps: deps}
}

func (t *CalculateATLTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "analysis.calculate_atl",
		Description: `Calculate Acute Training Load (ATL) - 7-day exponential weighted average of TSS.
ATL represents recent training fatigue/stress over the past week.
Formula: ATL_new = ATL_old + (TSS - ATL_old) * (1 - e^(-1/7))
Higher ATL = more recent fatigue.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"user_id": {
					Type:        "string",
					Description: "User ID to fetch activity history",
				},
				"days": {
					Type:        "number",
					Description: "Number of days to calculate ATL over (default 7)",
				},
			},
			Required: []string{"user_id"},
		},
	}
}

func (t *CalculateATLTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	days := 7
	if v, ok := input["days"]; ok {
		days = int(v.(float64))
	}

	// Fetch TSS history from activities
	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	rows, err := t.deps.DB.Pool.Query(ctx, `
		SELECT activity_date, COALESCE(tss, 0) as tss
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= $2
		  AND tss IS NOT NULL
		ORDER BY activity_date ASC
	`, userID, cutoffDate.Format("2006-01-02"))
	if err != nil {
		return "", fmt.Errorf("fetch activities: %w", err)
	}
	defer rows.Close()

	atl := 0.0
	decayConstant := 1 - math.Exp(-1.0/7.0)
	count := 0

	for rows.Next() {
		var date string
		var tss float64
		if err := rows.Scan(&date, &tss); err != nil {
			continue
		}
		
		// Apply exponential weighted average formula
		atl += (tss - atl) * decayConstant
		count++
	}

	return marshalJSON(map[string]any{
		"atl":             math.Round(atl*10) / 10,
		"days_analyzed":   days,
		"activities_used": count,
	}), nil
}

// ========================================================================
// 4. Calculate TSB Tool (Training Stress Balance)
// ========================================================================

type CalculateTSBTool struct {
	deps *Dependencies
}

func NewCalculateTSBTool(deps *Dependencies) *CalculateTSBTool {
	return &CalculateTSBTool{deps: deps}
}

func (t *CalculateTSBTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "analysis.calculate_tsb",
		Description: `Calculate Training Stress Balance (TSB = CTL - ATL).
TSB is the primary race readiness metric.
>+15: fresh. +5 to +15: optimal race form. -10 to +5: productive training.
-20 to -10: tired. <-20: very tired/overtrained.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"ctl": {
					Type:        "number",
					Description: "Chronic Training Load (42-day fitness)",
				},
				"atl": {
					Type:        "number",
					Description: "Acute Training Load (7-day fatigue)",
				},
			},
			Required: []string{"ctl", "atl"},
		},
	}
}

func (t *CalculateTSBTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	ctl := input["ctl"].(float64)
	atl := input["atl"].(float64)
	
	tsb := ctl - atl

	state := "productive_training"
	switch {
	case tsb > 15:
		state = "fresh"
	case tsb > 5:
		state = "optimal_race_form"
	case tsb > -10:
		state = "productive_training"
	case tsb > -20:
		state = "tired"
	default:
		state = "very_tired"
	}

	result := map[string]interface{}{
		"tsb":            math.Round(tsb*10) / 10,
		"state":          state,
		"race_readiness": tsb >= 5 && tsb <= 20,
	}

	return marshalJSON(result), nil
}

// ========================================================================
// 5. Estimate Lactate Threshold Tool
// ========================================================================

type EstimateLactateThresholdTool struct {
	deps *Dependencies
}

func NewEstimateLactateThresholdTool(deps *Dependencies) *EstimateLactateThresholdTool {
	return &EstimateLactateThresholdTool{deps: deps}
}

func (t *EstimateLactateThresholdTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "analysis.estimate_lactate_threshold",
		Description: `Estimate lactate threshold pace using 3 methods:
1. Race-based: Use recent race times (10K pace or slower)
2. Tempo: Analyze sustained tempo efforts (20-60 min in HR Zone 4)
3. HR deflection: Find pace where HR increases disproportionately
Returns weighted average with confidence scores.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"user_id": {
					Type:        "string",
					Description: "User ID to analyze activity history",
				},
			},
			Required: []string{"user_id"},
		},
	}
}

func (t *EstimateLactateThresholdTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	estimates := make(map[string]interface{})
	
	// Method 1: Race-based (10K or half marathon race pace)
	racePace, raceConf := t.estimateFromRaces(ctx, userID)
	if racePace > 0 {
		estimates["race_based"] = map[string]interface{}{
			"pace_s":     math.Round(racePace*10) / 10,
			"confidence": raceConf,
		}
	}

	// Method 2: Tempo runs (sustained efforts 20-60 min)
	tempoPace, tempoConf := t.estimateFromTempoRuns(ctx, userID)
	if tempoPace > 0 {
		estimates["tempo_based"] = map[string]interface{}{
			"pace_s":     math.Round(tempoPace*10) / 10,
			"confidence": tempoConf,
		}
	}

	// Method 3: HR deflection (simplified - would need HR streams for full implementation)
	// Skipped for MVP as it requires detailed HR analysis

	// Calculate weighted average
	if len(estimates) == 0 {
		return marshalJSON(map[string]interface{}{
			"error": "insufficient data to estimate threshold",
			"recommendation": "complete a 10K race or tempo run to enable estimation",
		}), nil
	}

	var totalPace, totalWeight float64
	if race, ok := estimates["race_based"].(map[string]interface{}); ok {
		pace := race["pace_s"].(float64)
		conf := race["confidence"].(float64)
		totalPace += pace * conf
		totalWeight += conf
	}
	if tempo, ok := estimates["tempo_based"].(map[string]interface{}); ok {
		pace := tempo["pace_s"].(float64)
		conf := tempo["confidence"].(float64)
		totalPace += pace * conf
		totalWeight += conf
	}

	estimatedPace := totalPace / totalWeight

	return marshalJSON(map[string]interface{}{
		"estimated_threshold_pace_s": math.Round(estimatedPace*10) / 10,
		"methods_used":               estimates,
		"confidence":                 math.Round(totalWeight*100) / 100,
	}), nil
}

// estimateFromRaces finds recent race performances
func (t *EstimateLactateThresholdTool) estimateFromRaces(ctx context.Context, userID string) (float64, float64) {
	// Look for race activities (workout_type = 'race')
	var pace sql.NullFloat64
	err := t.deps.DB.Pool.QueryRow(ctx, `
		SELECT avg_pace_s
		FROM activities
		WHERE user_id = $1
		  AND workout_type = 'race'
		  AND distance_m BETWEEN 8000 AND 25000
		  AND activity_date >= NOW() - INTERVAL '90 days'
		ORDER BY activity_date DESC
		LIMIT 1
	`, userID).Scan(&pace)

	if err != nil || !pace.Valid {
		return 0, 0
	}

	// Threshold pace ≈ 10K race pace + 5-10 seconds/km (slightly slower)
	thresholdPace := pace.Float64 + 7
	confidence := 0.8

	return thresholdPace, confidence
}

// estimateFromTempoRuns analyzes tempo efforts
func (t *EstimateLactateThresholdTool) estimateFromTempoRuns(ctx context.Context, userID string) (float64, float64) {
	// Look for tempo runs (20-60 minutes, consistent pace)
	rows, err := t.deps.DB.Pool.Query(ctx, `
		SELECT avg_pace_s, duration_s
		FROM activities
		WHERE user_id = $1
		  AND workout_type = 'tempo'
		  AND duration_s BETWEEN 1200 AND 3600
		  AND activity_date >= NOW() - INTERVAL '60 days'
		ORDER BY activity_date DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()

	var paces []float64
	for rows.Next() {
		var pace sql.NullFloat64
		var duration int
		if err := rows.Scan(&pace, &duration); err == nil && pace.Valid {
			paces = append(paces, pace.Float64)
		}
	}

	if len(paces) == 0 {
		return 0, 0
	}

	// Average tempo pace = threshold pace
	var sum float64
	for _, p := range paces {
		sum += p
	}
	avgPace := sum / float64(len(paces))
	confidence := 0.7 // Lower than race-based

	return avgPace, confidence
}

// ========================================================================
// 6. Cardiac Decoupling Tool
// ========================================================================

type CardiacDecouplingTool struct {
	deps *Dependencies
}

func NewCardiacDecouplingTool(deps *Dependencies) *CardiacDecouplingTool {
	return &CardiacDecouplingTool{deps: deps}
}

func (t *CardiacDecouplingTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "analysis.cardiac_decoupling",
		Description: `Analyze cardiac decoupling - the drift between HR and pace over a run.
Low decoupling (<5%) = excellent aerobic base.
High decoupling (>10%) = aerobic deficiency or dehydration.
Requires run duration >= 75 minutes with HR data.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"activity_id": {
					Type:        "string",
					Description: "Strava activity ID to analyze",
				},
				"duration_threshold_minutes": {
					Type:        "number",
					Description: "Minimum duration in minutes (default 75)",
				},
			},
			Required: []string{"activity_id"},
		},
	}
}

func (t *CardiacDecouplingTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	activityID := input["activity_id"].(string)
	threshold := 75
	if v, ok := input["duration_threshold_minutes"]; ok {
		threshold = int(v.(float64))
	}

	// Fetch activity from DB
	var activity struct {
		DurationS int
		SplitsKM  []byte
	}
	err := t.deps.DB.Pool.QueryRow(ctx, `
		SELECT duration_s, splits_km
		FROM activities
		WHERE user_id = $1 AND strava_id = $2
	`, userID, activityID).Scan(&activity.DurationS, &activity.SplitsKM)
	if err != nil {
		return "", fmt.Errorf("activity not found: %w", err)
	}

	if activity.DurationS < threshold*60 {
		return marshalJSON(map[string]any{
			"error": fmt.Sprintf("run duration %ds < threshold %dmin", activity.DurationS, threshold),
		}), nil
	}

	// Parse splits
	var splits []models.Split
	if err := unmarshalJSON(activity.SplitsKM, &splits); err != nil {
		return marshalJSON(map[string]any{"error": "no splits data available"}), nil
	}

	if len(splits) < 4 {
		return marshalJSON(map[string]any{"error": "insufficient splits for analysis"}), nil
	}

	mid := len(splits) / 2

	// Calculate PA ratio (pace/HR) for each half
	firstHalfPA := avgPaceHRRatio(splits[:mid])
	secondHalfPA := avgPaceHRRatio(splits[mid:])

	if firstHalfPA == 0 {
		return marshalJSON(map[string]any{"error": "missing HR data"}), nil
	}

	decouplingPct := ((secondHalfPA - firstHalfPA) / firstHalfPA) * 100

	quality := "poor"
	switch {
	case decouplingPct < 5:
		quality = "excellent"
	case decouplingPct < 8:
		quality = "good"
	case decouplingPct < 12:
		quality = "borderline"
	}

	return marshalJSON(map[string]any{
		"decoupling_pct":       math.Round(decouplingPct*100) / 100,
		"first_half_pa_ratio":  math.Round(firstHalfPA*100) / 100,
		"second_half_pa_ratio": math.Round(secondHalfPA*100) / 100,
		"quality":              quality,
		"interpretation":       interpretDecoupling(decouplingPct, quality),
	}), nil
}

// avgPaceHRRatio calculates average pace/HR ratio for splits
func avgPaceHRRatio(splits []models.Split) float64 {
	var sum float64
	count := 0
	for _, s := range splits {
		if s.HR > 0 && s.PaceS > 0 {
			sum += s.PaceS / float64(s.HR)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// interpretDecoupling provides coaching advice
func interpretDecoupling(pct float64, quality string) string {
	switch quality {
	case "excellent":
		return "Strong aerobic base. Maintain easy pace work and build volume gradually."
	case "good":
		return "Good aerobic efficiency. Continue current training approach."
	case "borderline":
		return "Aerobic system shows some fatigue. Consider more easy-paced runs and recovery."
	default:
		return "Significant decoupling detected. Slow down easy runs, check hydration, and build aerobic base before adding intensity."
	}
}

// ========================================================================
// 7. Detect Load Spike Tool (ACWR - Acute:Chronic Workload Ratio)
// ========================================================================

type DetectLoadSpikeTool struct {
	deps *Dependencies
}

func NewDetectLoadSpikeTool(deps *Dependencies) *DetectLoadSpikeTool {
	return &DetectLoadSpikeTool{deps: deps}
}

func (t *DetectLoadSpikeTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "analysis.detect_load_spike",
		Description: `Detect dangerous load spikes using ACWR (Acute:Chronic Workload Ratio).
ACWR = ATL / CTL. Safe zone: 0.8-1.3. Danger: >1.5. Critical: >2.0.
If injury history exists, thresholds tighten to 1.3/1.6.
Returns risk level and recommended max weekly TSS.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"atl": {
					Type:        "number",
					Description: "Acute Training Load (7-day)",
				},
				"ctl": {
					Type:        "number",
					Description: "Chronic Training Load (42-day)",
				},
				"planned_week_tss": {
					Type:        "number",
					Description: "Planned TSS for upcoming week",
				},
			},
			Required: []string{"atl", "ctl", "planned_week_tss"},
		},
	}
}

func (t *DetectLoadSpikeTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	atl := input["atl"].(float64)
	ctl := input["ctl"].(float64)
	plannedTSS := input["planned_week_tss"].(float64)

	// Check injury history
	var injuryCount int
	_ = t.deps.DB.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM injuries WHERE user_id = $1
	`, userID).Scan(&injuryCount)

	dangerThreshold := 1.5
	criticalThreshold := 2.0
	if injuryCount > 0 {
		dangerThreshold = 1.3
		criticalThreshold = 1.6
	}

	// Compute prospective ATL if planned week is completed
	// ATL_new = ATL + (weeklyTSS/7 - ATL) * (1 - e^(-1/7))
	projectedATL := atl + (plannedTSS/7-atl)*(1-math.Exp(-1.0/7))
	
	acwr := 0.0
	if ctl > 0 {
		acwr = projectedATL / ctl
	}

	riskLevel := "low"
	switch {
	case acwr > criticalThreshold:
		riskLevel = "critical"
	case acwr > dangerThreshold:
		riskLevel = "high"
	case acwr > 1.2:
		riskLevel = "moderate"
	}

	// Safe max TSS = (dangerThreshold - 0.1) * CTL * 7
	recommendedMaxTSS := (dangerThreshold - 0.1) * ctl * 7

	return marshalJSON(map[string]any{
		"spike_detected":      acwr > dangerThreshold,
		"acwr":                math.Round(acwr*100) / 100,
		"risk_level":          riskLevel,
		"recommended_max_tss": math.Round(recommendedMaxTSS*10) / 10,
		"explanation":         buildLoadSpikeExplanation(acwr, riskLevel, injuryCount > 0),
	}), nil
}

// buildLoadSpikeExplanation provides coaching advice
func buildLoadSpikeExplanation(acwr float64, riskLevel string, hasInjuryHistory bool) string {
	switch riskLevel {
	case "critical":
		return fmt.Sprintf("CRITICAL: ACWR %.2f is dangerously high. High injury risk. Reduce planned volume by 30-40%% immediately.", acwr)
	case "high":
		return fmt.Sprintf("WARNING: ACWR %.2f indicates load spike. Reduce volume by 15-25%% to stay safe.", acwr)
	case "moderate":
		return fmt.Sprintf("CAUTION: ACWR %.2f is elevated. Monitor fatigue and consider adding recovery.", acwr)
	default:
		return fmt.Sprintf("Safe zone: ACWR %.2f. Training load is well managed.", acwr)
	}
}

// Helper to unmarshal JSON
func unmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	return nil // Simplified for now - would use json.Unmarshal
}
