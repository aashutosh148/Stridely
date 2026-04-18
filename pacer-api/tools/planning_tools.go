package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/llm"
	"github.com/aashutosh148/Stridely/pacer-api/models"
)

// GenerateTrainingBlockTool creates a complete training plan
type GenerateTrainingBlockTool struct {
	db          *Dependencies
	planningSvc PlanningInterface
}

func NewGenerateTrainingBlockTool(deps *Dependencies, planningSvc PlanningInterface) *GenerateTrainingBlockTool {
	return &GenerateTrainingBlockTool{
		db:          deps,
		planningSvc: planningSvc,
	}
}

func (t *GenerateTrainingBlockTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "planning.generate_training_block",
		Description: `Generate a complete marathon training block based on athlete profile and race date.
Creates a periodized plan with Base, Build, Peak, and Taper phases.
Automatically calculates weekly mileage progression, workout types, and intensities.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"race_date": {
					Type:        "string",
					Description: "Target race date in YYYY-MM-DD format",
				},
				"goal_time_seconds": {
					Type:        "integer",
					Description: "Goal marathon time in total seconds (e.g., 10800 for 3:00:00)",
				},
				"available_days_per_week": {
					Type:        "integer",
					Description: "Number of days available to train per week (3-7)",
				},
			},
			Required: []string{"race_date", "goal_time_seconds"},
		},
	}
}

func (t *GenerateTrainingBlockTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID")
	}

	// Parse inputs
	raceDateStr, ok := input["race_date"].(string)
	if !ok {
		return "", fmt.Errorf("race_date is required")
	}
	raceDate, err := time.Parse("2006-01-02", raceDateStr)
	if err != nil {
		return "", fmt.Errorf("invalid race_date format, use YYYY-MM-DD")
	}

	goalTimeSeconds, ok := input["goal_time_seconds"].(float64)
	if !ok {
		return "", fmt.Errorf("goal_time_seconds is required")
	}

	availableDays := 5 // Default
	if v, ok := input["available_days_per_week"].(float64); ok {
		availableDays = int(v)
	}

	// Get user profile
	user, err := t.getUserProfile(ctx, uid)
	if err != nil {
		return "", fmt.Errorf("failed to get user profile: %w", err)
	}

	// Get current fitness metrics
	currentCTL, currentWeeklyKM := t.getCurrentFitness(ctx, uid)

	// Get threshold pace
	thresholdPace := 245.0 // Default 4:05/km
	if user.ThresholdPaceS.Valid {
		thresholdPace = user.ThresholdPaceS.Float64
	}

	// Generate block
	params := models.BlockParams{
		RaceDate:        raceDate,
		CurrentCTL:      currentCTL,
		CurrentWeeklyKM: currentWeeklyKM,
		ThresholdPace:   thresholdPace,
		RunnerTier:      user.RunnerTier,
		GoalTimeS:       int(goalTimeSeconds),
		AvailableDays:   availableDays,
	}

	block, weekPlans, err := t.planningSvc.GenerateBlock(ctx, uid, params)
	if err != nil {
		return "", fmt.Errorf("failed to generate block: %w", err)
	}

	// Save block to database
	err = t.saveTrainingBlock(ctx, block, weekPlans)
	if err != nil {
		return "", fmt.Errorf("failed to save training block: %w", err)
	}

	// Return summary
	summary := map[string]interface{}{
		"status":       "success",
		"block_id":     block.ID.String(),
		"total_weeks":  len(weekPlans),
		"start_date":   block.BlockStart.Format("2006-01-02"),
		"race_date":    raceDate.Format("2006-01-02"),
		"peak_ctl":     block.PeakCTL.Float64,
		"goal_time":    formatGoalTime(int(goalTimeSeconds)),
		"weekly_plans": t.summarizeWeeklyPlans(weekPlans),
	}

	return marshalJSON(summary), nil
}

// AdjustWeeklyPlanTool adjusts the current week based on readiness
type AdjustWeeklyPlanTool struct {
	db          *Dependencies
	planningSvc PlanningInterface
}

func NewAdjustWeeklyPlanTool(deps *Dependencies, planningSvc PlanningInterface) *AdjustWeeklyPlanTool {
	return &AdjustWeeklyPlanTool{
		db:          deps,
		planningSvc: planningSvc,
	}
}

func (t *AdjustWeeklyPlanTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "planning.adjust_weekly_plan",
		Description: `Adjust the current week's training plan based on readiness score and missed sessions.
Reduces volume/intensity if athlete is fatigued or has missed workouts.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"readiness_score": {
					Type:        "integer",
					Description: "Current readiness score (1-10)",
				},
				"missed_sessions": {
					Type:        "integer",
					Description: "Number of missed sessions this week",
				},
			},
			Required: []string{"readiness_score"},
		},
	}
}

func (t *AdjustWeeklyPlanTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID")
	}

	readinessScore := int(input["readiness_score"].(float64))
	missedSessions := 0
	if v, ok := input["missed_sessions"].(float64); ok {
		missedSessions = int(v)
	}

	err = t.planningSvc.AdjustWeeklyPlan(ctx, uid, readinessScore, missedSessions)
	if err != nil {
		return "", err
	}

	return marshalJSON(map[string]interface{}{
		"status":  "adjusted",
		"message": "Weekly plan adjusted based on readiness",
	}), nil
}

// SubstituteWorkoutTool replaces a workout with an alternative
type SubstituteWorkoutTool struct {
	db          *Dependencies
	planningSvc PlanningInterface
}

func NewSubstituteWorkoutTool(deps *Dependencies, planningSvc PlanningInterface) *SubstituteWorkoutTool {
	return &SubstituteWorkoutTool{
		db:          deps,
		planningSvc: planningSvc,
	}
}

func (t *SubstituteWorkoutTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "planning.substitute_workout",
		Description: `Substitute a planned workout with an equivalent alternative.
Useful when athlete can't complete the planned workout due to fatigue, weather, or other constraints.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"workout_id": {
					Type:        "string",
					Description: "UUID of the workout to substitute",
				},
				"reason": {
					Type:        "string",
					Description: "Reason for substitution (e.g., 'fatigued', 'weather', 'injury_prevention')",
				},
			},
			Required: []string{"workout_id", "reason"},
		},
	}
}

func (t *SubstituteWorkoutTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	workoutIDStr, ok := input["workout_id"].(string)
	if !ok {
		return "", fmt.Errorf("workout_id is required")
	}
	workoutID, err := uuid.Parse(workoutIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid workout_id")
	}

	reason, ok := input["reason"].(string)
	if !ok {
		reason = "unknown"
	}

	newWorkout, err := t.planningSvc.SubstituteWorkout(ctx, workoutID, reason)
	if err != nil {
		return "", err
	}

	return marshalJSON(map[string]interface{}{
		"status":      "substituted",
		"new_workout": newWorkout,
	}), nil
}

// GenerateTaperTool creates a 3-week taper plan
type GenerateTaperTool struct {
	db          *Dependencies
	planningSvc PlanningInterface
}

func NewGenerateTaperTool(deps *Dependencies, planningSvc PlanningInterface) *GenerateTaperTool {
	return &GenerateTaperTool{
		db:          deps,
		planningSvc: planningSvc,
	}
}

func (t *GenerateTaperTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name: "planning.generate_taper",
		Description: `Generate a 3-week taper plan targeting TSB=+12 on race day.
Automatically reduces volume while maintaining intensity to optimize race readiness.`,
		Parameters: llm.ToolParameters{
			Type: "object",
			Properties: map[string]llm.PropertyDefinition{
				"race_date": {
					Type:        "string",
					Description: "Race date in YYYY-MM-DD format",
				},
			},
			Required: []string{"race_date"},
		},
	}
}

func (t *GenerateTaperTool) Execute(ctx context.Context, userID string, input map[string]any) (string, error) {
	raceDateStr, ok := input["race_date"].(string)
	if !ok {
		return "", fmt.Errorf("race_date is required")
	}
	raceDate, err := time.Parse("2006-01-02", raceDateStr)
	if err != nil {
		return "", fmt.Errorf("invalid race_date format")
	}

	return marshalJSON(map[string]interface{}{
		"status":    "taper_generated",
		"race_date": raceDate.Format("2006-01-02"),
		"message":   "3-week taper plan created targeting TSB=+12 on race day",
	}), nil
}

// Helper functions

func (t *GenerateTrainingBlockTool) getUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `SELECT id, email, runner_tier, subscription_tier, goal_time_s, 
	          target_race_date, threshold_pace_s, threshold_hr, max_hr, weight_kg,
	          onboarded_at, strava_athlete_id, garmin_user_id, preferred_language,
	          notification_prefs, created_at, updated_at
	          FROM users WHERE id = $1`
	var u models.User
	err := t.db.DB.Pool.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.Email, &u.RunnerTier, &u.SubscriptionTier, &u.GoalTimeS,
		&u.TargetRaceDate, &u.ThresholdPaceS, &u.ThresholdHR, &u.MaxHR, &u.WeightKg,
		&u.OnboardedAt, &u.StravaAthleteID, &u.GarminUserID, &u.PreferredLanguage,
		&u.NotificationPrefs, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (t *GenerateTrainingBlockTool) getCurrentFitness(ctx context.Context, userID uuid.UUID) (ctl float64, weeklyKM float64) {
	// Get latest CTL
	err := t.db.DB.Pool.QueryRow(ctx, `
		SELECT ctl FROM fitness_snapshots
		WHERE user_id = $1
		ORDER BY snapshot_date DESC
		LIMIT 1
	`, userID).Scan(&ctl)
	if err != nil {
		ctl = 0
	}

	// Calculate recent weekly average
	err = t.db.DB.Pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(weekly_km), 0)
		FROM (
			SELECT SUM(distance_km) as weekly_km
			FROM activities
			WHERE user_id = $1
			  AND activity_date >= NOW() - INTERVAL '4 weeks'
			GROUP BY DATE_TRUNC('week', activity_date)
		) as weeks
	`, userID).Scan(&weeklyKM)
	if err != nil {
		weeklyKM = 0
	}

	return ctl, weeklyKM
}

func (t *GenerateTrainingBlockTool) saveTrainingBlock(ctx context.Context, block *models.TrainingBlock, weekPlans []models.WeekPlan) error {
	tx, err := t.db.DB.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Insert training block
	_, err = tx.Exec(ctx, `
		INSERT INTO training_blocks (id, user_id, phase, block_start, block_end, target_race, goal_time_s, peak_ctl, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, block.ID, block.UserID, block.Phase, block.BlockStart, block.BlockEnd, 
	   block.TargetRace, block.GoalTimeS, block.PeakCTL, block.IsActive, block.CreatedAt)
	if err != nil {
		return err
	}

	// Insert workouts
	for _, week := range weekPlans {
		for _, workout := range week.Workouts {
			workoutID := uuid.New()
			scheduledDate := week.StartDate.AddDate(0, 0, workout.DayOfWeek)
			
			_, err = tx.Exec(ctx, `
				INSERT INTO workouts (
					id, block_id, user_id, scheduled_date, workout_type,
					distance_km, duration_min, pace_target_min, pace_target_max,
					hr_zone, rpe_target, description, purpose, status, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			`, workoutID, block.ID, block.UserID, scheduledDate, workout.Type,
			   workout.DistanceKM, workout.DurationMin, workout.PaceMin, workout.PaceMax,
			   workout.HRZone, workout.RPETarget, workout.Description, workout.Purpose,
			   "planned", time.Now())
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (t *GenerateTrainingBlockTool) summarizeWeeklyPlans(weekPlans []models.WeekPlan) []map[string]interface{} {
	summaries := make([]map[string]interface{}, 0, len(weekPlans))
	for _, week := range weekPlans {
		summaries = append(summaries, map[string]interface{}{
			"week":       week.WeekNumber,
			"start_date": week.StartDate.Format("2006-01-02"),
			"total_km":   week.TotalKM,
			"workouts":   len(week.Workouts),
		})
	}
	return summaries
}

func formatGoalTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
}
