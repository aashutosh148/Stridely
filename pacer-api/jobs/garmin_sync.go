package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type GarminDailyData struct {
	Date          string
	HRV           *HRVData
	Sleep         *SleepData
	NewActivities []GarminActivityData
}

type GarminActivityData struct {
	ExternalID      string
	ActivityDate    string
	DistanceM       int
	DurationS       int
	AvgPaceS        float64
	AvgHR           int
	MaxHR           int
	TrainingLoad    float64
	CadenceSPM      int
	GroundContactMS int
	VerticalOscCM   float64
	LRBalancePct    float64
}

func RunGarminSync(deps *Dependencies) error {
	ctx := context.Background()
	users, err := listGarminUsers(ctx, deps)
	if err != nil {
		return err
	}

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")

	for _, uid := range users {
		for _, date := range []string{yesterday, today} {
			daily, err := deps.Garmin.SyncDailyData(ctx, uid, date)
			if err != nil {
				slog.Warn("garmin sync failed", "user_id", uid, "date", date, "error", err)
				continue
			}
			if daily == nil {
				continue
			}

			if err := upsertDailyHealth(ctx, deps, uid, daily.Date, daily.HRV, daily.Sleep, 0, "", ""); err != nil {
				slog.Warn("daily_health upsert failed during garmin sync", "user_id", uid, "date", date, "error", err)
			}

			if err := enrichActivitiesFromGarmin(ctx, deps, uid, daily.NewActivities); err != nil {
				slog.Warn("garmin activity enrichment failed", "user_id", uid, "date", date, "error", err)
			}
		}
	}

	slog.Info("garmin sync job complete", "users", len(users))
	return nil
}

func listGarminUsers(ctx context.Context, deps *Dependencies) ([]uuid.UUID, error) {
	rows, err := deps.DB.Pool.Query(ctx, `SELECT id FROM users WHERE garmin_user_id IS NOT NULL`)
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

func enrichActivitiesFromGarmin(ctx context.Context, deps *Dependencies, userID uuid.UUID, activities []GarminActivityData) error {
	for _, a := range activities {
		_, err := deps.DB.Pool.Exec(ctx, `
      INSERT INTO activities (
        user_id, strava_id, activity_date, workout_type,
        distance_m, duration_s, avg_pace_s, avg_hr, max_hr,
        garmin_training_load, garmin_cadence_spm, garmin_gct_ms,
        garmin_vert_osc_cm, garmin_lr_balance_pct, created_at
      ) VALUES (
        $1, $2, $3, 'unstructured',
        $4, $5, $6, NULLIF($7, 0), NULLIF($8, 0),
        NULLIF($9, 0), NULLIF($10, 0), NULLIF($11, 0),
        NULLIF($12, 0), NULLIF($13, 0), NOW()
      )
      ON CONFLICT (user_id, strava_id)
      DO UPDATE SET
        avg_hr = COALESCE(NULLIF(EXCLUDED.avg_hr, 0), activities.avg_hr),
        max_hr = COALESCE(NULLIF(EXCLUDED.max_hr, 0), activities.max_hr),
        garmin_training_load = COALESCE(NULLIF(EXCLUDED.garmin_training_load, 0), activities.garmin_training_load),
        garmin_cadence_spm = COALESCE(NULLIF(EXCLUDED.garmin_cadence_spm, 0), activities.garmin_cadence_spm),
        garmin_gct_ms = COALESCE(NULLIF(EXCLUDED.garmin_gct_ms, 0), activities.garmin_gct_ms),
        garmin_vert_osc_cm = COALESCE(NULLIF(EXCLUDED.garmin_vert_osc_cm, 0), activities.garmin_vert_osc_cm),
        garmin_lr_balance_pct = COALESCE(NULLIF(EXCLUDED.garmin_lr_balance_pct, 0), activities.garmin_lr_balance_pct)
    `,
			userID,
			"garmin:"+a.ExternalID,
			a.ActivityDate,
			a.DistanceM,
			a.DurationS,
			a.AvgPaceS,
			a.AvgHR,
			a.MaxHR,
			a.TrainingLoad,
			a.CadenceSPM,
			a.GroundContactMS,
			a.VerticalOscCM,
			a.LRBalancePct,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
