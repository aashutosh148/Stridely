package models

import (
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
	ID                 uuid.UUID        `json:"id"`
	Email              string           `json:"email"`
	FirstName          NullString       `json:"first_name,omitempty"`
	LastName           NullString       `json:"last_name,omitempty"`
	ProfilePictureURL  NullString       `json:"profile_picture_url,omitempty"`
	Bio                NullString       `json:"bio,omitempty"`
	City               NullString       `json:"city,omitempty"`
	State              NullString       `json:"state,omitempty"`
	RunnerTier         RunnerTier       `json:"runner_tier"`
	SubscriptionTier   SubscriptionTier `json:"subscription_tier"`
	GoalTimeS          NullInt32        `json:"goal_time_s,omitempty"`
	TargetRaceDate     NullTime         `json:"target_race_date,omitempty"`
	ThresholdPaceS     NullFloat64      `json:"threshold_pace_s,omitempty"`
	ThresholdHR        NullInt32        `json:"threshold_hr,omitempty"`
	MaxHR              NullInt32        `json:"max_hr,omitempty"`
	WeightKg           NullFloat64      `json:"weight_kg,omitempty"`
	OnboardedAt        NullTime         `json:"onboarded_at,omitempty"`
	StravaAthleteID    NullString       `json:"strava_athlete_id,omitempty"`
	GarminUserID       NullString       `json:"garmin_user_id,omitempty"`
	PreferredLanguage  string           `json:"preferred_language"`
	NotificationPrefs  map[string]any   `json:"notification_prefs"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}
