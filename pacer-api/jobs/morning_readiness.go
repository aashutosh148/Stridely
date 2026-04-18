package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type HRVData struct {
	LastNightAvg float64
	WeeklyAvg    float64
	Status       string
}

type SleepData struct {
	TotalSleepS int
	DeepSleepS  int
	RemSleepS   int
	SleepScore  int
}

type readinessContext struct {
	userID       uuid.UUID
	readinessTSB float64
}

func RunMorningReadiness(deps *Dependencies) error {
	ctx := context.Background()
	users, err := listActiveUsers(ctx, deps)
	if err != nil {
		return err
	}

	sem := make(chan struct{}, 50)
	var wg sync.WaitGroup

	for _, userID := range users {
		wg.Add(1)
		sem <- struct{}{}

		go func(uid uuid.UUID) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := processReadiness(ctx, deps, uid); err != nil {
				slog.Error("process readiness failed", "user_id", uid, "error", err)
			}
		}(userID)
	}

	wg.Wait()
	slog.Info("morning readiness job complete", "users", len(users))
	return nil
}

func processReadiness(ctx context.Context, deps *Dependencies, userID uuid.UUID) error {
	today := time.Now().UTC().Format("2006-01-02")

	hrv, _ := deps.Garmin.GetHRVData(ctx, userID, today)
	sleep, _ := deps.Garmin.GetSleepData(ctx, userID, today)

	rc, err := getReadinessContext(ctx, deps, userID)
	if err != nil {
		return err
	}

	score, level, note := calculateReadiness(hrv, sleep, rc.readinessTSB)

	workoutID, err := getTodaysWorkoutID(ctx, deps, userID)
	if err != nil {
		return err
	}

	adjusted := false
	if (level == "amber" || level == "red") && deps.Planning != nil {
		_ = deps.Planning.AdjustWeeklyPlan(ctx, userID, score, 0)
		if workoutID != uuid.Nil {
			adjusted = true
			adjustment := fmt.Sprintf("Auto-adjusted by morning readiness job (%s, score=%d)", level, score)
			_, _ = deps.DB.Pool.Exec(ctx, `
        UPDATE workouts
        SET status = 'modified',
            description = CONCAT(COALESCE(description, ''), CASE WHEN COALESCE(description, '') = '' THEN '' ELSE E'\\n\\n' END, $1)
        WHERE id = $2
      `, adjustment, workoutID)
		}
	}

	if err := upsertDailyHealth(ctx, deps, userID, today, hrv, sleep, score, level, note); err != nil {
		return err
	}

	if deps.Redis != nil {
		if _, err := deps.Redis.RebuildWorkingMemory(ctx, userID); err != nil {
			slog.Warn("working memory rebuild failed", "user_id", userID, "error", err)
		}
	}

	payload := map[string]any{
		"date":            today,
		"score":           score,
		"level":           level,
		"note":            note,
		"adjustedWorkout": adjusted,
	}
	_ = deps.Notifier.Push(ctx, userID, "readiness.updated", payload)

	return nil
}

func calculateReadiness(hrv *HRVData, sleep *SleepData, tsb float64) (int, string, string) {
	hrvScore := hrvSubScore(hrv)
	sleepScore := sleepSubScore(sleep)
	tsbScore := tsbSubScore(tsb)

	weighted := (hrvScore * 0.40) + (sleepScore * 0.35) + (tsbScore * 0.25)
	intScore := int(math.Round(weighted))
	if intScore < 1 {
		intScore = 1
	}
	if intScore > 10 {
		intScore = 10
	}

	level := "green"
	if intScore <= 4 {
		level = "red"
	} else if intScore <= 6 {
		level = "amber"
	}

	note := buildReadinessNote(hrv, sleep, tsb, intScore)
	return intScore, level, note
}

func hrvSubScore(hrv *HRVData) float64 {
	if hrv == nil {
		return 6.0
	}
	score := 8.5
	switch strings.ToUpper(hrv.Status) {
	case "POOR":
		score = 3.5
	case "UNBALANCED":
		score = 5.8
	case "BALANCED":
		score = 8.8
	}

	if hrv.WeeklyAvg > 0 && hrv.LastNightAvg > 0 {
		ratio := hrv.LastNightAvg / hrv.WeeklyAvg
		if ratio < 0.85 {
			score -= 1.0
		} else if ratio > 1.10 {
			score += 0.4
		}
	}

	return clamp(score, 1, 10)
}

func sleepSubScore(sleep *SleepData) float64 {
	if sleep == nil {
		return 6.0
	}

	hours := float64(sleep.TotalSleepS) / 3600.0
	score := 8.8
	switch {
	case hours < 5:
		score = 3.0
	case hours < 6:
		score = 4.5
	case hours < 6.5:
		score = 6.0
	case hours < 7:
		score = 7.0
	case hours < 8:
		score = 8.2
	default:
		score = 9.0
	}

	if sleep.SleepScore > 0 {
		normalized := float64(sleep.SleepScore) / 10.0
		score = (score * 0.7) + (normalized * 0.3)
	}

	return clamp(score, 1, 10)
}

func tsbSubScore(tsb float64) float64 {
	switch {
	case tsb <= -30:
		return 2.0
	case tsb <= -20:
		return 3.8
	case tsb <= -15:
		return 5.0
	case tsb <= -10:
		return 6.3
	case tsb <= 5:
		return 8.0
	case tsb <= 15:
		return 8.8
	default:
		return 9.2
	}
}

func buildReadinessNote(hrv *HRVData, sleep *SleepData, tsb float64, score int) string {
	hrvStatus := "unknown"
	if hrv != nil && hrv.Status != "" {
		hrvStatus = strings.ToLower(hrv.Status)
	}
	sleepHours := 0.0
	if sleep != nil {
		sleepHours = float64(sleep.TotalSleepS) / 3600.0
	}
	return fmt.Sprintf("HRV %s, sleep %.1fh, TSB %.1f. Readiness %d/10.", hrvStatus, sleepHours, tsb, score)
}

func getReadinessContext(ctx context.Context, deps *Dependencies, userID uuid.UUID) (*readinessContext, error) {
	out := &readinessContext{userID: userID, readinessTSB: 0}
	err := deps.DB.Pool.QueryRow(ctx, `
    SELECT COALESCE(tsb, 0)
    FROM fitness_snapshots
    WHERE user_id = $1
    ORDER BY snapshot_date DESC
    LIMIT 1
  `, userID).Scan(&out.readinessTSB)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return out, nil
}

func getTodaysWorkoutID(ctx context.Context, deps *Dependencies, userID uuid.UUID) (uuid.UUID, error) {
	var workoutID uuid.UUID
	err := deps.DB.Pool.QueryRow(ctx, `
    SELECT id
    FROM workouts
    WHERE user_id = $1
      AND scheduled_date = CURRENT_DATE
    ORDER BY created_at DESC
    LIMIT 1
  `, userID).Scan(&workoutID)
	if err == sql.ErrNoRows {
		return uuid.Nil, nil
	}
	return workoutID, err
}

func upsertDailyHealth(
	ctx context.Context,
	deps *Dependencies,
	userID uuid.UUID,
	healthDate string,
	hrv *HRVData,
	sleep *SleepData,
	score int,
	level string,
	note string,
) error {
	var hrvLastNightAvg any
	var hrvBaselineLow any
	var hrvBaselineHigh any
	var hrvStatus any
	if hrv != nil {
		hrvLastNightAvg = hrv.LastNightAvg
		hrvBaselineLow = hrv.WeeklyAvg * 0.9
		hrvBaselineHigh = hrv.WeeklyAvg * 1.1
		hrvStatus = strings.ToUpper(hrv.Status)
	}

	var sleepTotal any
	var sleepDeep any
	var sleepREM any
	var sleepScore any
	if sleep != nil {
		sleepTotal = sleep.TotalSleepS
		sleepDeep = sleep.DeepSleepS
		sleepREM = sleep.RemSleepS
		sleepScore = sleep.SleepScore
	}

	var readinessScore any
	var readinessLevel any
	var readinessNote any
	if score > 0 {
		readinessScore = score
	}
	if level != "" {
		readinessLevel = level
	}
	if note != "" {
		readinessNote = note
	}

	_, err := deps.DB.Pool.Exec(ctx, `
    INSERT INTO daily_health (
      user_id, health_date,
      hrv_last_night_avg, hrv_baseline_low, hrv_baseline_high, hrv_status,
      sleep_total_s, sleep_deep_s, sleep_rem_s, sleep_score,
      readiness_score, readiness_level, readiness_note
    ) VALUES (
      $1, $2,
      $3, $4, $5, $6,
      $7, $8, $9, $10,
      $11, $12, $13
    )
    ON CONFLICT (user_id, health_date)
    DO UPDATE SET
      hrv_last_night_avg = COALESCE(EXCLUDED.hrv_last_night_avg, daily_health.hrv_last_night_avg),
      hrv_baseline_low = COALESCE(EXCLUDED.hrv_baseline_low, daily_health.hrv_baseline_low),
      hrv_baseline_high = COALESCE(EXCLUDED.hrv_baseline_high, daily_health.hrv_baseline_high),
      hrv_status = COALESCE(EXCLUDED.hrv_status, daily_health.hrv_status),
      sleep_total_s = COALESCE(EXCLUDED.sleep_total_s, daily_health.sleep_total_s),
      sleep_deep_s = COALESCE(EXCLUDED.sleep_deep_s, daily_health.sleep_deep_s),
      sleep_rem_s = COALESCE(EXCLUDED.sleep_rem_s, daily_health.sleep_rem_s),
      sleep_score = COALESCE(EXCLUDED.sleep_score, daily_health.sleep_score),
      readiness_score = COALESCE(EXCLUDED.readiness_score, daily_health.readiness_score),
      readiness_level = COALESCE(EXCLUDED.readiness_level, daily_health.readiness_level),
      readiness_note = COALESCE(EXCLUDED.readiness_note, daily_health.readiness_note)
  `,
		userID, healthDate,
		hrvLastNightAvg, hrvBaselineLow, hrvBaselineHigh, hrvStatus,
		sleepTotal, sleepDeep, sleepREM, sleepScore,
		readinessScore, readinessLevel, readinessNote,
	)
	return err
}

func listActiveUsers(ctx context.Context, deps *Dependencies) ([]uuid.UUID, error) {
	rows, err := deps.DB.Pool.Query(ctx, `
    SELECT id
    FROM users
    WHERE strava_athlete_id IS NOT NULL
       OR garmin_user_id IS NOT NULL
  `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]uuid.UUID, 0)
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		users = append(users, uid)
	}
	return users, nil
}

func clamp(value, minV, maxV float64) float64 {
	if value < minV {
		return minV
	}
	if value > maxV {
		return maxV
	}
	return value
}
