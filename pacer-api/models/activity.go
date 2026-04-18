package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type WorkoutType string

const (
	WorkoutTypeEasy         WorkoutType = "easy"
	WorkoutTypeLong         WorkoutType = "long"
	WorkoutTypeTempo        WorkoutType = "tempo"
	WorkoutTypeInterval     WorkoutType = "interval"
	WorkoutTypeRace         WorkoutType = "race"
	WorkoutTypeRecovery     WorkoutType = "recovery"
	WorkoutTypeUnstructured WorkoutType = "unstructured"
)

type ZoneDistribution struct {
	Z1Pct float64 `json:"z1_pct"`
	Z2Pct float64 `json:"z2_pct"`
	Z3Pct float64 `json:"z3_pct"`
	Z4Pct float64 `json:"z4_pct"`
	Z5Pct float64 `json:"z5_pct"`
}

type Split struct {
	KM         int     `json:"km"`
	PaceS      float64 `json:"pace_s"`
	HR         int     `json:"hr,omitempty"`
	ElevDelta  float64 `json:"elev_delta,omitempty"`
}

type Activity struct {
	ID                    uuid.UUID         `json:"id"`
	UserID                uuid.UUID         `json:"user_id"`
	StravaID              string            `json:"strava_id"`
	ActivityDate          time.Time         `json:"activity_date"`
	WorkoutType           WorkoutType       `json:"workout_type"`
	DistanceM             int               `json:"distance_m"`
	DurationS             int               `json:"duration_s"`
	ElevationGainM        float64           `json:"elevation_gain_m"`
	AvgPaceS              sql.NullFloat64   `json:"avg_pace_s,omitempty"`
	AvgHR                 sql.NullInt32     `json:"avg_hr,omitempty"`
	MaxHR                 sql.NullInt32     `json:"max_hr,omitempty"`
	TSS                   sql.NullFloat64   `json:"tss,omitempty"`
	IntensityFactor       sql.NullFloat64   `json:"intensity_factor,omitempty"`
	ZoneDistribution      *ZoneDistribution `json:"zone_distribution,omitempty"`
	CardiacDecouplingPct  sql.NullFloat64   `json:"cardiac_decoupling_pct,omitempty"`
	GarminCadenceSPM      sql.NullInt32     `json:"garmin_cadence_spm,omitempty"`
	GarminGCTMs           sql.NullInt32     `json:"garmin_gct_ms,omitempty"`
	GarminVertOscCm       sql.NullFloat64   `json:"garmin_vert_osc_cm,omitempty"`
	GarminLRBalancePct    sql.NullFloat64   `json:"garmin_lr_balance_pct,omitempty"`
	GarminTrainingLoad    sql.NullFloat64   `json:"garmin_training_load,omitempty"`
	RPEReported           sql.NullInt32     `json:"rpe_reported,omitempty"`
	MatchedWorkoutID      uuid.NullUUID     `json:"matched_workout_id,omitempty"`
	AdherenceScore        sql.NullFloat64   `json:"adherence_score,omitempty"`
	SplitsKm              []Split           `json:"splits_km,omitempty"`
	StreamsS3Key          sql.NullString    `json:"streams_s3_key,omitempty"`
	GearID                sql.NullString    `json:"gear_id,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
}
