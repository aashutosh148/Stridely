package handlers

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourname/pacer-api/db"
)

type ReadinessHandler struct {
	db *db.Postgres
}

func NewReadinessHandler(database *db.Postgres) *ReadinessHandler {
	return &ReadinessHandler{db: database}
}

func (h *ReadinessHandler) Today(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	var score sql.NullInt32
	var level sql.NullString
	var note sql.NullString
	_ = h.db.Pool.QueryRow(c.Context(), `
		SELECT readiness_score, readiness_level, readiness_note
		FROM daily_health
		WHERE user_id = $1 AND health_date = CURRENT_DATE
		LIMIT 1
	`, uid).Scan(&score, &level, &note)

	type workout struct {
		ID          uuid.UUID       `json:"id"`
		WorkoutType string          `json:"workout_type"`
		DistanceKM  sql.NullFloat64 `json:"distance_km,omitempty"`
		DurationMin sql.NullInt32   `json:"duration_min,omitempty"`
		Description sql.NullString  `json:"description,omitempty"`
		Purpose     sql.NullString  `json:"purpose,omitempty"`
		Status      string          `json:"status"`
	}

	var w workout
	err = h.db.Pool.QueryRow(c.Context(), `
		SELECT id, workout_type, distance_km, duration_min, description, purpose, status
		FROM workouts
		WHERE user_id = $1 AND scheduled_date = CURRENT_DATE
		ORDER BY created_at DESC
		LIMIT 1
	`, uid).Scan(&w.ID, &w.WorkoutType, &w.DistanceKM, &w.DurationMin, &w.Description, &w.Purpose, &w.Status)
	if err == sql.ErrNoRows {
		return c.JSON(fiber.Map{
			"date":            time.Now().UTC().Format("2006-01-02"),
			"score":           int32Or(score, 6),
			"level":           stringOr(level, "amber"),
			"note":            stringOr(note, "No readiness note for today."),
			"planned_workout": nil,
			"adjusted_workout": nil,
			"factors": fiber.Map{
				"hrv_status": "unknown",
				"sleep_hours": 0,
			},
		})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	planned := fiber.Map{
		"id":           w.ID,
		"workout_type": w.WorkoutType,
		"distance_km":  float64Or(w.DistanceKM, 0),
		"duration_min": int32OrZero(w.DurationMin),
		"description":  stringOr(w.Description, ""),
		"purpose":      stringOr(w.Purpose, ""),
		"status":       w.Status,
	}

	var adjusted any = nil
	if w.Status == "modified" {
		adjusted = planned
	}

	return c.JSON(fiber.Map{
		"date":             time.Now().UTC().Format("2006-01-02"),
		"score":            int32Or(score, 6),
		"level":            stringOr(level, "amber"),
		"note":             stringOr(note, "Readiness generated."),
		"planned_workout":  planned,
		"adjusted_workout": adjusted,
		"factors": fiber.Map{
			"hrv_status": "balanced",
			"sleep_hours": 7.2,
		},
	})
}

func int32Or(v sql.NullInt32, fallback int32) int32 {
	if v.Valid {
		return v.Int32
	}
	return fallback
}

func int32OrZero(v sql.NullInt32) int32 {
	if v.Valid {
		return v.Int32
	}
	return 0
}

func float64Or(v sql.NullFloat64, fallback float64) float64 {
	if v.Valid {
		return v.Float64
	}
	return fallback
}

func stringOr(v sql.NullString, fallback string) string {
	if v.Valid {
		return v.String
	}
	return fallback
}
