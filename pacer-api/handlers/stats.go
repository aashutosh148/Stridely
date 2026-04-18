package handlers

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
)

type StatsHandler struct {
	db *db.Postgres
}

func NewStatsHandler(database *db.Postgres) *StatsHandler {
	return &StatsHandler{db: database}
}

func (h *StatsHandler) Overview(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	injuryRisk := h.computeInjuryRisk(c, uid)
	zoneCompliance := h.zoneCompliance(c, uid)
	monthlyMileage := h.monthlyMileage(c, uid)
	economy := h.runningEconomy(c, uid)
	facts := h.semanticFacts(c, uid)

	return c.JSON(fiber.Map{
		"injury_risk_score": injuryRisk,
		"zone_compliance":   zoneCompliance,
		"monthly_mileage":   monthlyMileage,
		"running_economy":   economy,
		"semantic_facts":    facts,
		"generated_at":      time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *StatsHandler) computeInjuryRisk(c *fiber.Ctx, userID uuid.UUID) float64 {
	var tsb sql.NullFloat64
	_ = h.db.Pool.QueryRow(c.Context(), `
		SELECT tsb
		FROM fitness_snapshots
		WHERE user_id = $1
		ORDER BY snapshot_date DESC
		LIMIT 1
	`, userID).Scan(&tsb)

	var acwr sql.NullFloat64
	_ = h.db.Pool.QueryRow(c.Context(), `
		WITH recent AS (
			SELECT COALESCE(SUM(tss), 0) AS load_7
			FROM activities
			WHERE user_id = $1 AND activity_date >= CURRENT_DATE - INTERVAL '7 day'
		), baseline AS (
			SELECT COALESCE(SUM(tss), 0) / 4.0 AS load_28
			FROM activities
			WHERE user_id = $1 AND activity_date >= CURRENT_DATE - INTERVAL '28 day'
		)
		SELECT CASE WHEN baseline.load_28 > 0 THEN recent.load_7 / baseline.load_28 ELSE 1 END
		FROM recent, baseline
	`, userID).Scan(&acwr)

	risk := 45.0
	if acwr.Valid {
		risk += (acwr.Float64 - 1.0) * 25
	}
	if tsb.Valid {
		if tsb.Float64 < -15 {
			risk += 20
		} else if tsb.Float64 < -8 {
			risk += 10
		} else if tsb.Float64 > 8 {
			risk -= 6
		}
	}

	if risk < 0 {
		risk = 0
	}
	if risk > 100 {
		risk = 100
	}
	return risk
}

func (h *StatsHandler) zoneCompliance(c *fiber.Ctx, userID uuid.UUID) []fiber.Map {
	rows, err := h.db.Pool.Query(c.Context(), `
		WITH w AS (
			SELECT DATE_TRUNC('week', activity_date)::date AS week_start,
			       AVG((zone_distribution->>'z1_pct')::float) AS z1,
			       AVG((zone_distribution->>'z2_pct')::float) AS z2,
			       AVG((zone_distribution->>'z3_pct')::float) AS z3,
			       AVG((zone_distribution->>'z4_pct')::float) AS z4,
			       AVG((zone_distribution->>'z5_pct')::float) AS z5
			FROM activities
			WHERE user_id = $1
			  AND activity_date >= CURRENT_DATE - INTERVAL '8 week'
			  AND zone_distribution IS NOT NULL
			GROUP BY 1
		)
		SELECT week_start, COALESCE(z1,0), COALESCE(z2,0), COALESCE(z3,0), COALESCE(z4,0), COALESCE(z5,0)
		FROM w
		ORDER BY week_start ASC
	`, userID)
	if err != nil {
		return []fiber.Map{}
	}
	defer rows.Close()

	out := []fiber.Map{}
	for rows.Next() {
		var week time.Time
		var z1, z2, z3, z4, z5 float64
		if err := rows.Scan(&week, &z1, &z2, &z3, &z4, &z5); err != nil {
			continue
		}
		out = append(out, fiber.Map{
			"week_start": week.Format("2006-01-02"),
			"actual": fiber.Map{"z1": z1, "z2": z2, "z3": z3, "z4": z4, "z5": z5},
			"target": fiber.Map{"z1": 15, "z2": 65, "z3": 10, "z4": 7, "z5": 3},
		})
	}
	return out
}

func (h *StatsHandler) monthlyMileage(c *fiber.Ctx, userID uuid.UUID) []fiber.Map {
	rows, err := h.db.Pool.Query(c.Context(), `
		SELECT DATE_TRUNC('month', activity_date)::date AS month_start,
		       COALESCE(SUM(distance_m), 0) / 1000.0 AS mileage_km
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= CURRENT_DATE - INTERVAL '6 month'
		GROUP BY 1
		ORDER BY month_start ASC
	`, userID)
	if err != nil {
		return []fiber.Map{}
	}
	defer rows.Close()

	out := []fiber.Map{}
	for rows.Next() {
		var month time.Time
		var km float64
		if err := rows.Scan(&month, &km); err != nil {
			continue
		}
		out = append(out, fiber.Map{"month": month.Format("Jan"), "mileage_km": km})
	}
	return out
}

func (h *StatsHandler) runningEconomy(c *fiber.Ctx, userID uuid.UUID) []fiber.Map {
	rows, err := h.db.Pool.Query(c.Context(), `
		SELECT DATE_TRUNC('week', activity_date)::date AS week_start,
		       AVG(garmin_cadence_spm),
		       AVG(garmin_gct_ms)
		FROM activities
		WHERE user_id = $1
		  AND activity_date >= CURRENT_DATE - INTERVAL '8 week'
		GROUP BY 1
		ORDER BY week_start ASC
	`, userID)
	if err != nil {
		return []fiber.Map{}
	}
	defer rows.Close()

	out := []fiber.Map{}
	for rows.Next() {
		var week time.Time
		var cadence sql.NullFloat64
		var gct sql.NullFloat64
		if err := rows.Scan(&week, &cadence, &gct); err != nil {
			continue
		}
		out = append(out, fiber.Map{
			"week_start": week.Format("2006-01-02"),
			"cadence":    nullFloat(cadence),
			"gct":        nullFloat(gct),
		})
	}
	return out
}

func (h *StatsHandler) semanticFacts(c *fiber.Ctx, userID uuid.UUID) []fiber.Map {
	rows, err := h.db.Pool.Query(c.Context(), `
		SELECT fact_key, notes, confidence
		FROM semantic_facts
		WHERE user_id = $1
		ORDER BY confidence DESC, last_updated DESC
		LIMIT 10
	`, userID)
	if err != nil {
		return []fiber.Map{}
	}
	defer rows.Close()

	out := []fiber.Map{}
	for rows.Next() {
		var key string
		var notes sql.NullString
		var confidence float64
		if err := rows.Scan(&key, &notes, &confidence); err != nil {
			continue
		}
		out = append(out, fiber.Map{
			"fact_key":    key,
			"notes":       stringOr(notes, key),
			"confidence":  confidence,
		})
	}
	return out
}

func nullFloat(v sql.NullFloat64) float64 {
	if v.Valid {
		return v.Float64
	}
	return 0
}
