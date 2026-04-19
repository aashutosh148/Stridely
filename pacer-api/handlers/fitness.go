package handlers

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/aashutosh148/Stridely/pacer-api/services"
)

type FitnessHandler struct {
	analysis *services.AnalysisService
}

func NewFitnessHandler(analysis *services.AnalysisService) *FitnessHandler {
	return &FitnessHandler{analysis: analysis}
}

func (h *FitnessHandler) GetMetrics(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	ctl, atl, tsb, err := h.analysis.GetLatestFitnessMetrics(c.Context(), uid)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		slog.Error("failed to fetch fitness metrics", "error", err, "user_id", userID)
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch fitness metrics"})
	}
	
	// If no fitness metrics found, use zeros
	if errors.Is(err, pgx.ErrNoRows) {
		ctl, atl, tsb = 0, 0, 0
	}

	state := "productive_training"
	raceReadiness := false
	switch {
	case tsb > 15:
		state = "fresh"
		raceReadiness = true
	case tsb > 5:
		state = "optimal_race_form"
		raceReadiness = true
	case tsb < -20:
		state = "very_tired"
	case tsb < -10:
		state = "tired"
	}

	// Get zone distribution for this week
	zoneDistThisWeek, err := h.analysis.GetZoneDistributionThisWeek(c.Context(), uid)
	if err != nil {
		slog.Error("failed to fetch zone distribution", "error", err, "user_id", userID)
		// Return empty zone distribution on error
		zoneDistThisWeek = map[string]float64{
			"z1_pct": 0,
			"z2_pct": 0,
			"z3_pct": 0,
			"z4_pct": 0,
			"z5_pct": 0,
		}
	}

	slog.Info("zone distribution fetched", "user_id", userID, "zones", zoneDistThisWeek)

	return c.JSON(fiber.Map{
		"ctl":                         ctl,
		"atl":                         atl,
		"tsb":                         tsb,
		"state":                       state,
		"race_readiness":              raceReadiness,
		"zone_distribution_this_week": zoneDistThisWeek,
	})
}

func (h *FitnessHandler) EstimateThreshold(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	if err := h.analysis.EstimateThreshold(c.Context(), uid); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	var threshold float64
	err = h.analysis.GetDB().Pool.QueryRow(c.Context(), `
    SELECT threshold_pace_s
    FROM users
    WHERE id = $1
  `, uid).Scan(&threshold)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read threshold"})
	}

	minutes := int(threshold) / 60
	seconds := int(threshold) % 60

	return c.JSON(fiber.Map{
		"threshold_pace_s": threshold,
		"threshold_pace_min_km": fiber.Map{
			"minutes": minutes,
			"seconds": seconds,
		},
		"message": "Threshold pace estimated and updated successfully",
	})
}
