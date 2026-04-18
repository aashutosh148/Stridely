package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourname/pacer-api/services"
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
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(fiber.Map{
				"ctl":            0,
				"atl":            0,
				"tsb":            0,
				"state":          "productive_training",
				"race_readiness": false,
			})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch fitness metrics"})
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

	return c.JSON(fiber.Map{
		"ctl":            ctl,
		"atl":            atl,
		"tsb":            tsb,
		"state":          state,
		"race_readiness": raceReadiness,
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
