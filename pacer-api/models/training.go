package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type PlanPhase string

const (
	PhaseBase  PlanPhase = "base"
	PhaseBuild PlanPhase = "build"
	PhasePeak  PlanPhase = "peak"
	PhaseTaper PlanPhase = "taper"
)

type TrainingBlock struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	Phase      PlanPhase       `json:"phase"`
	BlockStart time.Time       `json:"block_start"`
	BlockEnd   time.Time       `json:"block_end"`
	TargetRace sql.NullTime    `json:"target_race,omitempty"`
	GoalTimeS  sql.NullInt32   `json:"goal_time_s,omitempty"`
	PeakCTL    sql.NullFloat64 `json:"peak_ctl,omitempty"`
	IsActive   bool            `json:"is_active"`
	CreatedAt  time.Time       `json:"created_at"`
}

type Workout struct {
	ID                  uuid.UUID       `json:"id"`
	BlockID             uuid.UUID       `json:"block_id"`
	UserID              uuid.UUID       `json:"user_id"`
	ScheduledDate       time.Time       `json:"scheduled_date"`
	WorkoutType         WorkoutType     `json:"workout_type"`
	DistanceKM          sql.NullFloat64 `json:"distance_km,omitempty"`
	DurationMin         sql.NullInt32   `json:"duration_min,omitempty"`
	PaceTargetMin       sql.NullFloat64 `json:"pace_target_min,omitempty"`
	PaceTargetMax       sql.NullFloat64 `json:"pace_target_max,omitempty"`
	HRZone              sql.NullInt32   `json:"hr_zone,omitempty"`
	RPETarget           sql.NullInt32   `json:"rpe_target,omitempty"`
	Description         sql.NullString  `json:"description,omitempty"`
	Purpose             sql.NullString  `json:"purpose,omitempty"`
	Status              string          `json:"status"`
	CompletedActivityID *uuid.UUID      `json:"completed_activity_id,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
}

// Planning types used by both services and tools

type BlockParams struct {
	RaceDate        time.Time
	CurrentCTL      float64
	CurrentWeeklyKM float64
	ThresholdPace   float64 // s/km
	RunnerTier      RunnerTier
	GoalTimeS       int
	AvailableDays   int
}

type WeekPlan struct {
	WeekNumber int
	StartDate  time.Time
	TotalKM    float64
	Workouts   []WorkoutPlan
}

type WorkoutPlan struct {
	DayOfWeek   int
	Type        WorkoutType
	DistanceKM  float64
	DurationMin int
	PaceMin     float64 // seconds/km
	PaceMax     float64 // seconds/km
	HRZone      int
	RPETarget   int
	Description string
	Purpose     string
}
