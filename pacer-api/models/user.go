package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type RunnerTier string

const (
	RunnerTierBeginner     RunnerTier = "beginner"
	RunnerTierRecreational RunnerTier = "recreational"
	RunnerTierCompetitive  RunnerTier = "competitive"
	RunnerTierSerious      RunnerTier = "serious"
)

type SubscriptionTier string

const (
	SubscriptionTierFree SubscriptionTier = "free"
	SubscriptionTierCore SubscriptionTier = "core"
	SubscriptionTierPro  SubscriptionTier = "pro"
)

type User struct {
	ID                 uuid.UUID         `json:"id"`
	Email              string            `json:"email"`
	RunnerTier         RunnerTier        `json:"runner_tier"`
	SubscriptionTier   SubscriptionTier  `json:"subscription_tier"`
	GoalTimeS          sql.NullInt32     `json:"goal_time_s,omitempty"`
	TargetRaceDate     sql.NullTime      `json:"target_race_date,omitempty"`
	ThresholdPaceS     sql.NullFloat64   `json:"threshold_pace_s,omitempty"`
	ThresholdHR        sql.NullInt32     `json:"threshold_hr,omitempty"`
	MaxHR              sql.NullInt32     `json:"max_hr,omitempty"`
	WeightKg           sql.NullFloat64   `json:"weight_kg,omitempty"`
	OnboardedAt        sql.NullTime      `json:"onboarded_at,omitempty"`
	StravaAthleteID    sql.NullString    `json:"strava_athlete_id,omitempty"`
	GarminUserID       sql.NullString    `json:"garmin_user_id,omitempty"`
	PreferredLanguage  string            `json:"preferred_language"`
	NotificationPrefs  map[string]any    `json:"notification_prefs"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}
