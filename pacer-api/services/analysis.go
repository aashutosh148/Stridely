package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/models"
)

type AnalysisService struct {
	db *db.Postgres
}

func NewAnalysisService(dbConn *db.Postgres) *AnalysisService {
	return &AnalysisService{db: dbConn}
}

// GetDB returns the database connection
func (s *AnalysisService) GetDB() *db.Postgres {
	return s.db
}

// ProcessNewActivity analyzes and stores a Strava activity
func (s *AnalysisService) ProcessNewActivity(ctx context.Context, user *models.User, stravaActivity *StravaDetailedActivity) error {
	// Only process running activities
	if stravaActivity.Type != "Run" && stravaActivity.SportType != "Run" && stravaActivity.SportType != "TrailRun" {
		slog.Info("skipping non-running activity", "type", stravaActivity.Type, "sport_type", stravaActivity.SportType)
		return nil
	}

	// Parse activity date
	activityDate, err := time.Parse(time.RFC3339, stravaActivity.StartDate)
	if err != nil {
		return fmt.Errorf("parse activity date: %w", err)
	}

	// Classify workout type
	workoutType := s.classifyWorkoutType(stravaActivity)

	// Calculate pace (seconds per km)
	var avgPaceS sql.NullFloat64
	if stravaActivity.Distance > 0 {
		avgPaceS.Valid = true
		avgPaceS.Float64 = float64(stravaActivity.MovingTime) / (stravaActivity.Distance / 1000.0)
	}

	// Extract HR data
	var avgHR, maxHR sql.NullInt32
	if stravaActivity.AverageHeartrate > 0 {
		avgHR.Valid = true
		avgHR.Int32 = int32(stravaActivity.AverageHeartrate)
	}
	if stravaActivity.MaxHeartrate > 0 {
		maxHR.Valid = true
		maxHR.Int32 = int32(stravaActivity.MaxHeartrate)
	}

	// Compute zone distribution from HR data
	var zoneDistribution *models.ZoneDistribution
	if stravaActivity.HasHeartrate && len(stravaActivity.SplitsMetric) > 0 && user.MaxHR.Valid {
		zoneDistribution = s.computeZoneDistribution(stravaActivity.SplitsMetric, int(user.MaxHR.Int32))
	}

	// Calculate TSS (Training Stress Score)
	var tss sql.NullFloat64
	if user.ThresholdPaceS.Valid && avgPaceS.Valid {
		tss.Valid = true
		tss.Float64 = s.calculateTSS(stravaActivity.MovingTime, avgPaceS.Float64, user.ThresholdPaceS.Float64)
	}

	// Convert splits for storage
	var splits []models.Split
	for i, split := range stravaActivity.SplitsMetric {
		splits = append(splits, models.Split{
			KM:        i + 1,
			PaceS:     float64(split.MovingTime) / (split.Distance / 1000.0),
			HR:        int(split.AverageHeartrate),
			ElevDelta: split.ElevationDifference,
		})
	}

	// Convert to JSONB for storage
	zoneDistJSON, _ := json.Marshal(zoneDistribution)
	splitsJSON, _ := json.Marshal(splits)

	// Store activity in database
	stravaID := strconv.FormatInt(stravaActivity.ID, 10)
	var gearID sql.NullString
	if stravaActivity.GearID != "" {
		gearID.Valid = true
		gearID.String = stravaActivity.GearID
	}

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO activities (
			user_id, strava_id, activity_date, workout_type,
			distance_m, duration_s, elevation_gain_m,
			avg_pace_s, avg_hr, max_hr, tss,
			zone_distribution, splits_km, gear_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, now())
		ON CONFLICT (user_id, strava_id)
		DO UPDATE SET
			activity_date = EXCLUDED.activity_date,
			workout_type = EXCLUDED.workout_type,
			distance_m = EXCLUDED.distance_m,
			duration_s = EXCLUDED.duration_s,
			elevation_gain_m = EXCLUDED.elevation_gain_m,
			avg_pace_s = EXCLUDED.avg_pace_s,
			avg_hr = EXCLUDED.avg_hr,
			max_hr = EXCLUDED.max_hr,
			tss = EXCLUDED.tss,
			zone_distribution = EXCLUDED.zone_distribution,
			splits_km = EXCLUDED.splits_km,
			gear_id = EXCLUDED.gear_id
	`,
		user.ID, stravaID, activityDate.Format("2006-01-02"), workoutType,
		int(stravaActivity.Distance), stravaActivity.MovingTime, stravaActivity.TotalElevationGain,
		avgPaceS, avgHR, maxHR, tss,
		zoneDistJSON, splitsJSON, gearID,
	)

	if err != nil {
		return fmt.Errorf("store activity: %w", err)
	}

	slog.Info("activity processed",
		"user_id", user.ID,
		"strava_id", stravaID,
		"workout_type", workoutType,
		"distance_km", stravaActivity.Distance/1000.0,
		"tss", tss.Float64,
	)

	return nil
}

// classifyWorkoutType determines the workout type based on Strava data
func (s *AnalysisService) classifyWorkoutType(activity *StravaDetailedActivity) models.WorkoutType {
	// Use Strava's workout_type if available
	// 0 = default, 1 = race, 2 = long run, 3 = workout (intervals/tempo)
	switch activity.WorkoutType {
	case 1:
		return models.WorkoutTypeRace
	case 2:
		return models.WorkoutTypeLong
	case 3:
		// Could be tempo or intervals - use pace variability to decide
		if s.isPaceVariableWorkout(activity) {
			return models.WorkoutTypeInterval
		}
		return models.WorkoutTypeTempo
	}

	// Heuristic classification
	distanceKM := activity.Distance / 1000.0
	durationMin := float64(activity.MovingTime) / 60.0

	// Long run: > 90 minutes or > 20km
	if durationMin > 90 || distanceKM > 20 {
		return models.WorkoutTypeLong
	}

	// Recovery: < 30 minutes and slow pace
	if durationMin < 30 {
		return models.WorkoutTypeRecovery
	}

	// Check pace variability for intervals
	if s.isPaceVariableWorkout(activity) {
		return models.WorkoutTypeInterval
	}

	// Check if tempo pace (based on avg speed if we had threshold)
	// For now, default to easy
	return models.WorkoutTypeEasy
}

// isPaceVariableWorkout checks if pace varies significantly (intervals)
func (s *AnalysisService) isPaceVariableWorkout(activity *StravaDetailedActivity) bool {
	if len(activity.SplitsMetric) < 3 {
		return false
	}

	var paces []float64
	for _, split := range activity.SplitsMetric {
		if split.MovingTime > 0 && split.Distance > 0 {
			pace := float64(split.MovingTime) / (split.Distance / 1000.0)
			paces = append(paces, pace)
		}
	}

	if len(paces) < 3 {
		return false
	}

	// Calculate coefficient of variation
	mean := 0.0
	for _, pace := range paces {
		mean += pace
	}
	mean /= float64(len(paces))

	variance := 0.0
	for _, pace := range paces {
		variance += math.Pow(pace-mean, 2)
	}
	variance /= float64(len(paces))
	stdDev := math.Sqrt(variance)

	coefficientOfVariation := stdDev / mean

	// If pace varies > 10%, likely intervals
	return coefficientOfVariation > 0.10
}

// computeZoneDistribution calculates time spent in each HR zone
// Z1: < 60% max, Z2: 60-70%, Z3: 70-80%, Z4: 80-90%, Z5: > 90%
func (s *AnalysisService) computeZoneDistribution(splits []StravaSplit, maxHR int) *models.ZoneDistribution {
	if maxHR == 0 {
		return nil
	}

	var totalTime int
	var z1Time, z2Time, z3Time, z4Time, z5Time int

	for _, split := range splits {
		if split.AverageHeartrate == 0 {
			continue
		}

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

	if totalTime == 0 {
		return nil
	}

	return &models.ZoneDistribution{
		Z1Pct: float64(z1Time) / float64(totalTime) * 100,
		Z2Pct: float64(z2Time) / float64(totalTime) * 100,
		Z3Pct: float64(z3Time) / float64(totalTime) * 100,
		Z4Pct: float64(z4Time) / float64(totalTime) * 100,
		Z5Pct: float64(z5Time) / float64(totalTime) * 100,
	}
}

// calculateTSS computes Training Stress Score
// TSS = (duration_seconds * intensity_factor^2 * 100) / 3600
// Intensity Factor (IF) = pace_threshold / pace_actual (for running)
func (s *AnalysisService) calculateTSS(durationS int, avgPaceS, thresholdPaceS float64) float64 {
	if thresholdPaceS == 0 || avgPaceS == 0 {
		return 0
	}

	// Intensity Factor: faster pace = higher IF
	// For running: IF = threshold_pace / actual_pace
	intensityFactor := thresholdPaceS / avgPaceS

	// Cap IF at 1.2 (very hard effort)
	if intensityFactor > 1.2 {
		intensityFactor = 1.2
	}

	// TSS formula
	tss := (float64(durationS) * math.Pow(intensityFactor, 2) * 100) / 3600

	return math.Round(tss*10) / 10 // Round to 1 decimal
}

func CalculateTSS(durationS int, avgPaceS, thresholdPaceS float64) float64 {
	service := &AnalysisService{}
	return service.calculateTSS(durationS, avgPaceS, thresholdPaceS)
}

func CalculateCTLFromTSS(tssHistory []float64) float64 {
	ctl := 0.0
	decayConstant := 1 - math.Exp(-1.0/42.0)
	for _, tss := range tssHistory {
		ctl += (tss - ctl) * decayConstant
	}
	return ctl
}

func CalculateATLFromTSS(tssHistory []float64) float64 {
	atl := 0.0
	decayConstant := 1 - math.Exp(-1.0/7.0)
	for _, tss := range tssHistory {
		atl += (tss - atl) * decayConstant
	}
	return atl
}

func CalculateTSB(ctl, atl float64) float64 {
	return ctl - atl
}

// SyncStravaHistory syncs last N days of activities from Strava
func (s *AnalysisService) SyncStravaHistory(ctx context.Context, userID uuid.UUID, strava *StravaClient, days int) error {
	// Calculate timestamp for N days ago
	afterTime := time.Now().AddDate(0, 0, -days)
	afterTimestamp := afterTime.Unix()

	slog.Info("syncing strava history", "user_id", userID, "days", days, "after", afterTime.Format("2006-01-02"))

	// Fetch user profile
	var user models.User
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, email, runner_tier, subscription_tier, threshold_pace_s, max_hr
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Email, &user.RunnerTier, &user.SubscriptionTier,
		&user.ThresholdPaceS, &user.MaxHR,
	)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// Fetch activities in batches
	perPage := 200 // Strava max
	totalProcessed := 0

	for {
		activities, err := strava.GetActivities(ctx, userID.String(), afterTimestamp, perPage)
		if err != nil {
			return fmt.Errorf("get activities: %w", err)
		}

		if len(activities) == 0 {
			break
		}

		// Process each activity
		for _, activity := range activities {
			// Get detailed activity data
			detailed, err := strava.GetActivityDetail(ctx, userID.String(), activity.ID)
			if err != nil {
				slog.Warn("failed to get activity detail", "activity_id", activity.ID, "error", err)
				continue
			}

			// Process and store
			if err := s.ProcessNewActivity(ctx, &user, detailed); err != nil {
				slog.Warn("failed to process activity", "activity_id", activity.ID, "error", err)
				continue
			}

			totalProcessed++
		}

		// If we got fewer than requested, we're done
		if len(activities) < perPage {
			break
		}

		// Rate limiting: pause between batches
		time.Sleep(1 * time.Second)
	}

	slog.Info("strava history sync complete", "user_id", userID, "activities_processed", totalProcessed)
	
	// After sync, infer max_hr if not set
	go s.inferAndSetMaxHR(context.Background(), userID)
	
	return nil
}

// inferAndSetMaxHR infers max HR from activities and sets it on user profile if not already set
func (s *AnalysisService) inferAndSetMaxHR(ctx context.Context, userID uuid.UUID) {
	// Check if user already has max_hr set
	var currentMaxHR sql.NullInt32
	err := s.db.Pool.QueryRow(ctx, `SELECT max_hr FROM users WHERE id = $1`, userID).Scan(&currentMaxHR)
	if err != nil {
		slog.Warn("failed to check user max_hr", "user_id", userID, "error", err)
		return
	}
	
	// If already set, don't override
	if currentMaxHR.Valid && currentMaxHR.Int32 > 0 {
		slog.Info("user max_hr already set", "user_id", userID, "max_hr", currentMaxHR.Int32)
		return
	}
	
	// Get max HR from activities
	var inferredMaxHR sql.NullInt32
	err = s.db.Pool.QueryRow(ctx, `
		SELECT MAX(max_hr) FROM activities 
		WHERE user_id = $1 AND max_hr IS NOT NULL AND max_hr > 0
	`, userID).Scan(&inferredMaxHR)
	
	if err != nil || !inferredMaxHR.Valid || inferredMaxHR.Int32 == 0 {
		slog.Info("no max_hr data found in activities", "user_id", userID)
		return
	}
	
	// Update user with inferred max_hr
	_, err = s.db.Pool.Exec(ctx, `UPDATE users SET max_hr = $1 WHERE id = $2`, inferredMaxHR.Int32, userID)
	if err != nil {
		slog.Error("failed to set inferred max_hr", "user_id", userID, "error", err)
		return
	}
	
	slog.Info("✅ inferred and set max_hr from activities", "user_id", userID, "max_hr", inferredMaxHR.Int32)
	
	// Now recalculate zones for all activities
	s.recalculateZonesFromDB(ctx, userID, int(inferredMaxHR.Int32))
}

// recalculateZonesFromDB recalculates HR zones for all activities using existing splits data
func (s *AnalysisService) recalculateZonesFromDB(ctx context.Context, userID uuid.UUID, maxHR int) {
	slog.Info("🔄 recalculating HR zones from existing data", "user_id", userID, "max_hr", maxHR)
	
	// Get all activities with splits data
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, splits_km
		FROM activities
		WHERE user_id = $1 
		  AND splits_km IS NOT NULL 
		  AND splits_km != 'null'::jsonb
	`, userID)
	
	if err != nil {
		slog.Error("failed to query activities for zone recalc", "user_id", userID, "error", err)
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
			
			_, err = s.db.Pool.Exec(ctx, `
				UPDATE activities SET zone_distribution = $1 WHERE id = $2
			`, zoneJSON, activityID)
			
			if err == nil {
				updated++
			}
		}
	}
	
	slog.Info("✅ zone recalculation complete", "user_id", userID, "activities_updated", updated)
}

// GetZoneDistributionThisWeek aggregates zone distribution from activities in the last 7 days
func (s *AnalysisService) GetZoneDistributionThisWeek(ctx context.Context, userID uuid.UUID) (map[string]float64, error) {
	// Get all activities from the last 7 days with zone data
	rows, err := s.db.Pool.Query(ctx, `
		SELECT zone_distribution
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= CURRENT_DATE - INTERVAL '7 days'
		  AND zone_distribution IS NOT NULL
		  AND zone_distribution != 'null'::jsonb
	`, userID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var totalZ1, totalZ2, totalZ3, totalZ4, totalZ5 float64
	count := 0
	
	for rows.Next() {
		var zoneJSON []byte
		if err := rows.Scan(&zoneJSON); err != nil {
			continue
		}
		
		var zone models.ZoneDistribution
		if err := json.Unmarshal(zoneJSON, &zone); err != nil {
			continue
		}
		
		totalZ1 += zone.Z1Pct
		totalZ2 += zone.Z2Pct
		totalZ3 += zone.Z3Pct
		totalZ4 += zone.Z4Pct
		totalZ5 += zone.Z5Pct
		count++
	}
	
	if count == 0 {
		return map[string]float64{
			"z1_pct": 0,
			"z2_pct": 0,
			"z3_pct": 0,
			"z4_pct": 0,
			"z5_pct": 0,
		}, nil
	}
	
	// Return average distribution across the week
	return map[string]float64{
		"z1_pct": totalZ1 / float64(count),
		"z2_pct": totalZ2 / float64(count),
		"z3_pct": totalZ3 / float64(count),
		"z4_pct": totalZ4 / float64(count),
		"z5_pct": totalZ5 / float64(count),
	}, nil
}

// ComputeAndStoreCTLATLTSB calculates current fitness metrics and caches them in Redis
func (s *AnalysisService) ComputeAndStoreCTLATLTSB(ctx context.Context, userID uuid.UUID) (map[string]float64, error) {
	// Calculate CTL (42-day exponential weighted average)
	ctl := s.computeCTL(ctx, userID, 42)

	// Calculate ATL (7-day exponential weighted average)
	atl := s.computeATL(ctx, userID, 7)

	// Calculate TSB (Training Stress Balance)
	tsb := ctl - atl

	metrics := map[string]float64{
		"ctl": math.Round(ctl*10) / 10,
		"atl": math.Round(atl*10) / 10,
		"tsb": math.Round(tsb*10) / 10,
	}

	// Cache in Redis for 6 hours
	// TODO: Implement Redis caching when Redis integration is complete
	// For now, just return the computed values

	slog.Info("computed fitness metrics",
		"user_id", userID,
		"ctl", metrics["ctl"],
		"atl", metrics["atl"],
		"tsb", metrics["tsb"],
	)

	return metrics, nil
}

// computeCTL calculates Chronic Training Load (42-day EWA)
func (s *AnalysisService) computeCTL(ctx context.Context, userID uuid.UUID, days int) float64 {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	rows, err := s.db.Pool.Query(ctx, `
		SELECT activity_date, COALESCE(tss, 0) as tss
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= $2
		  AND tss IS NOT NULL
		ORDER BY activity_date ASC
	`, userID, cutoffDate.Format("2006-01-02"))
	if err != nil {
		slog.Warn("failed to fetch activities for CTL", "error", err)
		return 0
	}
	defer rows.Close()

	ctl := 0.0
	decayConstant := 1 - math.Exp(-1.0/42.0)

	for rows.Next() {
		var date string
		var tss float64
		if err := rows.Scan(&date, &tss); err != nil {
			continue
		}

		// Apply exponential weighted average formula
		ctl += (tss - ctl) * decayConstant
	}

	return ctl
}

// computeATL calculates Acute Training Load (7-day EWA)
func (s *AnalysisService) computeATL(ctx context.Context, userID uuid.UUID, days int) float64 {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	rows, err := s.db.Pool.Query(ctx, `
		SELECT activity_date, COALESCE(tss, 0) as tss
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= $2
		  AND tss IS NOT NULL
		ORDER BY activity_date ASC
	`, userID, cutoffDate.Format("2006-01-02"))
	if err != nil {
		slog.Warn("failed to fetch activities for ATL", "error", err)
		return 0
	}
	defer rows.Close()

	atl := 0.0
	decayConstant := 1 - math.Exp(-1.0/7.0)

	for rows.Next() {
		var date string
		var tss float64
		if err := rows.Scan(&date, &tss); err != nil {
			continue
		}

		// Apply exponential weighted average formula
		atl += (tss - atl) * decayConstant
	}

	return atl
}

// EstimateThreshold estimates lactate threshold pace and updates user profile
func (s *AnalysisService) EstimateThreshold(ctx context.Context, userID uuid.UUID) error {
	// Method 1: Race-based estimation (10K or half marathon)
	racePace := s.estimateThresholdFromRaces(ctx, userID)

	// Method 2: Tempo run estimation
	tempoPace := s.estimateThresholdFromTempoRuns(ctx, userID)

	// Use whichever method has data, preferring race-based
	var estimatedPace float64
	if racePace > 0 {
		estimatedPace = racePace
	} else if tempoPace > 0 {
		estimatedPace = tempoPace
	} else {
		return fmt.Errorf("insufficient data to estimate threshold")
	}

	// Update user's threshold pace
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE users
		SET threshold_pace_s = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, estimatedPace, userID)
	if err != nil {
		return fmt.Errorf("update threshold pace: %w", err)
	}

	slog.Info("updated threshold pace",
		"user_id", userID,
		"threshold_pace_s", estimatedPace,
		"source", map[bool]string{true: "race", false: "tempo"}[racePace > 0],
	)

	return nil
}

// estimateThresholdFromRaces finds threshold from recent race performances
func (s *AnalysisService) estimateThresholdFromRaces(ctx context.Context, userID uuid.UUID) float64 {
	var pace sql.NullFloat64
	err := s.db.Pool.QueryRow(ctx, `
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
		return 0
	}

	// Threshold pace ≈ 10K race pace + 5-10 seconds/km
	return pace.Float64 + 7
}

// estimateThresholdFromTempoRuns analyzes tempo efforts
func (s *AnalysisService) estimateThresholdFromTempoRuns(ctx context.Context, userID uuid.UUID) float64 {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT avg_pace_s
		FROM activities
		WHERE user_id = $1
		  AND workout_type = 'tempo'
		  AND duration_s BETWEEN 1200 AND 3600
		  AND activity_date >= NOW() - INTERVAL '60 days'
		ORDER BY activity_date DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var paces []float64
	for rows.Next() {
		var pace sql.NullFloat64
		if err := rows.Scan(&pace); err == nil && pace.Valid {
			paces = append(paces, pace.Float64)
		}
	}

	if len(paces) == 0 {
		return 0
	}

	// Average tempo pace = threshold pace
	var sum float64
	for _, p := range paces {
		sum += p
	}
	return sum / float64(len(paces))
}

// GetLatestFitnessMetrics returns the most recent CTL, ATL, TSB values for a user
func (s *AnalysisService) GetLatestFitnessMetrics(ctx context.Context, userID uuid.UUID) (ctl, atl, tsb float64, err error) {
	query := `
		SELECT ctl, atl, tsb
		FROM fitness_snapshots
		WHERE user_id = $1
		ORDER BY snapshot_date DESC
		LIMIT 1
	`
	err = s.db.Pool.QueryRow(ctx, query, userID).Scan(&ctl, &atl, &tsb)
	if err != nil {
		// If no snapshot exists, compute on-the-fly
		if err == sql.ErrNoRows {
			slog.Info("no fitness snapshot found, computing on-the-fly", "user_id", userID)
			ctl = s.computeCTL(ctx, userID, 42)
			atl = s.computeATL(ctx, userID, 7)
			tsb = ctl - atl
			return ctl, atl, tsb, nil
		}
		return 0, 0, 0, err
	}
	return ctl, atl, tsb, nil
}
