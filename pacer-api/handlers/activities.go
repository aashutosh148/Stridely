package handlers

import (
	"context"
	"database/sql"
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
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 50)
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Optional filters
	workoutType := c.Query("workout_type")
	startDate := c.Query("start_date")   // YYYY-MM-DD
	endDate := c.Query("end_date")       // YYYY-MM-DD

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

	if workoutType != "" && workoutType != "all" {
		argCount++
		query += " AND workout_type = $" + strconv.Itoa(argCount)
		args = append(args, workoutType)
	}

	if startDate != "" {
		argCount++
		query += " AND activity_date >= $" + strconv.Itoa(argCount)
		args = append(args, startDate)
	}

	if endDate != "" {
		argCount++
		query += " AND activity_date <= $" + strconv.Itoa(argCount)
		args = append(args, endDate)
	}

	query += " ORDER BY activity_date DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	// Execute query
	rows, err := h.db.Pool.Query(c.Context(), query, args...)
	if err != nil {
		slog.Error("failed to query activities", "error", err)
		return c.Status(500).JSON(fiber.Map{"error": "database query failed"})
	}
	defer rows.Close()

	// Parse results
	var activities []models.Activity
	for rows.Next() {
		var activity models.Activity
		var zoneDistJSON []byte
		var avgPaceS, tss sql.NullFloat64
		var avgHR, maxHR sql.NullInt32

		err := rows.Scan(
			&activity.ID, &activity.UserID, &activity.StravaID,
			&activity.ActivityDate, &activity.WorkoutType,
			&activity.DistanceM, &activity.DurationS, &activity.ElevationGainM,
			&avgPaceS, &avgHR, &maxHR,
			&tss, &zoneDistJSON, &activity.CreatedAt,
		)
		if err != nil {
			slog.Error("failed to scan activity", "error", err)
			continue
		}

		// Convert sql.Null types to pointers
		if avgPaceS.Valid {
			activity.AvgPaceS = &avgPaceS.Float64
		}
		if avgHR.Valid {
			avgHRInt := int(avgHR.Int32)
			activity.AvgHR = &avgHRInt
		}
		if maxHR.Valid {
			maxHRInt := int(maxHR.Int32)
			activity.MaxHR = &maxHRInt
		}
		if tss.Valid {
			activity.TSS = &tss.Float64
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
	countArgCount := 1

	if workoutType != "" && workoutType != "all" {
		countArgCount++
		countQuery += " AND workout_type = $" + strconv.Itoa(countArgCount)
		countArgs = append(countArgs, workoutType)
	}

	if startDate != "" {
		countArgCount++
		countQuery += " AND activity_date >= $" + strconv.Itoa(countArgCount)
		countArgs = append(countArgs, startDate)
	}

	if endDate != "" {
		countArgCount++
		countQuery += " AND activity_date <= $" + strconv.Itoa(countArgCount)
		countArgs = append(countArgs, endDate)
	}

	h.db.Pool.QueryRow(c.Context(), countQuery, countArgs...).Scan(&totalCount)

	totalPages := (totalCount + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	return c.JSON(fiber.Map{
		"activities":  activities,
		"total":       totalCount,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
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
	var avgPaceS, tss, intensityFactor, cardiacDecouplingPct sql.NullFloat64
	var avgHR, maxHR sql.NullInt32
	var gearID sql.NullString

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
		&avgPaceS, &avgHR, &maxHR,
		&tss, &intensityFactor, &zoneDistJSON,
		&cardiacDecouplingPct, &splitsJSON, &gearID,
		&activity.CreatedAt,
	)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "activity not found"})
	}

	// Convert sql.Null types to pointers
	if avgPaceS.Valid {
		activity.AvgPaceS = &avgPaceS.Float64
	}
	if avgHR.Valid {
		avgHRInt := int(avgHR.Int32)
		activity.AvgHR = &avgHRInt
	}
	if maxHR.Valid {
		maxHRInt := int(maxHR.Int32)
		activity.MaxHR = &maxHRInt
	}
	if tss.Valid {
		activity.TSS = &tss.Float64
	}
	if intensityFactor.Valid {
		activity.IntensityFactor = &intensityFactor.Float64
	}
	if cardiacDecouplingPct.Valid {
		activity.CardiacDecouplingPct = &cardiacDecouplingPct.Float64
	}
	if gearID.Valid {
		activity.GearID = &gearID.String
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

// RecalculateZones recalculates HR zone distributions for all activities
func (h *ActivitiesHandler) RecalculateZones(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	
	slog.Info("🔄 recalculating HR zones", "user_id", userID)
	
	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}
	
	// Start recalculation in background
	go func() {
		ctx := context.Background()
		
		// Get user max HR
		var maxHR int
		err := h.db.Pool.QueryRow(ctx, `SELECT max_hr FROM users WHERE id = $1`, uid).Scan(&maxHR)
		if err != nil || maxHR == 0 {
			slog.Error("❌ user has no max_hr set", "user_id", userID, "error", err)
			return
		}
		
		// Get all activities with splits data
		rows, err := h.db.Pool.Query(ctx, `
			SELECT id, strava_id, splits_km
			FROM activities
			WHERE user_id = $1 AND splits_km IS NOT NULL AND splits_km != 'null'::jsonb
		`, uid)
		if err != nil {
			slog.Error("❌ failed to query activities", "user_id", userID, "error", err)
			return
		}
		defer rows.Close()
		
		updated := 0
		for rows.Next() {
			var activityID uuid.UUID
			var stravaID string
			var splitsJSON []byte
			
			if err := rows.Scan(&activityID, &stravaID, &splitsJSON); err != nil {
				continue
			}
			
			// Parse splits
			var splits []models.Split
			if err := json.Unmarshal(splitsJSON, &splits); err != nil {
				continue
			}
			
			// Convert to StravaSplit format for zone calculation
			var stravaSplits []services.StravaSplit
			for _, split := range splits {
				if split.HR > 0 {
					stravaSplits = append(stravaSplits, services.StravaSplit{
						AverageHeartrate: float64(split.HR),
						MovingTime:       int(split.PaceS), // approximate time per km
					})
				}
			}
			
			if len(stravaSplits) == 0 {
				continue
			}
			
			// Calculate zone distribution
			var totalTime int
			var z1Time, z2Time, z3Time, z4Time, z5Time int
			
			for _, split := range stravaSplits {
				time := split.MovingTime
				totalTime += time
				
				hrPct := (split.AverageHeartrate / float64(maxHR)) * 100
				
				switch {
				case hrPct < 60:
					z1Time += time
				case hrPct < 70:
					z2Time += time
				case hrPct < 80:
					z3Time += time
				case hrPct < 90:
					z4Time += time
				default:
					z5Time += time
				}
			}
			
			if totalTime > 0 {
				zoneDist := models.ZoneDistribution{
					Z1Pct: float64(z1Time) / float64(totalTime) * 100,
					Z2Pct: float64(z2Time) / float64(totalTime) * 100,
					Z3Pct: float64(z3Time) / float64(totalTime) * 100,
					Z4Pct: float64(z4Time) / float64(totalTime) * 100,
					Z5Pct: float64(z5Time) / float64(totalTime) * 100,
				}
				
				zoneJSON, _ := json.Marshal(zoneDist)
				
				_, err = h.db.Pool.Exec(ctx, `
					UPDATE activities SET zone_distribution = $1 WHERE id = $2
				`, zoneJSON, activityID)
				
				if err == nil {
					updated++
				}
			}
		}
		
		slog.Info("✅ zone recalculation completed", "user_id", userID, "activities_updated", updated)
	}()
	
	return c.JSON(fiber.Map{
		"status":  "processing",
		"message": "HR zone recalculation started in background.",
	})
}

// TriggerZoneRecalc triggers zone recalculation from existing data (no Strava API calls)
func (h *ActivitiesHandler) TriggerZoneRecalc(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	
	slog.Info("🔄 triggering zone recalculation from DB", "user_id", userID)
	
	// Parse user ID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}
	
	// Get user max_hr
	var maxHR int
	err = h.db.Pool.QueryRow(c.Context(), `SELECT max_hr FROM users WHERE id = $1`, uid).Scan(&maxHR)
	if err != nil || maxHR == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "max_hr not set - sync activities first"})
	}
	
	// Start recalculation in background
	go func() {
		ctx := context.Background()
		
		// Get all activities with splits data
		rows, err := h.db.Pool.Query(ctx, `
			SELECT id, splits_km
			FROM activities
			WHERE user_id = $1 
			  AND splits_km IS NOT NULL 
			  AND splits_km != 'null'::jsonb
		`, uid)
		
		if err != nil {
			slog.Error("❌ failed to query activities", "user_id", userID, "error", err)
			return
		}
		defer rows.Close()
		
		updated := 0
		for rows.Next() {
			var activityID uuid.UUID
			var splitsJSON []byte
			
			if err := rows.Scan(&activityID, &splitsJSON); err != nil {
				continue
			}
			
			// Parse splits
			var splits []models.Split
			if err := json.Unmarshal(splitsJSON, &splits); err != nil {
				continue
			}
			
			// Check if any split has HR data
			hasHR := false
			for _, split := range splits {
				if split.HR > 0 {
					hasHR = true
					break
				}
			}
			
			if !hasHR {
				continue
			}
			
			// Calculate zone distribution
			var totalTime int
			var z1Time, z2Time, z3Time, z4Time, z5Time int
			
			for _, split := range splits {
				if split.HR == 0 {
					continue
				}
				
				// Estimate time per km from pace (pace_s is seconds per km)
				time := int(split.PaceS)
				totalTime += time
				
				hrPct := (float64(split.HR) / float64(maxHR)) * 100
				
				switch {
				case hrPct < 60:
					z1Time += time
				case hrPct < 70:
					z2Time += time
				case hrPct < 80:
					z3Time += time
				case hrPct < 90:
					z4Time += time
				default:
					z5Time += time
				}
			}
			
			if totalTime > 0 {
				zoneDist := models.ZoneDistribution{
					Z1Pct: float64(z1Time) / float64(totalTime) * 100,
					Z2Pct: float64(z2Time) / float64(totalTime) * 100,
					Z3Pct: float64(z3Time) / float64(totalTime) * 100,
					Z4Pct: float64(z4Time) / float64(totalTime) * 100,
					Z5Pct: float64(z5Time) / float64(totalTime) * 100,
				}
				
				zoneJSON, _ := json.Marshal(zoneDist)
				
				_, err = h.db.Pool.Exec(ctx, `
					UPDATE activities SET zone_distribution = $1 WHERE id = $2
				`, zoneJSON, activityID)
				
				if err == nil {
					updated++
				}
			}
		}
		
		slog.Info("✅ zone recalculation complete", "user_id", userID, "activities_updated", updated)
	}()
	
	return c.JSON(fiber.Map{
		"status":  "processing",
		"message": "Zone recalculation started from existing data. Refresh in a few seconds.",
		"max_hr":  maxHR,
	})
}
