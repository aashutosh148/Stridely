package handlers

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourname/pacer-api/db"
	"github.com/yourname/pacer-api/models"
	"github.com/yourname/pacer-api/services"
)

type PlansHandler struct {
	db          *db.Postgres
	planningSvc *services.PlanningService
}

func NewPlansHandler(database *db.Postgres, planningSvc *services.PlanningService) *PlansHandler {
	return &PlansHandler{
		db:          database,
		planningSvc: planningSvc,
	}
}

// GetActive returns the active training block for the user
func (h *PlansHandler) GetActive(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	query := `SELECT id, user_id, phase, block_start, block_end, target_race,
	          goal_time_s, peak_ctl, is_active, created_at
	          FROM training_blocks
	          WHERE user_id = $1 AND is_active = true
	          ORDER BY created_at DESC
	          LIMIT 1`

	var block models.TrainingBlock
	err = h.db.Pool.QueryRow(c.Context(), query, uid).Scan(
		&block.ID, &block.UserID, &block.Phase, &block.BlockStart, &block.BlockEnd,
		&block.TargetRace, &block.GoalTimeS, &block.PeakCTL, &block.IsActive, &block.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "no active training block found"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	return c.JSON(block)
}

// GetWeek returns workouts for a specific week
func (h *PlansHandler) GetWeek(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	// Parse week offset (0 = current week, -1 = last week, 1 = next week)
	weekOffset := c.QueryInt("offset", 0)
	
	// Calculate week start and end dates
	now := time.Now()
	startOfWeek := now.AddDate(0, 0, -int(now.Weekday())+weekOffset*7)
	endOfWeek := startOfWeek.AddDate(0, 0, 6)

	query := `SELECT id, block_id, user_id, scheduled_date, workout_type,
	          distance_km, duration_min, pace_target_min, pace_target_max,
	          hr_zone, rpe_target, description, purpose, status, 
	          completed_activity_id, created_at
	          FROM workouts
	          WHERE user_id = $1
	            AND scheduled_date >= $2
	            AND scheduled_date <= $3
	          ORDER BY scheduled_date ASC`

	rows, err := h.db.Pool.Query(c.Context(), query, uid, startOfWeek, endOfWeek)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}
	defer rows.Close()

	workouts := []models.Workout{}
	for rows.Next() {
		var w models.Workout
		err := rows.Scan(
			&w.ID, &w.BlockID, &w.UserID, &w.ScheduledDate, &w.WorkoutType,
			&w.DistanceKM, &w.DurationMin, &w.PaceTargetMin, &w.PaceTargetMax,
			&w.HRZone, &w.RPETarget, &w.Description, &w.Purpose, &w.Status,
			&w.CompletedActivityID, &w.CreatedAt,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scan error"})
		}
		workouts = append(workouts, w)
	}

	return c.JSON(fiber.Map{
		"week_start": startOfWeek.Format("2006-01-02"),
		"week_end":   endOfWeek.Format("2006-01-02"),
		"workouts":   workouts,
		"count":      len(workouts),
	})
}

// Generate creates a new training block
func (h *PlansHandler) Generate(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	var req struct {
		RaceDate      string `json:"race_date"`
		GoalTimeS     int    `json:"goal_time_s"`
		AvailableDays int    `json:"available_days"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// Parse race date
	raceDate, err := time.Parse("2006-01-02", req.RaceDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid race_date format, use YYYY-MM-DD"})
	}

	// Get user profile
	user, err := h.getUserProfile(c.Context(), uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to get user profile"})
	}

	// Get current fitness
	currentCTL, currentWeeklyKM := h.getCurrentFitness(c.Context(), uid)

	// Get threshold pace
	thresholdPace := 245.0 // Default
	if user.ThresholdPaceS.Valid {
		thresholdPace = user.ThresholdPaceS.Float64
	}

	// Generate plan
	params := models.BlockParams{
		RaceDate:        raceDate,
		CurrentCTL:      currentCTL,
		CurrentWeeklyKM: currentWeeklyKM,
		ThresholdPace:   thresholdPace,
		RunnerTier:      user.RunnerTier,
		GoalTimeS:       req.GoalTimeS,
		AvailableDays:   req.AvailableDays,
	}

	block, weekPlans, err := h.planningSvc.GenerateBlock(c.Context(), uid, params)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate plan"})
	}

	// Save to database
	err = h.saveTrainingBlock(c.Context(), block, weekPlans)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to save plan"})
	}

	return c.Status(201).JSON(fiber.Map{
		"block_id":    block.ID,
		"total_weeks": len(weekPlans),
		"start_date":  block.BlockStart.Format("2006-01-02"),
		"race_date":   raceDate.Format("2006-01-02"),
		"peak_ctl":    block.PeakCTL.Float64,
	})
}

// UpdateWorkout updates a specific workout
func (h *PlansHandler) UpdateWorkout(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	workoutID := c.Params("id")
	wid, err := uuid.Parse(workoutID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid workout ID"})
	}

	var req struct {
		Status      string  `json:"status"`
		Description string  `json:"description"`
		DistanceKM  float64 `json:"distance_km"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// Update workout
	query := `UPDATE workouts
	          SET status = COALESCE(NULLIF($1, ''), status),
	              description = COALESCE(NULLIF($2, ''), description),
	              distance_km = COALESCE(NULLIF($3, 0), distance_km)
	          WHERE id = $4 AND user_id = $5`

	result, err := h.db.Pool.Exec(c.Context(), query,
		req.Status, req.Description, req.DistanceKM, wid, uid)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "update failed"})
	}

	if result.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "workout not found"})
	}

	return c.JSON(fiber.Map{
		"status":  "updated",
		"message": "workout updated successfully",
	})
}

// Adjust adjusts the current week's plan based on readiness
func (h *PlansHandler) Adjust(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	var req struct {
		ReadinessScore int `json:"readiness_score"`
		MissedSessions int `json:"missed_sessions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	err = h.planningSvc.AdjustWeeklyPlan(c.Context(), uid, req.ReadinessScore, req.MissedSessions)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "adjustment failed"})
	}

	return c.JSON(fiber.Map{
		"status":  "adjusted",
		"message": "weekly plan adjusted based on readiness",
	})
}

// Helper methods

func (h *PlansHandler) getUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `SELECT id, email, runner_tier, subscription_tier, goal_time_s,
	          target_race_date, threshold_pace_s, threshold_hr, max_hr, weight_kg,
	          onboarded_at, strava_athlete_id, garmin_user_id, preferred_language,
	          notification_prefs, created_at, updated_at
	          FROM users WHERE id = $1`
	var u models.User
	err := h.db.Pool.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.Email, &u.RunnerTier, &u.SubscriptionTier, &u.GoalTimeS,
		&u.TargetRaceDate, &u.ThresholdPaceS, &u.ThresholdHR, &u.MaxHR, &u.WeightKg,
		&u.OnboardedAt, &u.StravaAthleteID, &u.GarminUserID, &u.PreferredLanguage,
		&u.NotificationPrefs, &u.CreatedAt, &u.UpdatedAt,
	)
	return &u, err
}

func (h *PlansHandler) getCurrentFitness(ctx context.Context, userID uuid.UUID) (ctl float64, weeklyKM float64) {
	h.db.Pool.QueryRow(ctx, `
		SELECT ctl FROM fitness_snapshots
		WHERE user_id = $1
		ORDER BY snapshot_date DESC
		LIMIT 1
	`, userID).Scan(&ctl)

	h.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(weekly_km), 0)
		FROM (
			SELECT SUM(distance_km) as weekly_km
			FROM activities
			WHERE user_id = $1
			  AND activity_date >= NOW() - INTERVAL '4 weeks'
			GROUP BY DATE_TRUNC('week', activity_date)
		) as weeks
	`, userID).Scan(&weeklyKM)

	return ctl, weeklyKM
}

func (h *PlansHandler) saveTrainingBlock(ctx context.Context, block *models.TrainingBlock, weekPlans []models.WeekPlan) error {
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Deactivate existing blocks
	_, err = tx.Exec(ctx, `UPDATE training_blocks SET is_active = false WHERE user_id = $1`, block.UserID)
	if err != nil {
		return err
	}

	// Insert new block
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
