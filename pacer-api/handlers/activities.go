package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/models"
	"github.com/aashutosh148/Stridely/pacer-api/services"
)

type ActivitiesHandler struct {
	db       *db.Postgres
	strava   *services.StravaClient
	analysis *services.AnalysisService
}

func NewActivitiesHandler(dbConn *db.Postgres, strava *services.StravaClient, analysis *services.AnalysisService) *ActivitiesHandler {
	return &ActivitiesHandler{
		db:       dbConn,
		strava:   strava,
		analysis: analysis,
	}
}

// List returns paginated list of activities
func (h *ActivitiesHandler) List(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	// Parse pagination parameters
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	// Optional date filters
	after := c.Query("after")   // YYYY-MM-DD
	before := c.Query("before") // YYYY-MM-DD

	// Build query
	query := `
		SELECT id, user_id, strava_id, activity_date, workout_type,
		       distance_m, duration_s, elevation_gain_m, avg_pace_s,
		       avg_hr, max_hr, tss, zone_distribution, created_at
		FROM activities
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if after != "" {
		argCount++
		query += " AND activity_date >= $" + strconv.Itoa(argCount)
		args = append(args, after)
	}

	if before != "" {
		argCount++
		query += " AND activity_date <= $" + strconv.Itoa(argCount)
		args = append(args, before)
	}

	query += " ORDER BY activity_date DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	// Execute query
	rows, err := h.db.Pool.Query(c.Context(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "database query failed"})
	}
	defer rows.Close()

	// Parse results
	var activities []models.Activity
	for rows.Next() {
		var activity models.Activity
		var zoneDistJSON []byte

		err := rows.Scan(
			&activity.ID, &activity.UserID, &activity.StravaID,
			&activity.ActivityDate, &activity.WorkoutType,
			&activity.DistanceM, &activity.DurationS, &activity.ElevationGainM,
			&activity.AvgPaceS, &activity.AvgHR, &activity.MaxHR,
			&activity.TSS, &zoneDistJSON, &activity.CreatedAt,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to parse activity"})
		}

		// Unmarshal zone distribution
		if len(zoneDistJSON) > 0 {
			var zoneDist models.ZoneDistribution
			if err := json.Unmarshal(zoneDistJSON, &zoneDist); err == nil {
				activity.ZoneDistribution = &zoneDist
			}
		}

		activities = append(activities, activity)
	}

	if activities == nil {
		activities = []models.Activity{} // Return empty array instead of null
	}

	// Get total count for pagination
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM activities WHERE user_id = $1"
	countArgs := []interface{}{userID}
	if after != "" {
		countQuery += " AND activity_date >= $2"
		countArgs = append(countArgs, after)
	}
	if before != "" {
		if after != "" {
			countQuery += " AND activity_date <= $3"
			countArgs = append(countArgs, before)
		} else {
			countQuery += " AND activity_date <= $2"
			countArgs = append(countArgs, before)
		}
	}

	h.db.Pool.QueryRow(c.Context(), countQuery, countArgs...).Scan(&totalCount)

	return c.JSON(fiber.Map{
		"activities": activities,
		"total":      totalCount,
		"limit":      limit,
		"offset":     offset,
	})
}

// Get returns single activity detail
func (h *ActivitiesHandler) Get(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	activityID := c.Params("id")

	// Parse UUID
	id, err := uuid.Parse(activityID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid activity ID"})
	}

	// Fetch activity
	var activity models.Activity
	var zoneDistJSON, splitsJSON []byte

	err = h.db.Pool.QueryRow(c.Context(), `
		SELECT id, user_id, strava_id, activity_date, workout_type,
		       distance_m, duration_s, elevation_gain_m, avg_pace_s,
		       avg_hr, max_hr, tss, intensity_factor, zone_distribution,
		       cardiac_decoupling_pct, splits_km, gear_id, created_at
		FROM activities
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(
		&activity.ID, &activity.UserID, &activity.StravaID,
		&activity.ActivityDate, &activity.WorkoutType,
		&activity.DistanceM, &activity.DurationS, &activity.ElevationGainM,
		&activity.AvgPaceS, &activity.AvgHR, &activity.MaxHR,
		&activity.TSS, &activity.IntensityFactor, &zoneDistJSON,
		&activity.CardiacDecouplingPct, &splitsJSON, &activity.GearID,
		&activity.CreatedAt,
	)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "activity not found"})
	}

	// Unmarshal JSON fields
	if len(zoneDistJSON) > 0 {
		var zoneDist models.ZoneDistribution
		if err := json.Unmarshal(zoneDistJSON, &zoneDist); err == nil {
			activity.ZoneDistribution = &zoneDist
		}
	}

	if len(splitsJSON) > 0 {
		var splits []models.Split
		if err := json.Unmarshal(splitsJSON, &splits); err == nil {
			activity.SplitsKm = splits
		}
	}

	return c.JSON(activity)
}

// Recent returns last 10 activities (for dashboard widget)
func (h *ActivitiesHandler) Recent(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	
	slog.Info("📊 fetching recent activities", "user_id", userID)

	rows, err := h.db.Pool.Query(c.Context(), `
		SELECT id, strava_id, activity_date, workout_type,
		       distance_m, duration_s, avg_pace_s, tss
		FROM activities
		WHERE user_id = $1
		ORDER BY activity_date DESC
		LIMIT 10
	`, userID)

	if err != nil {
		slog.Error("failed to query recent activities", "error", err, "user_id", userID)
		return c.Status(500).JSON(fiber.Map{"error": "database query failed"})
	}
	defer rows.Close()

	type RecentActivity struct {
		ID           uuid.UUID            `json:"id"`
		StravaID     string               `json:"strava_id"`
		ActivityDate time.Time            `json:"activity_date"`
		WorkoutType  models.WorkoutType   `json:"workout_type"`
		DistanceM    int                  `json:"distance_m"`
		DurationS    int                  `json:"duration_s"`
		AvgPaceS     *float64             `json:"avg_pace_s,omitempty"`
		TSS          *float64             `json:"tss,omitempty"`
	}

	var activities []RecentActivity
	for rows.Next() {
		var activity RecentActivity
		err := rows.Scan(
			&activity.ID, &activity.StravaID, &activity.ActivityDate,
			&activity.WorkoutType, &activity.DistanceM, &activity.DurationS,
			&activity.AvgPaceS, &activity.TSS,
		)
		if err != nil {
			slog.Error("failed to scan activity", "error", err)
			continue
		}
		activities = append(activities, activity)
	}

	if activities == nil {
		activities = []RecentActivity{} // Return empty array instead of null
	}

	slog.Info("✅ returning recent activities", "user_id", userID, "count", len(activities))

	return c.JSON(fiber.Map{"activities": activities})
}

// Sync triggers a manual sync of activities from Strava
func (h *ActivitiesHandler) Sync(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	
	// Parse optional days parameter (default: last 7 days)
	days := c.QueryInt("days", 7)
	if days < 1 || days > 365 {
		return c.Status(400).JSON(fiber.Map{"error": "days must be between 1 and 365"})
	}
	
	slog.Info("🔄 manual sync triggered", "user_id", userID, "days", days)
	
	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}
	
	// Start sync in background
	go func() {
		ctx := context.Background()
		if err := h.analysis.SyncStravaHistory(ctx, uid, h.strava, days); err != nil {
			slog.Error("❌ manual sync failed", "user_id", userID, "error", err)
		} else {
			slog.Info("✅ manual sync completed", "user_id", userID)
		}
	}()
	
	return c.JSON(fiber.Map{
		"status":  "syncing",
		"message": "Activity sync started in background. Check recent activities in a few moments.",
		"days":    days,
	})
}
