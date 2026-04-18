package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Redis wraps the go-redis client
type Redis struct {
	Client *redis.Client
	Pool   *pgxpool.Pool // For RebuildWorkingMemory
}

// WorkingMemory represents the current state stored in Redis
// LLD Section 8.1 - Fast-access memory for agent context
type WorkingMemory struct {
	UserID             uuid.UUID              `json:"user_id"`
	CTL                float64                `json:"ctl"`
	ATL                float64                `json:"atl"`
	TSB                float64                `json:"tsb"`
	ReadinessScore     float64                `json:"readiness_score"`
	TodayWorkout       *TodayWorkout          `json:"today_workout,omitempty"`
	LastActivity       *LastActivity          `json:"last_activity,omitempty"`
	WeekProgress       *WeekProgress          `json:"week_progress"`
	ActiveFlags        []string               `json:"active_flags"`
	LastRebuilt        time.Time              `json:"last_rebuilt"`
}

type TodayWorkout struct {
	ID           uuid.UUID `json:"id"`
	WorkoutType  string    `json:"workout_type"`
	DistanceKM   float64   `json:"distance_km"`
	Description  string    `json:"description"`
	PurposeText  string    `json:"purpose"`
}

type LastActivity struct {
	Date          time.Time `json:"date"`
	Type          string    `json:"type"`
	DistanceKM    float64   `json:"distance_km"`
	DurationS     int       `json:"duration_s"`
	AvgPaceS      int       `json:"avg_pace_s"`
	TSS           float64   `json:"tss"`
}

type WeekProgress struct {
	PlannedKM    float64 `json:"planned_km"`
	CompletedKM  float64 `json:"completed_km"`
	CompletedRuns int    `json:"completed_runs"`
	MissedRuns   int     `json:"missed_runs"`
}

// NewRedis creates a new Redis client connection
func NewRedis(ctx context.Context, redisURL string, pool *pgxpool.Pool) (*Redis, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to ping Redis: %w", err)
	}

	slog.Info("redis connection established", "addr", opts.Addr)

	return &Redis{Client: client, Pool: pool}, nil
}

// SetWorkingMemory stores working memory with 25-hour TTL
func (r *Redis) SetWorkingMemory(ctx context.Context, wm *WorkingMemory) error {
	key := fmt.Sprintf("wm:%s", wm.UserID)
	
	data, err := json.Marshal(wm)
	if err != nil {
		return fmt.Errorf("marshal working memory: %w", err)
	}
	
	ttl := 25 * time.Hour
	if err := r.Client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set working memory: %w", err)
	}
	
	return nil
}

// GetWorkingMemory retrieves working memory from Redis
func (r *Redis) GetWorkingMemory(ctx context.Context, userID uuid.UUID) (*WorkingMemory, error) {
	key := fmt.Sprintf("wm:%s", userID)
	
	data, err := r.Client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("get working memory: %w", err)
	}
	
	var wm WorkingMemory
	if err := json.Unmarshal(data, &wm); err != nil {
		return nil, fmt.Errorf("unmarshal working memory: %w", err)
	}
	
	return &wm, nil
}

// RebuildWorkingMemory fetches from Postgres and rebuilds Redis cache
// LLD Section 8.1 - Called by morning readiness cron or on cache miss
func (r *Redis) RebuildWorkingMemory(ctx context.Context, userID uuid.UUID) (*WorkingMemory, error) {
	wm := &WorkingMemory{
		UserID:      userID,
		ActiveFlags: []string{},
		LastRebuilt: time.Now(),
	}
	
	// 1. Fetch latest fitness metrics (CTL, ATL, TSB)
	err := r.Pool.QueryRow(ctx, `
		SELECT ctl, atl, tsb, readiness_score
		FROM fitness_snapshots
		WHERE user_id = $1
		ORDER BY snapshot_date DESC
		LIMIT 1
	`, userID).Scan(&wm.CTL, &wm.ATL, &wm.TSB, &wm.ReadinessScore)
	
	if err != nil && err.Error() != "no rows in result set" {
		return nil, fmt.Errorf("fetch fitness metrics: %w", err)
	}
	
	// 2. Fetch today's workout
	today := time.Now().Format("2006-01-02")
	var todayWorkout TodayWorkout
	err = r.Pool.QueryRow(ctx, `
		SELECT id, workout_type, distance_km, description, purpose
		FROM workouts
		WHERE user_id = $1
		  AND scheduled_date = $2
		  AND status != 'cancelled'
		ORDER BY scheduled_date
		LIMIT 1
	`, userID, today).Scan(
		&todayWorkout.ID,
		&todayWorkout.WorkoutType,
		&todayWorkout.DistanceKM,
		&todayWorkout.Description,
		&todayWorkout.PurposeText,
	)
	
	if err == nil {
		wm.TodayWorkout = &todayWorkout
	} else if err.Error() != "no rows in result set" {
		return nil, fmt.Errorf("fetch today's workout: %w", err)
	}
	
	// 3. Fetch last activity
	var lastActivity LastActivity
	err = r.Pool.QueryRow(ctx, `
		SELECT activity_date, activity_type, distance_km, duration_s, avg_pace_s, tss
		FROM activities
		WHERE user_id = $1
		ORDER BY activity_date DESC
		LIMIT 1
	`, userID).Scan(
		&lastActivity.Date,
		&lastActivity.Type,
		&lastActivity.DistanceKM,
		&lastActivity.DurationS,
		&lastActivity.AvgPaceS,
		&lastActivity.TSS,
	)
	
	if err == nil {
		wm.LastActivity = &lastActivity
	} else if err.Error() != "no rows in result set" {
		return nil, fmt.Errorf("fetch last activity: %w", err)
	}
	
	// 4. Calculate week progress (current week: Monday - Sunday)
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday())+1) // Monday
	if now.Weekday() == time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -7)
	}
	weekStartStr := weekStart.Format("2006-01-02")
	
	weekProgress := WeekProgress{}
	
	// Planned KM
	err = r.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(distance_km), 0)
		FROM workouts
		WHERE user_id = $1
		  AND scheduled_date >= $2
		  AND scheduled_date < $2::date + interval '7 days'
		  AND status != 'cancelled'
	`, userID, weekStartStr).Scan(&weekProgress.PlannedKM)
	
	if err != nil {
		return nil, fmt.Errorf("fetch planned km: %w", err)
	}
	
	// Completed activities
	err = r.Pool.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(distance_km), 0),
		  COUNT(*)
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= $2
		  AND activity_date < $2::date + interval '7 days'
	`, userID, weekStartStr).Scan(&weekProgress.CompletedKM, &weekProgress.CompletedRuns)
	
	if err != nil {
		return nil, fmt.Errorf("fetch completed activities: %w", err)
	}
	
	// Missed runs
	err = r.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM workouts
		WHERE user_id = $1
		  AND scheduled_date >= $2
		  AND scheduled_date < NOW()
		  AND status = 'planned'
	`, userID, weekStartStr).Scan(&weekProgress.MissedRuns)
	
	if err != nil {
		return nil, fmt.Errorf("fetch missed runs: %w", err)
	}
	
	wm.WeekProgress = &weekProgress
	
	// 5. Detect active flags (LLD Section 8.1 comments)
	// Flag: "recovery_week" - TSB > 10
	if wm.TSB > 10 {
		wm.ActiveFlags = append(wm.ActiveFlags, "recovery_week")
	}
	
	// Flag: "overreaching" - TSB < -30
	if wm.TSB < -30 {
		wm.ActiveFlags = append(wm.ActiveFlags, "overreaching")
	}
	
	// Flag: "missing_runs" - 2+ missed runs this week
	if weekProgress.MissedRuns >= 2 {
		wm.ActiveFlags = append(wm.ActiveFlags, "missing_runs")
	}
	
	// Flag: "low_readiness" - readiness < 5
	if wm.ReadinessScore < 5 {
		wm.ActiveFlags = append(wm.ActiveFlags, "low_readiness")
	}
	
	// Flag: "peak_fitness" - CTL at highest level (requires historical check)
	var maxCTL float64
	err = r.Pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(ctl), 0)
		FROM fitness_snapshots
		WHERE user_id = $1
		  AND snapshot_date >= NOW() - interval '90 days'
	`, userID).Scan(&maxCTL)
	
	if err == nil && wm.CTL > 0 && wm.CTL >= maxCTL*0.98 {
		wm.ActiveFlags = append(wm.ActiveFlags, "peak_fitness")
	}
	
	// Store in Redis with 25h TTL
	if err := r.SetWorkingMemory(ctx, wm); err != nil {
		return nil, fmt.Errorf("store working memory: %w", err)
	}
	
	slog.Info("working memory rebuilt", 
		"user_id", userID,
		"ctl", wm.CTL,
		"tsb", wm.TSB,
		"flags", wm.ActiveFlags,
	)
	
	return wm, nil
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	if err := r.Client.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}
	slog.Info("redis connection closed")
	return nil
}
